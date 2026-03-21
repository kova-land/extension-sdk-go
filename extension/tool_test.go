package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeToolExt is a minimal ToolExtension for testing.
type fakeToolExt struct {
	tools      []protocol.ToolDef
	callErr    error
	callResult *protocol.ToolCallResult
}

func (f *fakeToolExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{Tools: f.tools}, nil
}

func (f *fakeToolExt) HandleToolCall(_ context.Context, _ protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	if f.callErr != nil {
		return nil, f.callErr
	}
	if f.callResult != nil {
		return f.callResult, nil
	}
	return &protocol.ToolCallResult{Output: "ok"}, nil
}

func (f *fakeToolExt) Shutdown() error { return nil }

// runToolExt starts RunTool in the background and returns a FakeKova wired to it.
func runToolExt(t *testing.T, ext extension.ToolExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunTool(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunTool_FullLifecycle(t *testing.T) {
	tool := protocol.ToolDef{
		Name:        "echo",
		Description: "echoes input",
		InputSchema: map[string]any{"type": "object"},
	}
	ext := &fakeToolExt{
		tools:      []protocol.ToolDef{tool},
		callResult: &protocol.ToolCallResult{Output: "hello"},
	}

	fk := runToolExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.Tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(regs.Tools))
	}
	if regs.Tools[0].Name != "echo" {
		t.Errorf("tool name = %q, want echo", regs.Tools[0].Name)
	}

	result := fk.Call(protocol.MethodToolCall, protocol.ToolCallParams{
		Name:  "echo",
		Input: map[string]any{"text": "hello"},
	})

	toolResult := harness.MustUnmarshal[protocol.ToolCallResult](t, result)
	if toolResult.Output != "hello" {
		t.Errorf("output = %q, want hello", toolResult.Output)
	}
	if toolResult.IsError {
		t.Error("IsError = true, want false")
	}

	fk.Shutdown()
}

func TestRunTool_UnknownMethod(t *testing.T) {
	ext := &fakeToolExt{}
	fk := runToolExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("unknown_method", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRunTool_ToolError(t *testing.T) {
	ext := &fakeToolExt{
		callErr: errors.New("tool exploded"),
	}
	fk := runToolExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodToolCall, protocol.ToolCallParams{
		Name:  "boom",
		Input: map[string]any{},
	})
	if rpcErr.Code != protocol.ErrCodeToolFailed {
		t.Errorf("code = %d, want %d (ErrCodeToolFailed)", rpcErr.Code, protocol.ErrCodeToolFailed)
	}
	if rpcErr.Message != "tool exploded" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "tool exploded")
	}

	fk.Shutdown()
}
