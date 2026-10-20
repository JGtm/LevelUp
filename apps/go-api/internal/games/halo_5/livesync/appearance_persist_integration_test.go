//go:build integration

// Package livesync — appearance_persist_integration_test.go : round-trip du hook
// d'identité Spartan Halo 5. Vérifie que PersistAppearance écrit le service tag dans
// career_progression (append-only) + télécharge les PNG (Spartan + emblème) dans le
// cache d'assets sous le chemin servi local-first par le handler /api/v1/assets/spartan.
//
// Tag integration : OpenPlayerDB ouvre une vraie DuckDB (CGO).
package livesync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/dblease"
	syncpkg "levelup/go-api/internal/sync"
)

type fakeAppearanceSource struct {
	app        *halo5.H5Appearance
	spartanPNG []byte
	emblemPNG  []byte
	appErr     error
}

func (f *fakeAppearanceSource) GetAppearance(_ context.Context, _ string) (*halo5.H5Appearance, error) {
	return f.app, f.appErr
}
func (f *fakeAppearanceSource) GetSpartanRenderPNG(_ context.Context, _ string) ([]byte, string, error) {
	return f.spartanPNG, "image/png", nil
}
func (f *fakeAppearanceSource) GetEmblemPNG(_ context.Context, _ string) ([]byte, string, error) {
	return f.emblemPNG, "image/png", nil
}

var testPNG = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x42}

func TestPersistAppearance_RoundTrip(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	playerPath := filepath.Join(tmp, "player", "stats.duckdb")
	cacheRoot := filepath.Join(tmp, "cache")

	src := &fakeAppearanceSource{
		app:        &halo5.H5Appearance{Gamertag: "JGtm", ServiceTag: "OKLM", Emblem: halo5.H5AppearanceEmblem{EmblemId: 264}},
		spartanPNG: testPNG,
		emblemPNG:  testPNG,
	}

	out, err := PersistAppearance(ctx, src, playerPath, cacheRoot, "JGtm", "xuid-123")
	if err != nil {
		t.Fatalf("PersistAppearance: %v", err)
	}
	if out.ServiceTag != "OKLM" || !out.SpartanRendered || !out.EmblemRendered || !out.Persisted {
		t.Fatalf("résultat inattendu : %+v", out)
	}

	// PNG écrits au chemin servi local-first (kind/halo_5/{slug}.png).
	for _, kind := range []string{"spartan-banner", "spartan-emblem"} {
		p := filepath.Join(cacheRoot, kind, halo5.TitleSlug, "jgtm.png")
		if _, err := os.Stat(p); err != nil {
			t.Errorf("PNG attendu absent : %s (%v)", p, err)
		}
	}

	// career_progression relue : service tag (spartan_id) + slug relatif (banner/emblem).
	db, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB relecture: %v", err)
	}
	defer db.Close()
	var spartanID, banner, emblem string
	err = db.SQLDb().QueryRowContext(ctx, `
		SELECT COALESCE(spartan_id,''), COALESCE(banner_image_url,''), COALESCE(emblem_image_url,'')
		FROM career_progression WHERE xuid = ? ORDER BY recorded_at DESC LIMIT 1`, "xuid-123").
		Scan(&spartanID, &banner, &emblem)
	if err != nil {
		t.Fatalf("relecture career_progression: %v", err)
	}
	if spartanID != "OKLM" {
		t.Errorf("spartan_id = %q, want OKLM (service tag)", spartanID)
	}
	if banner != "jgtm" || emblem != "jgtm" {
		t.Errorf("banner=%q emblem=%q, want slug relatif 'jgtm'", banner, emblem)
	}
}

// TestPersistAppearance_DelaiCourt : l'écriture passe sous le verrou d'écrivain
// (KindPlayer) de la player DB, qui n'est pas réentrant. Aucun appelant de
// PersistAppearance ne le tient (cron de personnalisation, CLI h5-appearance-backfill) :
// sous un délai court, l'écriture aboutit. Quand un autre écrivain le tient, elle
// échoue à la fin du délai avec dblease.ErrDBLocked, sans rester bloquée.
func TestPersistAppearance_DelaiCourt(t *testing.T) {
	tmp := t.TempDir()
	playerPath := filepath.Join(tmp, "player", "stats.duckdb")
	cacheRoot := filepath.Join(tmp, "cache")
	src := &fakeAppearanceSource{
		app:       &halo5.H5Appearance{Gamertag: "Chocoboflor", ServiceTag: "OKLM"},
		emblemPNG: testPNG,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	out, err := PersistAppearance(ctx, src, playerPath, cacheRoot, "Chocoboflor", "xuid-choco")
	cancel()
	if err != nil || !out.Persisted {
		t.Fatalf("écriture sous délai court, verrou libre : persisted=%v err=%v", out.Persisted, err)
	}

	db, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB : %v", err)
	}
	defer db.Close()
	held, err := dblease.AcquireWriter(db.SQLDb(), db.Path(), dblease.KindPlayer, dblease.PlayerLeaseTimeout)
	if err != nil {
		t.Fatalf("prise du verrou par l'écrivain concurrent : %v", err)
	}
	defer held.Release()

	short, cancelShort := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelShort()
	start := time.Now()
	_, err = PersistAppearance(short, src, playerPath, cacheRoot, "Chocoboflor", "xuid-choco")
	if !errors.Is(err, dblease.ErrDBLocked) {
		t.Fatalf("écriture pendant que le verrou est tenu : err=%v, attendu dblease.ErrDBLocked", err)
	}
	if waited := time.Since(start); waited > 2*time.Second {
		t.Fatalf("écriture restée bloquée %v au-delà de son délai", waited)
	}
}
