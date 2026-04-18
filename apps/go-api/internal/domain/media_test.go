package domain

import "testing"

func TestMediaPageRequest_ResolvePage(t *testing.T) {
	tests := []struct {
		name string
		req  MediaPageRequest
		want int
	}{
		{"default", MediaPageRequest{}, 1},
		{"legacy page", MediaPageRequest{Page: 3}, 3},
		{"modern page", MediaPageRequest{Pagination: PaginationRequest{Page: 5}}, 5},
		{"modern wins", MediaPageRequest{Page: 2, Pagination: PaginationRequest{Page: 5}}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.ResolvePage(); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMediaPageRequest_ResolvePageSize(t *testing.T) {
	tests := []struct {
		name    string
		req     MediaPageRequest
		defSize int
		maxSize int
		want    int
	}{
		{"default", MediaPageRequest{}, 24, 100, 24},
		{"legacy", MediaPageRequest{PageSize: 50}, 24, 100, 50},
		{"modern", MediaPageRequest{Pagination: PaginationRequest{PageSize: 30}}, 24, 100, 30},
		{"capped", MediaPageRequest{PageSize: 200}, 24, 100, 100},
		{"no max", MediaPageRequest{PageSize: 200}, 24, 0, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.ResolvePageSize(tt.defSize, tt.maxSize); got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
