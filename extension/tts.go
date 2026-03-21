package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// TTSExtension is the interface that a TTS (text-to-speech) provider extension
// implements. TTS extensions bridge a speech synthesis API (OpenAI TTS, Piper,
// etc.) to kova's audio pipeline.
type TTSExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should set up its TTS client and return registrations
	// declaring the TTS provider(s) it offers.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleSynthesize processes a text-to-speech synthesis request and
	// returns the audio data as base64.
	HandleSynthesize(ctx context.Context, params protocol.TTSSynthesizeParams) (*protocol.TTSSynthesizeResult, error)

	// HandleVoices returns the list of available voices.
	HandleVoices(ctx context.Context) (*protocol.TTSVoicesResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunTTS starts the JSON-RPC event loop for a TTS extension. It handles
// the initialize/shutdown lifecycle and dispatches TTS-specific methods to
// the provided implementation. This function blocks until the host closes
// stdin, sends "shutdown", or an OS signal is received.
func RunTTS(ext TTSExtension, opts ...Option) error {
	ctx, cancel, transport, emitter := startRun(opts)
	defer cancel()

	d := &dispatcher{
		transport: transport,
		emitter:   emitter,
		onInitialize: func(params protocol.InitializeParams) (*protocol.Registrations, error) {
			return ext.Initialize(emitter, params.Config, params.ExtensionRoot)
		},
		onMethod: func(ctx context.Context, req *protocol.Request) error {
			return dispatchTTS(ctx, transport, ext, req)
		},
		onShutdown: ext.Shutdown,
	}

	return d.run(ctx)
}

func dispatchTTS(ctx context.Context, t *jsonrpc.Transport, ext TTSExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodTTSSynthesize:
		var params protocol.TTSSynthesizeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleSynthesize(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeTTSFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodTTSVoices:
		result, err := ext.HandleVoices(ctx)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeTTSFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
