package validate

import (
	"testing"
)

func TestUUID_Valid(t *testing.T) {
	err := UUID("agent_id", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("expected nil error for valid UUID, got: %v", err)
	}
}

func TestUUID_Empty(t *testing.T) {
	err := UUID("agent_id", "")
	if err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestUUID_Invalid(t *testing.T) {
	err := UUID("agent_id", "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestUUID_UpperCase(t *testing.T) {
	err := UUID("id", "550E8400-E29B-41D4-A716-446655440000")
	if err != nil {
		t.Fatalf("expected nil for uppercase UUID, got: %v", err)
	}
}

func TestTenantID_Valid(t *testing.T) {
	err := TenantID("my-tenant_123")
	if err != nil {
		t.Fatalf("expected nil for valid tenant ID, got: %v", err)
	}
}

func TestTenantID_Empty(t *testing.T) {
	err := TenantID("")
	if err == nil {
		t.Fatal("expected error for empty tenant ID")
	}
}

func TestTenantID_InvalidChars(t *testing.T) {
	err := TenantID("tenant with spaces")
	if err == nil {
		t.Fatal("expected error for tenant ID with spaces")
	}
}

func TestURL_HTTPS(t *testing.T) {
	err := URL("base_url", "https://api.zinetic.io")
	if err != nil {
		t.Fatalf("expected nil for HTTPS URL, got: %v", err)
	}
}

func TestURL_HTTPLocalhost(t *testing.T) {
	err := URL("base_url", "http://localhost:8080")
	if err != nil {
		t.Fatalf("expected nil for HTTP localhost, got: %v", err)
	}
}

func TestURL_HTTPLoopback(t *testing.T) {
	err := URL("base_url", "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("expected nil for HTTP 127.0.0.1, got: %v", err)
	}
}

func TestURL_HTTPRemote_Rejected(t *testing.T) {
	err := URL("base_url", "http://api.zinetic.io")
	if err == nil {
		t.Fatal("expected error for HTTP with remote host")
	}
}

func TestURL_Empty(t *testing.T) {
	err := URL("base_url", "")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestURL_InvalidScheme(t *testing.T) {
	err := URL("base_url", "ftp://files.zinetic.io")
	if err == nil {
		t.Fatal("expected error for ftp scheme")
	}
}

func TestScope_Valid(t *testing.T) {
	err := Scope("scope", "agents:read agents:write credentials:anchor")
	if err != nil {
		t.Fatalf("expected nil for valid scopes, got: %v", err)
	}
}

func TestScope_Empty(t *testing.T) {
	err := Scope("scope", "")
	if err == nil {
		t.Fatal("expected error for empty scope")
	}
}

func TestScope_InvalidChars(t *testing.T) {
	err := Scope("scope", "agents:re@d")
	if err == nil {
		t.Fatal("expected error for scope with invalid character @")
	}
}

func TestIP_Valid(t *testing.T) {
	err := IP("ip", "192.168.1.1")
	if err != nil {
		t.Fatalf("expected nil for valid IP, got: %v", err)
	}
}

func TestIP_IPv6(t *testing.T) {
	err := IP("ip", "::1")
	if err != nil {
		t.Fatalf("expected nil for valid IPv6, got: %v", err)
	}
}

func TestIP_Empty(t *testing.T) {
	err := IP("ip", "")
	if err != nil {
		t.Fatal("expected nil for empty IP (optional)")
	}
}

func TestIP_Invalid(t *testing.T) {
	err := IP("ip", "not-an-ip")
	if err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

func TestNonEmpty(t *testing.T) {
	if err := NonEmpty("name", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := NonEmpty("name", ""); err == nil {
		t.Fatal("expected error for empty value")
	}
	if err := NonEmpty("name", "   "); err == nil {
		t.Fatal("expected error for whitespace-only value")
	}
}
