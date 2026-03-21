package extension

import (
	"context"
	"fmt"

	"github.com/kova-land/extension-sdk-go/jsonrpc"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// RunOption configures which extension types are active in a composable Run call.
type RunOption func(*compositeConfig)

type compositeConfig struct {
	channel  ChannelExtension
	provider ProviderExtension
	tool     ToolExtension
	tts      TTSExtension
	hook     HookExtension
	http     HTTPExtension
	service  ServiceExtension
}

// WithChannel registers a channel extension handler.
func WithChannel(ext ChannelExtension) RunOption {
	return func(c *compositeConfig) { c.channel = ext }
}

// WithProvider registers a provider extension handler.
func WithProvider(ext ProviderExtension) RunOption {
	return func(c *compositeConfig) { c.provider = ext }
}

// WithTool registers a tool extension handler.
func WithTool(ext ToolExtension) RunOption {
	return func(c *compositeConfig) { c.tool = ext }
}

// WithTTS registers a TTS extension handler.
func WithTTS(ext TTSExtension) RunOption {
	return func(c *compositeConfig) { c.tts = ext }
}

// WithHook registers a hook extension handler.
func WithHook(ext HookExtension) RunOption {
	return func(c *compositeConfig) { c.hook = ext }
}

// WithHTTP registers an HTTP extension handler.
func WithHTTP(ext HTTPExtension) RunOption {
	return func(c *compositeConfig) { c.http = ext }
}

// WithService registers a service extension handler.
func WithService(ext ServiceExtension) RunOption {
	return func(c *compositeConfig) { c.service = ext }
}

// Run starts a composable extension that can implement multiple extension types
// simultaneously. It merges registrations from all provided handlers and routes
// incoming methods to the appropriate handler based on the method name.
//
// If a provider handler is registered, the buffer size defaults to 16 MB
// (same as RunProvider) unless overridden by WithBufferSize.
//
// Example:
//
//	extension.Run(
//	    extension.WithProvider(myProvider),
//	    extension.WithTTS(myTTS),
//	)
func Run(runOpts []RunOption, opts ...Option) error {
	cc := &compositeConfig{}
	for _, o := range runOpts {
		o(cc)
	}

	// Use large buffer if provider is registered (same behavior as RunProvider).
	if cc.provider != nil {
		opts = append([]Option{WithBufferSize(jsonrpc.LargeBufferSize)}, opts...)
	}

	ctx, cancel, transport, emitter := startRun(opts)
	defer cancel()

	d := &dispatcher{
		transport: transport,
		emitter:   emitter,
		onInitialize: func(params protocol.InitializeParams) (*protocol.Registrations, error) {
			return compositeInitialize(cc, emitter, params)
		},
		onMethod: func(ctx context.Context, req *protocol.Request) error {
			return compositeDispatch(ctx, transport, cc, req)
		},
		onShutdown: func() error {
			return compositeShutdown(cc)
		},
	}

	return d.run(ctx)
}

// compositeInitialize calls Initialize on each registered handler and merges
// all registrations into a single Registrations struct.
func compositeInitialize(cc *compositeConfig, emitter *Emitter, params protocol.InitializeParams) (*protocol.Registrations, error) {
	merged := &protocol.Registrations{}

	if cc.channel != nil {
		regs, err := cc.channel.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("channel init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}
	if cc.provider != nil {
		regs, err := cc.provider.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("provider init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}
	if cc.tool != nil {
		regs, err := cc.tool.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("tool init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}
	if cc.tts != nil {
		regs, err := cc.tts.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("tts init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}
	if cc.hook != nil {
		regs, err := cc.hook.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("hook init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}
	if cc.http != nil {
		regs, err := cc.http.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("http init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}
	if cc.service != nil {
		regs, err := cc.service.Initialize(emitter, params.Config, params.ExtensionRoot)
		if err != nil {
			return nil, fmt.Errorf("service init: %w", err)
		}
		mergeRegistrations(merged, regs)
	}

	return merged, nil
}

// mergeRegistrations appends all registrations from src into dst.
func mergeRegistrations(dst, src *protocol.Registrations) {
	if src == nil {
		return
	}
	dst.Tools = append(dst.Tools, src.Tools...)
	dst.HTTPRoutes = append(dst.HTTPRoutes, src.HTTPRoutes...)
	dst.Hooks = append(dst.Hooks, src.Hooks...)
	dst.Services = append(dst.Services, src.Services...)
	dst.Providers = append(dst.Providers, src.Providers...)
	dst.TTSProviders = append(dst.TTSProviders, src.TTSProviders...)
	dst.CLICommands = append(dst.CLICommands, src.CLICommands...)
}

// compositeDispatch routes a method call to the appropriate handler based on
// the method name. It tries each handler's methods in order.
func compositeDispatch(ctx context.Context, t *jsonrpc.Transport, cc *compositeConfig, req *protocol.Request) error {
	switch req.Method {
	// Channel methods
	case protocol.MethodChannelSend, protocol.MethodChannelTyping,
		protocol.MethodChannelReactionAdd, protocol.MethodChannelReactionRemove,
		protocol.MethodPromptUser, protocol.MethodChannelCapabilities:
		if cc.channel != nil {
			return dispatchChannel(ctx, t, cc.channel, req)
		}

	// Provider methods
	case protocol.MethodProviderCreateMessage, protocol.MethodProviderStreamMessage,
		protocol.MethodProviderCapabilities:
		if cc.provider != nil {
			return dispatchProvider(ctx, t, cc.provider, req)
		}

	// Tool methods
	case protocol.MethodToolCall:
		if cc.tool != nil {
			return dispatchTool(ctx, t, cc.tool, req)
		}

	// TTS methods
	case protocol.MethodTTSSynthesize, protocol.MethodTTSVoices:
		if cc.tts != nil {
			return dispatchTTS(ctx, t, cc.tts, req)
		}

	// Hook methods
	case protocol.MethodHookEvent:
		if cc.hook != nil {
			return dispatchHook(ctx, t, cc.hook, req)
		}

	// HTTP methods
	case protocol.MethodHTTPRequest:
		if cc.http != nil {
			return dispatchHTTP(ctx, t, cc.http, req)
		}

	// Service methods
	case protocol.MethodServiceStart, protocol.MethodServiceHealth, protocol.MethodServiceStop:
		if cc.service != nil {
			return dispatchService(ctx, t, cc.service, req)
		}
	}

	return t.SendError(req.ID, protocol.ErrCodeMethodNotFound, fmt.Sprintf("unknown method: %s", req.Method))
}

// compositeShutdown calls Shutdown on each registered handler, collecting errors.
func compositeShutdown(cc *compositeConfig) error {
	var errs []error

	if cc.channel != nil {
		if err := cc.channel.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("channel shutdown: %w", err))
		}
	}
	if cc.provider != nil {
		if err := cc.provider.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("provider shutdown: %w", err))
		}
	}
	if cc.tool != nil {
		if err := cc.tool.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("tool shutdown: %w", err))
		}
	}
	if cc.tts != nil {
		if err := cc.tts.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("tts shutdown: %w", err))
		}
	}
	if cc.hook != nil {
		if err := cc.hook.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("hook shutdown: %w", err))
		}
	}
	if cc.http != nil {
		if err := cc.http.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("http shutdown: %w", err))
		}
	}
	if cc.service != nil {
		if err := cc.service.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("service shutdown: %w", err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	// Combine multiple errors.
	msg := "multiple shutdown errors:"
	for _, e := range errs {
		msg += " " + e.Error() + ";"
	}
	return fmt.Errorf("%s", msg)
}
