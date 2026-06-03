package scim

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

func TestCreateUser_Success(t *testing.T) {
	mt := &mockTransport{
		result: &User{
			ID:       "user-001",
			UserName: "jdoe@example.com",
			Active:   true,
		},
	}
	svc := NewService(mt)

	resp, err := svc.CreateUser(context.Background(), &User{
		UserName: "jdoe@example.com",
		Active:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "user-001" {
		t.Fatalf("expected user-001, got %s", resp.ID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
	if mt.path != "/scim/v2/Users" {
		t.Fatalf("expected /scim/v2/Users, got %s", mt.path)
	}
}

func TestCreateUser_MissingUserName(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateUser(context.Background(), &User{Active: true})
	if err == nil {
		t.Fatal("expected error for missing userName")
	}
}

func TestGetUser_Success(t *testing.T) {
	mt := &mockTransport{
		result: &User{ID: "user-001", UserName: "jdoe@example.com"},
	}
	svc := NewService(mt)

	resp, err := svc.GetUser(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserName != "jdoe@example.com" {
		t.Fatalf("expected jdoe@example.com, got %s", resp.UserName)
	}
	if mt.path != "/scim/v2/Users/user-001" {
		t.Fatalf("expected /scim/v2/Users/user-001, got %s", mt.path)
	}
}

func TestGetUser_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetUser(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestDeleteUser_Success(t *testing.T) {
	mt := &mockTransport{}
	svc := NewService(mt)

	err := svc.DeleteUser(context.Background(), "user-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mt.method != "DELETE" {
		t.Fatalf("expected DELETE, got %s", mt.method)
	}
}

func TestDeleteUser_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.DeleteUser(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestListUsers_Success(t *testing.T) {
	mt := &mockTransport{
		result: &UserListResponse{
			TotalResults: 2,
			Resources: []User{
				{ID: "u-1", UserName: "alice@co.com"},
				{ID: "u-2", UserName: "bob@co.com"},
			},
		},
	}
	svc := NewService(mt)

	resp, err := svc.ListUsers(context.Background(), "", "", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalResults != 2 {
		t.Fatalf("expected 2, got %d", resp.TotalResults)
	}
}

func TestCreateGroup_Success(t *testing.T) {
	mt := &mockTransport{
		result: &Group{
			ID:          "grp-001",
			DisplayName: "Engineering",
		},
	}
	svc := NewService(mt)

	resp, err := svc.CreateGroup(context.Background(), &Group{
		DisplayName: "Engineering",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "grp-001" {
		t.Fatalf("expected grp-001, got %s", resp.ID)
	}
	if mt.method != "POST" {
		t.Fatalf("expected POST, got %s", mt.method)
	}
}

func TestCreateGroup_MissingDisplayName(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.CreateGroup(context.Background(), &Group{})
	if err == nil {
		t.Fatal("expected error for missing displayName")
	}
}

func TestGetGroup_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.GetGroup(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty group_id")
	}
}

func TestDeleteGroup_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	err := svc.DeleteGroup(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty group_id")
	}
}

func TestPatchUser_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.PatchUser(context.Background(), "", &PatchOp{})
	if err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestPatchGroup_EmptyID(t *testing.T) {
	svc := NewService(&mockTransport{})

	_, err := svc.PatchGroup(context.Background(), "", &PatchOp{})
	if err == nil {
		t.Fatal("expected error for empty group_id")
	}
}

func TestGetServiceProviderConfig_Success(t *testing.T) {
	mt := &mockTransport{
		result: &ServiceProviderConfig{
			Patch: SupportedFeature{Supported: true},
		},
	}
	svc := NewService(mt)

	resp, err := svc.GetServiceProviderConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Patch.Supported {
		t.Fatal("expected patch supported=true")
	}
	if mt.path != "/scim/v2/ServiceProviderConfig" {
		t.Fatalf("expected /scim/v2/ServiceProviderConfig, got %s", mt.path)
	}
}

func TestTransportError(t *testing.T) {
	mt := &mockTransport{err: fmt.Errorf("network error")}
	svc := NewService(mt)

	_, err := svc.GetUser(context.Background(), "user-001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateUser_Success(t *testing.T) {
	mt := &mockTransport{result: &User{ID: "u-1", UserName: "alice"}}
	svc := NewService(mt)

	resp, err := svc.UpdateUser(context.Background(), "u-1", &User{UserName: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.UserName != "alice" {
		t.Fatalf("expected alice, got %s", resp.UserName)
	}
	if mt.method != "PUT" {
		t.Fatalf("expected PUT, got %s", mt.method)
	}
}

func TestUpdateUser_RequiresID(t *testing.T) {
	svc := NewService(&mockTransport{})
	if _, err := svc.UpdateUser(context.Background(), "", &User{UserName: "x"}); err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestPatchUser_Success(t *testing.T) {
	mt := &mockTransport{result: &User{ID: "u-1", UserName: "alice"}}
	svc := NewService(mt)

	resp, err := svc.PatchUser(context.Background(), "u-1", &PatchOp{
		Operations: []Operation{{Op: "replace", Path: "active", Value: false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "u-1" {
		t.Fatalf("expected u-1, got %s", resp.ID)
	}
	if mt.method != "PATCH" {
		t.Fatalf("expected PATCH, got %s", mt.method)
	}
}

func TestUpdateGroup_Success(t *testing.T) {
	mt := &mockTransport{result: &Group{ID: "g-1", DisplayName: "Admins"}}
	svc := NewService(mt)

	resp, err := svc.UpdateGroup(context.Background(), "g-1", &Group{DisplayName: "Admins"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.DisplayName != "Admins" {
		t.Fatalf("expected Admins, got %s", resp.DisplayName)
	}
	if mt.method != "PUT" {
		t.Fatalf("expected PUT, got %s", mt.method)
	}
}

func TestUpdateGroup_RequiresID(t *testing.T) {
	svc := NewService(&mockTransport{})
	if _, err := svc.UpdateGroup(context.Background(), "", &Group{DisplayName: "x"}); err == nil {
		t.Fatal("expected error for empty group_id")
	}
}

func TestPatchGroup_Success(t *testing.T) {
	mt := &mockTransport{result: &Group{ID: "g-1", DisplayName: "Admins"}}
	svc := NewService(mt)

	resp, err := svc.PatchGroup(context.Background(), "g-1", &PatchOp{
		Operations: []Operation{{Op: "add", Path: "members", Value: "u-2"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "g-1" {
		t.Fatalf("expected g-1, got %s", resp.ID)
	}
}

func TestListGroups_Success(t *testing.T) {
	mt := &mockTransport{result: &GroupListResponse{TotalResults: 1, Resources: []Group{{ID: "g-1"}}}}
	svc := NewService(mt)

	resp, err := svc.ListGroups(context.Background(), "displayName eq \"Admins\"", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TotalResults != 1 {
		t.Fatalf("expected 1, got %d", resp.TotalResults)
	}
}

func TestGetResourceTypes_Success(t *testing.T) {
	mt := &mockTransport{result: []ResourceTypeResponse{{ID: "User"}, {ID: "Group"}}}
	svc := NewService(mt)

	resp, err := svc.GetResourceTypes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 resource types, got %d", len(resp))
	}
	if mt.path != "/scim/v2/ResourceTypes" {
		t.Fatalf("unexpected path %s", mt.path)
	}
}
