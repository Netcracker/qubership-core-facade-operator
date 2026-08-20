package lib

import (
	"strings"

	"github.com/netcracker/qubership-core-lib-go-actuator-common/v2/tracing"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

// Paths that must not be exported to Jaeger (platform tracing contract:
// probes, metrics and management endpoints are excluded).
// actuator-common RateLimitingSampler already drops /health and /static*,
// but not /ready — which is hit by readiness/startup probes every few seconds.
var excludedTracePaths = map[string]struct{}{
	"/health":      {},
	"/ready":       {},
	"/livez":       {},
	"/readyz":      {},
	"/healthz":     {},
	"/liveness":    {},
	"/readiness":   {},
	"/prometheus":  {},
	"/metrics":     {},
	"/api-version": {},
}

type probeFilteringSampler struct {
	delegate sdktrace.Sampler
}

func newProbeFilteringSampler(maxTracesPerSecond float64) sdktrace.Sampler {
	return probeFilteringSampler{
		delegate: tracing.NewRateLimitingSampler(maxTracesPerSecond),
	}
}

func (s probeFilteringSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	psc := trace.SpanContextFromContext(p.ParentContext)
	if isExcludedTracePath(p.Name) || isExcludedTracePath(httpTargetFromSamplingParams(p)) {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.Drop,
			Tracestate: psc.TraceState(),
		}
	}
	return s.delegate.ShouldSample(p)
}

func (s probeFilteringSampler) Description() string {
	return "ProbeFilteringSampler{" + s.delegate.Description() + "}"
}

func isExcludedTracePath(path string) bool {
	if path == "" {
		return false
	}
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	_, ok := excludedTracePaths[path]
	return ok
}

func httpTargetFromSamplingParams(p sdktrace.SamplingParameters) string {
	for _, attr := range p.Attributes {
		if attr.Key == semconv.HTTPTargetKey {
			if val := attr.Value.AsString(); val != "" {
				return val
			}
		}
	}
	return ""
}
