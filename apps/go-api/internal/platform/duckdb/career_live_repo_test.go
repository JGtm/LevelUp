//go:build integration

// Package duckdb — career_live_repo_test.go : tests CareerLiveRepo
// (flow live carrière, lecture per-field-merged + INSERT-if-changed +
// enrichment metadata). Couvre 5 méthodes 0% identifiées en Phase 3.
package duckdb

import (
	"context"
	"database/sql"
	"testing"
)

// TestCareerRankRow_IsEmpty : projection rank_row vide vs non-vide.
func TestCareerRankRow_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		row  *CareerRankRow
		want bool
	}{
		{"nil", nil, true},
		{"zero", &CareerRankRow{}, true},
		{"rank only", &CareerRankRow{Rank: 5}, false},
		{"xp only", &CareerRankRow{CurrentXP: 100}, false},
		{"spartan only", &CareerRankRow{SpartanID: "S1"}, false},
		{"banner only", &CareerRankRow{BannerImageURL: "url"}, false},
		// rank_name + xp_for_next_rank + tier + xp_total NE comptent PAS pour IsEmpty
		// car ils sont dérivés via EnrichFromMetadata sans pertinence d'identité.
		{"rank_name only does not save", &CareerRankRow{RankName: "Recruit"}, true},
	}
	for _, tc := range tests {
		if got := tc.row.IsEmpty(); got != tc.want {
			t.Errorf("%s: IsEmpty = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestCareerRankRowEqualForInsert : compare deux rows pour décider d'un INSERT.
func TestCareerRankRowEqualForInsert(t *testing.T) {
	base := &CareerRankRow{
		Rank: 10, CurrentXP: 500, IsMaxRank: false,
		SpartanID: "S1", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1",
	}
	tests := []struct {
		name string
		a, b *CareerRankRow
		want bool
	}{
		{"both nil", nil, nil, true},
		{"one nil", base, nil, false},
		{"identical", base, &CareerRankRow{
			Rank: 10, CurrentXP: 500, IsMaxRank: false,
			SpartanID: "S1", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1",
		}, true},
		// rank_name / rank_tier / xp_for_next_rank / xp_total / adornment_path
		// sont volontairement exclus du compare (cf. doc fonction).
		{"name diff ignored", base, &CareerRankRow{
			Rank: 10, CurrentXP: 500, RankName: "X",
			SpartanID: "S1", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1",
		}, true},
		{"rank diff", base, &CareerRankRow{Rank: 11, CurrentXP: 500,
			SpartanID: "S1", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1"}, false},
		{"xp diff", base, &CareerRankRow{Rank: 10, CurrentXP: 600,
			SpartanID: "S1", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1"}, false},
		{"spartan diff", base, &CareerRankRow{Rank: 10, CurrentXP: 500,
			SpartanID: "S2", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1"}, false},
	}
	for _, tc := range tests {
		if got := CareerRankRowEqualForInsert(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestBuildCareerRankNameDB : composition titre + tier + grade.
func TestBuildCareerRankNameDB(t *testing.T) {
	tests := []struct {
		name     string
		titleEN  string
		tierType string
		grade    sql.NullInt64
		want     string
	}{
		{"empty title", "", "Bronze", sql.NullInt64{Valid: true, Int64: 1}, ""},
		{"title only", "Recruit", "", sql.NullInt64{Valid: false}, "Recruit"},
		{"title + tier", "Lance Corporal", "Bronze", sql.NullInt64{Valid: false}, "Lance Corporal Bronze"},
		{"title + tier + grade", "Lance Corporal", "Bronze", sql.NullInt64{Valid: true, Int64: 3},
			"Lance Corporal Bronze 3"},
		{"grade zero ignored", "Recruit", "", sql.NullInt64{Valid: true, Int64: 0}, "Recruit"},
		{"whitespace trimmed", "  Recruit  ", "  Bronze  ", sql.NullInt64{Valid: true, Int64: 1},
			"Recruit Bronze 1"},
	}
	for _, tc := range tests {
		if got := buildCareerRankNameDB(tc.titleEN, tc.tierType, tc.grade); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestCareerLiveRepo_LoadLastCareerRank_EmptyDB : table career_progression
// vide pour ce xuid → (nil, nil) safe.
func TestCareerLiveRepo_LoadLastCareerRank_EmptyDB(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerLiveRepo(pdb)
	row, err := repo.LoadLastCareerRank(context.Background(), "unknown_xuid")
	if err != nil {
		t.Fatalf("LoadLastCareerRank empty: %v", err)
	}
	if row != nil {
		t.Errorf("attendu nil pour xuid inconnu, obtenu %+v", row)
	}
}

// TestCareerLiveRepo_LoadLastCareerRank_PerFieldMerge cadenasse la sémantique
// apparence (directive produit 2026-07-08, cas JGtm emblème 3806589 sans
// nameplate upstream) : bannière/emblème/backdrop sont des champs
// INDÉPENDANTS — chacun remonte sa dernière valeur non vide, sans couplage
// (« jamais vide » : un champ vide dans le snapshot le plus récent conserve
// sa dernière valeur connue, quel que soit l'état des autres champs).
func TestCareerLiveRepo_LoadLastCareerRank_PerFieldMerge(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// Wipe seed (1 row sans xuid, non pertinente pour ce test ciblé).
	if _, err := pdb.Player.Exec(ctx, `DELETE FROM career_progression`); err != nil {
		t.Fatalf("delete career_progression: %v", err)
	}

	xuidLive := "xuid_live_test"
	// Snapshot 1 (ancien) : banner_image_url renseigné.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
		(xuid, rank, current_xp, recorded_at, rank_name, rank_tier, xp_for_next_rank, xp_total,
		 is_max_rank, adornment_path, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url)
		VALUES (?, 10, 100, '2025-01-01 10:00:00+00', 'A', 'Bronze', 200, 1000, false,
		'/p1.png', 'S1', '/banner_v1.png', '/emblem_v1.png', '/backdrop_v1.png')`,
		xuidLive); err != nil {
		t.Fatalf("insert snapshot 1: %v", err)
	}
	// Snapshot 2 (récent) : banner vide (résolution nameplate échouée sur ce
	// cycle) → carry-forward de la dernière bannière connue.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
		(xuid, rank, current_xp, recorded_at, rank_name, rank_tier, xp_for_next_rank, xp_total,
		 is_max_rank, adornment_path, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url)
		VALUES (?, 11, 150, '2025-02-01 10:00:00+00', 'B', 'Bronze', 300, 1200, false,
		'/p2.png', 'S1', '', '/emblem_v1.png', '/backdrop_v2.png')`,
		xuidLive); err != nil {
		t.Fatalf("insert snapshot 2: %v", err)
	}

	repo := NewCareerLiveRepo(pdb)
	row, err := repo.LoadLastCareerRank(ctx, xuidLive)
	if err != nil {
		t.Fatalf("LoadLastCareerRank: %v", err)
	}
	if row == nil {
		t.Fatal("attendu row non-nil")
	}
	// Rank/XP/SpartanID : valeurs du snapshot 2 (le plus récent).
	if row.Rank != 11 {
		t.Errorf("Rank = %d, want 11", row.Rank)
	}
	if row.CurrentXP != 150 {
		t.Errorf("CurrentXP = %d, want 150", row.CurrentXP)
	}
	// Banner : ARG_MAX FILTER (NULLIF TRIM) remonte le snapshot 1 (non-vide).
	if row.BannerImageURL != "/banner_v1.png" {
		t.Errorf("BannerImageURL = %q, want /banner_v1.png (per-field merge)",
			row.BannerImageURL)
	}
	if row.EmblemImageURL != "/emblem_v1.png" {
		t.Errorf("EmblemImageURL = %q, want /emblem_v1.png", row.EmblemImageURL)
	}

	// Snapshot 3 (plus récent) : emblème mis à jour, banner vide (nameplate
	// irrésoluble upstream). Directive « jamais vide » + indépendance des
	// champs : l'emblème avance, la bannière conserve sa dernière valeur.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
		(xuid, rank, current_xp, recorded_at, rank_name, rank_tier, xp_for_next_rank, xp_total,
		 is_max_rank, adornment_path, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url)
		VALUES (?, 12, 50, '2025-03-01 10:00:00+00', 'C', 'Bronze', 400, 1400, false,
		'/p3.png', 'S1', '', '/emblem_v2.png', '/backdrop_v2.png')`,
		xuidLive); err != nil {
		t.Fatalf("insert snapshot 3: %v", err)
	}
	row, err = repo.LoadLastCareerRank(ctx, xuidLive)
	if err != nil {
		t.Fatalf("LoadLastCareerRank (emblème changé): %v", err)
	}
	if row == nil {
		t.Fatal("attendu row non-nil (emblème changé)")
	}
	if row.EmblemImageURL != "/emblem_v2.png" {
		t.Errorf("EmblemImageURL = %q, want /emblem_v2.png", row.EmblemImageURL)
	}
	if row.BannerImageURL != "/banner_v1.png" {
		t.Errorf("BannerImageURL = %q, want /banner_v1.png (jamais vide : dernière bannière connue)",
			row.BannerImageURL)
	}

	// Snapshot 4 : bannière propre à l'emblème v2 → elle prend le dessus.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
		(xuid, rank, current_xp, recorded_at, rank_name, rank_tier, xp_for_next_rank, xp_total,
		 is_max_rank, adornment_path, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url)
		VALUES (?, 12, 80, '2025-04-01 10:00:00+00', 'C', 'Bronze', 400, 1430, false,
		'/p3.png', 'S1', '/banner_v2.png', '/emblem_v2.png', '/backdrop_v2.png')`,
		xuidLive); err != nil {
		t.Fatalf("insert snapshot 4: %v", err)
	}
	// Snapshot 5 (le plus récent) : l'emblème change encore, bannière non
	// résolue sur ce cycle. Indépendance des champs : l'emblème avance vers
	// v1, la bannière reste la DERNIÈRE non vide (banner_v2) — aucun couplage
	// bannière↔emblème.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO career_progression
		(xuid, rank, current_xp, recorded_at, rank_name, rank_tier, xp_for_next_rank, xp_total,
		 is_max_rank, adornment_path, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url)
		VALUES (?, 13, 10, '2025-05-01 10:00:00+00', 'D', 'Bronze', 500, 1500, false,
		'/p4.png', 'S1', '', '/emblem_v1.png', '/backdrop_v2.png')`,
		xuidLive); err != nil {
		t.Fatalf("insert snapshot 5: %v", err)
	}
	row, err = repo.LoadLastCareerRank(ctx, xuidLive)
	if err != nil {
		t.Fatalf("LoadLastCareerRank (emblème re-changé): %v", err)
	}
	if row == nil {
		t.Fatal("attendu row non-nil (emblème re-changé)")
	}
	if row.EmblemImageURL != "/emblem_v1.png" {
		t.Errorf("EmblemImageURL = %q, want /emblem_v1.png", row.EmblemImageURL)
	}
	if row.BannerImageURL != "/banner_v2.png" {
		t.Errorf("BannerImageURL = %q, want /banner_v2.png (dernière bannière non vide, champs indépendants)",
			row.BannerImageURL)
	}
}

// TestCareerLiveRepo_InsertCareerProgressionIfChanged : skip si identique,
// insert si diff.
func TestCareerLiveRepo_InsertCareerProgressionIfChanged(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `DELETE FROM career_progression`); err != nil {
		t.Fatalf("delete: %v", err)
	}

	repo := NewCareerLiveRepo(pdb)
	xuid := "xuid_insert_test"

	// 1er insert : table vide → doit insérer.
	data := &CareerRankRow{
		Rank: 5, CurrentXP: 250, IsMaxRank: false,
		SpartanID: "S1", BannerImageURL: "b1", EmblemImageURL: "e1", BackdropImageURL: "k1",
	}
	inserted, err := repo.InsertCareerProgressionIfChanged(ctx, xuid, data)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !inserted {
		t.Error("attendu inserted=true pour 1er snapshot")
	}

	// 2e insert avec mêmes valeurs identité → no-op.
	inserted, err = repo.InsertCareerProgressionIfChanged(ctx, xuid, data)
	if err != nil {
		t.Fatalf("dup insert: %v", err)
	}
	if inserted {
		t.Error("attendu inserted=false pour snapshot identique")
	}

	// 3e insert avec rank diff → doit insérer.
	data.Rank = 6
	inserted, err = repo.InsertCareerProgressionIfChanged(ctx, xuid, data)
	if err != nil {
		t.Fatalf("diff insert: %v", err)
	}
	if !inserted {
		t.Error("attendu inserted=true pour rank différent")
	}

	// Edge cases : xuid vide / data nil → erreur défensive.
	if _, err := repo.InsertCareerProgressionIfChanged(ctx, "", data); err == nil {
		t.Error("attendu erreur sur xuid vide")
	}
	if _, err := repo.InsertCareerProgressionIfChanged(ctx, xuid, nil); err == nil {
		t.Error("attendu erreur sur data nil")
	}
}

// TestCareerLiveRepo_EnrichFromMetadata : hydrate rank_name/tier_type/
// xp_for_next_rank/xp_total/adornment_path depuis metadata.career_ranks.
// La table seedée n'a pas les colonnes tier_type/grade/xp_required → on les
// ajoute via ALTER pour ce test.
func TestCareerLiveRepo_EnrichFromMetadata(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()

	// Étendre career_ranks avec les colonnes attendues par EnrichFromMetadata.
	for _, col := range []string{
		`ALTER TABLE career_ranks ADD COLUMN tier_type VARCHAR`,
		`ALTER TABLE career_ranks ADD COLUMN grade BIGINT`,
		`ALTER TABLE career_ranks ADD COLUMN xp_required INTEGER`,
	} {
		if _, err := pdb.Metadata.Exec(ctx, col); err != nil {
			t.Fatalf("alter career_ranks: %v\nSQL: %s", err, col)
		}
	}
	// Hydrater la row rank_id=25 avec tier_type/grade/xp_required.
	if _, err := pdb.Metadata.Exec(ctx, `
		UPDATE career_ranks SET tier_type = 'Platinum', grade = 1, xp_required = 500
		WHERE rank_id = 25`); err != nil {
		t.Fatalf("update rank 25: %v", err)
	}
	// rank 1 (Recruit) : xp_required = 100, sans tier/grade.
	if _, err := pdb.Metadata.Exec(ctx, `
		UPDATE career_ranks SET xp_required = 100 WHERE rank_id = 1`); err != nil {
		t.Fatalf("update rank 1: %v", err)
	}

	repo := NewCareerLiveRepo(pdb)

	// Cas 1 : rank = 0 → no-op.
	row := &CareerRankRow{Rank: 0}
	if err := repo.EnrichFromMetadata(ctx, row); err != nil {
		t.Fatalf("EnrichFromMetadata rank=0: %v", err)
	}
	if row.RankName != "" {
		t.Errorf("rank=0: RankName = %q, want empty", row.RankName)
	}

	// Cas 2 : rank introuvable → no-op, pas d'erreur.
	row = &CareerRankRow{Rank: 999, CurrentXP: 50}
	if err := repo.EnrichFromMetadata(ctx, row); err != nil {
		t.Fatalf("EnrichFromMetadata rank=999: %v", err)
	}
	if row.RankName != "" {
		t.Errorf("rank=999: RankName = %q, want empty (no metadata)", row.RankName)
	}

	// Cas 3 : rank 25 valide → hydratation complète. xp_total = SUM(xp<25) + currentXP.
	// rank 1 : xp_required=100 (seul rank < 25 avec une valeur), rank 25 : xp_required=500.
	row = &CareerRankRow{Rank: 25, CurrentXP: 200}
	if err := repo.EnrichFromMetadata(ctx, row); err != nil {
		t.Fatalf("EnrichFromMetadata rank=25: %v", err)
	}
	if row.RankTier != "Platinum" {
		t.Errorf("RankTier = %q, want Platinum", row.RankTier)
	}
	if row.XPForNextRank != 500 {
		t.Errorf("XPForNextRank = %d, want 500", row.XPForNextRank)
	}
	// xp_total = 100 (sum xp < 25) + 200 (currentXP) = 300
	if row.XPTotal != 300 {
		t.Errorf("XPTotal = %d, want 300 (100 sum + 200 current)", row.XPTotal)
	}
	if row.AdornmentPath != "Progression/RewardTracks/CareerRanks/platinum1-adornment.png" {
		t.Errorf("AdornmentPath = %q", row.AdornmentPath)
	}
	// RankName : title_en="Lance Corporal" + tier_type="Platinum" + grade=1.
	if row.RankName != "Lance Corporal Platinum 1" {
		t.Errorf("RankName = %q, want \"Lance Corporal Platinum 1\"", row.RankName)
	}
}
