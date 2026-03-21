package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// errorChannelExt returns configurable errors for each method.
type errorChannelExt struct {
	sendErr       error
	typingErr     error
	reactionErr   error
	promptErr     error
	capsErr       error
	initErr       error
	reactionCalls int
}

func (f *errorChannelExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	if f.initErr != nil {
		return nil, f.initErr
	}
	return &protocol.Registrations{}, nil
}

func (f *errorChannelExt) HandleChannelSend(_ context.Context, _ protocol.ChannelSendParams) error {
	return f.sendErr
}

func (f *errorChannelExt) HandleChannelTyping(_ context.Context, _ protocol.ChannelTypingParams) error {
	return f.typingErr
}

func (f *errorChannelExt) HandleChannelReactionAdd(_ context.Context, _ protocol.ChannelReactionAddParams) error {
	f.reactionCalls++
	return f.reactionErr
}

func (f *errorChannelExt) HandleChannelReactionRemove(_ context.Context, _ protocol.ChannelReactionRemoveParams) error {
	f.reactionCalls++
	return f.reactionErr
}

func (f *errorChannelExt) HandlePromptUser(_ context.Context, _ protocol.PromptUserParams) (*protocol.PromptUserResult, error) {
	return nil, f.promptErr
}

func (f *errorChannelExt) HandleChannelCapabilities(_ context.Context, _ protocol.ChannelCapabilitiesParams) (*protocol.ChannelCapabilitiesResult, error) {
	return nil, f.capsErr
}

func (f *errorChannelExt) Shutdown() error { return nil }

func TestRunChannel_Reactions(t *testing.T) {
	ext := &fakeChannelExt{}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	// channel_reaction_add should succeed.
	fk.Call(protocol.MethodChannelReactionAdd, protocol.ChannelReactionAddParams{
		ChannelID: "ch-1",
		MessageID: "msg-1",
		Emoji:     "thumbsup",
	})

	// channel_reaction_remove should succeed.
	fk.Call(protocol.MethodChannelReactionRemove, protocol.ChannelReactionRemoveParams{
		ChannelID: "ch-1",
		MessageID: "msg-1",
		Emoji:     "thumbsup",
	})

	fk.Shutdown()
}

func TestRunChannel_SendError(t *testing.T) {
	ext := &errorChannelExt{sendErr: errors.New("send failed")}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodChannelSend, protocol.ChannelSendParams{
		ChannelID: "ch-1",
		Text:      "test",
	})
	if rpcErr.Code != protocol.ErrCodeInternal {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeInternal)
	}
	if rpcErr.Message != "send failed" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "send failed")
	}

	fk.Shutdown()
}

func TestRunChannel_TypingError(t *testing.T) {
	ext := &errorChannelExt{typingErr: errors.New("typing failed")}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodChannelTyping, protocol.ChannelTypingParams{
		ChannelID: "ch-1",
	})
	if rpcErr.Code != protocol.ErrCodeInternal {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeInternal)
	}

	fk.Shutdown()
}

func TestRunChannel_ReactionAddError(t *testing.T) {
	ext := &errorChannelExt{reactionErr: errors.New("reaction failed")}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodChannelReactionAdd, protocol.ChannelReactionAddParams{
		ChannelID: "ch-1",
		MessageID: "msg-1",
		Emoji:     "thumbsup",
	})
	if rpcErr.Code != protocol.ErrCodeInternal {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeInternal)
	}

	fk.Shutdown()
}

func TestRunChannel_ReactionRemoveError(t *testing.T) {
	ext := &errorChannelExt{reactionErr: errors.New("reaction failed")}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodChannelReactionRemove, protocol.ChannelReactionRemoveParams{
		ChannelID: "ch-1",
		MessageID: "msg-1",
		Emoji:     "thumbsup",
	})
	if rpcErr.Code != protocol.ErrCodeInternal {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeInternal)
	}

	fk.Shutdown()
}

