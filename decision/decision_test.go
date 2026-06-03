package decision

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
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

func TestEvaluate_Success(t *testing.T) {
	mt := &mockTransport{
		result: &AuthResponse{
			Decision:      "allow",
			ReasonCode:    "policy_match",
			PolicyID:      "policy-abc",
			PolicyVersion: "v2",
			EvaluatedAt:   time.Now(),
		},
	}
	svc := NewService(mt)

	resp, err := svc.Evaluate(context.Background(), &AuthRequest{
		SubjectID: "agent-123",
		Context:   &DecisionContext{IP: "192.168.1.1"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Decision != "allow" {
		t.Fatalf("expected allow, got %s", resp.Decision)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/decision/auth" {
		t.Fatalf("expected /v1/decision/auth, got %s", mt.path)
	}
}

func TestEvaluate_MissingSubjectID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Evaluate(context.Background(), &AuthRequest{
		Context: &DecisionContext{IP: "1.2.3.4"},
	})
	if err == nil {
		t.Fatal("expected error for missing subject_id")
	}
}

func TestEvaluate_MissingContext(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Evaluate(context.Background(), &AuthRequest{
		SubjectID: "agent-123",
	})
	if err == nil {
		t.Fatal("expected error for missing context")
	}
}

func TestEvaluate_TransportError(t *testing.T) {
	mt := &mockTransport{
		err: fmt.Errorf("service unavailable"),
	}
	svc := NewService(mt)

	_, err := svc.Evaluate(context.Background(), &AuthRequest{
		SubjectID: "agent-123",
		Context:   &DecisionContext{IP: "1.2.3.4"},
	})
	if err == nil {
		t.Fatal("expected transport error to propagate")
	}
}

func TestBatchEvaluate_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &BatchAuthResponse{
			Responses: []AuthResponse{
				{Decision: "allow"},
				{Decision: "deny"},
			},
		},
	}
	svc := NewService(mt)

	resp, err := svc.BatchEvaluate(context.Background(), &BatchAuthRequest{
		Requests: []AuthRequest{
			{SubjectID: "agent-1", Context: &DecisionContext{IP: "1.1.1.1"}},
			{SubjectID: "agent-2", Context: &DecisionContext{IP: "2.2.2.2"}},
		},
	})

	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported batch evaluation should not call transport")
	}
}

func TestBatchEvaluate_EmptyRequests(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.BatchEvaluate(context.Background(), &BatchAuthRequest{
		Requests: []AuthRequest{},
	})
	if err == nil {
		t.Fatal("expected error for empty requests")
	}
}

func TestBatchEvaluate_ExceedsMaxSize(t *testing.T) {
	svc := NewService(&mockTransport{})

	requests := make([]AuthRequest, 101)
	for i := range requests {
		requests[i] = AuthRequest{SubjectID: fmt.Sprintf("agent-%d", i), Context: &DecisionContext{}}
	}

	_, err := svc.BatchEvaluate(context.Background(), &BatchAuthRequest{
		Requests: requests,
	})
	if err == nil {
		t.Fatal("expected error for batch size > 100")
	}
}

func TestBatchEvaluate_AtMaxSize(t *testing.T) {
	mt := &mockTransport{
		result: &BatchAuthResponse{
			Responses: make([]AuthResponse, 100),
		},
	}
	svc := NewService(mt)

	requests := make([]AuthRequest, 100)
	for i := range requests {
		requests[i] = AuthRequest{SubjectID: fmt.Sprintf("agent-%d", i), Context: &DecisionContext{}}
	}

	_, err := svc.BatchEvaluate(context.Background(), &BatchAuthRequest{
		Requests: requests,
	})
	if err == nil {
		t.Fatal("expected unsupported error for exactly 100 requests")
	}
}

func TestSimulate_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &SimulateResponse{
			Decision:  "deny",
			Simulated: true,
		},
	}
	svc := NewService(mt)

	resp, err := svc.Simulate(context.Background(), &SimulateRequest{
		Request: &AuthRequest{
			SubjectID: "agent-123",
			Context:   &DecisionContext{IP: "10.0.0.1"},
		},
		PolicyVersion: "v3-draft",
	})

	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported simulation should not call transport")
	}
}

func TestSimulate_NilRequest(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Simulate(context.Background(), &SimulateRequest{})
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestSimulate_MissingSubjectID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Simulate(context.Background(), &SimulateRequest{
		Request: &AuthRequest{
			Context: &DecisionContext{IP: "1.2.3.4"},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing subject_id in simulate")
	}
}
