package protocol

// TTSSynthesizeParams holds the payload for a "tts_synthesize" request.
type TTSSynthesizeParams struct {
	// Text is the input text to synthesize. Required.
	Text string `json:"text"`
	// Voice is the voice ID to use. Required.
	Voice string `json:"voice"`
	// Model selects the TTS model (e.g., "tts-1", "tts-1-hd"). Optional.
	Model string `json:"model,omitempty"`
	// Format is the audio output format: "opus", "mp3", "flac", "wav", "pcm".
	Format string `json:"format,omitempty"`
	// Speed controls playback speed (0.25 to 4.0). Optional; defaults to 1.0.
	Speed float64 `json:"speed,omitempty"`
}

// TTSSynthesizeResult is the response for a "tts_synthesize" request.
// Audio data is returned as base64-encoded bytes to stay within the JSON-RPC
// protocol. For large audio payloads, extensions should consider chunking
// or streaming in future protocol versions.
type TTSSynthesizeResult struct {
	// AudioBase64 is the synthesized audio encoded as base64.
	AudioBase64 string `json:"audio_base64"`
	// Format is the audio format of the returned data.
	Format string `json:"format"`
	// ContentType is the MIME type (e.g., "audio/opus", "audio/mpeg").
	ContentType string `json:"content_type"`
}

// TTSVoicesResult is the response for a "tts_voices" request.
type TTSVoicesResult struct {
	Voices []TTSVoice `json:"voices"`
}

// TTSVoice describes an available voice from a TTS provider extension.
type TTSVoice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language,omitempty"`
}