func TestRunChannel_PromptUserError(t *testing.T) {
	ext := &errorChannelExt{promptErr: errors.New("prompt failed")}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodPromptUser, protocol.PromptUserParams{
		ChannelID: "ch-1",
		Message:   "Confirm?",
	})
	if rpcErr.Code != protocol.ErrCodeInternal {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeInternal)
	}

	fk.Shutdown()
}

func TestRunChannel_CapabilitiesError(t *testing.T) {
	ext := &errorChannelExt{capsErr: errors.New("caps failed")}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodChannelCapabilities, protocol.ChannelCapabilitiesParams{
		ChannelID: "ch-1",
	})
	if rpcErr.Code != protocol.ErrCodeInternal {
		t.Errorf("code = %d, want %d", rpcErr.Code, protocol.ErrCodeInternal)
	}

	fk.Shutdown()
}

func TestRunChannel_InitializeError(t *testing.T) {
	ext := &errorChannelExt{initErr: errors.New("init failed")}
	fk := runChannelExt(t, ext)

	rpcErr := fk.CallExpectError(protocol.MethodInitialize, protocol.InitializeParams{
		Config:        map[string]any{},
		ExtensionRoot: t.TempDir(),
	})
	if rpcErr.Code != protocol.ErrCodeNotReady {
		t.Errorf("code = %d, want %d (ErrCodeNotReady)", rpcErr.Code, protocol.ErrCodeNotReady)
	}
	if rpcErr.Message != "channel init: init failed" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "channel init: init failed")
	}

	fk.Shutdown()
}

func TestRunChannel_UnknownMethod(t *testing.T) {
	ext := &fakeChannelExt{}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("channel_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRunChannel_EmitLog(t *testing.T) {
	// Verify the Log emitter method works through the transport.
	ext := &loggingChannelExt{}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	// Trigger a send which causes the extension to emit a log notification.
	fk.Call(protocol.MethodChannelSend, protocol.ChannelSendParams{
		ChannelID: "ch-1",
		Text:      "test",
	})

	logs := fk.ReceivedNotifications(protocol.NotificationLog)
	if len(logs) != 1 {
		t.Fatalf("got %d log notifications, want 1", len(logs))
	}
	logParams := harness.MustUnmarshal[protocol.LogParams](t, logs[0])
	if logParams.Level != "info" {
		t.Errorf("level = %q, want info", logParams.Level)
	}
	if logParams.Message != "sent message" {
		t.Errorf("message = %q, want %q", logParams.Message, "sent message")
	}

	fk.Shutdown()
}

// loggingChannelExt emits a log notification when handling channel_send.
type loggingChannelExt struct {
	emitter *extension.Emitter
}

func (f *loggingChannelExt) Initialize(emitter *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	f.emitter = emitter
	return &protocol.Registrations{}, nil
}

func (f *loggingChannelExt) HandleChannelSend(_ context.Context, _ protocol.ChannelSendParams) error {
	return f.emitter.Log("info", "sent message")
}

func (f *loggingChannelExt) HandleChannelTyping(_ context.Context, _ protocol.ChannelTypingParams) error {
	return nil
}

func (f *loggingChannelExt) HandleChannelReactionAdd(_ context.Context, _ protocol.ChannelReactionAddParams) error {
	return nil
}

func (f *loggingChannelExt) HandleChannelReactionRemove(_ context.Context, _ protocol.ChannelReactionRemoveParams) error {
	return nil
}

func (f *loggingChannelExt) HandlePromptUser(_ context.Context, _ protocol.PromptUserParams) (*protocol.PromptUserResult, error) {
	return &protocol.PromptUserResult{}, nil
}

func (f *loggingChannelExt) HandleChannelCapabilities(_ context.Context, _ protocol.ChannelCapabilitiesParams) (*protocol.ChannelCapabilitiesResult, error) {
	return &protocol.ChannelCapabilitiesResult{}, nil
}

func (f *loggingChannelExt) Shutdown() error { return nil }
