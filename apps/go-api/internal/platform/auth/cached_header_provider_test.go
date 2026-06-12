package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCachedHeaderProvider_MemoizesAndRefreshes(t *testing.T) {
	var calls int
	now := time.Unix(1_700_000_000, 0)
	p := NewCachedHeaderProvider(time.Hour, func(_ context.Context) (string, error) {
		calls++
		return "header-v" + string(rune('0'+calls)), nil
	})
	p.now = func() time.Time { return now }

	// 1er appel → build.
	h1, err := p.Header(context.Background())
	if err != nil || h1 != "header-v1" {
		t.Fatalf("appel 1 = (%q, %v), want header-v1", h1, err)
	}
	// 2e appel dans le TTL → mémoïsé, pas de rebuild.
	if h2, _ := p.Header(context.Background()); h2 != "header-v1" || calls != 1 {
		t.Fatalf("appel 2 = %q (calls=%d), want header-v1 sans rebuild", h2, calls)
	}
	// Après expiration → rebuild.
	now = now.Add(2 * time.Hour)
	if h3, _ := p.Header(context.Background()); h3 != "header-v2" || calls != 2 {
		t.Fatalf("appel 3 = %q (calls=%d), want header-v2 après expiration", h3, calls)
	}
}

func TestCachedHeaderProvider_PropagatesBuildError(t *testing.T) {
	wantErr := errors.New("refresh KO")
	p := NewCachedHeaderProvider(time.Hour, func(_ context.Context) (string, error) {
		return "", wantErr
	})
	if _, err := p.Header(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
