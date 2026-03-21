package jsonrpc_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// newTransportPair creates a Transport that writes to a buffer for easy inspection.
func newTransportPair(t *testing.T) (*jsonrpc.Transport, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	tr := jsonrpc.NewTransport(strings.NewReader(""), &buf, jsonrpc.DefaultBufferSize)
	return tr, &buf
}

func TestSendResponse_RoundTrip(t *testing.T) {
	tr, buf := newTransportPair(t)

	type payload struct {
		Value string `json:"value"`
	}
	if err := tr.SendResponse(42, payload{Value: "hello"}); err != nil {
		t.Fatalf("SendResponse: %v", err)
	}

	// The output should be a single NDJSON line.
	line := strings.TrimRight(buf.String(), "\n")
	var resp protocol.Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != 42 {
		t.Errorf("ID = %d, want 42", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Error = %v, want nil", resp.Error)
	}

	var got payload
	if err := json.Unmarshal(resp.Result, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got.Value != "hello" {
		t.Errorf("Value = %q, want hello", got.Value)
	}
}

func TestSendError_RoundTrip(t *testing.T) {
	tr, buf := newTransportPair(t)

	if err := tr.SendError(7, protocol.ErrCodeMethodNotFound, "unknown method"); err != nil {
		t.Fatalf("SendError: %v", err)
	}

	line := strings.TrimRight(buf.String(), "\n")
	var resp protocol.Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != 7 {
		t.Errorf("ID = %d, want 7", resp.ID)
	}
	if resp.Error == nil {
		t.Fatal("Error is nil, want error")
	}
	if resp.Error.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("Code = %d, want %d", resp.Error.Code, protocol.ErrCodeMethodNotFound)
	}
	if resp.Error.Message != "unknown method" {
		t.Errorf("Message = %q, want %q", resp.Error.Message, "unknown method")
	}
}

func TestSendNotification_RoundTrip(t *testing.T) {
	tr, buf := newTransportPair(t)

	type params struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	if err := tr.SendNotification("log", params{Level: "info", Message: "started"}); err != nil {
		t.Fatalf("SendNotification: %v", err)
	}

	line := strings.TrimRight(buf.String(), "\n")
	var notif protocol.Notification
	if err := json.Unmarshal([]byte(line), &notif); err != nil {
		t.Fatalf("unmarshal notification: %v", err)
	}
	if notif.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", notif.JSONRPC)
	}
	if notif.Method != "log" {
		t.Errorf("Method = %q, want log", notif.Method)
	}

	var got params
	if err := json.Unmarshal(notif.Params, &got); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if got.Level != "info" || got.Message != "started" {
		t.Errorf("params = %+v, want {info started}", got)
	}
}

func TestReadRequest_EmptyLines(t *testing.T) {
	// Blank lines before a valid request should be skipped.
	input := "\n\n\n" + mustMarshalRequest(t, 1, "initialize", nil)
	tr := jsonrpc.NewTransport(strings.NewReader(input), io.Discard, jsonrpc.DefaultBufferSize)

	req, err := tr.ReadRequest()
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if req.Method != "initialize" {
		t.Errorf("Method = %q, want initialize", req.Method)
	}
}

func TestReadRequest_EOF(t *testing.T) {
	tr := jsonrpc.NewTransport(strings.NewReader(""), io.Discard, jsonrpc.DefaultBufferSize)

	_, err := tr.ReadRequest()
	if !errors.Is(err, io.EOF) {
		t.Errorf("err = %v, want io.EOF", err)
	}
}

func TestReadRequest_InvalidJSON(t *testing.T) {
	tr := jsonrpc.NewTransport(strings.NewReader("not-json\n"), io.Discard, jsonrpc.DefaultBufferSize)

	_, err := tr.ReadRequest()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestConcurrentWrites(t *testing.T) {
	var buf syncBuffer
	tr := jsonrpc.NewTransport(strings.NewReader(""), &buf, jsonrpc.DefaultBufferSize)

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			_ = tr.SendNotification("log", map[string]any{
				"level":   "info",
				"message": fmt.Sprintf("message-%d", n),
			})
		}(i)
	}
	wg.Wait()

	// Each line must be valid JSON — no interleaving.
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != goroutines {
		t.Fatalf("got %d lines, want %d", len(lines), goroutines)
	}
	for i, line := range lines {
		var notif protocol.Notification
		if err := json.Unmarshal([]byte(line), &notif); err != nil {
			t.Errorf("line %d is invalid JSON: %v\n%s", i, err, line)
		}
	}
}

// syncBuffer is a bytes.Buffer safe for concurrent use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *syncBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.String()
}

// mustMarshalRequest returns a newline-terminated JSON-RPC request line.
func mustMarshalRequest(t *testing.T, id int64, method string, params any) string {
	t.Helper()
	var rawParams json.RawMessage
	if params != nil {
		var err error
		rawParams, err = json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal params: %v", err)
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
		t.Fatalf("marshal request: %v", err)
	}
	return string(data) + "\n"
}
