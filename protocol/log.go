package protocol

import (
	"log/slog"
	"os"
)

// Log level constants for use with LogEntry.Level and Emitter.Log.
// These match the levels recognised by the kova parent process.
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// LogEntry is the structured JSON format extensions write to stderr.
// The kova parent process reads each line from the extension's stderr,
// unmarshals it as a LogEntry, and forwards the log at the correct level.
//
// Extensions should use [SetupLogging] to configure Go's slog package to
// emit this format automatically, rather than constructing LogEntry values
// by hand.
//
// Wire format (one JSON object per line, written to stderr):
//
//	{"time":"2006-01-02T15:04:05.999999999Z","level":"INFO","msg":"started","attrs":{"port":8080}}
type LogEntry struct {
	// Time is an RFC 3339 Nano timestamp (e.g. "2006-01-02T15:04:05.999999999Z").
	Time string `json:"time"`
	// Level is the severity: DEBUG, INFO, WARN, or ERROR.
	Level string `json:"level"`
	// Msg is the human-readable log message.
	Msg string `json:"msg"`
	// Attrs holds optional structured key-value pairs. Omitted when empty.
	Attrs map[string]any `json:"attrs,omitempty"`
}

// SetupLogging configures the default slog logger to write structured JSON
// log lines to stderr in the [LogEntry] format expected by the kova parent
// process. Call this once at extension startup, before any logging.
//
// Example:
//
//	func main() {
//	    protocol.SetupLogging(slog.LevelInfo)
//	    // slog.Info / slog.Debug / slog.Warn / slog.Error now write LogEntry JSON to stderr
//	    extension.Run(...)
//	}
func SetupLogging(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// LogParams holds the payload for a "log" notification from the extension.
// This is distinct from [LogEntry]: LogParams is sent over the JSON-RPC
// transport via [Emitter.Log], while LogEntry is written directly to stderr
// for the parent process to collect.
type LogParams struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}
