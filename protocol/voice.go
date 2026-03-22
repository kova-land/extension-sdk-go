package protocol

// VoiceJoinParams holds the payload for a "voice_join" request.
// The extension should join the specified voice channel and begin
// preparing to send/receive audio.
type VoiceJoinParams struct {
	// GuildID is the guild (server) containing the voice channel.
	GuildID string `json:"guild_id"`
	// ChannelID is the voice channel to join.
	ChannelID string `json:"channel_id"`
	// Mute controls whether the bot joins muted. Defaults to false.
	Mute bool `json:"mute,omitempty"`
	// Deaf controls whether the bot joins deafened. Defaults to false.
	Deaf bool `json:"deaf,omitempty"`
}

// VoiceJoinResult is the response for a "voice_join" request.
type VoiceJoinResult struct {
	// Success indicates whether the join was successful.
	Success bool `json:"success"`
	// Error is a human-readable error message if Success is false.
	Error string `json:"error,omitempty"`
}

// VoiceLeaveParams holds the payload for a "voice_leave" request.
// The extension should disconnect from the voice channel in the given guild.
type VoiceLeaveParams struct {
	// GuildID is the guild whose voice channel to leave.
	GuildID string `json:"guild_id"`
}

// VoiceLeaveResult is the response for a "voice_leave" request.
type VoiceLeaveResult struct {
	// Success indicates whether the leave was successful.
	Success bool `json:"success"`
	// Error is a human-readable error message if Success is false.
	Error string `json:"error,omitempty"`
}

// VoiceStreamParams holds the payload for a "voice_stream" request.
// Audio is sent as pre-encoded Opus frames (base64-encoded in JSON).
// Each frame should be a single Opus packet at 48kHz, stereo, 20ms frame size.
type VoiceStreamParams struct {
	// GuildID identifies which voice connection to stream to.
	GuildID string `json:"guild_id"`
	// OpusFrames is a list of base64-encoded Opus audio frames to send.
	// Each entry is one Opus packet. The extension sends them sequentially
	// to the voice connection's OpusSend channel.
	OpusFrames [][]byte `json:"opus_frames"`
}

// VoiceStreamResult is the response for a "voice_stream" request.
type VoiceStreamResult struct {
	// FramesSent is the number of Opus frames successfully sent.
	FramesSent int `json:"frames_sent"`
}
