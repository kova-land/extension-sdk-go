package extension_test

import (
	"context"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeChannelExt is a minimal ChannelExtension for testing.
type fakeChannelExt struct {
	emitter      *extension.Emitter
	capabilities protocol.ChannelCapabilities
	promptResult *protocol.PromptUserResult
}

func (f *fakeChannelExt) Initialize(emitter *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	f.emitter = emitter
	return &protocol.Registrations{}, nil
}

func (f *fakeChannelExt) HandleChannelSend(_ context.Context, _ protocol.ChannelSendParams) error {
	return nil
}

func (f *fakeChannelExt) HandleChannelTyping(_ context.Context, _ protocol.ChannelTypingParams) error {
	return nil
}

func (f *fakeChannelExt) HandlePromptUser(_ context.Context, _ protocol.PromptUserParams) (*protocol.PromptUserResult, error) {
	if f.promptResult != nil {
		return f.promptResult, nil
	}
	return &protocol.PromptUserResult{SelectedOption: "yes"}, nil
}

func (f *fakeChannelExt) HandleChannelCapabilities(_ context.Context, _ protocol.ChannelCapabilitiesParams) (*protocol.ChannelCapabilitiesResult, error) {
	return &protocol.ChannelCapabilitiesResult{Capabilities: f.capabilities}, nil
}

func (f *fakeChannelExt) Shutdown() error { return nil }

// runChannelExt starts RunChannel in the background and returns a FakeKova wired to it.
func runChannelExt(t *testing.T, ext extension.ChannelExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunChannel(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunChannel_SendAndTyping(t *testing.T) {
	ext := &fakeChannelExt{}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	// channel_send should succeed.
	fk.Call(protocol.MethodChannelSend, protocol.ChannelSendParams{
		ChannelID: "ch-1",
		Text:      "hello from agent",
	})

	// channel_typing should succeed.
	fk.Call(protocol.MethodChannelTyping, protocol.ChannelTypingParams{
		ChannelID: "ch-1",
	})

	fk.Shutdown()
}

func TestRunChannel_PromptUser(t *testing.T) {
	ext := &fakeChannelExt{
		promptResult: &protocol.PromptUserResult{SelectedOption: "confirm"},
	}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	result := fk.Call(protocol.MethodPromptUser, protocol.PromptUserParams{
		ChannelID: "ch-1",
		Message:   "Confirm action?",
		Options: []protocol.PromptOption{
			{Label: "Yes", Value: "confirm"},
			{Label: "No", Value: "deny"},
		},
	})

	promptResult := harness.MustUnmarshal[protocol.PromptUserResult](t, result)
	if promptResult.SelectedOption != "confirm" {
		t.Errorf("selected_option = %q, want confirm", promptResult.SelectedOption)
	}

	fk.Shutdown()
}

func TestRunChannel_Capabilities(t *testing.T) {
	ext := &fakeChannelExt{
		capabilities: protocol.ChannelCapabilities{
			SupportsText:    true,
			SupportsImages:  true,
			SupportsButtons: true,
			MaxMessageLen:   2000,
		},
	}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	result := fk.Call(protocol.MethodChannelCapabilities, protocol.ChannelCapabilitiesParams{
		ChannelID: "ch-1",
	})

	caps := harness.MustUnmarshal[protocol.ChannelCapabilitiesResult](t, result)
	if !caps.Capabilities.SupportsText {
		t.Error("SupportsText = false, want true")
	}
	if !caps.Capabilities.SupportsImages {
		t.Error("SupportsImages = false, want true")
	}
	if !caps.Capabilities.SupportsButtons {
		t.Error("SupportsButtons = false, want true")
	}
	if caps.Capabilities.MaxMessageLen != 2000 {
		t.Errorf("MaxMessageLen = %d, want 2000", caps.Capabilities.MaxMessageLen)
	}

	fk.Shutdown()
}

func TestRunChannel_EmitChannelMessage(t *testing.T) {
	// Use a channel extension that emits a channel_message notification when
	// it handles a channel_send request, allowing us to trigger the emission
	// synchronously from the extension's handler goroutine.
	ext := &emittingChannelExt{}
	fk := runChannelExt(t, ext)
	fk.Initialize(nil)

	// Emit the notification from the extension side by triggering channel_send.
	// The extension's HandleChannelSend will call EmitChannelMessage before responding.
	// The harness will collect the notification while reading the channel_send response.
	fk.Call(protocol.MethodChannelSend, protocol.ChannelSendParams{
		ChannelID: "ch-emit",
		Text:      "trigger",
	})

	notifs := fk.ReceivedNotifications(protocol.NotificationChannelMessage)
	if len(notifs) != 1 {
		t.Fatalf("got %d channel_message notifications, want 1", len(notifs))
	}

	msg := harness.MustUnmarshal[protocol.ChannelMessageParams](t, notifs[0])
	if msg.Text != "echo: trigger" {
		t.Errorf("text = %q, want %q", msg.Text, "echo: trigger")
	}
	if msg.Platform != "test" {
		t.Errorf("platform = %q, want test", msg.Platform)
	}

	fk.Shutdown()
}

// emittingChannelExt emits a channel_message notification when it handles channel_send.
type emittingChannelExt struct {
	emitter *extension.Emitter
}

func (f *emittingChannelExt) Initialize(emitter *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	f.emitter = emitter
	return &protocol.Registrations{}, nil
}

func (f *emittingChannelExt) HandleChannelSend(_ context.Context, params protocol.ChannelSendParams) error {
	// Emit a channel_message notification as a side effect.
	return f.emitter.EmitChannelMessage(protocol.ChannelMessageParams{
		ChannelID: params.ChannelID,
		UserID:    "user-1",
		Text:      "echo: " + params.Text,
		Platform:  "test",
	})
}

func (f *emittingChannelExt) HandleChannelTyping(_ context.Context, _ protocol.ChannelTypingParams) error {
	return nil
}

func (f *emittingChannelExt) HandlePromptUser(_ context.Context, _ protocol.PromptUserParams) (*protocol.PromptUserResult, error) {
	return &protocol.PromptUserResult{SelectedOption: "ok"}, nil
}

func (f *emittingChannelExt) HandleChannelCapabilities(_ context.Context, _ protocol.ChannelCapabilitiesParams) (*protocol.ChannelCapabilitiesResult, error) {
	return &protocol.ChannelCapabilitiesResult{}, nil
}

func (f *emittingChannelExt) Shutdown() error { return nil }
