package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

func TestProbeFilteringSamplerDropsProbesAndManagement(t *testing.T) {
	sampler := newProbeFilteringSampler(10)

	for _, path := range []string{"/health", "/ready", "/prometheus", "/api-version", "/metrics", "/ready?x=1"} {
		decision := sampler.ShouldSample(sdktrace.SamplingParameters{
			Name: "/",
			Attributes: []attribute.KeyValue{
				semconv.HTTPTargetKey.String(path),
			},
		}).Decision
		assert.Equal(t, sdktrace.Drop, decision, "expected drop for %s", path)
	}

	decision := sampler.ShouldSample(sdktrace.SamplingParameters{
		Name: "/ready",
	}).Decision
	assert.Equal(t, sdktrace.Drop, decision)
}

func TestProbeFilteringSamplerKeepsBusinessPaths(t *testing.T) {
	sampler := newProbeFilteringSampler(10)

	decision := sampler.ShouldSample(sdktrace.SamplingParameters{
		Name: "/api/v1/something",
		Attributes: []attribute.KeyValue{
			semconv.HTTPTargetKey.String("/api/v1/something"),
		},
	}).Decision
	assert.Equal(t, sdktrace.RecordAndSample, decision)
}

func TestIsExcludedTracePath(t *testing.T) {
	assert.True(t, isExcludedTracePath("/ready"))
	assert.True(t, isExcludedTracePath("/ready?foo=bar"))
	assert.False(t, isExcludedTracePath("/api/facades"))
	assert.False(t, isExcludedTracePath(""))
}
