package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeHookExt is a minimal HookExtension for testing.
type fakeHookExt struct {
	hooks       []protocol.HookDef
	eventResult *protocol.HookEventResult
	eventErr    error
}

func (f *fakeHookExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{Hooks: f.hooks}, nil
}

func (f *fakeHookExt) HandleHookEvent(_ context.Context, _ protocol.HookEventParams) (*protocol.HookEventResult, error) {
	if f.eventErr != nil {
		return nil, f.eventErr
	}
	if f.eventResult != nil {
		return f.eventResult, nil
	}
	return &protocol.HookEventResult{Action: "continue"}, nil
}

func (f *fakeHookExt) Shutdown() error { return nil }

// runHookExt starts RunHook in the background and returns a FakeKova wired to it.
func runHookExt(t *testing.T, ext extension.HookExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunHook(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunHook_FullLifecycle(t *testing.T) {
	ext := &fakeHookExt{
		hooks: []protocol.HookDef{{Events: []string{"UserPromptSubmit"}}},
		eventResult: &protocol.HookEventResult{
			Action:  "block",
			Message: "denied by policy",
		},
	}

	fk := runHookExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.Hooks) != 1 {
		t.Fatalf("got %d hooks, want 1", len(regs.Hooks))
	}
	if regs.Hooks[0].Events[0] != "UserPromptSubmit" {
		t.Errorf("hook event = %q, want UserPromptSubmit", regs.Hooks[0].Events[0])
	}

	result := fk.Call(protocol.MethodHookEvent, protocol.HookEventParams{
		Event: "UserPromptSubmit",
		Data:  map[string]any{"text": "hello"},
	})

	hookResult := harness.MustUnmarshal[protocol.HookEventResult](t, result)
	if hookResult.Action != "block" {
		t.Errorf("action = %q, want block", hookResult.Action)
	}
	if hookResult.Message != "denied by policy" {
		t.Errorf("message = %q, want %q", hookResult.Message, "denied by policy")
	}

	fk.Shutdown()
}

func TestRunHook_UnknownMethod(t *testing.T) {
	ext := &fakeHookExt{}
	fk := runHookExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("hook_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRunHook_EventError(t *testing.T) {
	ext := &fakeHookExt{
		eventErr: errors.New("hook exploded"),
	}
	fk := runHookExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodHookEvent, protocol.HookEventParams{
		Event: "PreToolUse",
		Data:  map[string]any{},
	})
	if rpcErr.Code != protocol.ErrCodeHookFailed {
		t.Errorf("code = %d, want %d (ErrCodeHookFailed)", rpcErr.Code, protocol.ErrCodeHookFailed)
	}
	if rpcErr.Message != "hook exploded" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "hook exploded")
	}

	fk.Shutdown()
}
