package health

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

type mockTransport struct {
	method    string
	path      string
	body      interface{}
	result    interface{}
	err       error
	callCount int
}

func (m *mockTransport) Do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	m.method = method
	m.path = path
	m.body = body
	m.callCount++
	if m.err != nil {
		return m.err
	}
	if m.result != nil && result != nil {
		data, _ := json.Marshal(m.result)
		json.Unmarshal(data, result)
	}
	return nil
}

func (m *mockTransport) DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error {
	return m.Do(ctx, method, path, body, result)
}

func (m *mockTransport) BuildQueryURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}
	return path + "?mocked=true"
}

func TestHealth_Success(t *testing.T) {
	mt := &mockTransport{
		result: &HealthResponse{
			Status:  "healthy",
			Details: map[string]string{"db": "connected"},
		},
	}
	svc := NewService(mt)

	resp, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "healthy" {
		t.Fatalf("expected healthy, got %s", resp.Status)
	}
	if mt.method != "GET" {
		t.Fatalf("expected GET, got %s", mt.method)
	}
	if mt.path != "/health" {
		t.Fatalf("expected /health, got %s", mt.path)
	}
}

func TestHealth_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("connection refused")}
	svc := NewService(mt)

	_, err := svc.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReady_Success(t *testing.T) {
	mt := &mockTransport{
		result: &ReadyResponse{
			Status: "ready",
			Dependencies: map[string]DepCheck{
				"postgres": {Status: "healthy", Latency: "2ms"},
				"redis":    {Status: "healthy", Latency: "1ms"},
			},
		},
	}
	svc := NewService(mt)

	resp, err := svc.Ready(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ready" {
		t.Fatalf("expected ready, got %s", resp.Status)
	}
	if len(resp.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(resp.Dependencies))
	}
	if mt.path != "/ready" {
		t.Fatalf("expected /ready, got %s", mt.path)
	}
}

func TestVersion_Success(t *testing.T) {
	mt := &mockTransport{
		result: &VersionResponse{
			Version:   "1.2.3",
			CommitSHA: "abc123",
			BuildTime: "2026-01-01T00:00:00Z",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Version(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %s", resp.Version)
	}
	if resp.CommitSHA != "abc123" {
		t.Fatalf("expected abc123, got %s", resp.CommitSHA)
	}
	if mt.path != "/version" {
		t.Fatalf("expected /version, got %s", mt.path)
	}
}

func TestMetrics_Success(t *testing.T) {
	mt := &mockTransport{
		result: "# HELP http_requests_total\nhttp_requests_total 42\n",
	}
	svc := NewService(mt)

	_, err := svc.Metrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.path != "/metrics" {
		t.Fatalf("expected /metrics, got %s", mt.path)
	}
}

func TestMetrics_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("timeout")}
	svc := NewService(mt)

	_, err := svc.Metrics(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
