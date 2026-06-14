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

	c.RunOnce(context.Background())

	if called != 1 {
		t.Fatalf("runner appelé %d fois, want 1", called)
	}
	if gotTitle != "halo_infinite" {
		t.Errorf("titre = %q, want halo_infinite", gotTitle)
	}
}

func TestAssetNameSweepCron_RunOnce_ErrorNoPanic(t *testing.T) {
	c := NewAssetNameSweepCron(func(_ context.Context, _ string) (assetnames.Result, error) {
		return assetnames.Result{}, errors.New("boom")
	}, "", 0)
	c.RunOnce(context.Background()) // ne doit pas paniquer ni bloquer
}

func TestAssetNameSweepCron_Defaults(t *testing.T) {
	c := NewAssetNameSweepCron(nil, "", 0)
	if c.interval != DefaultAssetNameSweepInterval {
		t.Errorf("interval défaut = %v, want %v", c.interval, DefaultAssetNameSweepInterval)
	}
	if c.titleSlug == "" {
		t.Error("titleSlug défaut ne doit pas être vide")
	}
	if c.bootDelay != assetSweepBootDelay {
		t.Errorf("bootDelay défaut = %v, want %v", c.bootDelay, assetSweepBootDelay)
	}
	c.RunOnce(context.Background()) // runner nil → no-op, pas de panic
}

// TestAssetNameSweepCron_Run_BootDelayThenStop : Run lance après bootDelay puis
// s'arrête sur ctx annulé sans balayer (bootDelay non écoulé) — pas de fuite de
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
