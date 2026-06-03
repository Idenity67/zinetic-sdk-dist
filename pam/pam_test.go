package pam

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
	q := path + "?"
	for k, v := range params {
		q += k + "=" + v + "&"
	}
	return q[:len(q)-1]
}

func TestRequestElevation_DurationClampsToMin30(t *testing.T) {
	mt := &mockTransport{result: &ElevationResponse{SessionID: "sess-1"}}
	svc := NewService(mt)

	req := &ElevationRequest{
		PrincipalID:    "user-123",
		TargetResource: "prod-db",
		AccessLevel:    "admin",
		Justification:  "emergency fix",
		Duration:       -5,
	}

	_, err := svc.RequestElevation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ElevationRequest
	json.Unmarshal(bodyData, &sent)
	if sent.Duration != 30 {
		t.Fatalf("expected duration defaulted to 30 for negative input, got %d", sent.Duration)
	}
}

func TestRequestElevation_DurationClampsToMax240(t *testing.T) {
	mt := &mockTransport{result: &ElevationResponse{SessionID: "sess-1"}}
	svc := NewService(mt)

	req := &ElevationRequest{
		PrincipalID:    "user-123",
		TargetResource: "prod-db",
		AccessLevel:    "admin",
		Justification:  "long migration",
		Duration:       500,
	}

	_, err := svc.RequestElevation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ElevationRequest
	json.Unmarshal(bodyData, &sent)
	if sent.Duration != 240 {
		t.Fatalf("expected duration clamped to 240 (maximum), got %d", sent.Duration)
	}
}

func TestRequestElevation_ZeroDurationDefaultsTo30(t *testing.T) {
	mt := &mockTransport{result: &ElevationResponse{SessionID: "sess-1"}}
	svc := NewService(mt)

	req := &ElevationRequest{
		PrincipalID:    "user-123",
		TargetResource: "prod-db",
		AccessLevel:    "admin",
		Justification:  "quick check",
		Duration:       0,
	}

	_, err := svc.RequestElevation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ElevationRequest
	json.Unmarshal(bodyData, &sent)
	if sent.Duration != 30 {
		t.Fatalf("expected 0 duration to default to 30, got %d", sent.Duration)
	}
}

func TestRequestElevation_NegativeDurationDefaultsTo30(t *testing.T) {
	mt := &mockTransport{result: &ElevationResponse{SessionID: "sess-1"}}
	svc := NewService(mt)

	req := &ElevationRequest{
		PrincipalID:    "user-123",
		TargetResource: "prod-db",
		AccessLevel:    "admin",
		Justification:  "test",
		Duration:       -10,
	}

	_, err := svc.RequestElevation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ElevationRequest
	json.Unmarshal(bodyData, &sent)
	if sent.Duration != 30 {
		t.Fatalf("expected negative duration to default to 30, got %d", sent.Duration)
	}
}

func TestRequestElevation_ValidDurationPassesThrough(t *testing.T) {
	mt := &mockTransport{result: &ElevationResponse{SessionID: "sess-1"}}
	svc := NewService(mt)

	req := &ElevationRequest{
		PrincipalID:    "user-123",
		TargetResource: "prod-db",
		AccessLevel:    "admin",
		Justification:  "scheduled maintenance",
		Duration:       120,
	}

	_, err := svc.RequestElevation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ElevationRequest
	json.Unmarshal(bodyData, &sent)
	if sent.Duration != 120 {
		t.Fatalf("expected duration 120 to pass through, got %d", sent.Duration)
	}
}

func TestRequestElevation_ShortPositiveDurationPassesThrough(t *testing.T) {
	mt := &mockTransport{result: &ElevationResponse{SessionID: "sess-1"}}
	svc := NewService(mt)

	req := &ElevationRequest{
		PrincipalID:    "user-123",
		TargetResource: "prod-db",
		AccessLevel:    "read",
		Justification:  "quick check",
		Duration:       5,
	}

	_, err := svc.RequestElevation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bodyData, _ := json.Marshal(mt.body)
	var sent ElevationRequest
	json.Unmarshal(bodyData, &sent)
	if sent.Duration != 5 {
		t.Fatalf("expected duration 5 to pass through (positive but < 30), got %d", sent.Duration)
	}
}

func TestRequestElevation_RequiresPrincipalID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestElevation(context.Background(), &ElevationRequest{
		TargetResource: "db",
		AccessLevel:    "admin",
		Justification:  "test",
	})
	if err == nil {
		t.Fatal("expected error for missing principal_id")
	}
}

func TestRequestElevation_RequiresTargetResource(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestElevation(context.Background(), &ElevationRequest{
		PrincipalID:   "user-1",
		AccessLevel:   "admin",
		Justification: "test",
	})
	if err == nil {
		t.Fatal("expected error for missing target_resource")
	}
}

func TestRequestElevation_RequiresAccessLevel(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestElevation(context.Background(), &ElevationRequest{
		PrincipalID:    "user-1",
		TargetResource: "db",
		Justification:  "test",
	})
	if err == nil {
		t.Fatal("expected error for missing access_level")
	}
}

func TestRequestElevation_RequiresJustification(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RequestElevation(context.Background(), &ElevationRequest{
		PrincipalID:    "user-1",
		TargetResource: "db",
		AccessLevel:    "admin",
	})
	if err == nil {
		t.Fatal("expected error for missing justification")
	}
}

