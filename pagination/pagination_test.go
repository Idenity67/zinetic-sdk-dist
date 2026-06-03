package pagination

import (
	"testing"
)

func TestPageParams_QueryValues_Default(t *testing.T) {
	p := DefaultPageParams()
	vals := p.QueryValues()
	if vals["limit"] != "50" {
		t.Fatalf("expected default limit 50, got %s", vals["limit"])
	}
	if _, ok := vals["cursor"]; ok {
		t.Fatal("expected no cursor in default params")
	}
}

func TestPageParams_QueryValues_WithCursor(t *testing.T) {
	p := &PageParams{Cursor: "abc123", Limit: 25}
	vals := p.QueryValues()
	if vals["cursor"] != "abc123" {
		t.Fatalf("expected cursor abc123, got %s", vals["cursor"])
	}
	if vals["limit"] != "25" {
		t.Fatalf("expected limit 25, got %s", vals["limit"])
	}
}

func TestPageParams_QueryValues_LimitCappedAt200(t *testing.T) {
	p := &PageParams{Limit: 500}
	vals := p.QueryValues()
	if vals["limit"] != "200" {
		t.Fatalf("expected limit capped at 200, got %s", vals["limit"])
	}
}

func TestPageParams_QueryValues_ZeroLimit(t *testing.T) {
	p := &PageParams{Limit: 0}
	vals := p.QueryValues()
	if _, ok := vals["limit"]; ok {
		t.Fatal("expected no limit key when limit is 0")
	}
}

func TestPageParams_QueryValues_Nil(t *testing.T) {
	var p *PageParams
	vals := p.QueryValues()
	if len(vals) != 0 {
		t.Fatalf("expected empty map for nil PageParams, got %v", vals)
	}
}
