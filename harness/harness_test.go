package harness_test

import (
	"encoding/json"
	"testing"

	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

func TestMustUnmarshal_Success(t *testing.T) {
	data := json.RawMessage(`{"output":"hello","is_error":false}`)
	result := harness.MustUnmarshal[protocol.ToolCallResult](t, data)
	if result.Output != "hello" {
		t.Errorf("Output = %q, want hello", result.Output)
	}
	if result.IsError {
		t.Error("IsError = true, want false")
	}
}

func TestMustUnmarshal_Complex(t *testing.T) {
	data := json.RawMessage(`{"capabilities":{"supports_text":true,"max_message_len":4096}}`)
	result := harness.MustUnmarshal[protocol.ChannelCapabilitiesResult](t, data)
	if !result.Capabilities.SupportsText {
		t.Error("SupportsText = false, want true")
	}
	if result.Capabilities.MaxMessageLen != 4096 {
		t.Errorf("MaxMessageLen = %d, want 4096", result.Capabilities.MaxMessageLen)
	}
}

func TestFormatError_Nil(t *testing.T) {
	got := harness.FormatError(nil)
	if got != "<nil>" {
		t.Errorf("FormatError(nil) = %q, want <nil>", got)
	}
}

func TestFormatError_WithError(t *testing.T) {
	err := &protocol.RPCError{Code: -32601, Message: "unknown method"}
	got := harness.FormatError(err)
	want := `code=-32601 message="unknown method"`
	if got != want {
		t.Errorf("FormatError = %q, want %q", got, want)
	}
}
