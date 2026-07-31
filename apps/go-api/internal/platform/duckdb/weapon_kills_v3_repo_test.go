//go:build integration

// Package duckdb — weapon_kills_v3_repo_test.go : tests WeaponKillsV3Repo.
//
// Round-trip sur DB :memory: avec la VRAIE migration (RunForDB(TargetShared) applique
// shared_weapon_kills_v3), comme objective_events_repo_test.go.
//
// Lancer avec : go test -tags=integration -run WeaponKillsV3 ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/migration"
)

const (
	wkv3TestMatchID = "m_weapon_v3_001"
	wkv3TestXUID    = "xuid(2533274000000001)"
)

// newWeaponKillsV3TestPlayerDB ouvre une mem DB, applique TOUTES les migrations shared
// (dont shared_weapon_kills_v3 — la "vraie" CREATE sous test), puis construit un
// PlayerDB dont le SharedReader pointe sur cette conn (RW en legacy).
func newWeaponKillsV3TestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sqlDB, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	if err := migration.RunForDB(sqlDB, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(sqlDB, ":memory:")

	return &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// sampleWeaponKillsV3 produit 2 rows : une attribution "high bit" (hash filmshell
// bit63=1) avec tous les champs v3 renseignés, et une row aux NULL-able vides.
func sampleWeaponKillsV3() []domain.WeaponKillV3Row {
	wid := uint64(0xf408190f42c9679f) // bit63=1 — vérifie le CAST UBIGINT
	recon := uint64(0x1234567890abcdef)
	delta := -250
	pIdx := 3
	high := uint32(0xf408190f)
	shotCtr := 7
	hit := true
	burst := false
	return []domain.WeaponKillV3Row{
		{
			TimeMS: 12000, WeaponID: &wid, ReconciledAs: &recon, DeltaMS: &delta,
			Confidence: "exact", AttributionPath: "fire_event",
			SwapDetected: true, DelayedDamage: false, PlayerIndex: &pIdx,
			SourceSignal: "fire_b5", HighWeaponID: &high, KillingShotHit: &hit,
			BurstFinal: &burst, ShotCounter: &shotCtr,
		},
		{
			// Tous les NULL-able vides (weapon non attribuée, signal indispo).
			TimeMS: 45000, Confidence: "none", AttributionPath: "none",
		},
	}
}

// ---------------------------------------------------------------------------
// Round-trip
// ---------------------------------------------------------------------------

func TestWeaponKillsV3Repo_WriteThenLoad_RoundTrip(t *testing.T) {
	pdb := newWeaponKillsV3TestPlayerDB(t)
	repo := NewWeaponKillsV3Repo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, sampleWeaponKillsV3()); err != nil {
		t.Fatalf("WriteMatch: %v", err)
	}

	got, err := repo.LoadMatch(ctx, wkv3TestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(got))
	}

	// Ordonné par (xuid, time_ms) : 12000 avant 45000.
	r0 := got[0]
	if r0.TimeMS != 12000 {
		t.Fatalf("r0.TimeMS = %d, want 12000", r0.TimeMS)
	}
	if r0.WeaponID == nil || *r0.WeaponID != 0xf408190f42c9679f {
		t.Errorf("r0.WeaponID = %v, want 0xf408190f42c9679f (UBIGINT bit63=1)", r0.WeaponID)
	}
	if r0.ReconciledAs == nil || *r0.ReconciledAs != 0x1234567890abcdef {
		t.Errorf("r0.ReconciledAs = %v", r0.ReconciledAs)
	}
	if r0.DeltaMS == nil || *r0.DeltaMS != -250 {
		t.Errorf("r0.DeltaMS = %v, want -250", r0.DeltaMS)
	}
	if r0.Confidence != "exact" || r0.AttributionPath != "fire_event" {
		t.Errorf("r0 conf/path = %q/%q", r0.Confidence, r0.AttributionPath)
	}
	if !r0.SwapDetected || r0.DelayedDamage {
		t.Errorf("r0 swap/delayed = %v/%v, want true/false", r0.SwapDetected, r0.DelayedDamage)
	}
	if r0.PlayerIndex == nil || *r0.PlayerIndex != 3 {
		t.Errorf("r0.PlayerIndex = %v, want 3", r0.PlayerIndex)
	}
	if r0.SourceSignal != "fire_b5" {
		t.Errorf("r0.SourceSignal = %q, want fire_b5", r0.SourceSignal)
	}
	if r0.HighWeaponID == nil || *r0.HighWeaponID != 0xf408190f {
		t.Errorf("r0.HighWeaponID = %v, want 0xf408190f", r0.HighWeaponID)
	}
	if r0.KillingShotHit == nil || !*r0.KillingShotHit {
		t.Errorf("r0.KillingShotHit = %v, want true", r0.KillingShotHit)
	}
	if r0.BurstFinal == nil || *r0.BurstFinal {
		t.Errorf("r0.BurstFinal = %v, want false", r0.BurstFinal)
	}
	if r0.ShotCounter == nil || *r0.ShotCounter != 7 {
		t.Errorf("r0.ShotCounter = %v, want 7", r0.ShotCounter)
	}

	// r1 : tous les NULL-able doivent revenir nil / vides.
	r1 := got[1]
	if r1.TimeMS != 45000 {
		t.Fatalf("r1.TimeMS = %d, want 45000", r1.TimeMS)
	}
	if r1.WeaponID != nil || r1.ReconciledAs != nil || r1.DeltaMS != nil ||
		r1.PlayerIndex != nil || r1.HighWeaponID != nil || r1.KillingShotHit != nil ||
		r1.BurstFinal != nil || r1.ShotCounter != nil {
		t.Errorf("r1 NULL-able non-nil: %+v", r1)
	}
	if r1.SourceSignal != "" {
		t.Errorf("r1.SourceSignal = %q, want empty", r1.SourceSignal)
	}
}

