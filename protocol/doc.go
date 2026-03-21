// Package protocol defines the JSON-RPC 2.0 message types and domain-specific
// parameter/result types for the Kova Extension Protocol.
//
// These types are the canonical Go representation of the kova-bridge-v1 protocol.
// They are copied from the kova source (internal/extension/bridge/protocol.go)
// and kept in sync. Extension authors use these types directly when implementing
// extension interfaces or working with the low-level JSON-RPC transport.
//
// The types are organized by domain:
//   - types.go: Core JSON-RPC envelope types (Request, Response, Notification, RPCError)
//   - constants.go: Method names, notification names, error codes
//   - initialize.go: Initialize handshake, Registrations, CLIDef, CallContext
//   - channel.go: Channel extension param/result types
//   - provider.go: Provider extension param/result types and content blocks
//   - tools.go: Tool extension param/result types
//   - services.go: Service lifecycle param/result types
//   - hooks.go: Hook event param/result types
//   - http.go: HTTP route param/result types
//   - tts.go: TTS provider param/result types
//   - log.go: Log notification param types
//
//go:generate go run ../cmd/generate-schemas/
package protocol
