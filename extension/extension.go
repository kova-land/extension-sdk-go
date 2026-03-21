package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// Emitter provides methods for sending notifications from the extension back
// to the kova host. It is passed to extensions during initialization and is
// safe for concurrent use.
type Emitter struct {
	transport *jsonrpc.Transport
}

// Log sends a structured log notification to the kova host. The level should
// be one of "debug", "info", "warn", or "error".
func (e *Emitter) Log(level, message string) error {
	return e.transport.SendNotification(protocol.NotificationLog, protocol.LogParams{
		Level:   level,
		Message: message,
	})
}

// EmitChannelMessage sends a channel_message notification to kova, pushing an
// inbound user message into the agent processing pipeline. Only meaningful for
// channel extensions.
func (e *Emitter) EmitChannelMessage(params protocol.ChannelMessageParams) error {
	return e.transport.SendNotification(protocol.NotificationChannelMessage, params)
}

// EmitStreamChunk sends a stream_chunk notification to kova during a streaming
// provider response. Only meaningful for provider extensions.
func (e *Emitter) EmitStreamChunk(params protocol.StreamChunkParams) error {
	return e.transport.SendNotification(protocol.NotificationStreamChunk, params)
}

// EmitStreamEnd sends a stream_end notification to kova, signaling that a
// streaming response is complete. Only meaningful for provider extensions.
func (e *Emitter) EmitStreamEnd(params protocol.StreamEndParams) error {
	return e.transport.SendNotification(protocol.NotificationStreamEnd, params)
}

// EmitStreamError sends a stream_error notification to kova, signaling a
// provider failure mid-stream. Only meaningful for provider extensions.
func (e *Emitter) EmitStreamError(params protocol.StreamErrorParams) error {
	return e.transport.SendNotification(protocol.NotificationStreamError, params)
}

// Option configures the extension runner.
type Option func(*runConfig)

type runConfig struct {
	stdin   io.Reader
	stdout  io.Writer
	bufSize int
}

// WithStdin overrides the default stdin reader. Useful for testing.
func WithStdin(r io.Reader) Option {
	return func(c *runConfig) { c.stdin = r }
}

// WithStdout overrides the default stdout writer. Useful for testing.
func WithStdout(w io.Writer) Option {
	return func(c *runConfig) { c.stdout = w }
}

// WithBufferSize sets the maximum NDJSON line buffer size. Defaults to 1 MB
// for most extension types. Provider extensions should use
// jsonrpc.LargeBufferSize (16 MB).
func WithBufferSize(size int) Option {
	return func(c *runConfig) { c.bufSize = size }
}

func defaultConfig() *runConfig {
	return &runConfig{
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		bufSize: jsonrpc.DefaultBufferSize,
	}
}

// dispatcher is the common request loop shared by all Run* functions. It handles
// initialize, shutdown, and delegates domain-specific methods to the handler.
type dispatcher struct {
	transport *jsonrpc.Transport
	emitter   *Emitter

	// onInitialize is called with the InitializeParams and must return registrations.
	onInitialize func(protocol.InitializeParams) (*protocol.Registrations, error)
	// onMethod is called for each non-lifecycle method.
	onMethod func(ctx context.Context, req *protocol.Request) error
	// onShutdown is called when the host sends "shutdown".
	onShutdown func() error
}

// run is the main request loop. It reads requests and dispatches them.
func (d *dispatcher) run(ctx context.Context) error {
	for {
		req, err := d.transport.ReadRequest()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			// Send parse error if we can identify the request.
			_ = d.transport.SendError(0, protocol.ErrCodeParseError, fmt.Sprintf("invalid request: %v", err))
			return err
		}

		if ctx.Err() != nil {
			return nil
		}

		switch req.Method {
		case protocol.MethodInitialize:
			d.handleInitialize(req)
		case protocol.MethodShutdown:
			d.handleShutdown(req)
			return nil
		default:
			if err := d.onMethod(ctx, req); err != nil {
				_ = d.transport.SendError(req.ID, protocol.ErrCodeInternal, err.Error())
			}
		}
	}
}

func (d *dispatcher) handleInitialize(req *protocol.Request) {
	var params protocol.InitializeParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			_ = d.transport.SendError(req.ID, protocol.ErrCodeInvalidParams, fmt.Sprintf("invalid params: %v", err))
			return
		}
	}

	regs, err := d.onInitialize(params)
	if err != nil {
		_ = d.transport.SendError(req.ID, protocol.ErrCodeNotReady, err.Error())
		return
	}

	result := protocol.InitializeResult{Registrations: *regs}
	_ = d.transport.SendResponse(req.ID, result)
}

func (d *dispatcher) handleShutdown(req *protocol.Request) {
	var shutdownErr error
	if d.onShutdown != nil {
		shutdownErr = d.onShutdown()
	}
	if shutdownErr != nil {
		_ = d.transport.SendError(req.ID, protocol.ErrCodeInternal, shutdownErr.Error())
		return
	}
	_ = d.transport.SendResponse(req.ID, struct{}{})
}

// startRun creates a context with signal handling, builds the transport and
// emitter, and returns them along with the cancellable context.
func startRun(opts []Option) (context.Context, context.CancelFunc, *jsonrpc.Transport, *Emitter) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	transport := jsonrpc.NewTransport(cfg.stdin, cfg.stdout, cfg.bufSize)
	emitter := &Emitter{transport: transport}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	return ctx, cancel, transport, emitter
}
