package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// ProviderExtension is the interface that an LLM provider extension implements.
// Provider extensions bridge an LLM API (Anthropic, OpenAI, Ollama, etc.) to
// kova's agent loop.
type ProviderExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should set up its API client and return registrations
	// declaring the provider(s) it offers.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleCreateMessage handles a non-streaming message creation request.
	// The extension should call its LLM API and return the complete response.
	HandleCreateMessage(ctx context.Context, params protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error)

	// HandleStreamMessage handles a streaming message creation request. The
	// extension should call its LLM API with streaming enabled, emit
	// stream_chunk and stream_end notifications via the Emitter, and then
	// return the final accumulated result.
	HandleStreamMessage(ctx context.Context, params protocol.ProviderCreateMessageParams) (*protocol.ProviderCreateMessageResult, error)

	// HandleCapabilities returns the provider's capabilities for the given
	// model (or default capabilities if no model is specified).
	HandleCapabilities(ctx context.Context, params protocol.ProviderCapabilitiesParams) (*protocol.ProviderCapabilitiesResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunProvider starts the JSON-RPC event loop for a provider extension. It
// handles the initialize/shutdown lifecycle and dispatches provider-specific
// methods to the provided implementation. The buffer size defaults to 16 MB
// to accommodate large conversation histories. This function blocks until the
// host closes stdin, sends "shutdown", or an OS signal is received.
func RunProvider(ext ProviderExtension, opts ...Option) error {
	// Default to large buffer for provider extensions.
	opts = append([]Option{WithBufferSize(jsonrpc.LargeBufferSize)}, opts...)

	ctx, cancel, transport, emitter := startRun(opts)
	defer cancel()

	d := &dispatcher{
		transport: transport,
		emitter:   emitter,
		onInitialize: func(params protocol.InitializeParams) (*protocol.Registrations, error) {
			return ext.Initialize(emitter, params.Config, params.ExtensionRoot)
		},
		onMethod: func(ctx context.Context, req *protocol.Request) error {
			return dispatchProvider(ctx, transport, ext, req)
		},
		onShutdown: ext.Shutdown,
	}

	return d.run(ctx)
}

func dispatchProvider(ctx context.Context, t *jsonrpc.Transport, ext ProviderExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodProviderCreateMessage:
		var params protocol.ProviderCreateMessageParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleCreateMessage(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeProviderFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodProviderStreamMessage:
		var params protocol.ProviderCreateMessageParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleStreamMessage(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeStreamFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodProviderCapabilities:
		var params protocol.ProviderCapabilitiesParams
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
			}
		}
		result, err := ext.HandleCapabilities(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeProviderFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
