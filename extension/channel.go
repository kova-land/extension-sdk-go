package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// ChannelExtension is the interface that a channel extension implements.
// Channel extensions bridge a chat platform (Discord, Slack, Signal, etc.)
// to the kova agent gateway.
type ChannelExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should set up its platform connection and return its
	// registrations. The emitter is provided for sending notifications
	// (e.g., channel_message) back to kova.
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleChannelSend delivers an agent response message to a channel.
	// The extension should forward the message to the specified channel
	// on its platform.
	HandleChannelSend(ctx context.Context, params protocol.ChannelSendParams) error

	// HandleChannelTyping shows a typing indicator in a channel. Extensions
	// should treat this as best-effort; errors are non-fatal.
	HandleChannelTyping(ctx context.Context, params protocol.ChannelTypingParams) error

	// HandleChannelReactionAdd adds an emoji reaction to a message. Extensions
	// should treat this as best-effort; errors are non-fatal. Extensions SHOULD NOT
	// return an error if the platform does not support reactions.
	HandleChannelReactionAdd(ctx context.Context, params protocol.ChannelReactionAddParams) error

	// HandleChannelReactionRemove removes an emoji reaction from a message. Extensions
	// should treat this as best-effort; errors are non-fatal. Extensions SHOULD NOT
	// return an error if the platform does not support reactions.
	HandleChannelReactionRemove(ctx context.Context, params protocol.ChannelReactionRemoveParams) error

	// HandlePromptUser presents an interactive prompt to a user and returns
	// their response. The extension renders the prompt using platform-native
	// UI (buttons, modals, numbered menus, etc.) and blocks until the user
	// responds or the timeout expires.
	HandlePromptUser(ctx context.Context, params protocol.PromptUserParams) (*protocol.PromptUserResult, error)

	// HandleChannelCapabilities returns the platform capabilities for the
	// given channel (or global defaults if no channel is specified).
	HandleChannelCapabilities(ctx context.Context, params protocol.ChannelCapabilitiesParams) (*protocol.ChannelCapabilitiesResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	// The extension should clean up platform connections and resources.
	Shutdown() error
}

// RunChannel starts the JSON-RPC event loop for a channel extension. It handles
// the initialize/shutdown lifecycle and dispatches channel-specific methods to
// the provided implementation. This function blocks until the host closes stdin,
// sends "shutdown", or an OS signal is received.
func RunChannel(ext ChannelExtension, opts ...Option) error {
	ctx, cancel, transport, emitter := startRun(opts)
	defer cancel()

	d := &dispatcher{
		transport: transport,
		emitter:   emitter,
		onInitialize: func(params protocol.InitializeParams) (*protocol.Registrations, error) {
			return ext.Initialize(emitter, params.Config, params.ExtensionRoot)
		},
		onMethod: func(ctx context.Context, req *protocol.Request) error {
			return dispatchChannel(ctx, transport, ext, req)
		},
		onShutdown: ext.Shutdown,
	}

	return d.run(ctx)
}

func dispatchChannel(ctx context.Context, t *jsonrpc.Transport, ext ChannelExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodChannelSend:
		var params protocol.ChannelSendParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		if err := ext.HandleChannelSend(ctx, params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
		}
		return t.SendResponse(req.ID, struct{}{})

	case protocol.MethodChannelTyping:
		var params protocol.ChannelTypingParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		if err := ext.HandleChannelTyping(ctx, params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
		}
		return t.SendResponse(req.ID, struct{}{})

	case protocol.MethodChannelReactionAdd:
		var params protocol.ChannelReactionAddParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		if err := ext.HandleChannelReactionAdd(ctx, params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
		}
		return t.SendResponse(req.ID, struct{}{})

	case protocol.MethodChannelReactionRemove:
		var params protocol.ChannelReactionRemoveParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		if err := ext.HandleChannelReactionRemove(ctx, params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
		}
		return t.SendResponse(req.ID, struct{}{})

	case protocol.MethodPromptUser:
		var params protocol.PromptUserParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandlePromptUser(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
		}
		return t.SendResponse(req.ID, result)

	case protocol.MethodChannelCapabilities:
		var params protocol.ChannelCapabilitiesParams
		if req.Params != nil {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
			}
		}
		result, err := ext.HandleChannelCapabilities(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
