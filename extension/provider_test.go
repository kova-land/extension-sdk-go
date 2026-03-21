package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeProviderExt is a minimal ProviderExtension for testing.
type fakeProviderExt struct {
	providers     []protocol.ProviderDef
	createResult  *protocol.ProviderCreateMessageResult
	createErr     error
	streamResult  *protocol.ProviderCreateMessageResult
	streamErr     error
	capsResult    *protocol.ProviderCapabilitiesResult
	capsErr       error
	emitter       *extension.Emitter
	streamChunks  []protocol.StreamChunkParams
	shutdownErr   error
	shutdownCalls int
}

func (f *fakeProviderExt) Initialize(emitter *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	f.emitter = emitter
	return &protocol.Registrations{Providers: f.providers}, nil
}

func (f *fakeProviderExt) HandleCreateMessage(_ context.Context, _ protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &protocol.ProviderCreateMessageResult{
		Content:    []protocol.ProviderContentBlock{{Type: "text", Text: "hello"}},
		StopReason: "end_turn",
	}, nil
}

func (f *fakeProviderExt) HandleStreamMessage(_ context.Context, params protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error) {
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	// Emit stream chunks if configured.
	for _, chunk := range f.streamChunks {
		if err := f.emitter.EmitStreamChunk(chunk); err != nil {
			return nil, err
		}
	}
	if err := f.emitter.EmitStreamEnd(protocol.StreamEndParams{
		StreamID:    params.StreamID,
		TotalChunks: len(f.streamChunks),
	}); err != nil {
		return nil, err
	}
	if f.streamResult != nil {
		return f.streamResult, nil
	}
	return &protocol.ProviderCreateMessageResult{
		Content:    []protocol.ProviderContentBlock{{Type: "text", Text: "streamed"}},
		StopReason: "end_turn",
	}, nil
}

func (f *fakeProviderExt) HandleCapabilities(_ context.Context, _ protocol.ProviderCapabilitiesParams) (*protocol.ProviderCapabilitiesResult, error) {
	if f.capsErr != nil {
		return nil, f.capsErr
	}
	if f.capsResult != nil {
		return f.capsResult, nil
	}
	return &protocol.ProviderCapabilitiesResult{
		Models:            []string{"test-model"},
		SupportsStreaming: true,
	}, nil
}

func (f *fakeProviderExt) Shutdown() error {
	f.shutdownCalls++
	return f.shutdownErr
}

