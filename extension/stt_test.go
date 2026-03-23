package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeSTTExt is a minimal STTExtension for testing.
type fakeSTTExt struct {
	sttProviders     []protocol.STTProviderDef
	transcribeResult *protocol.STTTranscribeResult
	transcribeErr    error
	modelsResult     *protocol.STTModelsResult
	modelsErr        error
	shutdownCalled   bool
}

func (f *fakeSTTExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{STTProviders: f.sttProviders}, nil
}

func (f *fakeSTTExt) HandleTranscribe(_ context.Context, _ protocol.STTTranscribeParams) (*protocol.STTTranscribeResult, error) {
	if f.transcribeErr != nil {
		return nil, f.transcribeErr
	}
	if f.transcribeResult != nil {
		return f.transcribeResult, nil
	}
	return &protocol.STTTranscribeResult{
		Text:       "hello world",
		Language:   "en",
		Confidence: 0.95,
	}, nil
}

func (f *fakeSTTExt) HandleModels(_ context.Context) (*protocol.STTModelsResult, error) {
	if f.modelsErr != nil {
		return nil, f.modelsErr
	}
	if f.modelsResult != nil {
		return f.modelsResult, nil
	}
	return &protocol.STTModelsResult{
		Models: []protocol.STTModel{{ID: "whisper-1", Name: "Whisper"}},
	}, nil
}

func (f *fakeSTTExt) Shutdown() error {
	f.shutdownCalled = true
	return nil
}

// runSTTExt starts RunSTT in the background and returns a FakeKova wired to it.
func runSTTExt(t *testing.T, ext extension.STTExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunSTT(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunSTT_FullLifecycle(t *testing.T) {
	ext := &fakeSTTExt{
		sttProviders: []protocol.STTProviderDef{{Name: "test-stt", Description: "test STT"}},
		transcribeResult: &protocol.STTTranscribeResult{
			Text:       "hello world",
			Language:   "en",
			Confidence: 0.98,
			Segments: []protocol.STTSegment{
				{Start: 0.0, End: 0.5, Text: "hello"},
				{Start: 0.5, End: 1.0, Text: "world"},
			},
		},
		modelsResult: &protocol.STTModelsResult{
			Models: []protocol.STTModel{
				{ID: "whisper-1", Name: "Whisper", Language: "en"},
				{ID: "large-v3", Name: "Large V3", Language: "multilingual"},
			},
		},
	}
	fk := runSTTExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.STTProviders) != 1 {
		t.Fatalf("got %d STT providers, want 1", len(regs.STTProviders))
	}
	if regs.STTProviders[0].Name != "test-stt" {
		t.Errorf("STT provider name = %q, want test-stt", regs.STTProviders[0].Name)
	}

	// Test transcribe.
	result := fk.Call(protocol.MethodSTTTranscribe, protocol.STTTranscribeParams{
		AudioBase64: "dGVzdA==",
		Format:      "opus",
		Language:    "en",
	})
	transcribeResult := harness.MustUnmarshal[protocol.STTTranscribeResult](t, result)
	if transcribeResult.Text != "hello world" {
		t.Errorf("text = %q, want hello world", transcribeResult.Text)
	}
	if transcribeResult.Language != "en" {
		t.Errorf("language = %q, want en", transcribeResult.Language)
	}
	if transcribeResult.Confidence != 0.98 {
		t.Errorf("confidence = %f, want 0.98", transcribeResult.Confidence)
	}
	if len(transcribeResult.Segments) != 2 {
		t.Fatalf("got %d segments, want 2", len(transcribeResult.Segments))
	}
	if transcribeResult.Segments[0].Text != "hello" {
		t.Errorf("segment[0].text = %q, want hello", transcribeResult.Segments[0].Text)
	}

	// Test models.
	modelsRaw := fk.Call(protocol.MethodSTTModels, nil)
	modelsResult := harness.MustUnmarshal[protocol.STTModelsResult](t, modelsRaw)
	if len(modelsResult.Models) != 2 {
		t.Fatalf("got %d models, want 2", len(modelsResult.Models))
	}
	if modelsResult.Models[0].ID != "whisper-1" {
		t.Errorf("model[0].id = %q, want whisper-1", modelsResult.Models[0].ID)
	}

	fk.Shutdown()
}

func TestRunSTT_TranscribeError(t *testing.T) {
	ext := &fakeSTTExt{
		transcribeErr: errors.New("transcription failed"),
	}
	fk := runSTTExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodSTTTranscribe, protocol.STTTranscribeParams{
		AudioBase64: "dGVzdA==",
		Format:      "wav",
	})
	if rpcErr.Code != protocol.ErrCodeSTTFailed {
		t.Errorf("code = %d, want %d (ErrCodeSTTFailed)", rpcErr.Code, protocol.ErrCodeSTTFailed)
	}
	if rpcErr.Message != "transcription failed" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "transcription failed")
	}

	fk.Shutdown()
}

func TestRunSTT_ModelsError(t *testing.T) {
	ext := &fakeSTTExt{
		modelsErr: errors.New("models unavailable"),
	}
	fk := runSTTExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodSTTModels, nil)
	if rpcErr.Code != protocol.ErrCodeSTTFailed {
		t.Errorf("code = %d, want %d (ErrCodeSTTFailed)", rpcErr.Code, protocol.ErrCodeSTTFailed)
	}

	fk.Shutdown()
}

func TestRunSTT_UnknownMethod(t *testing.T) {
	ext := &fakeSTTExt{}
	fk := runSTTExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("stt_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}
