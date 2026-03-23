package protocol

// InitializeParams holds the payload for the "initialize" request.
type InitializeParams struct {
	// Config is the extension-specific configuration from extension.json.
	Config map[string]any `json:"config"`
	// ExtensionRoot is the absolute path to the extension's directory.
	ExtensionRoot string `json:"extension_root"`
	// AgentName is the configured agent name (e.g. "kova"). Extensions can use
	// this for user-facing messages like boot announcements.
	AgentName string `json:"agent_name,omitempty"`
	// AgentVersion is the build version string (e.g. "0.14.0").
	AgentVersion string `json:"agent_version,omitempty"`
}

// InitializeResult is the response payload for the "initialize" request.
type InitializeResult struct {
	// ProtocolVersion is the extension protocol version this extension
	// implements (e.g. "0.5"). When set, kova validates compatibility.
	// Omit or leave empty to skip version negotiation.
	ProtocolVersion string        `json:"protocol_version,omitempty"`
	Registrations   Registrations `json:"registrations"`
}

// Registrations holds all integration points an extension declares during
// initialization. Fields map 1:1 to RegisterType values in the manifest.
type Registrations struct {
	Tools        []ToolDef        `json:"tools,omitempty"`
	HTTPRoutes   []HTTPRouteDef   `json:"http_routes,omitempty"`
	Hooks        []HookDef        `json:"hooks,omitempty"`
	Services     []ServiceDef     `json:"services,omitempty"`
	Providers    []ProviderDef    `json:"providers,omitempty"`
	TTSProviders []TTSProviderDef `json:"tts_providers,omitempty"`
	STTProviders []STTProviderDef `json:"stt_providers,omitempty"`
	CLICommands  []CLIDef         `json:"cli_commands,omitempty"`
}

// ToolDef describes a tool an extension registers during initialization.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// HTTPRouteDef describes an HTTP route an extension registers.
type HTTPRouteDef struct {
	Path    string   `json:"path"`
	Methods []string `json:"methods"`
}

// HookDef describes a lifecycle hook event an extension registers.
type HookDef struct {
	// Events is the list of lifecycle event names this hook listens to.
	// Each name must match a hooks.Event constant (e.g. "UserPromptSubmit", "PreToolUse").
	Events []string `json:"events"`
	// Matcher is an optional regex pattern applied to the tool name for tool-related
	// events (PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest).
	// An empty string or absent field matches all tools.
	Matcher string `json:"matcher,omitempty"`
}

// ServiceDef describes a background service an extension registers during
// initialization. Services are native-runtime-only and have explicit lifecycle
// (start/health/stop) managed by the Manager.
type ServiceDef struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

// ProviderDef describes an LLM provider an extension registers during
// initialization.
type ProviderDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// TTSProviderDef describes a TTS provider an extension registers during
// initialization. This is included in the Registrations response.
type TTSProviderDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CLIDef describes a CLI subcommand an extension declares so kova can expose
// it under `kova extension run <extension-name> <subcommand>`.
type CLIDef struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Command     string   `json:"command,omitempty"`
	Args        []string `json:"args,omitempty"`
}

// CallContext carries agent/session metadata with each tool call,
// allowing extensions to know which agent and session invoked them.
type CallContext struct {
	AgentID   string `json:"agent_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Platform  string `json:"platform,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}
