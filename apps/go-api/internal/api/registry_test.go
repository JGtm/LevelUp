package api

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/duckdb"
)

var errResolve = errors.New("player not found")

func failResolver(_ context.Context, _ string) (*duckdb.PlayerDB, error) {
	return nil, errResolve
}

func TestServiceRegistry_Career_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.Career(context.Background(), "unknown")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Filters_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.Filters(context.Background(), "unknown")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_LastMatch_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.LastMatch(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_MatchView_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.MatchView(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Media_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.Media(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_MediaUpload_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, _, _, _, err := reg.MediaUpload(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Sessions_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.Sessions(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_SessionCompare_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.SessionCompare(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Stats_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.Stats(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Timeseries_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.Timeseries(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_CitationsCtx_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.CitationsCtx(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_ExplorerCtx_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.ExplorerCtx(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_HomeCtx_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.HomeCtx(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_MatchHistoryCtx_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.MatchHistoryCtx(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_SquadCtx_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.SquadCtx(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_MatchExclusion_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, err := reg.MatchExclusion(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_TeammatesCtx_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.TeammatesCtx(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Compare_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.Compare(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestServiceRegistry_Leaderboard_ResolveError(t *testing.T) {
	reg := &ServiceRegistry{resolve: failResolver}
	_, _, _, err := reg.Leaderboard(context.Background(), "x")
	if !errors.Is(err, errResolve) {
		t.Fatalf("expected resolve error, got %v", err)
	}
}

func TestChallengeBadgeDirFromMetadataPath_LegacyWarehouse(t *testing.T) {
	metaPath := filepath.Join("data", "warehouse", "metadata.duckdb")
	want := filepath.Join("data", "cache", "challenge_badges")
	if got := challengeBadgeDirFromMetadataPath(metaPath); got != want {
		t.Fatalf("challengeBadgeDirFromMetadataPath() = %q, want %q", got, want)
	}
}

func TestChallengeBadgeDirFromMetadataPath_TitleAwareWarehouseUsesSharedDataRoot(t *testing.T) {
	metaPath := filepath.Join("data", "titles", "halo_infinite", "warehouse", "metadata.duckdb")
	want := filepath.Join("data", "cache", "challenge_badges")
	if got := challengeBadgeDirFromMetadataPath(metaPath); got != want {
		t.Fatalf("challengeBadgeDirFromMetadataPath() = %q, want %q", got, want)
	}
}
