package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// runComposite starts Run with the given options in the background and returns a FakeKova.
func runComposite(t *testing.T, runOpts []extension.RunOption) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.Run(runOpts,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRun_ProviderAndTTS(t *testing.T) {
	provider := &compositeProviderExt{
		providers: []protocol.ProviderDef{{Name: "openai", Description: "OpenAI"}},
		createResult: &protocol.ProviderCreateMessageResult{
			Content: []protocol.ProviderContentBlock{{Type: "text", Text: "hello"}},
		},
	}
	tts := &compositeTTSExt{
		providers: []protocol.TTSProviderDef{{Name: "openai-tts", Description: "OpenAI TTS"}},
		synthResult: &protocol.TTSSynthesizeResult{
			AudioBase64: "dGVzdA==",
			Format:      "mp3",
		},
	}

	fk := runComposite(t, []extension.RunOption{
		extension.WithProvider(provider),
		extension.WithTTS(tts),
	})

	// Initialize should merge registrations from both handlers.
	regs := fk.Initialize(nil)
	if len(regs.Providers) != 1 {
		t.Fatalf("got %d providers, want 1", len(regs.Providers))
	}
	if regs.Providers[0].Name != "openai" {
		t.Errorf("provider name = %q, want openai", regs.Providers[0].Name)
	}
	if len(regs.TTSProviders) != 1 {
		t.Fatalf("got %d tts providers, want 1", len(regs.TTSProviders))
	}
	if regs.TTSProviders[0].Name != "openai-tts" {
		t.Errorf("tts provider name = %q, want openai-tts", regs.TTSProviders[0].Name)
	}

	// Provider method should work.
	result := fk.Call(protocol.MethodProviderCreateMessage, protocol.ProviderCreateMessageParams{
		Model: "gpt-4",
	})
	providerResult := harness.MustUnmarshal[protocol.ProviderCreateMessageResult](t, result)
	if len(providerResult.Content) != 1 || providerResult.Content[0].Text != "hello" {
		t.Errorf("unexpected provider result: %+v", providerResult)
	}

	// TTS method should work.
	result = fk.Call(protocol.MethodTTSSynthesize, protocol.TTSSynthesizeParams{
		Text: "hello world",
	})
	ttsResult := harness.MustUnmarshal[protocol.TTSSynthesizeResult](t, result)
	if ttsResult.AudioBase64 != "dGVzdA==" {
		t.Errorf("audio = %q, want dGVzdA==", ttsResult.AudioBase64)
	}

	fk.Shutdown()
}

func TestRun_ToolAndHook(t *testing.T) {
	tool := &fakeToolExt{
		tools:      []protocol.ToolDef{{Name: "echo", Description: "echoes", InputSchema: map[string]any{"type": "object"}}},
		callResult: &protocol.ToolCallResult{Output: "ok"},
	}
	hook := &compositeHookExt{
		hooks:  []protocol.HookDef{{Events: []string{"PreToolUse"}}},
		result: &protocol.HookEventResult{Action: "continue"},
	}

	fk := runComposite(t, []extension.RunOption{
		extension.WithTool(tool),
		extension.WithHook(hook),
	})

	regs := fk.Initialize(nil)
	if len(regs.Tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(regs.Tools))
	}
	if len(regs.Hooks) != 1 {
		t.Fatalf("got %d hooks, want 1", len(regs.Hooks))
	}

	// Tool call should route to tool handler.
	result := fk.Call(protocol.MethodToolCall, protocol.ToolCallParams{
		Name: "echo", Input: map[string]any{},
	})
	toolResult := harness.MustUnmarshal[protocol.ToolCallResult](t, result)
	if toolResult.Output != "ok" {
		t.Errorf("output = %q, want ok", toolResult.Output)
	}

	// Hook event should route to hook handler.
	result = fk.Call(protocol.MethodHookEvent, protocol.HookEventParams{
		Event: "PreToolUse",
	})
	hookResult := harness.MustUnmarshal[protocol.HookEventResult](t, result)
	if hookResult.Action != "continue" {
		t.Errorf("action = %q, want continue", hookResult.Action)
	}

	fk.Shutdown()
}

