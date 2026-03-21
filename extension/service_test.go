package extension_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kova-land/extension-sdk-go/extension"
	"github.com/kova-land/extension-sdk-go/harness"
	"github.com/kova-land/extension-sdk-go/protocol"
)

// fakeServiceExt is a minimal ServiceExtension for testing.
type fakeServiceExt struct {
	services     []protocol.ServiceDef
	startResult  *protocol.ServiceStartResult
	startErr     error
	healthResult *protocol.ServiceHealthResult
	healthErr    error
	stopResult   *protocol.ServiceStopResult
	stopErr      error
}

func (f *fakeServiceExt) Initialize(_ *extension.Emitter, _ map[string]any, _ string) (*protocol.Registrations, error) {
	return &protocol.Registrations{Services: f.services}, nil
}

func (f *fakeServiceExt) HandleServiceStart(_ context.Context, _ protocol.ServiceStartParams) (*protocol.ServiceStartResult, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	if f.startResult != nil {
		return f.startResult, nil
	}
	return &protocol.ServiceStartResult{OK: true}, nil
}

func (f *fakeServiceExt) HandleServiceHealth(_ context.Context, _ protocol.ServiceHealthParams) (*protocol.ServiceHealthResult, error) {
	if f.healthErr != nil {
		return nil, f.healthErr
	}
	if f.healthResult != nil {
		return f.healthResult, nil
	}
	return &protocol.ServiceHealthResult{Status: protocol.ServiceHealthy}, nil
}

func (f *fakeServiceExt) HandleServiceStop(_ context.Context, _ protocol.ServiceStopParams) (*protocol.ServiceStopResult, error) {
	if f.stopErr != nil {
		return nil, f.stopErr
	}
	if f.stopResult != nil {
		return f.stopResult, nil
	}
	return &protocol.ServiceStopResult{OK: true}, nil
}

func (f *fakeServiceExt) Shutdown() error { return nil }

// runServiceExt starts RunService in the background and returns a FakeKova wired to it.
func runServiceExt(t *testing.T, ext extension.ServiceExtension) *harness.FakeKova {
	t.Helper()
	fk := harness.NewFakeKova(t)
	go func() {
		_ = extension.RunService(ext,
			extension.WithStdin(fk.ExtStdin),
			extension.WithStdout(fk.ExtStdout),
		)
	}()
	return fk
}

func TestRunService_FullLifecycle(t *testing.T) {
	ext := &fakeServiceExt{
		services: []protocol.ServiceDef{{ID: "metrics-collector", Description: "collects metrics"}},
		healthResult: &protocol.ServiceHealthResult{
			Status:  protocol.ServiceHealthy,
			Message: "all good",
		},
	}

	fk := runServiceExt(t, ext)

	regs := fk.Initialize(nil)
	if len(regs.Services) != 1 {
		t.Fatalf("got %d services, want 1", len(regs.Services))
	}
	if regs.Services[0].ID != "metrics-collector" {
		t.Errorf("service ID = %q, want metrics-collector", regs.Services[0].ID)
	}

	// Test service_start.
	startRaw := fk.Call(protocol.MethodServiceStart, protocol.ServiceStartParams{
		ID:     "metrics-collector",
		Config: map[string]any{"interval": "30s"},
	})
	startResult := harness.MustUnmarshal[protocol.ServiceStartResult](t, startRaw)
	if !startResult.OK {
		t.Error("start OK = false, want true")
	}

	// Test service_health.
	healthRaw := fk.Call(protocol.MethodServiceHealth, protocol.ServiceHealthParams{
		ID: "metrics-collector",
	})
	healthResult := harness.MustUnmarshal[protocol.ServiceHealthResult](t, healthRaw)
	if healthResult.Status != protocol.ServiceHealthy {
		t.Errorf("health status = %q, want healthy", healthResult.Status)
	}
	if healthResult.Message != "all good" {
		t.Errorf("health message = %q, want %q", healthResult.Message, "all good")
	}

	// Test service_stop.
	stopRaw := fk.Call(protocol.MethodServiceStop, protocol.ServiceStopParams{
		ID: "metrics-collector",
	})
	stopResult := harness.MustUnmarshal[protocol.ServiceStopResult](t, stopRaw)
	if !stopResult.OK {
		t.Error("stop OK = false, want true")
	}

	fk.Shutdown()
}

func TestRunService_UnknownMethod(t *testing.T) {
	ext := &fakeServiceExt{}
	fk := runServiceExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError("service_unknown", nil)
	if rpcErr.Code != protocol.ErrCodeMethodNotFound {
		t.Errorf("code = %d, want %d (ErrCodeMethodNotFound)", rpcErr.Code, protocol.ErrCodeMethodNotFound)
	}

	fk.Shutdown()
}

func TestRunService_StartError(t *testing.T) {
	ext := &fakeServiceExt{
		startErr: errors.New("failed to start"),
	}
	fk := runServiceExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodServiceStart, protocol.ServiceStartParams{
		ID: "broken",
	})
	if rpcErr.Code != protocol.ErrCodeServiceFailed {
		t.Errorf("code = %d, want %d (ErrCodeServiceFailed)", rpcErr.Code, protocol.ErrCodeServiceFailed)
	}
	if rpcErr.Message != "failed to start" {
		t.Errorf("message = %q, want %q", rpcErr.Message, "failed to start")
	}

	fk.Shutdown()
}

func TestRunService_HealthError(t *testing.T) {
	ext := &fakeServiceExt{
		healthErr: errors.New("health check failed"),
	}
	fk := runServiceExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodServiceHealth, protocol.ServiceHealthParams{
		ID: "broken",
	})
	if rpcErr.Code != protocol.ErrCodeServiceFailed {
		t.Errorf("code = %d, want %d (ErrCodeServiceFailed)", rpcErr.Code, protocol.ErrCodeServiceFailed)
	}

	fk.Shutdown()
}

func TestRunService_StopError(t *testing.T) {
	ext := &fakeServiceExt{
		stopErr: errors.New("failed to stop"),
	}
	fk := runServiceExt(t, ext)
	fk.Initialize(nil)

	rpcErr := fk.CallExpectError(protocol.MethodServiceStop, protocol.ServiceStopParams{
		ID: "stuck",
	})
	if rpcErr.Code != protocol.ErrCodeServiceFailed {
		t.Errorf("code = %d, want %d (ErrCodeServiceFailed)", rpcErr.Code, protocol.ErrCodeServiceFailed)
	}

	fk.Shutdown()
}
