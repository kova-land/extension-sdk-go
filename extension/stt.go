package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// STTExtension is the interface that an STT (speech-to-text) provider extension
// implements. STT extensions bridge a speech recognition API (OpenAI Whisper,
// Deepgram, etc.) to kova's audio pipeline.
type STTExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should set up its STT client and return registrations
	// declaring the STT provider(s) it offers.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleTranscribe processes a speech-to-text transcription request and
	// returns the recognized text.
	HandleTranscribe(ctx context.Context, params protocol.STTTranscribeParams) (*protocol.STTTranscribeResult, error)

	// HandleModels returns the list of available STT models.
	HandleModels(ctx context.Context) (*protocol.STTModelsResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunSTT starts the JSON-RPC event loop for an STT extension. It handles
// the initialize/shutdown lifecycle and dispatches STT-specific methods to
// the provided implementation. This function blocks until the host closes
// stdin, sends "shutdown", or an OS signal is received.
//
// This is a convenience wrapper around [Run] with [WithSTT].
func RunSTT(ext STTExtension, opts ...Option) error {
	return Run([]RunOption{WithSTT(ext)}, opts...)
}

func dispatchSTT(ctx context.Context, t *jsonrpc.Transport, ext STTExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodSTTTranscribe:
		var params protocol.STTTranscribeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleTranscribe(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeSTTFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodSTTModels:
		result, err := ext.HandleModels(ctx)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeSTTFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
