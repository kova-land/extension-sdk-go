package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// HTTPExtension is the interface that an HTTP route extension implements.
// HTTP extensions handle incoming HTTP requests that kova proxies to the
// extension based on registered route paths.
type HTTPExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should return registrations declaring the HTTP route(s)
	// it handles.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleHTTPRequest handles an HTTP request proxied by kova. The extension
	// processes the request and returns an HTTP response (status code, headers,
	// body).
	HandleHTTPRequest(ctx context.Context, params protocol.HTTPRequestParams) (*protocol.HTTPResponseResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunHTTP starts the JSON-RPC event loop for an HTTP route extension. It
// handles the initialize/shutdown lifecycle and dispatches http_request
// requests to the provided implementation. This function blocks until the
// host closes stdin, sends "shutdown", or an OS signal is received.
//
// This is a convenience wrapper around [Run] with [WithHTTP].
func RunHTTP(ext HTTPExtension, opts ...Option) error {
	return Run([]RunOption{WithHTTP(ext)}, opts...)
}

func dispatchHTTP(ctx context.Context, t *jsonrpc.Transport, ext HTTPExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodHTTPRequest:
		var params protocol.HTTPRequestParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleHTTPRequest(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeHTTPFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
