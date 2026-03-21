package protocol

// LogParams holds the payload for a "log" notification from the extension.
type LogParams struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}
