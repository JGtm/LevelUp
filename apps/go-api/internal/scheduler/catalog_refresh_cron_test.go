package scheduler

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestCatalogRefreshCron_RunOnce_CallsRunner(t *testing.T) {
	called := 0
	gotTitle := ""
	c := NewCatalogRefreshCron(func(_ context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
		called++
		gotTitle = titleSlug
		return domain.CatalogUGCDrainResult{Playlists: 3, Maps: 5}, nil
	}, "halo_infinite", 0)

	c.RunOnce(context.Background())

	if called != 1 {
		t.Fatalf("runner appelé %d fois, want 1", called)
	}
	if gotTitle != "halo_infinite" {
		t.Errorf("titre = %q, want halo_infinite", gotTitle)
	}
}

func TestCatalogRefreshCron_RunOnce_ErrorNoPanic(t *testing.T) {
	c := NewCatalogRefreshCron(func(_ context.Context, _ string) (domain.CatalogUGCDrainResult, error) {
		return domain.CatalogUGCDrainResult{}, errors.New("boom")
	}, "", 0)
	c.RunOnce(context.Background()) // ne doit pas paniquer ni bloquer
}

func TestCatalogRefreshCron_Defaults(t *testing.T) {
	c := NewCatalogRefreshCron(nil, "", 0)
	if c.interval != DefaultCatalogRefreshInterval {
		t.Errorf("interval défaut = %v, want %v", c.interval, DefaultCatalogRefreshInterval)
	}
	if c.titleSlug == "" {
		t.Error("titleSlug défaut ne doit pas être vide")
	}
	c.RunOnce(context.Background()) // runner nil → no-op, pas de panic
}
