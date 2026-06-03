package pagination

import "fmt"

type PageParams struct {
	Cursor string
	Limit  int
}

type Page[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

func (p *PageParams) QueryValues() map[string]string {
	vals := make(map[string]string)
	if p == nil {
		return vals
	}
	if p.Cursor != "" {
		vals["cursor"] = p.Cursor
	}
	if p.Limit > 0 {
		limit := min(p.Limit, 200)
		vals["limit"] = fmt.Sprintf("%d", limit)
	}
	return vals
}

func DefaultPageParams() *PageParams {
	return &PageParams{
		Limit: 50,
	}
}
