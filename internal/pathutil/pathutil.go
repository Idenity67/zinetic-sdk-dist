package pathutil

import (
	"fmt"
	"net/url"
	"strings"
)

func Segment(name, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if value == "." || value == ".." || strings.Contains(value, "/") {
		return "", fmt.Errorf("%s must be a single path segment", name)
	}
	return url.PathEscape(value), nil
}

func SlashPath(name, value string) (string, error) {
	value = strings.Trim(value, "/")
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	parts := strings.Split(value, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("%s contains an invalid path segment", name)
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/"), nil
}
