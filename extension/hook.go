package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// HookExtension is the interface that a lifecycle hook extension implements.
// Hook extensions receive lifecycle events from the kova agent (e.g.,
// UserPromptSubmit, PreToolUse) and can gate or observe agent behavior.
type HookExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should return registrations declaring the hook event(s)
	// it listens to.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleHookEvent handles a lifecycle event from the kova agent. The
	// extension inspects the event data and returns an action ("continue"
	// or "block") to control agent behavior.
	HandleHookEvent(ctx context.Context, params protocol.HookEventParams) (*protocol.HookEventResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunHook starts the JSON-RPC event loop for a hook extension. It handles
// the initialize/shutdown lifecycle and dispatches hook_event requests to the
// provided implementation. This function blocks until the host closes stdin,
// sends "shutdown", or an OS signal is received.
func RunHook(ext HookExtension, opts ...Option) error {
	ctx, cancel, transport, emitter := startRun(opts)
	defer cancel()

	d := &dispatcher{
		transport: transport,
		emitter:   emitter,
		onInitialize: func(params protocol.InitializeParams) (*protocol.Registrations, error) {
			return ext.Initialize(emitter, params.Config, params.ExtensionRoot)
		},
		onMethod: func(ctx context.Context, req *protocol.Request) error {
			return dispatchHook(ctx, transport, ext, req)
		},
		onShutdown: ext.Shutdown,
	}

	return d.run(ctx)
}

func dispatchHook(ctx context.Context, t *jsonrpc.Transport, ext HookExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodHookEvent:
		var params protocol.HookEventParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleHookEvent(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeHookFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
