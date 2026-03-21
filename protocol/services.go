package protocol

// ServiceStartParams is the payload for a "service_start" request.
type ServiceStartParams struct {
	ID     string         `json:"id"`
	Config map[string]any `json:"config,omitempty"`
}

// ServiceStartResult is the response payload for a "service_start" request.
type ServiceStartResult struct {
	OK bool `json:"ok"`
}

// ServiceHealthParams is the payload for a "service_health" request.
type ServiceHealthParams struct {
	ID string `json:"id"`
}

// ServiceHealthStatus enumerates the health states an extension can report for
// a background service.
type ServiceHealthStatus string

const (
	// ServiceHealthy indicates the service is operating normally.
	ServiceHealthy ServiceHealthStatus = "healthy"
	// ServiceDegraded indicates the service is operating in a reduced capacity.
	ServiceDegraded ServiceHealthStatus = "degraded"
	// ServiceUnhealthy indicates the service has failed or is unreachable.
	ServiceUnhealthy ServiceHealthStatus = "unhealthy"
)

// ServiceHealthResult is the response payload for a "service_health" request.
type ServiceHealthResult struct {
	Status  ServiceHealthStatus `json:"status"`
	Message string              `json:"message,omitempty"`
}

// ServiceStopParams is the payload for a "service_stop" request.
type ServiceStopParams struct {
	ID string `json:"id"`
}

// ServiceStopResult is the response payload for a "service_stop" request.
type ServiceStopResult struct {
	OK bool `json:"ok"`
}
