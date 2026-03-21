package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeHTTPExt is a minimal HTTPExtension for testing.
type fakeHTTPExt struct {
	routes     []protocol.HTTPRouteDef
	httpResult *protocol.HTTPResponseResult
	httpErr    error
}

func (f *fakeHTTPExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{HTTPRoutes: f.routes}, nil
}

func (f *fakeHTTPExt) HandleHTTPRequest(_ context.Context, _ protocol.HTTPRequestParams) (*protocol.HTTPResponseResult, error) {
	if f.httpErr != nil {
		return nil, f.httpErr
	}
	if f.httpResult != nil {
		return f.httpResult, nil
	}
	return &protocol.HTTPResponseResult{StatusCode: 200, Body: []byte("ok")}, nil
}

func (f *fakeHTTPExt) Shutdown() error { return nil }

// runHTTPExt starts RunHTTP in the background and returns a FakeKova wired to it.
func runHTTPExt(t *testing.T, ext extension.HTTPExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunHTTP(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunHTTP_FullLifecycle(t *testing.T) {
	ext := &fakeHTTPExt{
		routes: []protocol.HTTPRouteDef{{Path: "/webhook", Methods: []string{"POST"}}},
		httpResult: &protocol.HTTPResponseResult{
			StatusCode: 201,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"created":true}`),
		},
	}

	fk := runHTTPExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.HTTPRoutes) != 1 {
		t.Fatalf("got %d HTTP routes, want 1", len(regs.HTTPRoutes))
	}
	if regs.HTTPRoutes[0].Path != "/webhook" {
		t.Errorf("route path = %q, want /webhook", regs.HTTPRoutes[0].Path)
	}

	result := fk.Call(protocol.MethodHTTPRequest, protocol.HTTPRequestParams{
		Method:  "POST",
		Path:    "/webhook",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"event":"push"}`),
	})

	httpResult := harness.MustUnmarshal[protocol.HTTPResponseResult](t, result)
	if httpResult.StatusCode != 201 {
		t.Errorf("status_code = %d, want 201", httpResult.StatusCode)
	}
	if httpResult.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", httpResult.Headers["Content-Type"])
	}

	fk.Shutdown()
}

func TestRunHTTP_UnknownMethod(t *testing.T) {
	ext := &fakeHTTPExt{}
	fk := runHTTPExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("http_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRunHTTP_RequestError(t *testing.T) {
	ext := &fakeHTTPExt{
		httpErr: errors.New("handler failed"),
	}
	fk := runHTTPExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodHTTPRequest, protocol.HTTPRequestParams{
		Method: "GET",
		Path:   "/health",
	})
	if rpcErr.Code != protocol.ErrCodeHTTPFailed {
		t.Errorf("code = %d, want %d (ErrCodeHTTPFailed)", rpcErr.Code, protocol.ErrCodeHTTPFailed)
	}
	if rpcErr.Message != "handler failed" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "handler failed")
	}

	fk.Shutdown()
}
