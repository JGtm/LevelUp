//go:build integration

// Package duckdb — spartan_identity_repo_test.go : tests SpartanIdentityRepo
// (PLAN_SPARTAN_IDENTITY_REFACTOR §11 Phase 1, 2026-05-25).
//
// Couvre :
//   - IsEmpty sur tous les cas (nil / vide / partiel)
//   - Load avec table absente (no-op gracieux)
//   - Load avec row absente (sql.ErrNoRows → nil, nil)
//   - Upsert insert (xuid jamais vu)
//   - Upsert update fresh data (remplace tout + last_refreshed_at)
//   - Upsert update status-only (échec api_empty préserve URLs précédentes)
//   - Upsert erreurs (xuid vide, pdb nil)
package duckdb

import (
	"context"
	"strings"
	"testing"
	"time"
)

const spartanIdentityCreateSQL = `
CREATE TABLE IF NOT EXISTS spartan_identity (
	xuid                VARCHAR PRIMARY KEY,
	spartan_id          VARCHAR,
	banner_image_url    VARCHAR,
	emblem_image_url    VARCHAR,
	backdrop_image_url  VARCHAR,
	last_refreshed_at   TIMESTAMP,
	last_attempt_at     TIMESTAMP,
	last_attempt_status VARCHAR
)`

// seedSpartanIdentityTable crée la table sur la player DB de test.
// Réplique la migration `create_spartan_identity_table` (steps_player_spartan_identity.go).
func seedSpartanIdentityTable(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	if _, err := pdb.Player.Exec(context.Background(), spartanIdentityCreateSQL); err != nil {
		t.Fatalf("create spartan_identity: %v", err)
	}
}

