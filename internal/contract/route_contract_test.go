package contract

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestSDKRoutesExistInBackendOpenAPI(t *testing.T) {
	sdkRoot, err := findSDKRoot()
	if err != nil {
		t.Fatal(err)
	}
	candidates := []string{
		filepath.Join(sdkRoot, ".zinetic-backend-contract", "api", "openapi", "zinetic.yaml"),
		filepath.Join(sdkRoot, "..", "zinetic-backend", "api", "openapi", "zinetic.yaml"),
		filepath.Join(sdkRoot, "zinetic-backend", "api", "openapi", "zinetic.yaml"),
	}
	openAPIPath, err := firstExistingFile(candidates)
	if err != nil {
		if os.Getenv("ZINETIC_REQUIRE_BACKEND_OPENAPI") == "1" || os.Getenv("ZINETIC_REQUIRE_BACKEND_OPENAPI") == "true" {
			t.Fatalf("backend OpenAPI was required but not found: %v", err)
		}
		t.Skipf("sibling backend OpenAPI not available: %v", err)
	}
	raw, err := os.ReadFile(openAPIPath)
	if err != nil {
		t.Fatalf("read backend OpenAPI %s: %v", openAPIPath, err)
	}

	available := openAPIPathSet(raw)
	missing := []string{}
	for _, route := range sdkRouteTemplates(t, sdkRoot) {
		canonical := canonicalBackendRoute(route)
		if isAllowedRootSystemRoute(canonical) {
			continue
		}
		if !available[contractRouteKey(canonical)] {
			missing = append(missing, route+" -> "+canonical)
		}
	}

	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("SDK routes missing from backend OpenAPI:\n%s", strings.Join(missing, "\n"))
	}
}

func firstExistingFile(candidates []string) (string, error) {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", os.ErrNotExist
}

func findSDKRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modPath := filepath.Join(wd, "go.mod")
		if raw, err := os.ReadFile(modPath); err == nil && bytes.Contains(raw, []byte("module sdk.zinetic.net")) {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", os.ErrNotExist
		}
		wd = parent
	}
}

func openAPIPathSet(raw []byte) map[string]bool {
	paths := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "/") || !strings.HasSuffix(line, ":") {
			continue
		}
		paths[contractRouteKey(strings.TrimSuffix(line, ":"))] = true
	}
	return paths
}

func sdkRouteTemplates(t *testing.T, sdkRoot string) []string {
	t.Helper()
	routes := map[string]bool{}
	doLiteral := regexp.MustCompile(`Do(?:WithHeaders|Raw)?\([^\n]*,\s*"(?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)"\s*,\s*"([^"]+)"`)
	sprintfLiteral := regexp.MustCompile(`fmt\.Sprintf\("(/[^"]+)"`)
	buildQueryLiteral := regexp.MustCompile(`BuildQueryURL\("(/[^"]+)"`)
	formatParam := regexp.MustCompile(`%[sdv]`)

	err := filepath.WalkDir(sdkRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".zinetic-backend-contract", "docs", "examples", "vendor":
				return filepath.SkipDir
			case "contract":
				if strings.HasSuffix(path, filepath.Join("internal", "contract")) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		for _, match := range doLiteral.FindAllStringSubmatch(text, -1) {
			routes[match[1]] = true
		}
		for _, match := range sprintfLiteral.FindAllStringSubmatch(text, -1) {
			routes[formatParam.ReplaceAllString(match[1], "{id}")] = true
		}
		for _, match := range buildQueryLiteral.FindAllStringSubmatch(text, -1) {
			routes[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	out := make([]string, 0, len(routes))
	for route := range routes {
		out = append(out, route)
	}
	sort.Strings(out)
	return out
}

func TestSDKRouteTemplatesSkipsBackendContractCheckout(t *testing.T) {
	sdkRoot := t.TempDir()
	writeRouteFixture(t, filepath.Join(sdkRoot, "user.go"), `package zinetic

func f() {
	client.Do(ctx, "GET", "/v1/users/me", nil)
}
`)
	writeRouteFixture(t, filepath.Join(sdkRoot, ".zinetic-backend-contract", "services", "gateway", "external.go"), `package gateway

func f() {
	client.Do(ctx, "GET", "/admin/realms/master/users", nil)
}
`)

	routes := sdkRouteTemplates(t, sdkRoot)
	if !containsRoute(routes, "/v1/users/me") {
		t.Fatalf("expected SDK route to be collected, got %v", routes)
	}
	if containsRoute(routes, "/admin/realms/master/users") {
		t.Fatalf("backend contract checkout route was collected: %v", routes)
	}
}

func writeRouteFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsRoute(routes []string, route string) bool {
	for _, candidate := range routes {
		if candidate == route {
			return true
		}
	}
	return false
}

func canonicalBackendRoute(route string) string {
	route = strings.TrimSpace(route)
	if idx := strings.Index(route, "?"); idx >= 0 {
		route = route[:idx]
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	if route == "/api/v1" || strings.HasPrefix(route, "/api/v1/") {
		return route
	}
	if isRootRoute(route) {
		return route
	}
	if route == "/v1" {
		route = "/"
	} else {
		route = strings.TrimPrefix(route, "/v1")
	}
	if route == "" {
		route = "/"
	}
	if route == "/" {
		return "/api/v1"
	}
	return "/api/v1" + route
}

func isRootRoute(route string) bool {
	for _, prefix := range []string{"/health", "/ready", "/version", "/metrics", "/docs", "/.well-known", "/scim", "/oid4vci", "/oauth", "/auth/token", "/pam"} {
		if route == prefix || strings.HasPrefix(route, prefix+"/") {
			return true
		}
	}
	return false
}

func isAllowedRootSystemRoute(route string) bool {
	for _, prefix := range []string{"/health", "/ready", "/version", "/metrics"} {
		if route == prefix || strings.HasPrefix(route, prefix+"/") {
			return true
		}
	}
	return false
}

func contractRouteKey(route string) string {
	route = strings.TrimRight(strings.TrimSpace(route), "/")
	if route == "" {
		route = "/"
	}
	parts := strings.Split(route, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			parts[i] = "{}"
		}
	}
	return strings.Join(parts, "/")
}
