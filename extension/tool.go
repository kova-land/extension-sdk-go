package extension

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// ToolExtension is the interface that a tool extension implements. Tool
// extensions expose custom tools that the kova agent can invoke during
// conversations.
type ToolExtension interface {
	// Initialize is called when the host sends the "initialize" request.
	// The extension should return registrations declaring the tool(s) it
	// provides (names, descriptions, input schemas).
	Initialize(emitter *Emitter, config map[string]any, extensionRoot string) (*protocol.Registrations, error)

	// HandleToolCall handles a tool invocation from the agent. The extension
	// should execute the tool logic and return the output text and whether
	// the output represents an error.
	HandleToolCall(ctx context.Context, params protocol.ToolCallParams) (*protocol.ToolCallResult, error)

	// Shutdown is called when the host sends the "shutdown" request.
	Shutdown() error
}

// RunTool starts the JSON-RPC event loop for a tool extension. It handles
// the initialize/shutdown lifecycle and dispatches tool_call requests to the
// provided implementation. This function blocks until the host closes stdin,
// sends "shutdown", or an OS signal is received.
//
// This is a convenience wrapper around [Run] with [WithTool].
func RunTool(ext ToolExtension, opts ...Option) error {
	return Run([]RunOption{WithTool(ext)}, opts...)
}

func dispatchTool(ctx context.Context, t *jsonrpc.Transport, ext ToolExtension, req *protocol.Request) error {
	switch req.Method {
	case protocol.MethodToolCall:
		var params protocol.ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return t.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
		}
		result, err := ext.HandleToolCall(ctx, params)
		if err != nil {
			return t.SendError(req.ID, protocol.ErrCodeToolFailed, err.Error())
		}
		return t.SendResponse(req.ID, result)

	default:
		return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
	}
}
