// overtime_wiring_test.go — branchement du flag « Prolongation » sur les DEUX
// surfaces consommatrices : le header de la Match View et les lignes
// Explorer/historique. Couvre variante connue / inconnue / titre sans table.
package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// haloInfiniteRegulation : extrait de la table réelle (regulation.toml).
var haloInfiniteRegulation = map[string]int{
	"CTF:Arena":         720,
	"Team Slayer:Arena": 720,
}

// --- Match View header -----------------------------------------------------

func TestApplyMatchHeaderOvertime_KnownVariant(t *testing.T) {
	h := domain.MatchViewHeader{}
	meta := &domain.MatchMetaRaw{
		GameVariantName: strPtr("CTF:Arena"),
		ElapsedSeconds:  intPtr(774),
	}
	applyMatchHeaderOvertime(&h, meta, haloInfiniteRegulation)
	if !h.IsOvertime || h.OvertimeSeconds != 54 {
		t.Errorf("header = (%v, %d), want (true, 54)", h.IsOvertime, h.OvertimeSeconds)
	}
}

func TestApplyMatchHeaderOvertime_KnownVariantWithinRegulation(t *testing.T) {
	h := domain.MatchViewHeader{}
	meta := &domain.MatchMetaRaw{
		GameVariantName: strPtr("CTF:Arena"),
		ElapsedSeconds:  intPtr(728),
	}
	applyMatchHeaderOvertime(&h, meta, haloInfiniteRegulation)
	if h.IsOvertime || h.OvertimeSeconds != 0 {
		t.Errorf("header = (%v, %d), want (false, 0)", h.IsOvertime, h.OvertimeSeconds)
	}
}

// Variante absente de la table (renommage saisonnier, BTB, mode à manches) :
// jamais de flag, jamais d'erreur.
func TestApplyMatchHeaderOvertime_UnknownVariant(t *testing.T) {
	h := domain.MatchViewHeader{}
	meta := &domain.MatchMetaRaw{
		GameVariantName: strPtr("BTB:Slayer"),
		ElapsedSeconds:  intPtr(990),
	}
	applyMatchHeaderOvertime(&h, meta, haloInfiniteRegulation)
	if h.IsOvertime || h.OvertimeSeconds != 0 {
		t.Errorf("variante inconnue : header = (%v, %d), want (false, 0)", h.IsOvertime, h.OvertimeSeconds)
	}
}

// Titre sans table réglementaire (Halo 5) → aucun flag.
func TestApplyMatchHeaderOvertime_TitleWithoutRegulation(t *testing.T) {
	for _, table := range []map[string]int{nil, {}} {
		h := domain.MatchViewHeader{}
		meta := &domain.MatchMetaRaw{
			GameVariantName: strPtr("CTF:Arena"),
			ElapsedSeconds:  intPtr(990),
		}
		applyMatchHeaderOvertime(&h, meta, table)
		if h.IsOvertime || h.OvertimeSeconds != 0 {
			t.Errorf("titre sans table : header = (%v, %d), want (false, 0)", h.IsOvertime, h.OvertimeSeconds)
		}
	}
}

// Durée non estimable (participation jamais backfillée) → aucun flag.
func TestApplyMatchHeaderOvertime_NoElapsed(t *testing.T) {
	h := domain.MatchViewHeader{}
	meta := &domain.MatchMetaRaw{GameVariantName: strPtr("CTF:Arena")}
	applyMatchHeaderOvertime(&h, meta, haloInfiniteRegulation)
	if h.IsOvertime {
		t.Error("durée absente : aucun flag attendu")
	}
	// meta nil : no-op, pas de panic.
	applyMatchHeaderOvertime(&h, nil, haloInfiniteRegulation)
}

// --- Lignes Explorer / historique ------------------------------------------

func overtimeRows() []domain.MatchHistoryRawRow {
	now := time.Now()
	mapName := aquariusMap
	pair := slayerMode
	return []domain.MatchHistoryRawRow{
		{ // prolongation sur variante connue
			MatchID: "ot", StartTime: &now, MapName: &mapName, PairName: &pair, Outcome: 2,
			GameVariantName: strPtr("Team Slayer:Arena"), ElapsedSeconds: intPtr(763),
		},
		{ // dans le temps
			MatchID: "reg", StartTime: &now, MapName: &mapName, PairName: &pair, Outcome: 3,
			GameVariantName: strPtr("Team Slayer:Arena"), ElapsedSeconds: intPtr(721),
		},
		{ // variante inconnue, très longue → jamais flaguée
			MatchID: "unknown", StartTime: &now, MapName: &mapName, PairName: &pair, Outcome: 2,
			GameVariantName: strPtr("BTB:Total Control"), ElapsedSeconds: intPtr(1200),
		},
	}
}

func explorerRowsByID(t *testing.T, svc *MatchHistoryService) map[string]domain.MatchHistoryRow {
	t.Helper()
	resp, err := svc.GetPage(context.Background(), domain.MatchHistoryQueryRequest{
		Pagination: domain.PaginationRequest{Page: 1, PageSize: 20},
	})
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}
	out := map[string]domain.MatchHistoryRow{}
	for _, it := range resp.Table.Items {
		out[it.MatchID] = it
	}
	return out
}

func TestMatchHistoryService_OvertimeFlags(t *testing.T) {
	svc := NewMatchHistoryService(&mockMatchHistoryRepo{rows: overtimeRows()}, "Player").
		WithRegulation(haloInfiniteRegulation)

	rows := explorerRowsByID(t, svc)
	if got := rows["ot"]; !got.IsOvertime || got.OvertimeSeconds != 43 {
		t.Errorf("ligne ot = (%v, %d), want (true, 43)", got.IsOvertime, got.OvertimeSeconds)
	}
	if got := rows["reg"]; got.IsOvertime || got.OvertimeSeconds != 0 {
		t.Errorf("ligne reg = (%v, %d), want (false, 0)", got.IsOvertime, got.OvertimeSeconds)
	}
	if got := rows["unknown"]; got.IsOvertime || got.OvertimeSeconds != 0 {
		t.Errorf("ligne variante inconnue = (%v, %d), want (false, 0)", got.IsOvertime, got.OvertimeSeconds)
	}
}

// Service SANS table injectée (titre sans regulation.toml) : aucune ligne flaguée.
func TestMatchHistoryService_NoRegulationNeverFlags(t *testing.T) {
	svc := NewMatchHistoryService(&mockMatchHistoryRepo{rows: overtimeRows()}, "Player")

	for id, row := range explorerRowsByID(t, svc) {
		if row.IsOvertime || row.OvertimeSeconds != 0 {
			t.Errorf("ligne %s flaguée sans table réglementaire : (%v, %d)", id, row.IsOvertime, row.OvertimeSeconds)
		}
	}
}
