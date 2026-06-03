package validate

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"sdk.zinetic.net/apierr"
)

var (
	uuidPattern     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	tenantIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	scopePattern    = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)
)

func UUID(field, value string) error {
	if value == "" {
		return apierr.NewValidationError(field, "must not be empty")
	}
	if !uuidPattern.MatchString(strings.ToLower(value)) {
		return apierr.NewValidationError(field, "must be a valid UUID")
	}
	return nil
}

func TenantID(value string) error {
	if value == "" {
		return apierr.NewValidationError("tenant_id", "must not be empty")
	}
	if !tenantIDPattern.MatchString(value) {
		return apierr.NewValidationError("tenant_id", "must match pattern [a-zA-Z0-9_-]{1,128}")
	}
	return nil
}

func NonEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return apierr.NewValidationError(field, "must not be empty")
	}
	return nil
}

func URL(field, value string) error {
	if value == "" {
		return apierr.NewValidationError(field, "must not be empty")
	}
	u, err := url.Parse(value)
	if err != nil {
		return apierr.NewValidationError(field, "must be a valid URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return apierr.NewValidationError(field, "must use https or http scheme")
	}
	if u.Host == "" {
		return apierr.NewValidationError(field, "must have a valid host")
	}
	hostname := u.Hostname()
	if u.Scheme == "http" && hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
		return apierr.NewValidationError(field, "http scheme only allowed for localhost; use https for remote hosts")
	}
	return nil
}

func IP(field, value string) error {
	if value == "" {
		return nil
	}
	if net.ParseIP(value) == nil {
		return apierr.NewValidationError(field, "must be a valid IP address")
	}
	return nil
}

func Scope(field, value string) error {
	if value == "" {
		return apierr.NewValidationError(field, "must not be empty")
	}
	parts := strings.Fields(value)
	for _, p := range parts {
		if !scopePattern.MatchString(p) {
			return apierr.NewValidationError(field, fmt.Sprintf("invalid scope value: %s", p))
		}
	}
	return nil
}

func MaxDuration(field string, value, max time.Duration) error {
	if value > max {
		return apierr.NewValidationError(field, fmt.Sprintf("must not exceed %s", max))
	}
	return nil
}

func PositiveInt(field string, value int) error {
	if value <= 0 {
		return apierr.NewValidationError(field, "must be a positive integer")
	}
	return nil
}

func Enum[T comparable](field string, value T, allowed []T) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return apierr.NewValidationError(field, "invalid value")
}