// runProviderExt starts RunProvider in the background and returns a FakeKova wired to it.
func runProviderExt(t *testing.T, ext extension.ProviderExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunProvider(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunProvider_CreateMessage(t *testing.T) {
	ext := &fakeProviderExt{
		providers: []protocol.ProviderDef{{Name: "test-provider", Description: "test"}},
		createResult: &protocol.ProviderCreateMessageResult{
			Content:      []protocol.ProviderContentBlock{{Type: "text", Text: "response text"}},
			StopReason:   "end_turn",
			InputTokens:  100,
			OutputTokens: 50,
		},
	}
	fk := runProviderExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.Providers) != 1 {
		t.Fatalf("got %d providers, want 1", len(regs.Providers))
	}
	if regs.Providers[0].Name != "test-provider" {
		t.Errorf("provider name = %q, want test-provider", regs.Providers[0].Name)
	}

	result := fk.Call(protocol.MethodProviderCreateMessage, protocol.ProviderCreateMessageParams{
		Model:     "test-model",
		MaxTokens: 1024,
		Messages: []protocol.ProviderMessage{
			{Role: "user", Content: []protocol.ProviderContentBlock{{Type: "text", Text: "hello"}}},
		},
	})

	msg := harness.MustUnmarshal[protocol.ProviderCreateMessageResult](t, result)
	if len(msg.Content) != 1 || msg.Content[0].Text != "response text" {
		t.Errorf("unexpected content: %+v", msg.Content)
	}
	if msg.StopReason != "end_turn" {
		t.Errorf("stop_reason = %q, want end_turn", msg.StopReason)
	}
	if msg.InputTokens != 100 {
		t.Errorf("input_tokens = %d, want 100", msg.InputTokens)
	}

	fk.Shutdown()
}

func TestRunProvider_StreamMessage(t *testing.T) {
	ext := &fakeProviderExt{
		streamChunks: []protocol.StreamChunkParams{
			{StreamID: "s1", Type: "text_delta", Text: "hello ", Index: 0},
			{StreamID: "s1", Type: "text_delta", Text: "world", Index: 1},
		},
		streamResult: &protocol.ProviderCreateMessageResult{
			Content:    []protocol.ProviderContentBlock{{Type: "text", Text: "hello world"}},
			StopReason: "end_turn",
		},
	}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	result := fk.Call(protocol.MethodProviderStreamMessage, protocol.ProviderCreateMessageParams{
		Model:     "test-model",
		MaxTokens: 1024,
		StreamID:  "s1",
		Messages: []protocol.ProviderMessage{
			{Role: "user", Content: []protocol.ProviderContentBlock{{Type: "text", Text: "hi"}}},
		},
	})

	msg := harness.MustUnmarshal[protocol.ProviderCreateMessageResult](t, result)
	if msg.Content[0].Text != "hello world" {
		t.Errorf("text = %q, want %q", msg.Content[0].Text, "hello world")
	}

	// Verify stream notifications were received.
	chunks := fk.ReceivedNotifications(protocol.NotificationStreamChunk)
	if len(chunks) != 2 {
		t.Fatalf("got %d stream_chunk notifications, want 2", len(chunks))
	}
	ends := fk.ReceivedNotifications(protocol.NotificationStreamEnd)
	if len(ends) != 1 {
		t.Fatalf("got %d stream_end notifications, want 1", len(ends))
	}
	endParams := harness.MustUnmarshal[protocol.StreamEndParams](t, ends[0])
	if endParams.TotalChunks != 2 {
		t.Errorf("total_chunks = %d, want 2", endParams.TotalChunks)
	}

	fk.Shutdown()
}

func TestRunProvider_Capabilities(t *testing.T) {
	ext := &fakeProviderExt{
		capsResult: &protocol.ProviderCapabilitiesResult{
			Models:            []string{"model-a", "model-b"},
			SupportsStreaming: true,
			SupportsVision:    true,
			MaxTokens:         4096,
			ContextWindow:     200000,
		},
	}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	result := fk.Call(protocol.MethodProviderCapabilities, protocol.ProviderCapabilitiesParams{
		Model: "model-a",
	})

	caps := harness.MustUnmarshal[protocol.ProviderCapabilitiesResult](t, result)
	if len(caps.Models) != 2 {
		t.Fatalf("got %d models, want 2", len(caps.Models))
	}
	if !caps.SupportsStreaming {
		t.Error("SupportsStreaming = false, want true")
	}
	if !caps.SupportsVision {
		t.Error("SupportsVision = false, want true")
	}
	if caps.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", caps.MaxTokens)
	}

	fk.Shutdown()
}

func TestRunProvider_CapabilitiesEmptyParams(t *testing.T) {
	ext := &fakeProviderExt{}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	// Call with nil params to exercise the nil-params branch.
	result := fk.Call(protocol.MethodProviderCapabilities, nil)
	caps := harness.MustUnmarshal[protocol.ProviderCapabilitiesResult](t, result)
	if !caps.SupportsStreaming {
		t.Error("default SupportsStreaming = false, want true")
	}

	fk.Shutdown()
}

func TestRunProvider_CreateMessageError(t *testing.T) {
	ext := &fakeProviderExt{
		createErr: errors.New("API rate limited"),
	}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodProviderCreateMessage, protocol.ProviderCreateMessageParams{
		Model:     "test-model",
		MaxTokens: 1024,
		Messages:  []protocol.ProviderMessage{{Role: "user", Content: []protocol.ProviderContentBlock{{Type: "text", Text: "hi"}}}},
	})
	if rpcErr.Code != protocol.ErrCodeProviderFailed {
		t.Errorf("code = %d, want %d (ErrCodeProviderFailed)", rpcErr.Code, protocol.ErrCodeProviderFailed)
	}
	if rpcErr.Message != "API rate limited" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "API rate limited")
	}

	fk.Shutdown()
}