func TestRevokeElevation_RequiresSessionID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.RevokeElevation(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}
}

func TestRevokeElevation_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)
	err := svc.RevokeElevation(context.Background(), "sess-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/pam/elevate/sess-123/revoke" {
		t.Fatalf("expected revoke path, got %s", mt.path)
	}
}

func TestGetSession_RequiresID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetSession(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}
}

func TestRequestElevation_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("timeout")}
	svc := NewService(mt)
	_, err := svc.RequestElevation(context.Background(), &ElevationRequest{
		PrincipalID:    "user-1",
		TargetResource: "db",
		AccessLevel:    "admin",
		Justification:  "test",
		Duration:       60,
	})
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestGetSession_Success(t *testing.T) {
	mt := &mockTransport{
		result: &ElevationSession{
			SessionID:      "sess-123",
			PrincipalID:    "user-1",
			TargetResource: "prod-db",
			Status:         "active",
		},
	}
	svc := NewService(mt)

	sess, err := svc.GetSession(context.Background(), "sess-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sess.SessionID != "sess-123" {
		t.Fatalf("expected sess-123, got %s", sess.SessionID)
	}
	if mt.path != "/pam/elevations/sess-123" {
		t.Fatalf("expected session path, got %s", mt.path)
	}
}

func TestListSessions_Success(t *testing.T) {
	mt := &mockTransport{
		result: &SessionListResponse{
			Data:    []ElevationSession{{SessionID: "s1"}, {SessionID: "s2"}},
			HasMore: true,
		},
	}
	svc := NewService(mt)

	resp, err := svc.ListSessions(context.Background(), "active", "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(resp.Data))
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
}

func TestListSessions_NoParams(t *testing.T) {
	mt := &mockTransport{result: &SessionListResponse{}}
	svc := NewService(mt)

	_, err := svc.ListSessions(context.Background(), "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.path != "/pam/elevations" {
		t.Fatalf("expected /pam/elevations, got %s", mt.path)
	}
}

func TestGetEphemeralCredential_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &EphemeralCredential{
			ResourceType: "database",
			ResourceID:   "prod-db-01",
		},
	}
	svc := NewService(mt)

	cred, err := svc.GetEphemeralCredential(context.Background(), "sess-1", "database", "prod-db-01")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if cred != nil {
		t.Fatalf("expected nil credential, got %#v", cred)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported credential retrieval should not call transport")
	}
}

func TestGetEphemeralCredential_MissingSessionID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetEphemeralCredential(context.Background(), "", "database", "db-1")
	if err == nil {
		t.Fatal("expected error for missing session_id")
	}
}

func TestGetEphemeralCredential_MissingResourceType(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetEphemeralCredential(context.Background(), "sess-1", "", "db-1")
	if err == nil {
		t.Fatal("expected error for missing resource_type")
	}
}

func TestGetEphemeralCredential_MissingResourceID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetEphemeralCredential(context.Background(), "sess-1", "database", "")
	if err == nil {
		t.Fatal("expected error for missing resource_id")
	}
}

func TestGetSessionRecording_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &SessionRecording{
			SessionID: "sess-1",
			Commands:  []Command{{Input: "ls"}},
		},
	}
	svc := NewService(mt)

	rec, err := svc.GetSessionRecording(context.Background(), "sess-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if rec != nil {
		t.Fatalf("expected nil recording, got %#v", rec)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported recording retrieval should not call transport")
	}
}

func TestGetSessionRecording_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetSessionRecording(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty session_id")
	}
}

func TestGetAnalytics_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &AnalyticsResponse{
			PrincipalID:      "user-1",
			TotalElevations:  42,
			AveragesDuration: 60,
		},
	}
	svc := NewService(mt)

	resp, err := svc.GetAnalytics(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil analytics, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported analytics retrieval should not call transport")
	}
}

func TestGetAnalytics_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetAnalytics(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty principal_id")
	}
}

func TestApproveElevation_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.ApproveElevation(context.Background(), "sess-1", "approver-1", "looks good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/pam/elevate/sess-1/approve" {
		t.Fatalf("expected approve path, got %s", mt.path)
	}
}

func TestApproveElevation_MissingSessionID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.ApproveElevation(context.Background(), "", "approver-1", "ok")
	if err == nil {
		t.Fatal("expected error for missing session_id")
	}
}

func TestApproveElevation_MissingApproverID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.ApproveElevation(context.Background(), "sess-1", "", "ok")
	if err == nil {
		t.Fatal("expected error for missing approver_id")
	}
}

func TestDenyElevation_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DenyElevation(context.Background(), "sess-1", "approver-1", "too risky")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.path != "/pam/elevate/sess-1/approve" {
		t.Fatalf("expected approve path, got %s", mt.path)
	}
}

func TestDenyElevation_MissingSessionID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.DenyElevation(context.Background(), "", "approver-1", "reason")
	if err == nil {
		t.Fatal("expected error for missing session_id")
	}
}

func TestDenyElevation_MissingApproverID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.DenyElevation(context.Background(), "sess-1", "", "reason")
	if err == nil {
		t.Fatal("expected error for missing approver_id")
	}
}
