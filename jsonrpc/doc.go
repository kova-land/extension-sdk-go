// Package jsonrpc provides a low-level JSON-RPC 2.0 transport for the Kova
// Extension Protocol. It reads newline-delimited JSON (NDJSON) from an
// io.Reader and writes to an io.Writer, handling request parsing, response
// serialization, and notification emission.
//
// Extension authors typically use the higher-level extension package instead of
// this package directly.
package jsonrpc
