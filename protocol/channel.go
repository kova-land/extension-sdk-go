package protocol

// ChannelMessageParams is sent by an extension (via channel_message notification)
// to push an incoming user message to the kova gateway. Extensions that implement
// a channel (e.g. a Discord bot subprocess) use this to deliver messages from
// their platform into kova's agent processing pipeline.
type ChannelMessageParams struct {
	// ChannelID is the platform-specific channel or conversation identifier.
	ChannelID string `json:"channel_id"`
	// MessageID is the platform-specific identifier of the incoming message.
	// Used by core to reference the message for reactions (ack/error indicators).
	MessageID string `json:"message_id,omitempty"`
	// UserID is the platform-specific identifier of the message author (may be empty).
	UserID string `json:"user_id,omitempty"`
	// Username is the human-readable display name of the message author (may be empty).
	Username string `json:"username,omitempty"`
	// Text is the raw message text.
	Text string `json:"text"`
	// Platform is the platform name reported by the extension (e.g. "discord", "slack").
	Platform string `json:"platform"`
	// IsSlashCommand indicates this message originated from a platform slash command
	// (e.g. Discord interaction) rather than a regular chat message. When true, core
	// handles the command directly (tools, agents, skills, clear, etc.) without LLM
	// processing and sends the response back via channel_send so the extension can
	// route it through the original interaction.
	IsSlashCommand bool `json:"is_slash_command,omitempty"`
	// InteractionID is the platform-specific interaction identifier (e.g. Discord
	// interaction ID). Extensions set this when IsSlashCommand is true so that core
	// can echo it back in the channel_send response, allowing the extension to match
	// the response to the correct pending interaction when multiple slash commands
	// are in flight in the same channel.
	InteractionID string `json:"interaction_id,omitempty"`

	// Images holds inbound image attachments (base64-encoded in JSON via []byte).
	// Extensions that support receiving images should populate this from their
	// platform's attachment mechanism.
	Images []ChannelMessageImage `json:"images,omitempty"`
}

// ChannelMessageImage carries an inbound image attachment over the bridge protocol.
// Data is base64-encoded by the JSON marshaler ([]byte -> base64 string).
type ChannelMessageImage struct {
	// Data is the raw image bytes (base64-encoded in JSON).
	Data []byte `json:"data"`
	// MIMEType is the MIME type (e.g. "image/png", "image/jpeg").
	MIMEType string `json:"mime_type"`
	// Filename is the original filename (may be empty).
	Filename string `json:"filename,omitempty"`
}

// ChannelSendParams is sent by kova to deliver an agent response to the extension.
// The extension is expected to forward the message to the given channel on its platform.
type ChannelSendParams struct {
	// ChannelID identifies the target channel on the extension's platform.
	ChannelID string `json:"channel_id"`
	// Text is the response text to deliver.
	Text string `json:"text"`
	// Images holds image attachments. Extensions that don't support images silently
	// ignore this slice.
	Images []ChannelSendAttachment `json:"images,omitempty"`
	// Audio holds audio attachments.
	Audio []ChannelSendAttachment `json:"audio,omitempty"`
	// Files holds file attachments.
	Files []ChannelSendAttachment `json:"files,omitempty"`

	// Components holds rows of interactive buttons (platform-dependent support).
	Components [][]ChannelSendComponent `json:"components,omitempty"`

	// IsSlashCommandResponse marks this response as the result of a slash command.
	// When true, the extension should route the response through the original
	// platform interaction (e.g. Discord ephemeral follow-up) instead of sending
	// a regular channel message.
	IsSlashCommandResponse bool `json:"is_slash_command_response,omitempty"`
	// InteractionID is echoed back from the inbound ChannelMessageParams.InteractionID.
	// Extensions use this to match the response to the correct pending interaction
	// when multiple slash commands are in flight in the same channel.
	InteractionID string `json:"interaction_id,omitempty"`
}

// ChannelSendAttachment carries a file attachment over the bridge protocol.
// Data is base64-encoded by the JSON marshaler ([]byte -> base64 string).
type ChannelSendAttachment struct {
	// Data is the raw file content (base64-encoded in JSON).
	Data []byte `json:"data"`
	// MIMEType is the MIME type (e.g. "image/png", "audio/opus").
	MIMEType string `json:"mime_type"`
	// Filename is the suggested filename.
	Filename string `json:"filename"`
}

// ChannelSendComponent represents an interactive button component sent over
// the bridge protocol.
type ChannelSendComponent struct {
	Label    string `json:"label"`
	CustomID string `json:"custom_id"`
	Style    int    `json:"style"` // 1=Primary, 2=Secondary, 3=Success, 4=Danger
}

