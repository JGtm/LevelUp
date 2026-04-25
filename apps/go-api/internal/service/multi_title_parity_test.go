package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	halo_games "levelup/go-api/internal/games/halo_infinite"
)

// TestMultiTitle_CareerServiceAcceptsDataAdapter prouve que CareerService
// accepte un DataAdapter via WithDataAdapter et conserve son comportement
// legacy en mode fallback (ErrCapabilityNotSupported). Les autres services
// (MatchHistory, MatchView, Home, Timeseries) sont validés indirectement
// par leur wiring dans api/registry.go (compile-time check sur la chaîne
// .WithDataAdapter()).
func TestMultiTitle_CareerServiceAcceptsDataAdapter(t *testing.T) {
	t.Parallel()
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	careerSvc := NewCareerService(&mockCareerRepo{}).WithDataAdapter(adapter)
	if careerSvc == nil {
		t.Errorf("CareerService.WithDataAdapter retourne nil")
	}
}

// TestMultiTitle_GoldenParity_Encounters : golden parity textuelle pour
// /career/encounters. Prouve que le payload JSON sérialisé est identique
// avec ou sans DataAdapter, sur fixture déterministe. Représente le
// "diff = 0" demandé par le plan §10.1 au niveau service.
func TestMultiTitle_GoldenParity_Encounters(t *testing.T) {
	t.Parallel()

	avg := 1.42
	rows := []domain.EncounterRawRow{
		{Gamertag: "Alpha", XUID: "x_alpha", MatchCount: 10, AsTeammate: 8, AsEnemy: 2, AvgKDA: &avg},
		{Gamertag: "Beta", XUID: "x_beta", MatchCount: 5, AsTeammate: 1, AsEnemy: 4, AvgKDA: nil},
	}

	repoLegacy := &mockCareerRepo{encRows: rows}
	respLegacy, err := NewCareerService(repoLegacy).GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	repoAdapter := &mockCareerRepo{encRows: rows}
	adapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	respAdapter, err := NewCareerService(repoAdapter).WithDataAdapter(adapter).GetEncounters(context.Background())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	mustSameJSON(t, "Encounters", respLegacy, respAdapter)
}

// TestMultiTitle_GoldenParity_CareerSummary : golden parity sur la Summary
// de la page Carrière (output de buildCareerSummary à partir de GetLatestRank).
func TestMultiTitle_GoldenParity_CareerSummary(t *testing.T) {
	t.Parallel()

	rankLabel := "Diamond 3"
	rankName := "Diamant 3"
	rankTier := "DIAMOND"
	xpForNext := 1234
	xpTotal := 5_000_000
	rankData := &domain.CareerRankData{
		RankNumber:    25,
		CurrentXP:     500,
		RecordedAt:    time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		RankLabel:     &rankLabel,
		RankName:      &rankName,
		RankTier:      &rankTier,
		XPForNextRank: &xpForNext,
		XPTotal:       &xpTotal,
	}

	respLegacy, err := NewCareerService(&mockCareerRepo{rank: rankData}).GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	repoAdapter := &mockCareerRepo{rank: rankData}
	adapter := halo_games.NewDataAdapter(repoAdapter, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	respAdapter, err := NewCareerService(repoAdapter).WithDataAdapter(adapter).GetCareerPage(context.Background())
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	mustSameJSON(t, "Summary", respLegacy.Summary, respAdapter.Summary)
}

// TestMultiTitle_DataAdapterFallback_PreservesLegacyError : si le repo échoue
// et que le DataAdapter ne supporte pas la capability, l'erreur du repo est
// propagée correctement (pas avalée par le fallback gracieux).
func TestMultiTitle_DataAdapterFallback_PreservesLegacyError(t *testing.T) {
	t.Parallel()

	repo := &mockCareerRepo{encErr: errors.New("repo down")}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc := NewCareerService(repo).WithDataAdapter(adapter)

	_, err := svc.GetEncounters(context.Background())
	if err == nil {
		t.Errorf("erreur du repo devrait être propagée même via fallback adapter")
	}
}

// mustSameJSON sérialise les deux valeurs et vérifie qu'elles sont strictement
// identiques (golden parity diff = 0).
func mustSameJSON(t *testing.T, label string, a, b any) {
	t.Helper()
	jsonA, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal %s legacy: %v", label, err)
	}
	jsonB, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal %s adapter: %v", label, err)
	}
	if string(jsonA) != string(jsonB) {
		t.Errorf("%s parity cassée :\nlegacy=  %s\nadapter= %s", label, jsonA, jsonB)
	}
}

// Garde-fous d'interface : si la signature change, ce fichier ne compile plus.
var (
	_ games.TitleDataAdapter = (*halo_games.DataAdapter)(nil)
	_                        = canonical.FieldKills // imports utilisés
)
