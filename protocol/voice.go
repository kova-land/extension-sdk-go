package protocol

// VoiceJoinParams holds the payload for a "voice_join" request.
// The extension should join the specified voice channel and begin
// preparing to send/receive audio.
type VoiceJoinParams struct {
	// ChannelID is the voice channel to join.
	ChannelID string `json:"channel_id"`
	// Options are platform-specific join options (e.g., mute, deaf for Discord).
	Options map[string]any `json:"options,omitempty"`
}

// VoiceJoinResult is the response for a "voice_join" request.
type VoiceJoinResult struct {
	// Success indicates whether the join was successful.
	Success bool `json:"success"`
	// Error is a human-readable error message if Success is false.
	Error string `json:"error,omitempty"`
}

// VoiceLeaveParams holds the payload for a "voice_leave" request.
// The extension should disconnect from the voice channel identified by ChannelID.
type VoiceLeaveParams struct {
	// ChannelID is the voice channel to leave.
	ChannelID string `json:"channel_id"`
}

// VoiceLeaveResult is the response for a "voice_leave" request.
type VoiceLeaveResult struct {
	// Success indicates whether the leave was successful.
	Success bool `json:"success"`
	// Error is a human-readable error message if Success is false.
	Error string `json:"error,omitempty"`
}

// VoiceStreamParams holds the payload for a "voice_stream" request.
// Audio is sent as a list of encoded frames; the encoding is indicated by Format.
type VoiceStreamParams struct {
	// ChannelID identifies which voice connection to stream to.
	ChannelID string `json:"channel_id"`
	// AudioData is the audio payload as a list of encoded frames.
	// The encoding depends on Format.
	AudioData [][]byte `json:"audio_data"`
	// Format indicates the audio encoding (e.g., "opus", "pcm", "mp3").
	// Extensions declare their supported codec via capabilities.
	Format string `json:"format,omitempty"`
	// SampleRate is the audio sample rate in Hz (e.g., 48000).
	// Optional — defaults to the platform's requirement.
	SampleRate int `json:"sample_rate,omitempty"`
	// Channels is the number of audio channels: 1 for mono, 2 for stereo.
	// Optional — defaults to the platform's requirement.
	Channels int `json:"channels,omitempty"`
}

// VoiceStreamResult is the response for a "voice_stream" request.
type VoiceStreamResult struct {
	// FramesSent is the number of audio frames successfully sent.
	FramesSent int `json:"frames_sent"`
	// Error is a human-readable error message if any frames failed to send.
	Error string `json:"error,omitempty"`
}
