package scim

import (
	"context"
	"fmt"
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

func (s *Service) CreateUser(ctx context.Context, user *User) (*User, error) {
	if user.UserName == "" {
		return nil, fmt.Errorf("userName is required")
	}

	if len(user.Schemas) == 0 {
		user.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}
	}

	var resp User
	err := s.transport.Do(ctx, "POST", "/scim/v2/Users", user, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	var resp User
	err := s.transport.Do(ctx, "GET", fmt.Sprintf("/scim/v2/Users/%s", userID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) UpdateUser(ctx context.Context, userID string, user *User) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	var resp User
	err := s.transport.Do(ctx, "PUT", fmt.Sprintf("/scim/v2/Users/%s", userID), user, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) PatchUser(ctx context.Context, userID string, patch *PatchOp) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if len(patch.Schemas) == 0 {
		patch.Schemas = []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"}
	}

	var resp User
	err := s.transport.Do(ctx, "PATCH", fmt.Sprintf("/scim/v2/Users/%s", userID), patch, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/scim/v2/Users/%s", userID), nil, nil)
}

func (s *Service) ListUsers(ctx context.Context, filter string, cursor string, count int) (*UserListResponse, error) {
	params := map[string]string{}
	if filter != "" {
		params["filter"] = filter
	}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if count > 0 {
		if count > 200 {
			count = 200
		}
		params["count"] = fmt.Sprintf("%d", count)
	}

	path := s.transport.BuildQueryURL("/scim/v2/Users", params)

	var resp UserListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) CreateGroup(ctx context.Context, group *Group) (*Group, error) {
	if group.DisplayName == "" {
		return nil, fmt.Errorf("displayName is required")
	}

	if len(group.Schemas) == 0 {
		group.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:Group"}
	}

	var resp Group
	err := s.transport.Do(ctx, "POST", "/scim/v2/Groups", group, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	if groupID == "" {
		return nil, fmt.Errorf("group_id is required")
	}

	var resp Group
	err := s.transport.Do(ctx, "GET", fmt.Sprintf("/scim/v2/Groups/%s", groupID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) UpdateGroup(ctx context.Context, groupID string, group *Group) (*Group, error) {
	if groupID == "" {
		return nil, fmt.Errorf("group_id is required")
	}

	var resp Group
	err := s.transport.Do(ctx, "PUT", fmt.Sprintf("/scim/v2/Groups/%s", groupID), group, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) PatchGroup(ctx context.Context, groupID string, patch *PatchOp) (*Group, error) {
	if groupID == "" {
		return nil, fmt.Errorf("group_id is required")
	}

	if len(patch.Schemas) == 0 {
		patch.Schemas = []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"}
	}

	var resp Group
	err := s.transport.Do(ctx, "PATCH", fmt.Sprintf("/scim/v2/Groups/%s", groupID), patch, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) DeleteGroup(ctx context.Context, groupID string) error {
	if groupID == "" {
		return fmt.Errorf("group_id is required")
	}
	return s.transport.Do(ctx, "DELETE", fmt.Sprintf("/scim/v2/Groups/%s", groupID), nil, nil)
}

func (s *Service) ListGroups(ctx context.Context, filter string, cursor string, count int) (*GroupListResponse, error) {
	params := map[string]string{}
	if filter != "" {
		params["filter"] = filter
	}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if count > 0 {
		params["count"] = fmt.Sprintf("%d", count)
	}

	path := s.transport.BuildQueryURL("/scim/v2/Groups", params)

	var resp GroupListResponse
	err := s.transport.Do(ctx, "GET", path, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetServiceProviderConfig(ctx context.Context) (*ServiceProviderConfig, error) {
	var resp ServiceProviderConfig
	err := s.transport.Do(ctx, "GET", "/scim/v2/ServiceProviderConfig", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) GetResourceTypes(ctx context.Context) ([]ResourceTypeResponse, error) {
	var resp []ResourceTypeResponse
	err := s.transport.Do(ctx, "GET", "/scim/v2/ResourceTypes", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
