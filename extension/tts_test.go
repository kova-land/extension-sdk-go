package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeTTSExt is a minimal TTSExtension for testing.
type fakeTTSExt struct {
	ttsProviders   []protocol.TTSProviderDef
	synthResult    *protocol.TTSSynthesizeResult
	synthErr       error
	voicesResult   *protocol.TTSVoicesResult
	voicesErr      error
	shutdownCalled bool
}

func (f *fakeTTSExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{TTSProviders: f.ttsProviders}, nil
}

func (f *fakeTTSExt) HandleSynthesize(_ context.Context, _ protocol.TTSSynthesizeParams) (*protocol.TTSSynthesizeResult, error) {
	if f.synthErr != nil {
		return nil, f.synthErr
	}
	if f.synthResult != nil {
		return f.synthResult, nil
	}
	return &protocol.TTSSynthesizeResult{
		AudioBase64: "dGVzdA==",
		Format:      "opus",
		ContentType: "audio/opus",
	}, nil
}

func (f *fakeTTSExt) HandleVoices(_ context.Context) (*protocol.TTSVoicesResult, error) {
	if f.voicesErr != nil {
		return nil, f.voicesErr
	}
	if f.voicesResult != nil {
		return f.voicesResult, nil
	}
	return &protocol.TTSVoicesResult{
		Voices: []protocol.TTSVoice{{ID: "v1", Name: "Default"}},
	}, nil
}

func (f *fakeTTSExt) Shutdown() error {
	f.shutdownCalled = true
	return nil
}

// runTTSExt starts RunTTS in the background and returns a FakeKova wired to it.
func runTTSExt(t *testing.T, ext extension.TTSExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunTTS(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunTTS_FullLifecycle(t *testing.T) {
	ext := &fakeTTSExt{
		ttsProviders: []protocol.TTSProviderDef{{Name: "test-tts", Description: "test TTS"}},
		synthResult: &protocol.TTSSynthesizeResult{
			AudioBase64: "YXVkaW8=",
			Format:      "mp3",
			ContentType: "audio/mpeg",
		},
		voicesResult: &protocol.TTSVoicesResult{
			Voices: []protocol.TTSVoice{
				{ID: "alloy", Name: "Alloy", Language: "en"},
				{ID: "echo", Name: "Echo", Language: "en"},
			},
		},
	}
	fk := runTTSExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.TTSProviders) != 1 {
		t.Fatalf("got %d TTS providers, want 1", len(regs.TTSProviders))
	}
	if regs.TTSProviders[0].Name != "test-tts" {
		t.Errorf("TTS provider name = %q, want test-tts", regs.TTSProviders[0].Name)
	}

	// Test synthesize.
	result := fk.Call(protocol.MethodTTSSynthesize, protocol.TTSSynthesizeParams{
		Text:  "Hello world",
		Voice: "alloy",
		Model: "tts-1",
	})
	synthResult := harness.MustUnmarshal[protocol.TTSSynthesizeResult](t, result)
	if synthResult.AudioBase64 != "YXVkaW8=" {
		t.Errorf("audio_base64 = %q, want YXVkaW8=", synthResult.AudioBase64)
	}
	if synthResult.Format != "mp3" {
		t.Errorf("format = %q, want mp3", synthResult.Format)
	}
	if synthResult.ContentType != "audio/mpeg" {
		t.Errorf("content_type = %q, want audio/mpeg", synthResult.ContentType)
	}

	// Test voices.
	voicesRaw := fk.Call(protocol.MethodTTSVoices, nil)
	voicesResult := harness.MustUnmarshal[protocol.TTSVoicesResult](t, voicesRaw)
	if len(voicesResult.Voices) != 2 {
		t.Fatalf("got %d voices, want 2", len(voicesResult.Voices))
	}
	if voicesResult.Voices[0].ID != "alloy" {
		t.Errorf("voice[0].id = %q, want alloy", voicesResult.Voices[0].ID)
	}

	fk.Shutdown()
}

func TestRunTTS_SynthesizeError(t *testing.T) {
	ext := &fakeTTSExt{
		synthErr: errors.New("synthesis failed"),
	}
	fk := runTTSExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodTTSSynthesize, protocol.TTSSynthesizeParams{
		Text:  "test",
		Voice: "alloy",
	})
	if rpcErr.Code != protocol.ErrCodeTTSFailed {
		t.Errorf("code = %d, want %d (ErrCodeTTSFailed)", rpcErr.Code, protocol.ErrCodeTTSFailed)
	}
	if rpcErr.Message != "synthesis failed" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "synthesis failed")
	}

	fk.Shutdown()
}

func TestRunTTS_VoicesError(t *testing.T) {
	ext := &fakeTTSExt{
		voicesErr: errors.New("voices unavailable"),
	}
	fk := runTTSExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodTTSVoices, nil)
	if rpcErr.Code != protocol.ErrCodeTTSFailed {
		t.Errorf("code = %d, want %d (ErrCodeTTSFailed)", rpcErr.Code, protocol.ErrCodeTTSFailed)
	}

	fk.Shutdown()
}

func TestRunTTS_UnknownMethod(t *testing.T) {
	ext := &fakeTTSExt{}
	fk := runTTSExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("tts_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}
