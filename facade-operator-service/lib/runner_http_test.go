package lib

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpServerResponses(t *testing.T) {
	port := startHttpServer(t)
	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantBody        string
	}{
		{name: "health probe", path: "/health", wantStatus: http.StatusOK, wantContentType: "application/json", wantBody: `"ok"`},
		{name: "readiness probe", path: "/ready", wantStatus: http.StatusOK, wantContentType: "application/json", wantBody: `"ok"`},
		{name: "unknown path", path: "/missing", wantStatus: http.StatusNotFound, wantContentType: "text/plain; charset=utf-8", wantBody: "Not Found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := get(t, fmt.Sprintf("http://127.0.0.1:%d%s", port, tt.path))

			assert.Equal(t, tt.wantStatus, resp.status, "status")
			assert.Equal(t, tt.wantContentType, resp.contentType, "Content-Type")
			assert.Equal(t, tt.wantBody, resp.body, "body")
		})
	}
}

func TestHttpServerAcceptsIPv4AndIPv6(t *testing.T) {
	port := startHttpServer(t)
	for _, host := range []string{"127.0.0.1", "::1"} {
		t.Run(host, func(t *testing.T) {
			if host == "::1" && !ipv6LoopbackAvailable() {
				t.Skip("IPv6 loopback is not available on this machine")
			}
			url := fmt.Sprintf("http://%s/health", net.JoinHostPort(host, strconv.Itoa(port)))

			resp := get(t, url)

			assert.Equal(t, http.StatusOK, resp.status, "status of GET %s", url)
		})
	}
}

type httpResult struct {
	status      int
	contentType string
	body        string
}

func startHttpServer(t *testing.T) int {
	t.Helper()
	registerFiberSecurityForTests()
	usePrometheusRegistry(t)
	port := freePort(t)
	t.Setenv("HTTP_SERVER_BIND", ":"+strconv.Itoa(port))
	t.Setenv("PPROF_PORT", "0")
	t.Setenv("MICROSERVICE_NAME", "facade-operator")
	t.Setenv("MICROSERVICE_NAMESPACE", "test")
	configloader.Init(configloader.EnvPropertySource())
	app, err := newHttpApp()
	require.NoError(t, err)
	go listenHttp(app)
	t.Cleanup(func() { _ = app.Shutdown() })
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		resp, err := http.Get(url)
		if assert.NoError(c, err, "GET %s", url) {
			_ = resp.Body.Close()
		}
	}, 10*time.Second, 50*time.Millisecond)
	return port
}

func get(t *testing.T, url string) httpResult {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	require.NoError(t, err, "GET %s", url)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "read body of GET %s", url)
	return httpResult{status: resp.StatusCode, contentType: resp.Header.Get("Content-Type"), body: string(body)}
}

func usePrometheusRegistry(t *testing.T) {
	t.Helper()
	registry := prometheus.NewRegistry()
	previousRegisterer, previousGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = registry, registry
	t.Cleanup(func() {
		prometheus.DefaultRegisterer, prometheus.DefaultGatherer = previousRegisterer, previousGatherer
	})
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func ipv6LoopbackAvailable() bool {
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
