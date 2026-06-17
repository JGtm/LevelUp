package config

// csr_season_title_test.go — PMT-4 PR-1 : résolveur CSR par titre.
//   parité global (sans overlay) · overlay titre prioritaire · dégradation
//   CapRanked absente → "" · override env prioritaire.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

func writeCSROverlay(t *testing.T, repoRoot, slug, content string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "data", "titles", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write overlay: %v", err)
	}
}

func TestCSRSeasonIDForTitle(t *testing.T) {
	t.Setenv("LEVELUP_CSR_SEASON_ID", "") // neutralise un éventuel override d'env réel
	root := t.TempDir()
	cfg := &AppConfig{RepoRoot: root, CurrentCSRSeasonID: "CsrSeason13-1"}
	ctx := context.Background()

	reg := titlePkg.NewRegistry() // halo_infinite (CapRanked) déjà enregistré
	reg.Register(&titlePkg.TitleDescriptor{
		Slug:         "synthetic_no_ranked",
		Name:         "Synth",
		Status:       titlePkg.StatusActive,
		Capabilities: []titlePkg.Capability{titlePkg.CapMatchmaking}, // PAS CapRanked
	})

	// (a) Halo sans overlay → fallback global.
	if got := cfg.CSRSeasonIDForTitle(ctx, "halo_infinite", reg); got != "CsrSeason13-1" {
		t.Errorf("halo sans overlay = %q, want CsrSeason13-1 (global)", got)
	}

	// (b) Halo avec overlay → overlay prioritaire sur global.
	writeCSROverlay(t, root, "halo_infinite", `{"csr_season_id":"CsrSeason14-1"}`)
	if got := cfg.CSRSeasonIDForTitle(ctx, "halo_infinite", reg); got != "CsrSeason14-1" {
		t.Errorf("halo avec overlay = %q, want CsrSeason14-1 (overlay)", got)
	}

	// Dégradation : titre sans CapRanked → "" même si l'overlay déclare une saison.
	writeCSROverlay(t, root, "synthetic_no_ranked", `{"csr_season_id":"WhateverSeason"}`)
	if got := cfg.CSRSeasonIDForTitle(ctx, "synthetic_no_ranked", reg); got != "" {
		t.Errorf("titre sans CapRanked = %q, want \"\" (dégradation, sync CSR skippé)", got)
	}

	// Titre inconnu du registre → "" (dégradation).
	if got := cfg.CSRSeasonIDForTitle(ctx, "unknown_title", reg); got != "" {
		t.Errorf("titre inconnu = %q, want \"\"", got)
	}

	// Override env → précédence absolue (parité loadCSRSeasonID).
	t.Setenv("LEVELUP_CSR_SEASON_ID", "EnvSeason")
	if got := cfg.CSRSeasonIDForTitle(ctx, "halo_infinite", reg); got != "EnvSeason" {
		t.Errorf("env override = %q, want EnvSeason", got)
	}
}
