//go:build integration

// Package duckdb — career_progression_partial_test.go : tests :memory: pour
// InsertCareerProgressionPartial.
//
// Couvre l'isolation per-field : une nouvelle ligne INSERT avec seulement
// quelques colonnes set ne doit JAMAIS écraser les autres (qui restent
// récupérables via ARG_MAX FILTER côté lecture).
package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

func partialPtr[T any](v T) *T { return &v }

func TestInsertPartial_EmptyPartial_Skip(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()

	inserted, err := repo.InsertCareerProgressionPartial(ctx, "test_xuid", &CareerProgressionPartial{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if inserted {
		t.Error("empty partial should not INSERT")
	}
}

func TestInsertPartial_NilPartial_Skip(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()

	inserted, err := repo.InsertCareerProgressionPartial(ctx, "test_xuid", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if inserted {
		t.Error("nil partial should not INSERT")
	}
}

func TestInsertPartial_EmptyXUID_Error(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()

	_, err := repo.InsertCareerProgressionPartial(ctx, "", &CareerProgressionPartial{Rank: partialPtr(100)})
	if err == nil {
		t.Error("expected error on empty xuid")
	}
}

// TestInsertPartial_BannerOnly_OtherFieldsRemainNullable : INSERT avec
// uniquement banner_image_url. La row insérée doit avoir banner_image_url
// set et tous les autres champs NULL.
func TestInsertPartial_BannerOnly_OtherFieldsRemainNullable(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()
	xuid := "2535469190789936"

	inserted, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		BannerImageURL: partialPtr("https://cdn/banner.png"),
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !inserted {
		t.Fatal("expected INSERT")
	}

	var (
		rankNullable sql.NullInt64
		bannerURL    sql.NullString
		emblemURL    sql.NullString
		spartanID    sql.NullString
	)
	row := pdb.Player.QueryRow(ctx,
		`SELECT rank, banner_image_url, emblem_image_url, spartan_id
		 FROM career_progression WHERE xuid = ?`, xuid)
	if err := row.Scan(&rankNullable, &bannerURL, &emblemURL, &spartanID); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if rankNullable.Valid {
		t.Errorf("rank should be NULL, got %v", rankNullable.Int64)
	}
	if !bannerURL.Valid || bannerURL.String != "https://cdn/banner.png" {
		t.Errorf("banner: %+v", bannerURL)
	}
	if emblemURL.Valid {
		t.Errorf("emblem should be NULL, got %q", emblemURL.String)
	}
	if spartanID.Valid {
		t.Errorf("spartan_id should be NULL, got %q", spartanID.String)
	}
}

// TestInsertPartial_RankOnly_DoesNotPolluteOldBanner : scénario clé du fix.
// Row historique avec banner+emblem ; nouvelle row avec rank only.
// La lecture doit récupérer banner+emblem (historiques) ET rank (frais).
func TestInsertPartial_RankOnly_DoesNotPolluteOldBanner(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()
	xuid := "2535469190789936"

	_, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		BannerImageURL: partialPtr("https://cdn/old-banner.png"),
		EmblemImageURL: partialPtr("https://cdn/old-emblem.png"),
		SpartanID:      partialPtr("OLDID"),
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	inserted, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:      partialPtr(150),
		CurrentXP: partialPtr(2500),
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !inserted {
		t.Fatal("expected INSERT")
	}

	last, err := repo.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if last == nil {
		t.Fatal("nil")
	}
	if last.Rank != 150 {
		t.Errorf("Rank: got %d, want 150", last.Rank)
	}
	if last.CurrentXP != 2500 {
		t.Errorf("CurrentXP: got %d, want 2500", last.CurrentXP)
	}
	if last.BannerImageURL != "https://cdn/old-banner.png" {
		t.Errorf("BannerImageURL: got %q, want carry-forward", last.BannerImageURL)
	}
	if last.EmblemImageURL != "https://cdn/old-emblem.png" {
		t.Errorf("EmblemImageURL: got %q, want carry-forward", last.EmblemImageURL)
	}
	if last.SpartanID != "OLDID" {
		t.Errorf("SpartanID: got %q, want carry-forward", last.SpartanID)
	}
}

// TestInsertPartial_BannerUpdate_PreservesRank : nouveau banner via custom-only
// fetch alors qu'on a un rank existant. Le rank doit être préservé.
func TestInsertPartial_BannerUpdate_PreservesRank(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()
	xuid := "2535469190789936"

	_, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:      partialPtr(180),
		CurrentXP: partialPtr(3000),
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	inserted, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		BannerImageURL: partialPtr("https://cdn/new-banner.png"),
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !inserted {
		t.Fatal("expected INSERT")
	}

	last, err := repo.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if last.Rank != 180 {
		t.Errorf("Rank should carry-forward 180: got %d", last.Rank)
	}
	if last.CurrentXP != 3000 {
		t.Errorf("CurrentXP should carry-forward 3000: got %d", last.CurrentXP)
	}
	if last.BannerImageURL != "https://cdn/new-banner.png" {
		t.Errorf("BannerImageURL should be new: got %q", last.BannerImageURL)
	}
}

func TestInsertPartial_IdenticalToLast_Skip(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()
	xuid := "2535469190789936"

	_, _ = repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:           partialPtr(100),
		BannerImageURL: partialPtr("https://cdn/banner.png"),
	})

	inserted, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:           partialPtr(100),
		BannerImageURL: partialPtr("https://cdn/banner.png"),
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if inserted {
		t.Error("should skip INSERT when identical to last")
	}
}

// TestInsertPartial_MultipleRows_CarryForwardPerField : 3 INSERT successifs
// avec des champs différents. La lecture finale doit récupérer la dernière
// valeur non-vide par champ indépendamment.
func TestInsertPartial_MultipleRows_CarryForwardPerField(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()
	xuid := "2535469190789936"

	_, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:      partialPtr(100),
		CurrentXP: partialPtr(1000),
	})
	if err != nil {
		t.Fatalf("t1: %v", err)
	}
	_, err = repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		BannerImageURL: partialPtr("https://cdn/banner.png"),
		EmblemImageURL: partialPtr("https://cdn/emblem.png"),
	})
	if err != nil {
		t.Fatalf("t2: %v", err)
	}
	_, err = repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:      partialPtr(150),
		CurrentXP: partialPtr(2500),
	})
	if err != nil {
		t.Fatalf("t3: %v", err)
	}

	last, err := repo.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if last.Rank != 150 {
		t.Errorf("Rank: got %d, want 150 (from T3)", last.Rank)
	}
	if last.CurrentXP != 2500 {
		t.Errorf("CurrentXP: got %d, want 2500 (from T3)", last.CurrentXP)
	}
	if last.BannerImageURL != "https://cdn/banner.png" {
		t.Errorf("BannerImageURL: got %q, want from T2", last.BannerImageURL)
	}
	if last.EmblemImageURL != "https://cdn/emblem.png" {
		t.Errorf("EmblemImageURL: got %q, want from T2", last.EmblemImageURL)
	}
}

