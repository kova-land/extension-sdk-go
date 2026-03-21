// Package extension provides high-level interfaces and runners for building
// Kova extensions. It handles the JSON-RPC lifecycle (initialize, dispatch,
// shutdown) so extension authors only need to implement domain-specific logic.
//
// Each extension type has a corresponding interface and Run function:
//
//   - [ChannelExtension] + [RunChannel]: Chat platform adapters (Discord, Slack, etc.)
//   - [ProviderExtension] + [RunProvider]: LLM provider backends (Anthropic, OpenAI, etc.)
//   - [ToolExtension] + [RunTool]: Custom tools exposed to the agent
//   - [TTSExtension] + [RunTTS]: Text-to-speech provider backends
//
// The Run functions set up stdin/stdout JSON-RPC transport, handle OS signals
// (SIGTERM, SIGINT), and dispatch incoming requests to the appropriate interface
// method. They also provide an [Emitter] for sending notifications back to kova.
package extension
