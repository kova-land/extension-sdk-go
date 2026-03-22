# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Commands

```bash
go build ./...
go test -v ./...
go test -v -race ./...
go fmt ./...
go vet ./...
```

## Linting

```bash
golangci-lint run ./...
```

## Schema Generation

```bash
make generate          # Regenerate JSON schemas from Go protocol types
make check-schemas     # Verify schemas are up to date (used in CI)
```

## Architecture

This is the Go SDK for building [Kova](https://github.com/kova-land/kova) extensions. It provides protocol types, a JSON-RPC 2.0 transport layer, high-level extension runner interfaces, and test utilities.

### Package Map

| Package | Purpose |
|---------|---------|
| `protocol` | JSON-RPC 2.0 envelope types and all domain-specific param/result types |
| `jsonrpc` | Low-level NDJSON transport (read requests, write responses/notifications) |
| `extension` | High-level interfaces and runners for each extension type + composable `Run()` |
| `harness` | `FakeKova` test helper that simulates the kova host |
| `retry` | Retry utilities for transient failures |
| `schemas` | JSON schema generation from Go types (`cmd/generate-schemas/`) |

### Extension Types

Extensions implement one of: `ToolExtension`, `ChannelExtension`, `ProviderExtension`, `TTSExtension`, `HookExtension`, `HTTPExtension`, `ServiceExtension`. Each has a corresponding `Run*()` function. For multi-type extensions, use `extension.Run()` with `With*()` options.

## Key Patterns

- Zero external dependencies for core packages (`protocol`, `jsonrpc`, `extension`, `harness`) -- stdlib only
- `schemas` package uses `github.com/google/jsonschema-go` for schema generation
- Protocol types are the canonical source -- Python and TypeScript SDKs generate from the JSON schemas produced here
- `FakeKova` test harness simulates the kova host over in-memory pipes for unit testing extensions
- `Emitter` is passed during `Initialize` for sending notifications (channel messages, logs) back to kova

## Protocol Design Rules

### Params-Capabilities Symmetry

When adding fields to `ChannelMessageParams` that describe platform-specific behavior (threads, DMs, mentions, etc.), always add a corresponding `Supports*` field to `ChannelCapabilities`. This ensures:

1. Extensions declare what their platform supports (capabilities)
2. Extensions send what actually happened in each message (params)
3. Kova core can validate and route based on declared capabilities
4. No implicit assumptions about what a platform can do

**Pattern:**

| `ChannelMessageParams` field | `ChannelCapabilities` field |
|------------------------------|----------------------------|
| `Thread` (bool) | `SupportsThreads` (bool) |
| `DirectMessage` (bool) | `SupportsDMs` (bool) |
| `Explicit` (bool) | `SupportsMentions` (bool) |

**Why:** If params exist without capabilities, kova cannot distinguish "platform doesn't support threads" from "extension forgot to set the field." Both produce `Thread: false`, but the meaning is different. Capabilities make the distinction explicit.

## Relationship to Kova

- Protocol types mirror `kova/internal/extension/bridge/protocol.go` and must stay in sync
- This SDK is what third-party extension authors import to build kova extensions in Go
- The JSON schemas in `schemas/` are the single source of truth for all SDK language bindings
- Kova's built-in extensions (`extensions/discord`, `extensions/signal`, `extensions/anthropic`, `extensions/openai`) use the bridge protocol directly, not this SDK

## Related Repositories

| Repo | Purpose |
|------|---------|
| [kova-land/kova](https://github.com/kova-land/kova) | Core agent framework (defines the extension protocol) |
| [kova-land/extension-sdk-python](https://github.com/kova-land/extension-sdk-python) | Python SDK (types generated from this repo's schemas) |
| [kova-land/extension-sdk-typescript](https://github.com/kova-land/extension-sdk-typescript) | TypeScript SDK (types generated from this repo's schemas) |