// ChannelTypingParams is sent by kova to request that the extension show a typing
// indicator in the given channel. Extensions should treat this as best-effort.
type ChannelTypingParams struct {
	// ChannelID identifies the target channel on the extension's platform.
	ChannelID string `json:"channel_id"`
}

// ChannelReactionAddParams is sent by kova to add an emoji reaction to a message.
// Extensions should treat this as best-effort.
type ChannelReactionAddParams struct {
	// ChannelID identifies the target channel on the extension's platform.
	ChannelID string `json:"channel_id"`
	// MessageID identifies the message to react to.
	MessageID string `json:"message_id"`
	// Emoji is the reaction emoji (e.g. "👀", "❌").
	Emoji string `json:"emoji"`
}

// ChannelReactionRemoveParams is sent by kova to remove an emoji reaction from a message.
// Extensions should treat this as best-effort.
type ChannelReactionRemoveParams struct {
	// ChannelID identifies the target channel on the extension's platform.
	ChannelID string `json:"channel_id"`
	// MessageID identifies the message to remove the reaction from.
	MessageID string `json:"message_id"`
	// Emoji is the reaction emoji to remove.
	Emoji string `json:"emoji"`
}

// PromptOption represents a single selectable option in a prompt_user request.
// Extensions render these as platform-native UI elements (e.g. Discord buttons,
// Signal numbered menus).
type PromptOption struct {
	// Label is the human-readable text displayed to the user.
	Label string `json:"label"`
	// Value is the machine-readable value returned when this option is selected.
	Value string `json:"value"`
	// Style hints at visual treatment. Values: "primary", "secondary", "success", "danger".
	// Extensions may ignore this if the platform has no styling support.
	Style string `json:"style,omitempty"`
}

// PromptUserParams is sent by kova to request interactive user input via the
// extension's platform-native UI. The extension renders the prompt using whatever
// mechanism the platform supports (buttons, modals, numbered text menus, etc.)
// and returns the user's response.
type PromptUserParams struct {
	// ChannelID identifies the target channel for the prompt.
	ChannelID string `json:"channel_id"`
	// UserID optionally restricts which user may respond. Empty means any user.
	UserID string `json:"user_id,omitempty"`
	// Message is the prompt text displayed to the user.
	Message string `json:"message"`
	// Options is the list of selectable choices. May be empty if only freeform
	// text input is desired.
	Options []PromptOption `json:"options,omitempty"`
	// AllowText indicates whether the user may provide freeform text input
	// in addition to (or instead of) selecting an option.
	AllowText bool `json:"allow_text,omitempty"`
	// TextPlaceholder is optional hint text for the freeform input field.
	TextPlaceholder string `json:"text_placeholder,omitempty"`
	// TimeoutSeconds is the maximum time to wait for a response. 0 means use
	// the extension's default timeout.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// PromptUserResult is the response payload for a "prompt_user" request.
type PromptUserResult struct {
	// SelectedOption is the Value of the option the user selected. Empty if
	// the user provided only freeform text or did not select an option.
	SelectedOption string `json:"selected_option,omitempty"`
	// Text is freeform text input from the user. Empty if the user only
	// selected an option.
	Text string `json:"text,omitempty"`
	// TimedOut indicates the prompt expired before the user responded.
	TimedOut bool `json:"timed_out,omitempty"`
	// Dismissed indicates the user explicitly dismissed/cancelled the prompt.
	Dismissed bool `json:"dismissed,omitempty"`
}

// ChannelCapabilities describes what features a channel extension's platform
// supports. Returned by the extension in response to a "channel_capabilities"
// request.
type ChannelCapabilities struct {
	SupportsText      bool `json:"supports_text"`
	SupportsImages    bool `json:"supports_images,omitempty"`
	SupportsAudio     bool `json:"supports_audio,omitempty"`
	SupportsFiles     bool `json:"supports_files,omitempty"`
	SupportsVoice     bool `json:"supports_voice,omitempty"`
	SupportsReactions bool `json:"supports_reactions,omitempty"`
	SupportsButtons   bool `json:"supports_buttons,omitempty"`
	SupportsModals    bool `json:"supports_modals,omitempty"`
	SupportsTyping    bool `json:"supports_typing,omitempty"`
	// MaxMessageLen is the platform's maximum message length. 0 means unlimited.
	MaxMessageLen int `json:"max_message_len,omitempty"`
}

// ChannelCapabilitiesParams is sent by kova to query the extension's platform
// capabilities. The channel_id is optional; when provided, capabilities may
// vary per channel (e.g. a DM channel vs a group channel).
type ChannelCapabilitiesParams struct {
	ChannelID string `json:"channel_id,omitempty"`
}

// ChannelCapabilitiesResult is the response payload for a
// "channel_capabilities" request.
type ChannelCapabilitiesResult struct {
	Capabilities ChannelCapabilities `json:"capabilities"`
}
