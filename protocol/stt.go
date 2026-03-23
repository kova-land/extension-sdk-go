package protocol

// STTTranscribeParams holds the payload for a "stt_transcribe" request.
type STTTranscribeParams struct {
	// AudioBase64 is the input audio encoded as base64. Required.
	AudioBase64 string `json:"audio_base64"`
	// Format is the audio format of the input data: "opus", "mp3", "flac", "wav", "pcm".
	// Required so the STT provider knows how to decode.
	Format string `json:"format"`
	// SampleRate is the audio sample rate in Hz (e.g., 16000, 48000). Optional.
	SampleRate int `json:"sample_rate,omitempty"`
	// Channels is the number of audio channels: 1 for mono, 2 for stereo. Optional.
	Channels int `json:"channels,omitempty"`
	// Language is the BCP-47 language code hint (e.g., "en", "fr"). Optional.
	Language string `json:"language,omitempty"`
	// Model selects the STT model (e.g., "whisper-1", "large-v3"). Optional.
	Model string `json:"model,omitempty"`
	// UserID identifies the user who produced the audio. Optional.
	UserID string `json:"user_id,omitempty"`
	// ChannelID identifies the channel the audio came from. Optional.
	ChannelID string `json:"channel_id,omitempty"`
}

// STTTranscribeResult is the response for a "stt_transcribe" request.
type STTTranscribeResult struct {
	// Text is the full transcription of the audio.
	Text string `json:"text"`
	// Language is the detected or confirmed BCP-47 language code.
	Language string `json:"language,omitempty"`
	// Confidence is the overall transcription confidence (0.0 to 1.0). Optional.
	Confidence float64 `json:"confidence,omitempty"`
	// Segments are word- or phrase-level timing segments. Optional.
	Segments []STTSegment `json:"segments,omitempty"`
}

// STTSegment represents a timed segment within a transcription.
type STTSegment struct {
	// Start is the segment start time in seconds.
	Start float64 `json:"start"`
	// End is the segment end time in seconds.
	End float64 `json:"end"`
	// Text is the transcribed text for this segment.
	Text string `json:"text"`
}

// STTModelsResult is the response for a "stt_models" request.
type STTModelsResult struct {
	Models []STTModel `json:"models"`
}

// STTModel describes an available model from an STT provider extension.
type STTModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language,omitempty"`
}

// STTProviderDef describes an STT provider an extension registers during
// initialization. This is included in the Registrations response.
type STTProviderDef struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