func TestRunProvider_StreamMessageError(t *testing.T) {
	ext := &fakeProviderExt{
		streamErr: errors.New("stream interrupted"),
	}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodProviderStreamMessage, protocol.ProviderCreateMessageParams{
		Model:     "test-model",
		MaxTokens: 1024,
		StreamID:  "s1",
		Messages:  []protocol.ProviderMessage{{Role: "user", Content: []protocol.ProviderContentBlock{{Type: "text", Text: "hi"}}}},
	})
	if rpcErr.Code != protocol.ErrCodeStreamFailed {
		t.Errorf("code = %d, want %d (ErrCodeStreamFailed)", rpcErr.Code, protocol.ErrCodeStreamFailed)
	}

	fk.Shutdown()
}

func TestRunProvider_CapabilitiesError(t *testing.T) {
	ext := &fakeProviderExt{
		capsErr: errors.New("model not found"),
	}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodProviderCapabilities, protocol.ProviderCapabilitiesParams{
		Model: "nonexistent",
	})
	if rpcErr.Code != protocol.ErrCodeProviderFailed {
		t.Errorf("code = %d, want %d (ErrCodeProviderFailed)", rpcErr.Code, protocol.ErrCodeProviderFailed)
	}

	fk.Shutdown()
}

func TestRunProvider_UnknownMethod(t *testing.T) {
	ext := &fakeProviderExt{}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("provider_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRunProvider_EmitStreamError(t *testing.T) {
	// Verify the EmitStreamError helper works through the transport.
	ext := &streamErrorProviderExt{}
	fk := runProviderExt(t, ext)
	fk.Initialize(nil)

	// The extension will emit a stream_error notification as a side effect
	// of handling stream_message, then return an error.
	rpcErr := fk.CallExpectError(protocol.MethodProviderStreamMessage, protocol.ProviderCreateMessageParams{
		Model:     "test-model",
		MaxTokens: 1024,
		StreamID:  "s1",
		Messages:  []protocol.ProviderMessage{{Role: "user", Content: []protocol.ProviderContentBlock{{Type: "text", Text: "hi"}}}},
	})
	if rpcErr.Code != protocol.ErrCodeStreamFailed {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeStreamFailed)
	}

	streamErrors := fk.ReceivedNotifications(protocol.NotificationStreamError)
	if len(streamErrors) != 1 {
		t.Fatalf("got %d stream_error notifications, want 1", len(streamErrors))
	}
	errParams := harness.MustUnmarshal[protocol.StreamErrorParams](t, streamErrors[0])
	if errParams.StreamID != "s1" {
		t.Errorf("stream_id = %q, want s1", errParams.StreamID)
	}
	if errParams.Code != protocol.ErrCodeProviderFailed {
		t.Errorf("error code = %d, want %d", errParams.Code, protocol.ErrCodeProviderFailed)
	}

	fk.Shutdown()
}

// streamErrorProviderExt emits a stream_error notification then returns an error.
type streamErrorProviderExt struct {
	emitter *extension.Emitter
}

func (f *streamErrorProviderExt) Initialize(emitter *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	f.emitter = emitter
	return &protocol.Registrations{}, nil
}

func (f *streamErrorProviderExt) HandleCreateMessage(_ context.Context, _ protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error) {
	return &protocol.ProviderCreateMessageResult{}, nil
}

func (f *streamErrorProviderExt) HandleStreamMessage(_ context.Context, params protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error) {
	_ = f.emitter.EmitStreamError(protocol.StreamErrorParams{
		StreamID: params.StreamID,
		Code:     protocol.ErrCodeProviderFailed,
		Message:  "upstream failure",
	})
	return nil, errors.New("upstream failure")
}

func (f *streamErrorProviderExt) HandleCapabilities(_ context.Context, _ protocol.ProviderCapabilitiesParams) (*protocol.ProviderCapabilitiesResult, error) {
	return &protocol.ProviderCapabilitiesResult{}, nil
}

func (f *streamErrorProviderExt) Shutdown() error { return nil }
