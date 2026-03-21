package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// ServiceExtension is the interface that a background service extension
// implements. Service extensions run long-lived background tasks with
// explicit lifecycle management (start, health check, stop) by the kova
// Manager.
type ServiceExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should return registrations declaring the service(s) it
	// provides.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleServiceStart starts a background service identified by ID. The
	// extension should begin its background work and return OK. This is
	// called by the Manager after initialization completes for each declared
	// service.
	HandleServiceStart(ctx context.Context, params protocol.ServiceStartParams) (*protocol.ServiceStartResult, error)

	// HandleServiceHealth checks the health of a running service. The
	// extension should return the current health status (healthy, degraded,
	// or unhealthy).
	HandleServiceHealth(ctx context.Context, params protocol.ServiceHealthParams) (*protocol.ServiceHealthResult, error)

	// HandleServiceStop stops a running service identified by ID. The
	// extension should clean up resources for this service. This is called
	// by the Manager during shutdown.
	HandleServiceStop(ctx context.Context, params protocol.ServiceStopParams) (*protocol.ServiceStopResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunService starts the JSON-RPC event loop for a service extension. It
// handles the initialize/shutdown lifecycle and dispatches service_start,
// service_health, and service_stop requests to the provided implementation.
// This function blocks until the host closes stdin, sends "shutdown", or an
// OS signal is received.
//
// This is a convenience wrapper around [Run] with [WithService].
func RunService(ext ServiceExtension, opts ...Option) error {
	return Run([]RunOption{WithService(ext)}, opts...)
}

func dispatchService(ctx context.Context, t *jsonrpc.Transport, ext ServiceExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodServiceStart:
		var params protocol.ServiceStartParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleServiceStart(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeServiceFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodServiceHealth:
		var params protocol.ServiceHealthParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleServiceHealth(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeServiceFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodServiceStop:
		var params protocol.ServiceStopParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleServiceStop(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeServiceFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
