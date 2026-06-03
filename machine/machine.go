package machine

import "context"

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

func (s *Service) EnrollGitHub(ctx context.Context, req *GitHubEnrollRequest) (*GitHubIdentity, error) {
	var resp GitHubIdentity
	if err := s.transport.Do(ctx, "POST", "/v1/machine/github/enroll", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) MintGitHubToken(ctx context.Context, req *GitHubTokenRequest) (*TokenResponse, error) {
	var resp TokenResponse
	if err := s.transport.Do(ctx, "POST", "/v1/machine/github/tokens/mint", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) EnrollKubernetes(ctx context.Context, req *KubernetesEnrollRequest) (*KubernetesIdentity, error) {
	var resp KubernetesIdentity
	if err := s.transport.Do(ctx, "POST", "/v1/machine/k8s/enroll", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) MintKubernetesToken(ctx context.Context, req *KubernetesTokenRequest) (*TokenResponse, error) {
	var resp TokenResponse
	if err := s.transport.Do(ctx, "POST", "/v1/machine/k8s/tokens/mint", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