// ---------------------------------------------------------------------------
// Idempotence DELETE-replace
// ---------------------------------------------------------------------------

func TestWeaponKillsV3Repo_WriteMatch_ReplaceIdempotent(t *testing.T) {
	pdb := newWeaponKillsV3TestPlayerDB(t)
	repo := NewWeaponKillsV3Repo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, sampleWeaponKillsV3()); err != nil {
		t.Fatalf("WriteMatch #1: %v", err)
	}
	// Ré-écrit avec 1 seule row : le DELETE doit purger les 2 anciennes.
	replaced := []domain.WeaponKillV3Row{{
		TimeMS: 999, Confidence: "approx", AttributionPath: "timeline",
	}}
	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, replaced); err != nil {
		t.Fatalf("WriteMatch #2: %v", err)
	}

	got, err := repo.LoadMatch(ctx, wkv3TestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(rows) = %d, want 1 (replace)", len(got))
	}
	if got[0].TimeMS != 999 || got[0].Confidence != "approx" {
		t.Errorf("got[0] = %+v, want time=999 conf=approx", got[0])
	}
}

// Le DELETE est scopé par (match_id, xuid) : un autre joueur du même match survit.
func TestWeaponKillsV3Repo_WriteMatch_ScopedByXUID(t *testing.T) {
	pdb := newWeaponKillsV3TestPlayerDB(t)
	repo := NewWeaponKillsV3Repo(pdb)
	ctx := context.Background()
	otherXUID := "xuid(2533274000000002)"

	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, sampleWeaponKillsV3()); err != nil {
		t.Fatalf("WriteMatch self: %v", err)
	}
	if err := repo.WriteMatch(ctx, wkv3TestMatchID, otherXUID, sampleWeaponKillsV3()); err != nil {
		t.Fatalf("WriteMatch other: %v", err)
	}
	// Réécrit self avec 1 row : ne doit PAS toucher les rows de otherXUID.
	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID,
		[]domain.WeaponKillV3Row{{TimeMS: 1, Confidence: "none"}}); err != nil {
		t.Fatalf("WriteMatch self #2: %v", err)
	}

	got, err := repo.LoadMatch(ctx, wkv3TestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	// 1 (self réduit) + 2 (other intact) = 3.
	if len(got) != 3 {
		t.Errorf("len(rows) = %d, want 3 (1 self + 2 other)", len(got))
	}
}

// ---------------------------------------------------------------------------
// No-op garde-fou
// ---------------------------------------------------------------------------

func TestWeaponKillsV3Repo_WriteMatch_EmptyIsNoOp(t *testing.T) {
	pdb := newWeaponKillsV3TestPlayerDB(t)
	repo := NewWeaponKillsV3Repo(pdb)
	ctx := context.Background()

	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, sampleWeaponKillsV3()); err != nil {
		t.Fatalf("WriteMatch seed: %v", err)
	}
	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, nil); err != nil {
		t.Fatalf("WriteMatch(nil): %v", err)
	}
	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, []domain.WeaponKillV3Row{}); err != nil {
		t.Fatalf("WriteMatch([]): %v", err)
	}

	got, err := repo.LoadMatch(ctx, wkv3TestMatchID)
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len(rows) = %d, want 2 (no-op préserve)", len(got))
	}
}

// LoadMatch sur un match absent retourne un slice vide, pas une erreur.
func TestWeaponKillsV3Repo_LoadMatch_Empty(t *testing.T) {
	pdb := newWeaponKillsV3TestPlayerDB(t)
	repo := NewWeaponKillsV3Repo(pdb)

	got, err := repo.LoadMatch(context.Background(), "no_such_match")
	if err != nil {
		t.Fatalf("LoadMatch: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// Capability gating
// ---------------------------------------------------------------------------

func TestWeaponKillsV3Repo_CapabilityNotSupported_NoTable(t *testing.T) {
	pdb := newWeaponKillsV3TestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, "DROP TABLE weapon_kills_v3"); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	repo := NewWeaponKillsV3Repo(pdb)

	if err := repo.WriteMatch(ctx, wkv3TestMatchID, wkv3TestXUID, sampleWeaponKillsV3()); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("WriteMatch err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := repo.LoadMatch(ctx, wkv3TestMatchID); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("LoadMatch err = %v, want ErrCapabilityNotSupported", err)
	}
}
