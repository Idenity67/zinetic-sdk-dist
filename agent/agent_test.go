package agent

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

func TestRegister_Success(t *testing.T) {
	mt := &mockTransport{
		result: &RegisterResponse{
			Agent: &Agent{
				ID:        "agent-123",
				TenantID:  "tenant-1",
				Name:      "test-agent",
				AgentType: "llm",
				State:     "active",
			},
			AnchorID: "anchor-456",
		},
	}
	svc := NewService(mt)

	resp, err := svc.Register(context.Background(), &RegisterRequest{
		TenantID:  "tenant-1",
		Name:      "test-agent",
		AgentType: "llm",
		OwnerID:   "owner-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Agent.ID != "agent-123" {
		t.Fatalf("expected agent-123, got %s", resp.Agent.ID)
	}
	if resp.AnchorID != "anchor-456" {
		t.Fatalf("expected anchor-456, got %s", resp.AnchorID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/agents" {
		t.Fatalf("expected /v1/agents, got %s", mt.path)
	}
}

func TestRegister_MissingTenantID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		Name:      "test-agent",
		AgentType: "llm",
		OwnerID:   "owner-1",
	})
	if err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
}

func TestRegister_MissingName(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		TenantID:  "tenant-1",
		AgentType: "llm",
		OwnerID:   "owner-1",
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestRegister_MissingAgentType(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		TenantID: "tenant-1",
		Name:     "test-agent",
		OwnerID:  "owner-1",
	})
	if err == nil {
		t.Fatal("expected error for missing agent_type")
	}
}

func TestRegister_MissingOwnerID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Register(context.Background(), &RegisterRequest{
		TenantID:  "tenant-1",
		Name:      "test-agent",
		AgentType: "llm",
	})
	if err == nil {
		t.Fatal("expected error for missing owner_id")
	}
}

func TestGet_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Agent{
			ID:        "agent-123",
			TenantID:  "tenant-1",
			Name:      "my-agent",
			AgentType: "llm",
			State:     "active",
			CreatedAt: time.Now(),
		},
	}
	svc := NewService(mt)

	agent, err := svc.Get(context.Background(), "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.ID != "agent-123" {
		t.Fatalf("expected agent-123, got %s", agent.ID)
	}
	if mt.path != "/v1/agents/agent-123" {
		t.Fatalf("expected /v1/agents/agent-123, got %s", mt.path)
	}
}

func TestGet_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestGet_TransportError(t *testing.T) {
	mt := &mockTransport{
		err: fmt.Errorf("connection refused"),
	}
	svc := NewService(mt)

	_, err := svc.Get(context.Background(), "agent-123")
	if err == nil {
		t.Fatal("expected transport error to propagate")
	}
}

func TestUpdate_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Agent{
			ID:    "agent-123",
			Name:  "updated-name",
			State: "active",
		},
	}
	svc := NewService(mt)

	agent, err := svc.Update(context.Background(), "agent-123", &UpdateRequest{
		Name: "updated-name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent.Name != "updated-name" {
		t.Fatalf("expected updated-name, got %s", agent.Name)
	}
	if mt.method != "PATCH" {
		t.Fatalf("expected PATCH, got %s", mt.method)
	}
}

func TestUpdate_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.Update(context.Background(), "", &UpdateRequest{Name: "x"})
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestDecommission_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.Decommission(context.Background(), "agent-123", "end of life")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "DELETE" {
		t.Fatalf("expected DELETE, got %s", mt.method)
	}
	if mt.path != "/v1/agents/agent-123?mocked=true" {
		t.Fatalf("expected decommission path, got %s", mt.path)
	}
}

