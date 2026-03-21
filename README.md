# extension-sdk-go

Go SDK for building [Kova](https://github.com/kova-land/kova) extensions. Provides protocol types, a JSON-RPC transport layer, high-level extension runner interfaces, and test utilities.

## Installation

```bash
go get github.com/kova-land/extension-sdk-go
```

Core packages (`protocol`, `jsonrpc`, `extension`, `harness`) have zero external dependencies and use only the Go standard library. The `schemas` package uses `github.com/google/jsonschema-go` for schema generation.

## Package Overview

| Package | Purpose |
|---------|---------|
| `protocol` | JSON-RPC 2.0 envelope types and all domain-specific param/result types |
| `jsonrpc` | Low-level NDJSON transport (read requests, write responses/notifications) |
| `extension` | High-level interfaces and runners for each extension type |
| `harness` | `FakeKova` test helper that simulates the kova host |

## Quick Start: Tool Extension

A minimal tool extension that returns the current time:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/protocol"
)

type timeTool struct{}

func (t *timeTool) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{
		Tools: []protocol.ToolDef{{
			Name:        "current_time",
			Description: "Returns the current UTC time",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		}},
	}, nil
}

func (t *timeTool) HandleToolCall(_ context.Context, params protocol.ToolCallParams) (*protocol.ToolCallResult, error) {
	return &protocol.ToolCallResult{
		Output: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (t *timeTool) Shutdown() error { return nil }

func main() {
	if err := extension.RunTool(&timeTool{}); err != nil {
		log.Fatal(err)
	}
}
```

## Extension Types

### Channel Extension

Bridges a chat platform (Discord, Slack, etc.) to kova. Implement `extension.ChannelExtension` and call `extension.RunChannel`.

```go
extension.RunChannel(&myChannel{})
```

### Provider Extension

Bridges an LLM API (Anthropic, OpenAI, etc.) to kova. Implement `extension.ProviderExtension` and call `extension.RunProvider`. The buffer size defaults to 16 MB for large conversation histories.

```go
extension.RunProvider(&myProvider{})
```

### Tool Extension

Exposes custom tools to the agent. Implement `extension.ToolExtension` and call `extension.RunTool`.

```go
extension.RunTool(&myTool{})
```

### TTS Extension

Bridges a text-to-speech API (OpenAI TTS, Piper, etc.) to kova. Implement `extension.TTSExtension` and call `extension.RunTTS`.

```go
extension.RunTTS(&myTTS{})
```

### Multi-Type Limitation

Each `Run*` function handles a single registration type. Extensions that register multiple types (e.g., both tools and hooks) are not yet supported by the SDK — use the low-level `jsonrpc.Transport` directly for multi-type extensions.

## Testing Extensions

The `harness` package provides `FakeKova`, which simulates the kova host over in-memory pipes:

```go
func TestTimeTool(t *testing.T) {
	tool := &timeTool{}
	fk := harness.NewFakeKova(t)

	// Start the extension in a goroutine.
	go extension.RunTool(tool,
		extension.WithStdin(fk.ExtStdin),
		extension.WithStdout(fk.ExtStdout),
	)

	// Initialize and verify registrations.
	regs := fk.Initialize(nil)
	if len(regs.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(regs.Tools))
	}

	// Call the tool.
	result := fk.Call("tool_call", protocol.ToolCallParams{
		Name:  "current_time",
		Input: map[string]any{},
	})

	var toolResult protocol.ToolCallResult
	json.Unmarshal(result, &toolResult)
	if toolResult.Output == "" {
		t.Fatal("expected non-empty output")
	}

	fk.Shutdown()
}
```

## Emitter

The `Emitter` is passed to your extension during `Initialize`. Use it to send notifications back to kova:

```go
func (ext *myChannel) Initialize(emitter *extension.Emitter, ...) (*protocol.Registrations, error) {
	ext.emitter = emitter
	// Later, when a user message arrives on your platform:
	emitter.EmitChannelMessage(protocol.ChannelMessageParams{
		ChannelID: "123",
		Text:      "Hello from the user",
		Platform:  "my-platform",
	})
	// Structured logging:
	emitter.Log("info", "channel extension initialized")
	return &protocol.Registrations{}, nil
}
```

## Protocol Compatibility

The types in `protocol/` are copied from [`kova/internal/extension/bridge/protocol.go`](https://github.com/kova-land/kova/blob/main/internal/extension/bridge/protocol.go) and kept in sync. They use identical field names, JSON tags, and Go types to ensure wire compatibility.

## License

Apache 2.0. See [LICENSE](LICENSE).
