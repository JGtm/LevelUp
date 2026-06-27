package scheduler

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/assetnames"
)

func TestAssetNameSweepCron_RunOnce_CallsRunner(t *testing.T) {
	called := 0
	gotTitle := ""
	c := NewAssetNameSweepCron(func(_ context.Context, titleSlug string) (assetnames.Result, error) {
		called++
		gotTitle = titleSlug
		return assetnames.Result{Requested: 4, Resolved: 3, Skipped: 1}, nil
	}, "halo_infinite", 0)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if called != 1 {
		t.Fatalf("runner appelé %d fois, want 1 (seul halo_infinite a CapForge)", called)
	}
	if gotTitle != "halo_infinite" {
		t.Errorf("titre = %q, want halo_infinite", gotTitle)
	}
}

// TestAssetNameSweepCron_RunOnce_SkipsTitleWithoutForge vérifie la brique
// title-aware : un titre actif SANS CapForge (modèle Halo 5, noms résolus
// metadata-side via cmd/h5-metadata-fetch) est skippé proprement — aucun sweep
// lancé pour lui — tandis que halo_infinite (avec la cap) reste traité.
func TestAssetNameSweepCron_RunOnce_SkipsTitleWithoutForge(t *testing.T) {
	var swept []string
	c := NewAssetNameSweepCron(func(_ context.Context, titleSlug string) (assetnames.Result, error) {
		swept = append(swept, titleSlug)
		return assetnames.Result{Requested: 1, Resolved: 1}, nil
	}, "", 0)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if len(swept) != 1 || swept[0] != "halo_infinite" {
		t.Fatalf("titres balayés = %v, want [halo_infinite] uniquement (title_no_forge skippé)", swept)
	}
}

func TestAssetNameSweepCron_RunOnce_ErrorNoPanic(t *testing.T) {
	c := NewAssetNameSweepCron(func(_ context.Context, _ string) (assetnames.Result, error) {
		return assetnames.Result{}, errors.New("boom")
	}, "", 0)
	c.registry = forgeRegistry()
	c.RunOnce(context.Background()) // l'erreur d'un titre n'interrompt pas le cycle
}

func TestAssetNameSweepCron_Defaults(t *testing.T) {
	c := NewAssetNameSweepCron(nil, "", 0)
	if c.interval != DefaultAssetNameSweepInterval {
		t.Errorf("interval défaut = %v, want %v", c.interval, DefaultAssetNameSweepInterval)
	}
	if c.registry == nil {
		t.Error("registry défaut ne doit pas être nil (DefaultRegistry)")
	}
	if c.bootDelay != assetSweepBootDelay {
		t.Errorf("bootDelay défaut = %v, want %v", c.bootDelay, assetSweepBootDelay)
	}
	c.RunOnce(context.Background()) // runner nil → no-op, pas de panic
}

// TestAssetNameSweepCron_Run_CtxCancelBeforeBootDelay : Run lance après bootDelay
// puis s'arrête sur ctx annulé sans balayer (bootDelay non écoulé) — pas de fuite de
// goroutine ni de panic. bootDelay forcé court via le champ.
func TestAssetNameSweepCron_Run_CtxCancelBeforeBootDelay(t *testing.T) {
	called := 0
	c := NewAssetNameSweepCron(func(_ context.Context, _ string) (assetnames.Result, error) {
		called++
		return assetnames.Result{}, nil
	}, "", 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé avant le bootDelay → Run doit retourner sans appeler le runner

	c.Run(ctx)

	if called != 0 {
		t.Errorf("runner appelé %d fois alors que ctx annulé avant bootDelay, want 0", called)
	}
}
