package agent

import (
	"context"
	"fmt"

	"sdk.zinetic.net/internal/pathutil"
)

type Transport interface {
	Do(ctx context.Context, method, path string, body interface{}, result interface{}) error
	DoWithHeaders(ctx context.Context, method, path string, body interface{}, result interface{}, headers map[string]string) error
	BuildQueryURL(path string, params map[string]string) string
}

type Service struct {
	transport Transport
}

func NewService(t Transport) *Service {
	return &Service{transport: t}
}

func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.AgentType == "" {
		return nil, fmt.Errorf("agent_type is required")
	}
	if req.OwnerID == "" {
		return nil, fmt.Errorf("owner_id is required")
	}

	var resp RegisterResponse
	err := s.transport.Do(ctx, "POST", "/v1/agents", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Get(ctx context.Context, agentID string) (*Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	agentID, err := pathutil.Segment("agent_id", agentID)
	if err != nil {
		return nil, err
	}

	var resp Agent
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/agents/%s", agentID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, agentID string, req *UpdateRequest) (*Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	agentID, err := pathutil.Segment("agent_id", agentID)
	if err != nil {
		return nil, err
	}

	var resp Agent
	err = s.transport.Do(ctx, "PATCH", fmt.Sprintf("/v1/agents/%s", agentID), req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Decommission(ctx context.Context, agentID string, reason string) error {
	if agentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if reason == "" {
		return fmt.Errorf("reason is required")
	}

	agentID, err := pathutil.Segment("agent_id", agentID)
	if err != nil {
		return err
	}
	path := s.transport.BuildQueryURL(fmt.Sprintf("/v1/agents/%s", agentID), map[string]string{"reason": reason})
	return s.transport.Do(ctx, "DELETE", path, nil, nil)
}

func (s *Service) List(ctx context.Context, cursor string, limit int) (*ListResponse, error) {
	params := map[string]string{}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	path := s.transport.BuildQueryURL("/v1/agents", params)

	var resp ListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetCard(ctx context.Context, agentID string) (*AgentCard, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	return nil, fmt.Errorf("agent card retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) Delegate(ctx context.Context, req *DelegationRequest) (*DelegationResponse, error) {
	if req.FromAgentID == "" {
		return nil, fmt.Errorf("from_agent_id is required")
	}
	if req.ToAgentID == "" {
		return nil, fmt.Errorf("to_agent_id is required")
	}
	if len(req.Capabilities) == 0 {
		return nil, fmt.Errorf("at least one capability is required")
	}

	var resp DelegationResponse
	err := s.transport.Do(ctx, "POST", "/v1/delegation/delegate", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetDelegationChain(ctx context.Context, delegationID string) (*DelegationChain, error) {
	if delegationID == "" {
		return nil, fmt.Errorf("delegation_id is required")
	}

	return nil, fmt.Errorf("delegation chain retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) GetBaseline(ctx context.Context, agentID string) (*BehavioralBaseline, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	return nil, fmt.Errorf("behavioral baseline retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) RecalibrateBaseline(ctx context.Context, agentID string) (*BehavioralBaseline, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	agentID, err := pathutil.Segment("agent_id", agentID)
	if err != nil {
		return nil, err
	}

	var resp BehavioralBaseline
	err = s.transport.Do(ctx, "POST", fmt.Sprintf("/v1/agents/%s/recalibrate-baseline", agentID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetDriftEvents(ctx context.Context, agentID string, cursor string, limit int) ([]DriftEvent, string, error) {
	if agentID == "" {
		return nil, "", fmt.Errorf("agent_id is required")
	}

	params := map[string]string{}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	agentID, err := pathutil.Segment("agent_id", agentID)
	if err != nil {
		return nil, "", err
	}

	path := s.transport.BuildQueryURL(fmt.Sprintf("/v1/agents/%s/drift-history", agentID), params)

	var resp struct {
		Data       []DriftEvent `json:"data"`
		NextCursor string       `json:"next_cursor"`
	}
	err = s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, "", err
	}
	return resp.Data, resp.NextCursor, nil
}

func (s *Service) GetSnapshot(ctx context.Context, agentID string) (*ForensicSnapshot, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	return nil, fmt.Errorf("forensic snapshot retrieval is not supported by the current backend OpenAPI contract")
}

func (s *Service) KillSwitch(ctx context.Context, req *KillSwitchRequest) (*KillSwitchResponse, error) {
	if req.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if req.Reason == "" {
		return nil, fmt.Errorf("reason is required")
	}
	if req.Actor == "" {
		return nil, fmt.Errorf("actor is required")
	}

	var resp KillSwitchResponse
	err := s.transport.Do(ctx, "POST", "/v1/agents/kill-all", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Inventory(ctx context.Context) (*InventoryResponse, error) {
	var resp InventoryResponse
	err := s.transport.Do(ctx, "GET", "/v1/agents/inventory", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) RegisterMCPServer(ctx context.Context, server *MCPServer) (*MCPServer, error) {
	if server.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if server.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	var resp MCPServer
	err := s.transport.Do(ctx, "POST", "/v1/mcp/servers", server, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) ListMCPServers(ctx context.Context, cursor string, limit int) (*MCPServerListResponse, error) {
	params := map[string]string{}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if limit > 0 {
		if limit > 200 {
			limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", limit)
	}

	path := s.transport.BuildQueryURL("/v1/mcp/servers", params)

	var resp MCPServerListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) AuthorizeMCPTool(ctx context.Context, req *MCPToolAuthRequest) (*MCPToolAuthResponse, error) {
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if req.MCPServer == "" {
		return nil, fmt.Errorf("mcp_server is required")
	}
	if req.ToolName == "" {
		return nil, fmt.Errorf("tool_name is required")
	}

	var resp MCPToolAuthResponse
	err := s.transport.Do(ctx, "POST", "/v1/mcp/authorize", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Suspend(ctx context.Context, agentID string, reason string) (*Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("reason is required")
	}

	return nil, fmt.Errorf("agent suspend is not supported by the current backend OpenAPI contract")
}

func (s *Service) Reactivate(ctx context.Context, agentID string) (*Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	return nil, fmt.Errorf("agent reactivate is not supported by the current backend OpenAPI contract")
}

func (s *Service) DeregisterMCPServer(ctx context.Context, serverID string) error {
	if serverID == "" {
		return fmt.Errorf("server_id is required")
	}
	serverID, err := pathutil.Segment("server_id", serverID)
	if err != nil {
		return err
	}
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/v1/mcp/servers/%s", serverID), nil, nil)
}

func (s *Service) GetMCPServerTools(ctx context.Context, serverID string) ([]MCPTool, error) {
	if serverID == "" {
		return nil, fmt.Errorf("server_id is required")
	}
	serverID, err := pathutil.Segment("server_id", serverID)
	if err != nil {
		return nil, err
	}
	var resp []MCPTool
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/mcp/servers/%s/tools", serverID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) CallMCPTool(ctx context.Context, req *MCPToolCallRequest) (*MCPToolCallResponse, error) {
	if req.ServerID == "" {
		return nil, fmt.Errorf("server_id is required")
	}
	if req.Tool == "" {
		return nil, fmt.Errorf("tool is required")
	}
	var resp MCPToolCallResponse
	err := s.transport.Do(ctx, "POST", "/v1/mcp/call", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