// TestInsertPartial_AllFieldsSet_RoundTrip : INSERT complet, lecture doit tout
// récupérer.
func TestInsertPartial_AllFieldsSet_RoundTrip(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	ctx := context.Background()
	xuid := "2535469190789936"

	maxRank := true
	_, err := repo.InsertCareerProgressionPartial(ctx, xuid, &CareerProgressionPartial{
		Rank:             partialPtr(255),
		CurrentXP:        partialPtr(0),
		XPForNextRank:    partialPtr(5000),
		XPTotal:          partialPtr(999999),
		IsMaxRank:        &maxRank,
		RankName:         partialPtr("Hero"),
		RankTier:         partialPtr("Onyx"),
		SpartanID:        partialPtr("OKLM"),
		BannerImageURL:   partialPtr("https://cdn/banner.png"),
		EmblemImageURL:   partialPtr("https://cdn/emblem.png"),
		BackdropImageURL: partialPtr("https://cdn/backdrop.png"),
		AdornmentPath:    partialPtr("rank/255.png"),
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	last, err := repo.LoadLastCareerRank(ctx, xuid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if last == nil {
		t.Fatal("nil")
	}
	if last.Rank != 255 || !last.IsMaxRank || last.SpartanID != "OKLM" ||
		last.BannerImageURL != "https://cdn/banner.png" ||
		last.AdornmentPath != "rank/255.png" {
		t.Errorf("round-trip mismatch: %+v", last)
	}
}
