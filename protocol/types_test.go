package protocol_test

import (
	"encoding/json"
	"testing"

	"github.com/kova-land/extension-sdk-go/protocol"
)

func TestRequest_JSON_RoundTrip(t *testing.T) {
	params := json.RawMessage(`{"key":"value"}`)
	req := protocol.Request{
		JSONRPC: "2.0",
		Method:  "tool_call",
		Params:  params,
		ID:      99,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got protocol.Request
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", got.JSONRPC)
	}
	if got.Method != "tool_call" {
		t.Errorf("Method = %q, want tool_call", got.Method)
	}
	if got.ID != 99 {
		t.Errorf("ID = %d, want 99", got.ID)
	}
	if string(got.Params) != `{"key":"value"}` {
		t.Errorf("Params = %s, want {\"key\":\"value\"}", string(got.Params))
	}
}

func TestRPCError_Error(t *testing.T) {
	e := &protocol.RPCError{
		Code:    protocol.ErrCodeMethodNotFound,
		Message: "unknown method: foo",
	}

	if e.Error() != "unknown method: foo" {
		t.Errorf("Error() = %q, want %q", e.Error(), "unknown method: foo")
	}
}

func TestRegistrations_Omitempty(t *testing.T) {
	// Empty Registrations should omit all slice fields.
	regs := protocol.Registrations{}
	data, err := json.Marshal(regs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Unmarshal into a raw map to verify omitempty behavior.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}

	omitFields := []string{"tools", "http_routes", "hooks", "services", "providers", "tts_providers", "cli_commands"}
	for _, field := range omitFields {
		if _, present := m[field]; present {
			t.Errorf("field %q should be omitted when empty, but was present in JSON", field)
		}
	}
}
