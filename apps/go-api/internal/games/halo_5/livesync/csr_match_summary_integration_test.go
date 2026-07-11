//go:build integration

// Package livesync — csr_match_summary_integration_test.go : instrumentation D1 du
// backfill CSR par match. Vérifie que PersistPerMatchRatings VENTILE les skips par
// raison (aucun continue silencieux) sur une DuckDB in-memory + un carnageGetter
// fake — ce qui départage la cause du CSR manquant (carnage KO / joueur absent /
// placement CSR null).
//
// Tag integration : importe DuckDB (CGO) — ne compile pas sur Windows par défaut.
package livesync

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/skill"
)

// fakeCarnage implémente carnageGetter : réponses/erreurs par matchID.
type fakeCarnage struct {
	byID map[string]*halo5.H5CarnageResponse
	err  map[string]error
}

func (f *fakeCarnage) GetMatchCarnage(_ context.Context, matchID, _ string) (*halo5.H5CarnageResponse, error) {
	if e, ok := f.err[matchID]; ok {
		return nil, e
	}
	if r, ok := f.byID[matchID]; ok {
		return r, nil
	}
	return nil, errors.New("fake: match inconnu")
}

func carnageForOwner(gt string, csr *halo5.H5Csr) *halo5.H5CarnageResponse {
	return &halo5.H5CarnageResponse{
		PlayerStats: []halo5.H5CarnagePlayer{
			{Player: halo5.H5PlayerRef{Gamertag: gt}, CurrentCsr: csr},
		},
	}
}

func TestPersistPerMatchRatings_SummaryVentilatesSkips(t *testing.T) {
	ctx := context.Background()
	player, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = player.Close() })
	if err := syncpkg.EnsurePlayerSchema(ctx, player); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}

	shared, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	if _, err := shared.ExecContext(ctx, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, is_ranked BOOLEAN,
		start_time TIMESTAMP, start_time_utc TIMESTAMPTZ)`); err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	// m_csr : classé + CSR → écrit. m_place : classé sans CSR → placement.
	// m_social : non classé (pas de CSR attendu). m_carn : carnage KO. m_absent :
	// carnage sans le joueur. m_noreg : absent du registre.
	for _, q := range []string{
		`INSERT INTO match_registry VALUES ('m_csr', TRUE,  '2026-06-23 06:48:00', '2026-06-23 06:48:00+00')`,
		`INSERT INTO match_registry VALUES ('m_place', TRUE, '2026-06-23 06:49:00', '2026-06-23 06:49:00+00')`,
		`INSERT INTO match_registry VALUES ('m_social', FALSE,'2026-06-23 06:50:00', '2026-06-23 06:50:00+00')`,
		`INSERT INTO match_registry VALUES ('m_carn', TRUE,  '2026-06-23 06:51:00', '2026-06-23 06:51:00+00')`,
		`INSERT INTO match_registry VALUES ('m_absent', TRUE, '2026-06-23 06:52:00', '2026-06-23 06:52:00+00')`,
	} {
		if _, err := shared.ExecContext(ctx, q); err != nil {
			t.Fatalf("insert registry: %v\nSQL: %s", err, q)
		}
	}

	const gt = "JGtm"
	src := &fakeCarnage{
		byID: map[string]*halo5.H5CarnageResponse{
			"m_csr":    carnageForOwner(gt, &halo5.H5Csr{DesignationId: 5, Tier: 0, Csr: 1700}),  // Onyx
			"m_place":  carnageForOwner(gt, nil),                                                 // placement (CurrentCsr nil)
			"m_social": carnageForOwner(gt, nil),                                                 // social : pas de CSR attendu
			"m_absent": carnageForOwner("SomeoneElse", &halo5.H5Csr{DesignationId: 2, Csr: 900}), // owner absent
		},
		err: map[string]error{"m_carn": errors.New("HTTP 401 token mort")},
	}

	ids := []string{"m_csr", "m_place", "m_social", "m_carn", "m_absent", "m_noreg"}
	sum := PersistPerMatchRatings(ctx, src, player, shared, gt, "xuid1", ids)

	if sum.Processed != 6 {
		t.Errorf("Processed = %d, want 6", sum.Processed)
	}
	// D3 : m_csr (CSR réel) + m_place (ligne Placement) → 2 lignes CSR écrites.
	if sum.CSRWritten != 2 {
		t.Errorf("CSRWritten = %d, want 2 (m_csr réel + m_place placement)", sum.CSRWritten)
	}
	if sum.PlacementCSRNull != 1 {
		t.Errorf("PlacementCSRNull = %d, want 1 (m_place classé sans CSR)", sum.PlacementCSRNull)
	}
	if sum.SkipCarnage != 1 {
		t.Errorf("SkipCarnage = %d, want 1 (m_carn 401)", sum.SkipCarnage)
	}
	if sum.SkipOwnerAbsent != 1 {
		t.Errorf("SkipOwnerAbsent = %d, want 1 (m_absent)", sum.SkipOwnerAbsent)
	}
	if sum.SkipRegistry != 1 {
		t.Errorf("SkipRegistry = %d, want 1 (m_noreg)", sum.SkipRegistry)
	}

	// m_social (classé=false, CSR nil) ne produit AUCUNE ligne CSR (ni réelle ni
	// placement — le placement n'est écrit que pour les CLASSÉS). → 2 lignes au total.
	var csrRows int
	if err := player.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank WHERE rating_type='CSR'`).Scan(&csrRows); err != nil {
		t.Fatalf("count CSR: %v", err)
	}
	if csrRows != 2 {
		t.Errorf("match_skill_rank CSR rows = %d, want 2 (m_csr + m_place)", csrRows)
	}

	// La ligne m_place doit être une ligne « Placement » (tier), pas un rang réel.
	var placeTier string
	if err := player.QueryRowContext(ctx,
		`SELECT COALESCE(tier,'') FROM match_skill_rank WHERE match_id='m_place' AND rating_type='CSR'`).Scan(&placeTier); err != nil {
		t.Fatalf("query m_place tier: %v", err)
	}
	if placeTier != skill.TierLabelPlacement {
		t.Errorf("m_place tier = %q, want %q", placeTier, skill.TierLabelPlacement)
	}
}