func TestRun_UnregisteredMethodReturnsError(t *testing.T) {
	// Only register a tool handler, then try calling a provider method.
	tool := &fakeToolExt{
		tools: []protocol.ToolDef{{Name: "echo", Description: "echoes", InputSchema: map[string]any{"type": "object"}}},
	}

	fk := runComposite(t, []extension.RunOption{
		extension.WithTool(tool),
	})

	fk.Initialize(nil)

	// Provider method should fail since no provider is registered.
	rpcErr := fk.CallExpectError(protocol.MethodProviderCreateMessage, protocol.ProviderCreateMessageParams{
		Model: "gpt-4",
	})
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRun_ShutdownCallsAllHandlers(t *testing.T) {
	providerShutdown := false
	ttsShutdown := false

	provider := &compositeProviderExt{
		providers:    []protocol.ProviderDef{{Name: "p1"}},
		shutdownFunc: func() error { providerShutdown = true; return nil },
	}
	tts := &compositeTTSExt{
		providers:    []protocol.TTSProviderDef{{Name: "t1"}},
		shutdownFunc: func() error { ttsShutdown = true; return nil },
	}

	fk := runComposite(t, []extension.RunOption{
		extension.WithProvider(provider),
		extension.WithTTS(tts),
	})

	fk.Initialize(nil)
	fk.Shutdown()

	if !providerShutdown {
		t.Error("provider shutdown not called")
	}
	if !ttsShutdown {
		t.Error("tts shutdown not called")
	}
}

func TestRun_InitErrorStopsEarly(t *testing.T) {
	provider := &compositeProviderExt{
		initErr: errors.New("provider broken"),
	}
	tts := &compositeTTSExt{
		providers: []protocol.TTSProviderDef{{Name: "t1"}},
	}

	fk := runComposite(t, []extension.RunOption{
		extension.WithProvider(provider),
		extension.WithTTS(tts),
	})

	rpcErr := fk.CallExpectError(protocol.MethodInitialize, protocol.InitializeParams{
		Config:        map[string]any{},
		ExtensionRoot: t.TempDir(),
	})
	if rpcErr.Code != protocol.ErrCodeNotReady {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeNotReady)
	}

	fk.Shutdown()
}

// --- Fake provider extension for composite tests ---

type compositeProviderExt struct {
	providers    []protocol.ProviderDef
	createResult *protocol.ProviderCreateMessageResult
	initErr      error
	shutdownFunc func() error
}

func (f *compositeProviderExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}
	return &protocol.Registrations{Providers: f.providers}, nil
}

func (f *compositeProviderExt) HandleCreateMessage(_ context.Context, _ protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error) {
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &protocol.ProviderCreateMessageResult{}, nil
}

func (f *compositeProviderExt) HandleStreamMessage(_ context.Context, _ protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error) {
	return &protocol.ProviderCreateMessageResult{}, nil
}

func (f *compositeProviderExt) HandleCapabilities(_ context.Context, _ protocol.ProviderCapabilitiesParams) (*protocol.ProviderCapabilitiesResult, error) {
	return &protocol.ProviderCapabilitiesResult{}, nil
}

func (f *compositeProviderExt) Shutdown() error {
	if f.shutdownFunc != nil {
		return f.shutdownFunc()
	}
	return nil
}

// --- Fake TTS extension for composite tests ---

type compositeTTSExt struct {
	providers    []protocol.TTSProviderDef
	synthResult  *protocol.TTSSynthesizeResult
	shutdownFunc func() error
}

func (f *compositeTTSExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{TTSProviders: f.providers}, nil
}

func (f *compositeTTSExt) HandleSynthesize(_ context.Context, _ protocol.TTSSynthesizeParams) (*protocol.TTSSynthesizeResult, error) {
	if f.synthResult != nil {
		return f.synthResult, nil
	}
	return &protocol.TTSSynthesizeResult{}, nil
}

func (f *compositeTTSExt) HandleVoices(_ context.Context) (*protocol.TTSVoicesResult, error) {
	return &protocol.TTSVoicesResult{}, nil
}

func (f *compositeTTSExt) Shutdown() error {
	if f.shutdownFunc != nil {
		return f.shutdownFunc()
	}
	return nil
}

// --- Fake hook extension for composite tests ---

type compositeHookExt struct {
	hooks  []protocol.HookDef
	result *protocol.HookEventResult
}

func (f *compositeHookExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{Hooks: f.hooks}, nil
}

func (f *compositeHookExt) HandleHookEvent(_ context.Context, _ protocol.HookEventParams) (*protocol.HookEventResult, error) {
	if f.result != nil {
		return f.result, nil
	}
	return &protocol.HookEventResult{Action: "continue"}, nil
}

func (f *compositeHookExt) Shutdown() error { return nil }
