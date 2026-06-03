package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	q := path + "?"
	for k, v := range params {
		q += k + "=" + v + "&"
	}
	return q[:len(q)-1]
}

func TestSearch_Success(t *testing.T) {
	now := time.Now()
	mt := &mockTransport{
		result: &SearchResponse{
			Data: []Event{
				{
					EventID:   "evt-001",
					Timestamp: now,
					Action:    "credential.issued",
					Outcome:   "success",
				},
			},
			HasMore:    true,
			NextCursor: "cursor-abc",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Search(context.Background(), &SearchRequest{
		ActorID: "user-123",
		Action:  "credential.issued",
		Limit:   50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Data))
	}
	if resp.Data[0].EventID != "evt-001" {
		t.Fatalf("expected evt-001, got %s", resp.Data[0].EventID)
	}
	if !resp.HasMore {
		t.Fatal("expected has_more=true")
	}
	if mt.method != "GET" {
		t.Fatalf("expected GET, got %s", mt.method)
	}
	if !strings.HasPrefix(mt.path, "/v1/audit-logs") {
		t.Fatalf("expected audit logs route, got %s", mt.path)
	}
}

func TestSearch_LimitCappedAt200(t *testing.T) {
	mt := &mockTransport{result: &SearchResponse{}}
	svc := NewService(mt)

	req := &SearchRequest{Limit: 500}
	_, err := svc.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Limit != 200 {
		t.Fatalf("expected limit capped at 200, got %d", req.Limit)
	}
}

func TestSearch_TimeRangeParams(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	mt := &mockTransport{result: &SearchResponse{}}
	svc := NewService(mt)

	_, err := svc.Search(context.Background(), &SearchRequest{
		TimeRangeStart: &start,
		TimeRangeEnd:   &end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetEvent_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Event{
			EventID: "evt-002",
			Action:  "nhi.rotated",
			Outcome: "success",
		},
	}
	svc := NewService(mt)

	resp, err := svc.GetEvent(context.Background(), "evt-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.EventID != "evt-002" {
		t.Fatalf("expected evt-002, got %s", resp.EventID)
	}
	if mt.path != "/v1/audit-logs/evt-002" {
		t.Fatalf("expected /v1/audit-logs/evt-002, got %s", mt.path)
	}
}

func TestGetEvent_EscapesID(t *testing.T) {
	mt := &mockTransport{result: &Event{EventID: "evt #2"}}
	svc := NewService(mt)

	_, err := svc.GetEvent(context.Background(), "evt #2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.path != "/v1/audit-logs/evt%20%232" {
		t.Fatalf("unexpected escaped path: %s", mt.path)
	}
}

func TestGetEvent_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetEvent(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty event_id")
	}
}

func TestCreateSIEMConfig_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &SIEMConfigResponse{
			ID:         "siem-001",
			TargetType: "splunk",
			Endpoint:   "https://splunk.example.com",
			Format:     ExportFormatCEF,
			Status:     "active",
		},
	}
	svc := NewService(mt)

	resp, err := svc.CreateSIEMConfig(context.Background(), &SIEMConfigRequest{
		TargetType: "splunk",
		Endpoint:   "https://splunk.example.com",
		Format:     ExportFormatCEF,
	})
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported SIEM config creation should not call transport")
	}
}

func TestCreateSIEMConfig_MissingTargetType(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateSIEMConfig(context.Background(), &SIEMConfigRequest{
		Endpoint: "https://splunk.example.com",
		Format:   ExportFormatCEF,
	})
	if err == nil {
		t.Fatal("expected error for missing target_type")
	}
}

func TestCreateSIEMConfig_MissingEndpoint(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateSIEMConfig(context.Background(), &SIEMConfigRequest{
		TargetType: "splunk",
		Format:     ExportFormatCEF,
	})
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestCreateSIEMConfig_MissingFormat(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateSIEMConfig(context.Background(), &SIEMConfigRequest{
		TargetType: "splunk",
		Endpoint:   "https://splunk.example.com",
	})
	if err == nil {
		t.Fatal("expected error for missing format")
	}
}

func TestListSIEMConfigs_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: struct {
			Data []SIEMConfigResponse `json:"data"`
		}{
			Data: []SIEMConfigResponse{
				{ID: "siem-001", Status: "active"},
				{ID: "siem-002", Status: "paused"},
			},
		},
	}
	svc := NewService(mt)

	resp, err := svc.ListSIEMConfigs(context.Background())
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported SIEM config listing should not call transport")
	}
}

func TestDeleteSIEMConfig_Unsupported(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DeleteSIEMConfig(context.Background(), "siem-001")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported SIEM config deletion should not call transport")
	}
}

func TestDeleteSIEMConfig_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.DeleteSIEMConfig(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty config_id")
	}
}

func TestSearch_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("network error")}
	svc := NewService(mt)

	_, err := svc.Search(context.Background(), &SearchRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