// TestSpartanIdentityRow_IsEmpty : projection vide vs non-vide.
func TestSpartanIdentityRow_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		row  *SpartanIdentityRow
		want bool
	}{
		{"nil", nil, true},
		{"zero", &SpartanIdentityRow{}, true},
		{"xuid only", &SpartanIdentityRow{XUID: "123"}, true},
		{"spartan_id set", &SpartanIdentityRow{SpartanID: "OKLM"}, false},
		{"banner set", &SpartanIdentityRow{BannerImageURL: "url"}, false},
		{"emblem set", &SpartanIdentityRow{EmblemImageURL: "url"}, false},
		{"backdrop set", &SpartanIdentityRow{BackdropImageURL: "url"}, false},
		{"only timestamps set", &SpartanIdentityRow{
			LastRefreshedAt: time.Now(), LastAttemptAt: time.Now(),
		}, true},
	}
	for _, tc := range tests {
		if got := tc.row.IsEmpty(); got != tc.want {
			t.Errorf("%s: IsEmpty = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestSpartanIdentityRepo_Load_TableMissing : no-op gracieux (nil, nil) si
// la migration n'a pas tourné (table absente).
func TestSpartanIdentityRepo_Load_TableMissing(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSpartanIdentityRepo(pdb)

	row, err := repo.Load(context.Background(), "xuid123")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if row != nil {
		t.Errorf("Load returned %v, want nil for missing table", row)
	}
}

// TestSpartanIdentityRepo_Load_RowMissing : table présente mais aucune row
// pour ce xuid → (nil, nil) sans erreur.
func TestSpartanIdentityRepo_Load_RowMissing(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedSpartanIdentityTable(t, pdb)
	repo := NewSpartanIdentityRepo(pdb)

	row, err := repo.Load(context.Background(), "xuid-never-seen")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if row != nil {
		t.Errorf("Load returned %v, want nil for missing row", row)
	}
}

// TestSpartanIdentityRepo_Upsert_InsertNew : 1er upsert pour un xuid → INSERT.
func TestSpartanIdentityRepo_Upsert_InsertNew(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedSpartanIdentityTable(t, pdb)
	repo := NewSpartanIdentityRepo(pdb)
	ctx := context.Background()

	xuid := "2533274823110022"
	data := &SpartanIdentityRow{
		XUID:             xuid,
		SpartanID:        "OKLM",
		BannerImageURL:   "https://halo.api/banner.png",
		EmblemImageURL:   "https://halo.api/emblem.png",
		BackdropImageURL: "https://halo.api/backdrop.png",
	}
	if err := repo.Upsert(ctx, xuid, data, SpartanIdentityStatusOK); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.Load(ctx, xuid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil after Upsert")
	}
	if got.SpartanID != "OKLM" {
		t.Errorf("SpartanID = %q, want OKLM", got.SpartanID)
	}
	if got.BannerImageURL != "https://halo.api/banner.png" {
		t.Errorf("BannerImageURL = %q", got.BannerImageURL)
	}
	if got.LastAttemptStatus != SpartanIdentityStatusOK {
		t.Errorf("LastAttemptStatus = %q, want ok", got.LastAttemptStatus)
	}
	if got.LastRefreshedAt.IsZero() {
		t.Error("LastRefreshedAt is zero, want non-zero after fresh insert")
	}
	if got.LastAttemptAt.IsZero() {
		t.Error("LastAttemptAt is zero")
	}
}

// TestSpartanIdentityRepo_Upsert_UpdateExistingFresh : row existe + data
// fresh → UPDATE complet, last_refreshed_at MAJ.
func TestSpartanIdentityRepo_Upsert_UpdateExistingFresh(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedSpartanIdentityTable(t, pdb)
	repo := NewSpartanIdentityRepo(pdb)
	ctx := context.Background()

	xuid := "2533274823110022"
	// Premier insert.
	first := &SpartanIdentityRow{
		XUID: xuid, SpartanID: "OLD", BannerImageURL: "old-banner",
		EmblemImageURL: "old-emblem", BackdropImageURL: "old-backdrop",
	}
	if err := repo.Upsert(ctx, xuid, first, SpartanIdentityStatusOK); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	firstLoad, _ := repo.Load(ctx, xuid)
	firstRefreshed := firstLoad.LastRefreshedAt

	// Attendre un tick pour garantir que les timestamps diffèrent.
	time.Sleep(2 * time.Millisecond)

	// Second upsert avec nouvelles valeurs.
	second := &SpartanIdentityRow{
		XUID: xuid, SpartanID: "NEW", BannerImageURL: "new-banner",
		EmblemImageURL: "new-emblem", BackdropImageURL: "new-backdrop",
	}
	if err := repo.Upsert(ctx, xuid, second, SpartanIdentityStatusOK); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	got, err := repo.Load(ctx, xuid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.SpartanID != "NEW" {
		t.Errorf("SpartanID = %q, want NEW", got.SpartanID)
	}
	if got.BannerImageURL != "new-banner" {
		t.Errorf("BannerImageURL = %q, want new-banner", got.BannerImageURL)
	}
	if !got.LastRefreshedAt.After(firstRefreshed) {
		t.Errorf("LastRefreshedAt did not advance: first=%v, got=%v", firstRefreshed, got.LastRefreshedAt)
	}
}

// TestSpartanIdentityRepo_Upsert_StatusOnlyPreservesURLs : Upsert avec data
// nil + status non-OK ne doit PAS écraser les URLs précédentes (contrat
// UI-first : "la bannière ne disparaît jamais à cause d'un échec API").
func TestSpartanIdentityRepo_Upsert_StatusOnlyPreservesURLs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedSpartanIdentityTable(t, pdb)
	repo := NewSpartanIdentityRepo(pdb)
	ctx := context.Background()

	xuid := "2533274823110022"
	// Insert initial avec données fraîches.
	fresh := &SpartanIdentityRow{
		XUID: xuid, SpartanID: "OKLM", BannerImageURL: "banner.png",
		EmblemImageURL: "emblem.png", BackdropImageURL: "backdrop.png",
	}
	if err := repo.Upsert(ctx, xuid, fresh, SpartanIdentityStatusOK); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Échec API silencieux : data nil + status api_empty.
	if err := repo.Upsert(ctx, xuid, nil, SpartanIdentityStatusAPIEmpty); err != nil {
		t.Fatalf("status-only Upsert: %v", err)
	}

	got, err := repo.Load(ctx, xuid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// URLs/SpartanID DOIVENT être préservés.
	if got.SpartanID != "OKLM" {
		t.Errorf("SpartanID = %q, want OKLM (preserved)", got.SpartanID)
	}
	if got.BannerImageURL != "banner.png" {
		t.Errorf("BannerImageURL = %q, want banner.png (preserved)", got.BannerImageURL)
	}
	// Status DOIT être mis à jour.
	if got.LastAttemptStatus != SpartanIdentityStatusAPIEmpty {
		t.Errorf("LastAttemptStatus = %q, want api_empty", got.LastAttemptStatus)
	}
}

// TestSpartanIdentityRepo_Upsert_StatusOnlyOnEmptyRow : Upsert status-only
// sur xuid jamais vu → INSERT row vide juste pour tracer le status (utile
// pour le diag "auth_missing").
func TestSpartanIdentityRepo_Upsert_StatusOnlyOnEmptyRow(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedSpartanIdentityTable(t, pdb)
	repo := NewSpartanIdentityRepo(pdb)
	ctx := context.Background()

	xuid := "xuid-never-seen"
	if err := repo.Upsert(ctx, xuid, nil, SpartanIdentityStatusAuthMissing); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := repo.Load(ctx, xuid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got == nil {
		t.Fatal("Load returned nil, want row with status")
	}
	if got.LastAttemptStatus != SpartanIdentityStatusAuthMissing {
		t.Errorf("LastAttemptStatus = %q, want auth_missing", got.LastAttemptStatus)
	}
	if !got.IsEmpty() {
		t.Errorf("row should be IsEmpty=true (no URLs), got %+v", got)
	}
	// last_refreshed_at doit être zéro car aucune donnée fresh n'a été stockée.
	if !got.LastRefreshedAt.IsZero() {
		t.Errorf("LastRefreshedAt = %v, want zero (no fresh data ever)", got.LastRefreshedAt)
	}
}

// TestSpartanIdentityRepo_Upsert_EmptyXUID : erreur sur xuid vide.
func TestSpartanIdentityRepo_Upsert_EmptyXUID(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedSpartanIdentityTable(t, pdb)
	repo := NewSpartanIdentityRepo(pdb)
	if err := repo.Upsert(context.Background(), "", nil, SpartanIdentityStatusOK); err == nil {
		t.Error("expected error on empty xuid")
	} else if !strings.Contains(err.Error(), "xuid") {
		t.Errorf("error = %v, want mention of xuid", err)
	}
}
