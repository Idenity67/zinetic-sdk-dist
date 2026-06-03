package audit

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

func (s *Service) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	params := map[string]string{}

	if req.TimeRangeStart != nil {
		params["time_range_start"] = req.TimeRangeStart.UTC().Format("2006-01-02T15:04:05Z")
	}
	if req.TimeRangeEnd != nil {
		params["time_range_end"] = req.TimeRangeEnd.UTC().Format("2006-01-02T15:04:05Z")
	}
	if req.ActorID != "" {
		params["actor_id"] = req.ActorID
	}
	if req.ActorType != "" {
		params["actor_type"] = req.ActorType
	}
	if req.Action != "" {
		params["action"] = req.Action
	}
	if req.ResourceType != "" {
		params["resource_type"] = req.ResourceType
	}
	if req.ResourceID != "" {
		params["resource_id"] = req.ResourceID
	}
	if req.Outcome != "" {
		params["outcome"] = req.Outcome
	}
	if req.CorrelationID != "" {
		params["correlation_id"] = req.CorrelationID
	}
	if req.Cursor != "" {
		params["cursor"] = req.Cursor
	}
	if req.Limit > 0 {
		if req.Limit > 200 {
			req.Limit = 200
		}
		params["limit"] = fmt.Sprintf("%d", req.Limit)
	}

	path := s.transport.BuildQueryURL("/v1/audit-logs", params)

	var resp SearchResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetEvent(ctx context.Context, eventID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}

	eventID, err := pathutil.Segment("event_id", eventID)
	if err != nil {
		return nil, err
	}

	var resp Event
	err = s.transport.Do(ctx, "GET", fmt.Sprintf("/v1/audit-logs/%s", eventID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) CreateSIEMConfig(ctx context.Context, req *SIEMConfigRequest) (*SIEMConfigResponse, error) {
	if req.TargetType == "" {
		return nil, fmt.Errorf("target_type is required")
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if req.Format == "" {
		return nil, fmt.Errorf("format is required")
	}

	return nil, fmt.Errorf("SIEM config creation is not supported by the current backend OpenAPI contract")
}

func (s *Service) ListSIEMConfigs(ctx context.Context) ([]SIEMConfigResponse, error) {
	return nil, fmt.Errorf("SIEM config listing is not supported by the current backend OpenAPI contract")
}

func (s *Service) DeleteSIEMConfig(ctx context.Context, configID string) error {
	if configID == "" {
		return fmt.Errorf("config_id is required")
	}
	return fmt.Errorf("SIEM config deletion is not supported by the current backend OpenAPI contract")
}
