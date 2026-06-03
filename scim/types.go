package scim

import "time"

type User struct {
	Schemas      []string   `json:"schemas"`
	ID           string     `json:"id,omitempty"`
	ExternalID   string     `json:"externalId,omitempty"`
	UserName     string     `json:"userName"`
	Name         *Name      `json:"name,omitempty"`
	DisplayName  string     `json:"displayName,omitempty"`
	Emails       []Email    `json:"emails,omitempty"`
	PhoneNumbers []Phone    `json:"phoneNumbers,omitempty"`
	Active       bool       `json:"active"`
	Groups       []GroupRef `json:"groups,omitempty"`
	Meta         *Meta      `json:"meta,omitempty"`
}

type Name struct {
	Formatted       string `json:"formatted,omitempty"`
	FamilyName      string `json:"familyName,omitempty"`
	GivenName       string `json:"givenName,omitempty"`
	MiddleName      string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

type Email struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type Phone struct {
	Value string `json:"value"`
	Type  string `json:"type,omitempty"`
}

type GroupRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

type Meta struct {
	ResourceType string    `json:"resourceType,omitempty"`
	Created      time.Time `json:"created,omitempty"`
	LastModified time.Time `json:"lastModified,omitempty"`
	Location     string    `json:"location,omitempty"`
	Version      string    `json:"version,omitempty"`
}

type Group struct {
	Schemas     []string `json:"schemas"`
	ID          string   `json:"id,omitempty"`
	ExternalID  string   `json:"externalId,omitempty"`
	DisplayName string   `json:"displayName"`
	Members     []Member `json:"members,omitempty"`
	Meta        *Meta    `json:"meta,omitempty"`
}

type Member struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	Type    string `json:"type,omitempty"`
}

type ListResponse struct {
	Schemas      []string      `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	ItemsPerPage int           `json:"itemsPerPage"`
	StartIndex   int           `json:"startIndex"`
	NextCursor   string        `json:"nextCursor,omitempty"`
	Resources    []interface{} `json:"Resources"`
}

type UserListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	ItemsPerPage int      `json:"itemsPerPage"`
	StartIndex   int      `json:"startIndex"`
	NextCursor   string   `json:"nextCursor,omitempty"`
	Resources    []User   `json:"Resources"`
}

type GroupListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	ItemsPerPage int      `json:"itemsPerPage"`
	StartIndex   int      `json:"startIndex"`
	NextCursor   string   `json:"nextCursor,omitempty"`
	Resources    []Group  `json:"Resources"`
}

type PatchOp struct {
	Schemas    []string    `json:"schemas"`
	Operations []Operation `json:"Operations"`
}

type Operation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type ServiceProviderConfig struct {
	Schemas               []string         `json:"schemas"`
	DocumentationURI      string           `json:"documentationUri,omitempty"`
	Patch                 SupportedFeature `json:"patch"`
	Bulk                  BulkFeature      `json:"bulk"`
	Filter                FilterFeature    `json:"filter"`
	ChangePassword        SupportedFeature `json:"changePassword"`
	Sort                  SupportedFeature `json:"sort"`
	ETag                  SupportedFeature `json:"etag"`
	AuthenticationSchemes []AuthScheme     `json:"authenticationSchemes"`
}

type SupportedFeature struct {
	Supported bool `json:"supported"`
}

type BulkFeature struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

type FilterFeature struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

type AuthScheme struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ResourceTypeResponse struct {
	Schemas     []string `json:"schemas"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Endpoint    string   `json:"endpoint"`
	Schema      string   `json:"schema"`
}

type ErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}
