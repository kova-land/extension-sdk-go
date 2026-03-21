package harness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/kova-land/extension-sdk-go/protocol"
)

// FakeKova simulates the kova host for testing extensions. It provides methods
// to send JSON-RPC requests and collect notifications from the extension under
// test.
//
// Use NewFakeKova to create an instance, then wire its Stdin/Stdout to the
// extension's I/O (typically via the WithStdin/WithStdout options on the
// extension runner).
type FakeKova struct {
	t      testing.TB
	nextID atomic.Int64

	// toExt is the writer that feeds the extension's stdin.
	toExt io.Writer
	// fromExt is the reader that reads from the extension's stdout.
	fromExt io.Reader

	// ExtStdin is connected to toExt — pass this as the extension's stdin.
	ExtStdin io.Reader
	// ExtStdout is connected to fromExt — pass this as the extension's stdout.
	ExtStdout io.Writer

	mu            sync.Mutex
	notifications map[string][]json.RawMessage
	scanner       *bufio.Scanner
}

// NewFakeKova creates a FakeKova instance with in-memory pipes. The returned
// FakeKova's ExtStdin and ExtStdout should be passed to the extension runner
// via WithStdin and WithStdout options.
func NewFakeKova(t testing.TB) *FakeKova {
	// Pipe for host -> extension (host writes to toExt, extension reads from extStdin).
	extStdinR, toExtW := io.Pipe()
	// Pipe for extension -> host (extension writes to extStdout, host reads from fromExt).
	fromExtR, extStdoutW := io.Pipe()

	fk := &FakeKova{
		t:             t,
		toExt:         toExtW,
		fromExt:       fromExtR,
		ExtStdin:      extStdinR,
		ExtStdout:     extStdoutW,
		notifications: make(map[string][]json.RawMessage),
	}

	fk.scanner = bufio.NewScanner(fromExtR)
	fk.scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	return fk
}

// Initialize sends the "initialize" request with the given config and returns
// the extension's registrations.
func (fk *FakeKova) Initialize(config map[string]any) *protocol.Registrations {
	fk.t.Helper()
	if config == nil {
		config = map[string]any{}
	}

	result := fk.Call(protocol.MethodInitialize, protocol.InitializeParams{
		Config:        config,
		ExtensionRoot: fk.t.TempDir(),
	})

	var initResult protocol.InitializeResult
	if err := json.Unmarshal(result, &initResult); err != nil {
		fk.t.Fatalf("unmarshal initialize result: %v", err)
	}
	return &initResult.Registrations
}

// Call sends a JSON-RPC request and returns the result. It collects any
// notifications received before the response and stores them for later
// retrieval via ReceivedNotifications. Fails the test on error responses.
func (fk *FakeKova) Call(method string, params any) json.RawMessage {
	fk.t.Helper()
	return fk.callInternal(method, params)
}

// CallExpectError sends a JSON-RPC request and expects an error response.
// Returns the RPCError. Fails the test if a success response is received.
func (fk *FakeKova) CallExpectError(method string, params any) *protocol.RPCError {
	fk.t.Helper()

	id := fk.nextID.Add(1)
	fk.sendRequest(id, method, params)

	for {
		resp, notif := fk.readMessage()
		if notif != nil {
			fk.storeNotification(notif)
			continue
		}
		if resp.ID != id {
			fk.t.Fatalf("response ID mismatch: got %d, want %d", resp.ID, id)
		}
		if resp.Error == nil {
			fk.t.Fatalf("expected error response for %s, got success", method)
		}
		return resp.Error
	}
}

// Shutdown sends the "shutdown" request and closes the host-side pipes.
func (fk *FakeKova) Shutdown() {
	fk.t.Helper()
	_ = fk.Call(protocol.MethodShutdown, nil)
	// Close the write pipe so the extension sees EOF on stdin.
	if closer, ok := fk.toExt.(io.Closer); ok {
		_ = closer.Close()
	}
}

// ReceivedNotifications returns all notifications received for the given method
// since the last call to ReceivedNotifications for that method.
func (fk *FakeKova) ReceivedNotifications(method string) []json.RawMessage {
	fk.mu.Lock()
	defer fk.mu.Unlock()
	result := fk.notifications[method]
	delete(fk.notifications, method)
	return result
}

func (fk *FakeKova) callInternal(method string, params any) json.RawMessage {
	fk.t.Helper()

	id := fk.nextID.Add(1)
	fk.sendRequest(id, method, params)

	// Read messages until we get the response for our ID.
	for {
		resp, notif := fk.readMessage()
		if notif != nil {
			fk.storeNotification(notif)
			continue
		}
		if resp.ID != id {
			fk.t.Fatalf("response ID mismatch: got %d, want %d", resp.ID, id)
		}
		if resp.Error != nil {
			fk.t.Fatalf("%s returned error %d: %s", method, resp.Error.Code, resp.Error.Message)
		}
		return resp.Result
	}
}

func (fk *FakeKova) sendRequest(id int64, method string, params any) {
	fk.t.Helper()

	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			fk.t.Fatalf("marshal params: %v", err)
		}
	}

	req := protocol.Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
		ID:      id,
	}

	data, err := json.Marshal(req)
	if err != nil {
		fk.t.Fatalf("marshal request: %v", err)
	}
	data = append(data, '\n')
	if _, err := fk.toExt.Write(data); err != nil {
		fk.t.Fatalf("write request: %v", err)
	}
}

// readMessage reads the next line from the extension and classifies it as
// either a response or a notification.
func (fk *FakeKova) readMessage() (*protocol.Response, *protocol.Notification) {
	fk.t.Helper()

	for {
		if !fk.scanner.Scan() {
			err := fk.scanner.Err()
			if err != nil {
				fk.t.Fatalf("read from extension: %v", err)
			}
			fk.t.Fatal("extension closed stdout unexpectedly")
		}

		line := fk.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Peek to determine message type.
		var peek map[string]json.RawMessage
		if err := json.Unmarshal(line, &peek); err != nil {
			fk.t.Fatalf("invalid JSON from extension: %v\nline: %s", err, string(line))
		}

		// Notifications have a "method" field and no numeric "id" (or id is absent).
		if _, hasMethod := peek["method"]; hasMethod {
			var notif protocol.Notification
			if err := json.Unmarshal(line, &notif); err != nil {
				fk.t.Fatalf("unmarshal notification: %v", err)
			}
			return nil, &notif
		}

		// Otherwise it's a response.
		var resp protocol.Response
		if err := json.Unmarshal(line, &resp); err != nil {
			fk.t.Fatalf("unmarshal response: %v", err)
		}
		return &resp, nil
	}
}

func (fk *FakeKova) storeNotification(notif *protocol.Notification) {
	fk.mu.Lock()
	defer fk.mu.Unlock()
	fk.notifications[notif.Method] = append(fk.notifications[notif.Method], notif.Params)
}

// MustUnmarshal is a test helper that unmarshals JSON into the target type,
// failing the test on error.
func MustUnmarshal[T any](t testing.TB, data json.RawMessage) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal %T: %v\njson: %s", v, err, string(data))
	}
	return v
}

// FormatError formats an RPCError for display in test output.
func FormatError(err *protocol.RPCError) string {
	if err == nil {
		return "<nil>"
	}
	return fmt.Sprintf("code=%d message=%q", err.Code, err.Message)
}
