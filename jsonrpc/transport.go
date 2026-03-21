package jsonrpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/kova-land/extension-sdk-go/protocol"
)

// DefaultBufferSize is the default maximum line size for the NDJSON scanner (1 MB).
const DefaultBufferSize = 1 << 20

// LargeBufferSize is the recommended buffer size for provider extensions that
// handle large conversation histories (16 MB).
const LargeBufferSize = 16 << 20

// Transport provides JSON-RPC 2.0 message framing over NDJSON (newline-delimited
// JSON). It reads requests from an io.Reader and writes responses and notifications
// to an io.Writer. All write operations are mutex-protected and safe for concurrent use.
type Transport struct {
	scanner *bufio.Scanner
	writer  io.Writer
	mu      sync.Mutex
}

// NewTransport creates a Transport that reads from r and writes to w.
// The scanner buffer is sized to bufSize bytes. Use DefaultBufferSize for most
// extensions or LargeBufferSize for provider extensions.
func NewTransport(r io.Reader, w io.Writer, bufSize int) *Transport {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), bufSize)
	return &Transport{
		scanner: scanner,
		writer:  w,
	}
}

// ReadRequest reads the next JSON-RPC request from the input stream. It blocks
// until a complete line is available. Returns io.EOF when the input stream is
// closed.
func (t *Transport) ReadRequest() (*protocol.Request, error) {
	for {
		if !t.scanner.Scan() {
			if err := t.scanner.Err(); err != nil {
				return nil, fmt.Errorf("read: %w", err)
			}
			return nil, io.EOF
		}
		line := t.scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		return &req, nil
	}
}

// SendResponse writes a successful JSON-RPC response with the given request ID
// and result payload. The result is marshaled to JSON as the "result" field.
func (t *Transport) SendResponse(id int64, result any) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	resp := protocol.Response{
		JSONRPC: "2.0",
		Result:  data,
		ID:      id,
	}
	return t.writeLine(resp)
}

// SendError writes a JSON-RPC error response with the given request ID,
// error code, and human-readable message.
func (t *Transport) SendError(id int64, code int, message string) error {
	resp := protocol.Response{
		JSONRPC: "2.0",
		Error:   &protocol.RPCError{Code: code, Message: message},
		ID:      id,
	}
	return t.writeLine(resp)
}

// SendNotification writes a JSON-RPC notification (no ID, no response expected)
// with the given method name and params payload.
func (t *Transport) SendNotification(method string, params any) error {
	data, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal notification params: %w", err)
	}
	notif := protocol.Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  data,
	}
	return t.writeLine(notif)
}

// writeLine marshals v to JSON and writes it as a single NDJSON line.
func (t *Transport) writeLine(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, err := t.writer.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return fmt.Errorf("write newline: %w", err)
	}
	return nil
}
