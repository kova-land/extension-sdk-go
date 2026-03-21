// Package harness provides test utilities for Kova extensions. The FakeKova
// type simulates the kova host, allowing extension authors to test their
// implementations without running a real kova instance.
//
// FakeKova communicates with the extension over in-memory pipes using the
// same NDJSON JSON-RPC 2.0 protocol that kova uses in production. This ensures
// that tests exercise the full serialization/deserialization path.
package harness