func TestDecommission_EmptyAgentID(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.Decommission(context.Background(), "", "reason")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestDecommission_EmptyReason(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.Decommission(context.Background(), "agent-123", "")
	if err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestList_Success(t *testing.T) {
	mt := &mockTransport{
		result: &ListResponse{
			Data:       []Agent{{ID: "a1"}, {ID: "a2"}},
			NextCursor: "cursor-abc",
			HasMore:    true,
		},
	}
	svc := NewService(mt)

	resp, err := svc.List(context.Background(), "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(resp.Data))
	}
	if resp.NextCursor != "cursor-abc" {
		t.Fatalf("expected cursor-abc, got %s", resp.NextCursor)
	}
	if mt.method != "GET" {
		t.Fatalf("expected GET, got %s", mt.method)
	}
}

func TestList_LimitCapped(t *testing.T) {
	mt := &mockTransport{result: &ListResponse{}}
	svc := NewService(mt)

	_, err := svc.List(context.Background(), "cur", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetCard_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &AgentCard{
			AgentID:  "agent-123",
			Name:     "card-agent",
			Endpoint: "https://example.com/agent",
		},
	}
	svc := NewService(mt)

	card, err := svc.GetCard(context.Background(), "agent-123")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if card != nil {
		t.Fatalf("expected nil card, got %#v", card)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported card retrieval should not call transport")
	}
}

func TestGetCard_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetCard(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestDelegate_Success(t *testing.T) {
	mt := &mockTransport{
		result: &DelegationResponse{
			DelegationID: "del-1",
			Chain:        []string{"a1", "a2"},
		},
	}
	svc := NewService(mt)

	resp, err := svc.Delegate(context.Background(), &DelegationRequest{
		FromAgentID:  "a1",
		ToAgentID:    "a2",
		Capabilities: []string{"read"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DelegationID != "del-1" {
		t.Fatalf("expected del-1, got %s", resp.DelegationID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/delegation/delegate" {
		t.Fatalf("expected delegation route, got %s", mt.path)
	}
}

func TestDelegate_MissingFromAgent(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Delegate(context.Background(), &DelegationRequest{
		ToAgentID:    "a2",
		Capabilities: []string{"read"},
	})
	if err == nil {
		t.Fatal("expected error for missing from_agent_id")
	}
}

func TestDelegate_MissingToAgent(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Delegate(context.Background(), &DelegationRequest{
		FromAgentID:  "a1",
		Capabilities: []string{"read"},
	})
	if err == nil {
		t.Fatal("expected error for missing to_agent_id")
	}
}

func TestDelegate_EmptyCapabilities(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Delegate(context.Background(), &DelegationRequest{
		FromAgentID: "a1",
		ToAgentID:   "a2",
	})
	if err == nil {
		t.Fatal("expected error for empty capabilities")
	}
}

func TestGetDelegationChain_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &DelegationChain{
			DelegationID: "del-1",
			Links:        []DelegationLink{{FromAgentID: "a1", ToAgentID: "a2", Depth: 1}},
		},
	}
	svc := NewService(mt)

	chain, err := svc.GetDelegationChain(context.Background(), "del-1")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if chain != nil {
		t.Fatalf("expected nil chain, got %#v", chain)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported delegation chain retrieval should not call transport")
	}
}

func TestGetDelegationChain_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetDelegationChain(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty delegation_id")
	}
}

func TestGetBaseline_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &BehavioralBaseline{
			AgentID:     "agent-123",
			APIPatterns: map[string]int{"GET /v1/agents": 100},
		},
	}
	svc := NewService(mt)

	bl, err := svc.GetBaseline(context.Background(), "agent-123")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if bl != nil {
		t.Fatalf("expected nil baseline, got %#v", bl)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported baseline retrieval should not call transport")
	}
}

func TestGetBaseline_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetBaseline(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestRecalibrateBaseline_Success(t *testing.T) {
	mt := &mockTransport{
		result: &BehavioralBaseline{AgentID: "agent-123"},
	}
	svc := NewService(mt)

	bl, err := svc.RecalibrateBaseline(context.Background(), "agent-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bl.AgentID != "agent-123" {
		t.Fatalf("expected agent-123, got %s", bl.AgentID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/agents/agent-123/recalibrate-baseline" {
		t.Fatalf("expected recalibrate route, got %s", mt.path)
	}
}

func TestRecalibrateBaseline_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RecalibrateBaseline(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestGetDriftEvents_Success(t *testing.T) {
	mt := &mockTransport{
		result: &struct {
			Data       []DriftEvent `json:"data"`
			NextCursor string       `json:"next_cursor"`
		}{
			Data:       []DriftEvent{{AgentID: "agent-123", Severity: "high"}},
			NextCursor: "next",
		},
	}
	svc := NewService(mt)

	events, cursor, err := svc.GetDriftEvents(context.Background(), "agent-123", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if cursor != "next" {
		t.Fatalf("expected next cursor, got %s", cursor)
	}
}

func TestGetDriftEvents_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, _, err := svc.GetDriftEvents(context.Background(), "", "", 10)
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestGetSnapshot_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &ForensicSnapshot{
			AgentID: "agent-123",
			Actions: []AgentAction{{ActionID: "act-1", Action: "read"}},
		},
	}
	svc := NewService(mt)

	snap, err := svc.GetSnapshot(context.Background(), "agent-123")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if snap != nil {
		t.Fatalf("expected nil snapshot, got %#v", snap)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported snapshot retrieval should not call transport")
	}
}

func TestGetSnapshot_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetSnapshot(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestKillSwitch_Success(t *testing.T) {
	mt := &mockTransport{
		result: &KillSwitchResponse{
			AgentsRevoked: 5,
			TokensRevoked: 12,
		},
	}
	svc := NewService(mt)

	resp, err := svc.KillSwitch(context.Background(), &KillSwitchRequest{
		TenantID: "t1",
		Reason:   "breach",
		Actor:    "admin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AgentsRevoked != 5 {
		t.Fatalf("expected 5 agents revoked, got %d", resp.AgentsRevoked)
	}
	if mt.path != "/v1/agents/kill-all" {
		t.Fatalf("expected kill-all path, got %s", mt.path)
	}
}

func TestKillSwitch_MissingTenantID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.KillSwitch(context.Background(), &KillSwitchRequest{
		Reason: "breach",
		Actor:  "admin",
	})
	if err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
}

func TestKillSwitch_MissingReason(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.KillSwitch(context.Background(), &KillSwitchRequest{
		TenantID: "t1",
		Actor:    "admin",
	})
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

func TestKillSwitch_MissingActor(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.KillSwitch(context.Background(), &KillSwitchRequest{
		TenantID: "t1",
		Reason:   "breach",
	})
	if err == nil {
		t.Fatal("expected error for missing actor")
	}
}

func TestInventory_Success(t *testing.T) {
	mt := &mockTransport{
		result: &InventoryResponse{
			TotalActive:    10,
			TotalSuspended: 2,
			TotalRevoked:   1,
		},
	}
	svc := NewService(mt)

	inv, err := svc.Inventory(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.TotalActive != 10 {
		t.Fatalf("expected 10 active, got %d", inv.TotalActive)
	}
	if mt.path != "/v1/agents/inventory" {
		t.Fatalf("expected inventory path, got %s", mt.path)
	}
}

func TestInventory_TransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("timeout")}
	svc := NewService(mt)

	_, err := svc.Inventory(context.Background())
	if err == nil {
		t.Fatal("expected transport error")
	}
}

func TestRegisterMCPServer_Success(t *testing.T) {
	mt := &mockTransport{
		result: &MCPServer{
			ID:       "srv-1",
			Name:     "my-server",
			Endpoint: "https://mcp.example.com",
		},
	}
	svc := NewService(mt)

	srv, err := svc.RegisterMCPServer(context.Background(), &MCPServer{
		Name:     "my-server",
		Endpoint: "https://mcp.example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv.ID != "srv-1" {
		t.Fatalf("expected srv-1, got %s", srv.ID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/v1/mcp/servers" {
		t.Fatalf("expected /v1/mcp/servers, got %s", mt.path)
	}
}

func TestRegisterMCPServer_MissingName(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RegisterMCPServer(context.Background(), &MCPServer{
		Endpoint: "https://mcp.example.com",
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestRegisterMCPServer_MissingEndpoint(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.RegisterMCPServer(context.Background(), &MCPServer{
		Name: "my-server",
	})
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestListMCPServers_Success(t *testing.T) {
	mt := &mockTransport{
		result: &MCPServerListResponse{
			Data:    []MCPServer{{ID: "srv-1"}},
			HasMore: false,
		},
	}
	svc := NewService(mt)

	resp, err := svc.ListMCPServers(context.Background(), "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 server, got %d", len(resp.Data))
	}
}

func TestAuthorizeMCPTool_Success(t *testing.T) {
	mt := &mockTransport{
		result: &MCPToolAuthResponse{
			Authorized: true,
			PolicyID:   "pol-1",
		},
	}
	svc := NewService(mt)

	resp, err := svc.AuthorizeMCPTool(context.Background(), &MCPToolAuthRequest{
		AgentID:   "a1",
		MCPServer: "srv-1",
		ToolName:  "read_file",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Authorized {
		t.Fatal("expected authorized=true")
	}
	if mt.path != "/v1/mcp/authorize" {
		t.Fatalf("expected authorize path, got %s", mt.path)
	}
}

func TestAuthorizeMCPTool_MissingAgentID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.AuthorizeMCPTool(context.Background(), &MCPToolAuthRequest{
		MCPServer: "srv-1",
		ToolName:  "tool",
	})
	if err == nil {
		t.Fatal("expected error for missing agent_id")
	}
}

func TestAuthorizeMCPTool_MissingServer(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.AuthorizeMCPTool(context.Background(), &MCPToolAuthRequest{
		AgentID:  "a1",
		ToolName: "tool",
	})
	if err == nil {
		t.Fatal("expected error for missing mcp_server")
	}
}

func TestAuthorizeMCPTool_MissingTool(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.AuthorizeMCPTool(context.Background(), &MCPToolAuthRequest{
		AgentID:   "a1",
		MCPServer: "srv-1",
	})
	if err == nil {
		t.Fatal("expected error for missing tool_name")
	}
}

func TestSuspend_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &Agent{ID: "agent-123", State: "suspended"},
	}
	svc := NewService(mt)

	agent, err := svc.Suspend(context.Background(), "agent-123", "policy violation")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if agent != nil {
		t.Fatalf("expected nil agent, got %#v", agent)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported suspend should not call transport")
	}
}

func TestSuspend_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Suspend(context.Background(), "", "reason")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestReactivate_Unsupported(t *testing.T) {
	mt := &mockTransport{
		result: &Agent{ID: "agent-123", State: "active"},
	}
	svc := NewService(mt)

	agent, err := svc.Reactivate(context.Background(), "agent-123")
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if agent != nil {
		t.Fatalf("expected nil agent, got %#v", agent)
	}
	if mt.callCount != 0 {
		t.Fatal("unsupported reactivate should not call transport")
	}
}

func TestReactivate_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.Reactivate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
}

func TestDeregisterMCPServer_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DeregisterMCPServer(context.Background(), "srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "DELETE" {
		t.Fatalf("expected DELETE, got %s", mt.method)
	}
	if mt.path != "/v1/mcp/servers/srv-1" {
		t.Fatalf("expected /v1/mcp/servers/srv-1, got %s", mt.path)
	}
}

func TestDeregisterMCPServer_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	err := svc.DeregisterMCPServer(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty server_id")
	}
}

func TestGetMCPServerTools_Success(t *testing.T) {
	mt := &mockTransport{
		result: []MCPTool{
			{Name: "read_file", Description: "Read a file"},
			{Name: "write_file", Description: "Write a file"},
		},
	}
	svc := NewService(mt)

	tools, err := svc.GetMCPServerTools(context.Background(), "srv-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if mt.path != "/v1/mcp/servers/srv-1/tools" {
		t.Fatalf("expected tools path, got %s", mt.path)
	}
}

func TestGetMCPServerTools_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.GetMCPServerTools(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty server_id")
	}
}

func TestCallMCPTool_Success(t *testing.T) {
	mt := &mockTransport{
		result: &MCPToolCallResponse{
			ID:      "call-1",
			AgentID: "a1",
			Tool:    "read_file",
		},
	}
	svc := NewService(mt)

	resp, err := svc.CallMCPTool(context.Background(), &MCPToolCallRequest{
		ServerID: "srv-1",
		AgentID:  "a1",
		Tool:     "read_file",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "call-1" {
		t.Fatalf("expected call-1, got %s", resp.ID)
	}
	if mt.path != "/v1/mcp/call" {
		t.Fatalf("expected /v1/mcp/call, got %s", mt.path)
	}
}

func TestCallMCPTool_MissingServerID(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.CallMCPTool(context.Background(), &MCPToolCallRequest{
		Tool: "read_file",
	})
	if err == nil {
		t.Fatal("expected error for missing server_id")
	}
}

func TestCallMCPTool_MissingTool(t *testing.T) {
	svc := NewService(&mockTransport{})
	_, err := svc.CallMCPTool(context.Background(), &MCPToolCallRequest{
		ServerID: "srv-1",
	})
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}
