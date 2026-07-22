package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	halo_games "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/port"
)

// TestExplorerService_PlayerIntersection_DataAdapterParity (HIGH-B Path C) prouve
// que la bascule de l'intersection 2-joueurs (matchs communs + kills croisés) vers
// TitleDataAdapter.LoadPlayerIntersection produit STRICTEMENT le même
// ExplorerPlayerQueryResponse complet (common_matches, badges, encounter_stats,
// wins/losses, activity_heatmap) que le repo legacy. Fixture hétérogène : allié +
// ennemi + égalité, team IDs nil, kills croisés non nuls.
func TestExplorerService_PlayerIntersection_DataAdapterParity(t *testing.T) {
	t.Parallel()
	t1, t2 := 1, 2
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, MapUI: "Aquarius", ModeUI: "Slayer", Player1TeamID: &t1, Player2TeamID: &t1, Player1Outcome: 2, Player1Kills: 20, Player1Deaths: 5, Player1KDA: 4.5},  // alliés, win
		{MatchID: "m2", StartTime: now, MapUI: "Live Fire", ModeUI: "CTF", Player1TeamID: &t1, Player2TeamID: &t2, Player1Outcome: 3, Player1Kills: 8, Player1Deaths: 12, Player1KDA: 0.7},    // ennemis, loss
		{MatchID: "m3", StartTime: now, MapUI: "Streets", ModeUI: "Oddball", Player1TeamID: nil, Player2TeamID: nil, Player1Outcome: 1, Player1Kills: 10, Player1Deaths: 10, Player1KDA: 1.0}, // team nil, tie
	}
	kv := domain.KillerVictimAggregate{KillsDealt: 7, DeathsSuffered: 3}

	legacy, err := NewExplorerService(&mockExplorerRepo{xuid: "target-x", matches: matches, kv: kv}, "self").
		GetCommonMatches(context.Background(), "TargetGT", "", 1)
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}

	repoAdapter := &mockExplorerRepo{xuid: "target-x", matches: matches, kv: kv}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil))).WithCrossPlayerSource(repoAdapter)
	viaAdapter, err := NewExplorerService(repoAdapter, "self").WithDataAdapter(adapter).
		GetCommonMatches(context.Background(), "TargetGT", "", 1)
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	jsonLegacy, _ := json.Marshal(legacy)
	jsonAdapter, _ := json.Marshal(viaAdapter)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("player intersection parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestExplorerService_PlayerIntersection_AdapterFallbackOnUnsupported : adapter sans
// CrossPlayerSource → ErrCapabilityNotSupported → fallback silencieux sur repo.
func TestExplorerService_PlayerIntersection_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()
	t1 := 1
	matches := []domain.CommonMatchRaw{{MatchID: "m1", Player1TeamID: &t1, Player2TeamID: &t1, Player1Outcome: 2, Player1Kills: 10, Player1Deaths: 5}}
	repo := &mockExplorerRepo{xuid: "target-x", matches: matches}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil))) // pas de CrossPlayerSource
	resp, err := NewExplorerService(repo, "self").WithDataAdapter(adapter).
		GetCommonMatches(context.Background(), "TargetGT", "", 1)
	if err != nil {
		t.Fatalf("fallback devrait être silencieux, got %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("TotalCount via fallback = %d, want 1", resp.TotalCount)
	}
}

// --- HIGH-B Path A : parité profil de combat récent via l'adapter ---

// TestExplorerService_CombatProfileLocal_DataAdapterParity prouve que la bascule
// du profil de combat récent vers TitleDataAdapter.LoadTargetRecentMatches produit
// STRICTEMENT le même []ExplorerTargetRecentMatch que le repo legacy. Fixture
// hétérogène : Rank nil (DNF), KDA fractionnaire, PerfectKills>0.
func TestExplorerService_CombatProfileLocal_DataAdapterParity(t *testing.T) {
	t.Parallel()
	rank2 := 2
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	rows := []domain.ExplorerTargetRecentMatch{
		{MatchID: "m1", StartTime: now, MapUI: "Aquarius", ModeUI: "Slayer", Outcome: 2, Rank: &rank2, Kills: 20, Deaths: 5, Assists: 3, KDA: 4.5, Score: 1500, DamageDealt: 3000, DamageTaken: 1000, MaxKillingSpree: 5, PerfectKills: 2},
		{MatchID: "m2", StartTime: now, MapUI: "Live Fire", ModeUI: "CTF", Outcome: 4, Rank: nil, Kills: 1, Deaths: 1, KDA: 1.0}, // DNF + Rank nil
	}

	legacy := NewExplorerService(&mockExplorerRepo{recentMatches: rows}, "self").
		computeTargetCombatProfileLocal(context.Background(), "target")

	repoAdapter := &mockExplorerRepo{recentMatches: rows}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil))).WithRecentSource(repoAdapter)
	viaAdapter := NewExplorerService(repoAdapter, "self").WithDataAdapter(adapter).
		computeTargetCombatProfileLocal(context.Background(), "target")

	jsonLegacy, _ := json.Marshal(legacy)
	jsonAdapter, _ := json.Marshal(viaAdapter)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("combat profile parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestExplorerService_SampleStats_DataAdapterParity (HIGH-B Path B) prouve que la
// bascule de l'agrégat sample stats vers TitleDataAdapter.LoadParticipantStats
// produit STRICTEMENT le même ExplorerTargetSampleStats que le repo legacy. Fixture
// avec DamageDealt/DamageTaken float64 non-entiers (piège de la troncature int).
func TestExplorerService_SampleStats_DataAdapterParity(t *testing.T) {
	t.Parallel()
	agg := &domain.ParticipantStatsAggregate{
		Kills: 50, Deaths: 30, Assists: 20, Wins: 6, Losses: 4, Draws: 0,
		ShotsFired: 500, ShotsHit: 250, DamageDealt: 12345.67, DamageTaken: 9876.54,
		HeadshotKills: 12, MeleeKills: 3, PowerWeaponKills: 5, GrenadeKills: 2,
		TimePlayedSeconds: 3600, PersonalScore: 8000,
	}
	rawMatches := []domain.CommonMatchRaw{{MatchID: "m1"}, {MatchID: "m2"}}

	legacy := NewExplorerService(&mockExplorerRepo{participants: agg}, "self").
		computeTargetSampleStats(context.Background(), "target", rawMatches)

	repoAdapter := &mockExplorerRepo{participants: agg}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil))).WithParticipantSource(repoAdapter)
	viaAdapter := NewExplorerService(repoAdapter, "self").WithDataAdapter(adapter).
		computeTargetSampleStats(context.Background(), "target", rawMatches)

	jsonLegacy, _ := json.Marshal(legacy)
	jsonAdapter, _ := json.Marshal(viaAdapter)
	if string(jsonLegacy) != string(jsonAdapter) {
		t.Errorf("sample stats parity cassée :\nlegacy=  %s\nadapter= %s", jsonLegacy, jsonAdapter)
	}
}

// TestExplorerService_SampleStats_AdapterFallbackOnUnsupported : adapter sans
// ParticipantSource → ErrCapabilityNotSupported → fallback silencieux sur repo.
func TestExplorerService_SampleStats_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()
	agg := &domain.ParticipantStatsAggregate{Kills: 10, Deaths: 5, Wins: 1}
	repo := &mockExplorerRepo{participants: agg}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil))) // pas de ParticipantSource
	out := NewExplorerService(repo, "self").WithDataAdapter(adapter).
		computeTargetSampleStats(context.Background(), "target", []domain.CommonMatchRaw{{MatchID: "m1"}})
	if out == nil {
		t.Error("sample stats via fallback repo = nil, attendu non-nil")
	}
}

// TestExplorerService_TargetFragDistribution_PopulatedFromWeaponRows : quand le loader
// weapon_kills renvoie des armes résolues (classe/rôle), l'encart cible expose la
// « Répartition des frags » v2 (sunburst) + le top armes enrichi. Classes gun (registre)
// + Mêlée/Grenade (compteurs API de l'agrégat) + résidu « Non attribué » ; Σ == total.
func TestExplorerService_TargetFragDistribution_PopulatedFromWeaponRows(t *testing.T) {
	t.Parallel()
	agg := &domain.ParticipantStatsAggregate{Kills: 20, Deaths: 10, MeleeKills: 2, GrenadeKills: 1}
	wk := &fakeExplorerWeaponKillsRepo{rows: []port.WeaponKillRow{
		{XUID: "target", WeaponID: 100, Kills: 8, Label: "BR75", Class: "shoulder", Role: "precision"},
		{XUID: "target", WeaponID: 200, Kills: 5, Label: "Sidekick", Class: "sidearm", Role: "sidearm"},
	}}
	out := NewExplorerService(&mockExplorerRepo{participants: agg}, "self").
		WithWeaponKillsRepo(wk).
		computeTargetSampleStats(context.Background(), "target", []domain.CommonMatchRaw{{MatchID: "m1"}})
	if out == nil || out.FragDistribution == nil {
		t.Fatalf("frag_distribution attendue non-nil, got %+v", out)
	}
	if out.FragDistribution.TotalKills != 20 {
		t.Errorf("TotalKills = %d, want 20", out.FragDistribution.TotalKills)
	}
	if len(out.TopWeaponKills) != 2 {
		t.Errorf("top_weapon_kills = %d, want 2", len(out.TopWeaponKills))
	}
	classes := map[string]int{}
	sum := 0
	for _, c := range out.FragDistribution.Classes {
		classes[c.Class] = c.Kills
		sum += c.Kills
	}
	if classes["shoulder"] != 8 || classes["sidearm"] != 5 {
		t.Errorf("classes gun inattendues: %+v", classes)
	}
	if classes["melee"] != 2 || classes["grenade"] != 1 {
		t.Errorf("classes API mêlée/grenade inattendues: %+v", classes)
	}
	if sum != out.FragDistribution.TotalKills {
		t.Errorf("Σ classes = %d, want == TotalKills %d (unattributed absorbe le résidu)", sum, out.FragDistribution.TotalKills)
	}
}

// TestExplorerService_TargetFragDistribution_NilOnRepoError : échec du loader
// weapon_kills → frag_distribution + top_weapon_kills nil best-effort (les stats
// legacy restent calculées, le front retombe sur le donut kill-type).
func TestExplorerService_TargetFragDistribution_NilOnRepoError(t *testing.T) {
	t.Parallel()
	agg := &domain.ParticipantStatsAggregate{Kills: 12, Deaths: 4}
	wk := &fakeExplorerWeaponKillsRepo{rowsErr: errors.New("db down")}
	out := NewExplorerService(&mockExplorerRepo{participants: agg}, "self").
		WithWeaponKillsRepo(wk).
		computeTargetSampleStats(context.Background(), "target", []domain.CommonMatchRaw{{MatchID: "m1"}})
	if out == nil {
		t.Fatal("sample stats non-nil attendues (best-effort)")
	}
	if out.FragDistribution != nil {
		t.Errorf("frag_distribution attendue nil sur échec repo, got %+v", out.FragDistribution)
	}
	if out.TopWeaponKills != nil {
		t.Errorf("top_weapon_kills attendu nil sur échec repo, got %+v", out.TopWeaponKills)
	}
}

// TestExplorerService_CombatProfileLocal_AdapterFallbackOnUnsupported : adapter sans
// RecentSource → ErrCapabilityNotSupported → fallback silencieux sur repo.
func TestExplorerService_CombatProfileLocal_AdapterFallbackOnUnsupported(t *testing.T) {
	t.Parallel()
	rows := []domain.ExplorerTargetRecentMatch{{MatchID: "m1", Outcome: 2, Kills: 10, Deaths: 5}}
	repo := &mockExplorerRepo{recentMatches: rows}
	adapter := halo_games.NewDataAdapter(nil, slog.New(slog.NewJSONHandler(io.Discard, nil))) // pas de RecentSource
	out := NewExplorerService(repo, "self").WithDataAdapter(adapter).
		computeTargetCombatProfileLocal(context.Background(), "target")
	if len(out) != 1 {
		t.Errorf("profil via fallback repo = %d matchs, want 1", len(out))
	}
}

// --- mock ---

type mockExplorerRepo struct {
	xuid     string
	xuidErr  error
	matches  []domain.CommonMatchRaw
	matchErr error
	kv       domain.KillerVictimAggregate
	kvErr    error
	// Encart target_profile : sample stats sur common_matches.
	participants    *domain.ParticipantStatsAggregate
	participantsErr error
	medals          *domain.MedalCountsAggregate
	medalsErr       error
	startTimes      []time.Time
	startTimesErr   error
	recentMatches   []domain.ExplorerTargetRecentMatch
	recentErr       error
	topWeapons      []domain.WeaponHighlight
	topWeaponsErr   error
}

func (m *mockExplorerRepo) ResolveXUIDByGamertag(_ context.Context, _ string) (string, error) {
	return m.xuid, m.xuidErr
}
func (m *mockExplorerRepo) GetCommonMatches(_ context.Context, _, _ string) ([]domain.CommonMatchRaw, error) {
	return m.matches, m.matchErr
}
func (m *mockExplorerRepo) GetKillerVictimBetween(_ context.Context, _, _ string) (domain.KillerVictimAggregate, error) {
	return m.kv, m.kvErr
}
func (m *mockExplorerRepo) GetParticipantStatsForMatches(_ context.Context, _ string, _ []string) (*domain.ParticipantStatsAggregate, error) {
	return m.participants, m.participantsErr
}
func (m *mockExplorerRepo) GetMedalCountsForMatches(_ context.Context, _ string, _ []string) (*domain.MedalCountsAggregate, error) {
	return m.medals, m.medalsErr
}
func (m *mockExplorerRepo) GetMatchStartTimesForXUID(_ context.Context, _ string) ([]time.Time, error) {
	return m.startTimes, m.startTimesErr
}
func (m *mockExplorerRepo) GetTargetRecentMatches(_ context.Context, _ string, _ int) ([]domain.ExplorerTargetRecentMatch, error) {
	return m.recentMatches, m.recentErr
}
func (m *mockExplorerRepo) TranslateModeUIsFR(_ context.Context, _ []domain.ExplorerTargetRecentMatch) {
}
func (m *mockExplorerRepo) GetTopWeaponsForMatches(_ context.Context, _ string, _ []string, _ int) ([]domain.WeaponHighlight, error) {
	return m.topWeapons, m.topWeaponsErr
}

// fakeExplorerWeaponKillsRepo simule port.WeaponKillsRepository + la capability
// OPTIONNELLE LoadKillMechanicsAggregated (explorerKillMechanicsLoader) pour la
// « Répartition des frags » v2 de l'encart cible.
type fakeExplorerWeaponKillsRepo struct {
	rows    []port.WeaponKillRow
	rowsErr error
	mechs   []port.KillMechanicsRow
}

func (f *fakeExplorerWeaponKillsRepo) LoadWeaponKillsAggregated(_ context.Context, _ string, _ port.WeaponKillFilters) ([]port.WeaponKillRow, error) {
	return f.rows, f.rowsErr
}
func (f *fakeExplorerWeaponKillsRepo) LoadKillMechanicsAggregated(_ context.Context, _ port.WeaponKillFilters) ([]port.KillMechanicsRow, error) {
	return f.mechs, nil
}

// --- tests ---

func TestExplorerService_GetCommonMatches_OK(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 0
	repo := &mockExplorerRepo{
		xuid: "other-xuid",
		matches: []domain.CommonMatchRaw{
			{
				MatchID:       "m1",
				StartTime:     now,
				MapUI:         "Aquarius",
				ModeUI:        "Slayer",
				Player1TeamID: &tid1,
				Player2TeamID: &tid2,
			},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "OtherPlayer", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TargetGamertag != "OtherPlayer" {
		t.Errorf("TargetGamertag = %q, want OtherPlayer", resp.TargetGamertag)
	}
	if resp.TargetXUID != "other-xuid" {
		t.Errorf("TargetXUID = %q, want other-xuid", resp.TargetXUID)
	}
	if resp.Total != 1 {
		t.Errorf("Total = %d, want 1", resp.Total)
	}
	if !resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = true (same team ID)")
	}
}

func TestExplorerService_GetCommonMatches_Empty(t *testing.T) {
	repo := &mockExplorerRepo{
		xuid:    "other-xuid",
		matches: []domain.CommonMatchRaw{},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "Player", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("Total = %d, want 0", resp.Total)
	}
}

func TestExplorerService_GetCommonMatches_ResolveError(t *testing.T) {
	repo := &mockExplorerRepo{xuidErr: errors.New("not found")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Unknown", "", 1)
	if err == nil {
		t.Error("expected error")
	}
}

// TestExplorerService_GetCommonMatches_XUIDProvidedSkipsResolve : quand le xuid
// est fourni (cas du Classement), la résolution gamertag→xuid locale est sautée —
// même si elle échouerait — et la réponse part avec le xuid fourni (le profil live
// reste servi, l'intersection est simplement vide pour un inconnu).
func TestExplorerService_GetCommonMatches_XUIDProvidedSkipsResolve(t *testing.T) {
	// xuidErr forcé : si ResolveXUIDByGamertag était appelé, le test échouerait.
	repo := &mockExplorerRepo{xuidErr: errors.New("not found locally"), matches: []domain.CommonMatchRaw{}}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "WorldStranger", "stranger-xuid", 1)
	if err != nil {
		t.Fatalf("xuid fourni → résolution locale sautée attendue, got %v", err)
	}
	if resp.TargetXUID != "stranger-xuid" {
		t.Errorf("TargetXUID = %q, want stranger-xuid", resp.TargetXUID)
	}
	if resp.TargetGamertag != "WorldStranger" {
		t.Errorf("TargetGamertag = %q, want WorldStranger", resp.TargetGamertag)
	}
}

// TestExplorerService_GetCommonMatches_LiveFallbackResolves : joueur jamais croisé
// (résolution locale → sql.ErrNoRows) mais résolveur live câblé → on continue avec
// le xuid résolu, l'intersection est vide, la réponse part en 200 (le profil cible
// public est servi pour ce xuid).
func TestExplorerService_GetCommonMatches_LiveFallbackResolves(t *testing.T) {
	repo := &mockExplorerRepo{
		xuidErr: fmt.Errorf("ResolveXUIDByGamertag: %w", sql.ErrNoRows),
		matches: []domain.CommonMatchRaw{},
	}
	res := &stubResolver{xuid: "live-xuid-123"}
	svc := NewExplorerService(repo, "my-xuid").WithLiveGamertagResolver(res)

	resp, err := svc.GetCommonMatches(context.Background(), "NeverCrossed", "", 1)
	if err != nil {
		t.Fatalf("fallback live attendu, got %v", err)
	}
	if res.calls != 1 {
		t.Fatalf("résolveur live attendu 1 appel, got %d", res.calls)
	}
	if resp.TargetXUID != "live-xuid-123" {
		t.Errorf("TargetXUID = %q, want live-xuid-123", resp.TargetXUID)
	}
}

// TestExplorerService_GetCommonMatches_LiveFallbackAlsoFails : local ET live
// échouent → erreur (gamertag réellement introuvable).
func TestExplorerService_GetCommonMatches_LiveFallbackAlsoFails(t *testing.T) {
	repo := &mockExplorerRepo{xuidErr: fmt.Errorf("ResolveXUIDByGamertag: %w", sql.ErrNoRows)}
	res := &stubResolver{err: errors.New("xbox profile: aucun profil")}
	svc := NewExplorerService(repo, "my-xuid").WithLiveGamertagResolver(res)

	if _, err := svc.GetCommonMatches(context.Background(), "Ghost", "", 1); err == nil {
		t.Error("erreur attendue quand local ET live échouent")
	}
}

func TestExplorerService_GetCommonMatches_QueryError(t *testing.T) {
	repo := &mockExplorerRepo{xuid: "other", matchErr: errors.New("db fail")}
	svc := NewExplorerService(repo, "my-xuid")

	_, err := svc.GetCommonMatches(context.Background(), "Player", "", 1)
	if err == nil {
		t.Error("expected error")
	}
}

// TestExplorerService_GetCommonMatches_WithStats — vérifie kills/deaths/kda + OutcomeLabel.
func TestExplorerService_GetCommonMatches_WithStats(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 1 // équipes différentes
	repo := &mockExplorerRepo{
		xuid: "enemy-xuid",
		matches: []domain.CommonMatchRaw{
			{
				MatchID:        "golden-match-1",
				StartTime:      now,
				MapUI:          "Recharge",
				ModeUI:         "Slayer",
				Player1TeamID:  &tid1,
				Player2TeamID:  &tid2,
				Player1Outcome: 2, // WIN
				Player1Kills:   18,
				Player1Deaths:  7,
				Player1KDA:     2.57,
			},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "EnemyPlayer", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Fatalf("TotalCount = %d, want 1", resp.TotalCount)
	}
	m := resp.CommonMatches[0]
	if m.Kills != 18 {
		t.Errorf("Kills = %d, want 18", m.Kills)
	}
	if m.Deaths != 7 {
		t.Errorf("Deaths = %d, want 7", m.Deaths)
	}
	if m.KDA == 0.0 {
		t.Error("KDA = 0.0 : kills/deaths/kda ne sont pas propagés depuis match_participants")
	}
	if m.WereTeammates {
		t.Error("WereTeammates attendu false (équipes différentes)")
	}
	if m.PlayerOutcome != 2 {
		t.Errorf("PlayerOutcome = %d, want 2 (WIN)", m.PlayerOutcome)
	}
	if m.OutcomeLabel == "" {
		t.Error("OutcomeLabel vide — doit être résolu via outcomeLabel()")
	}
}

func TestExplorerService_GetCommonMatches_DifferentTeams(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 1
	repo := &mockExplorerRepo{
		xuid: "other",
		matches: []domain.CommonMatchRaw{
			{MatchID: "m1", StartTime: now, Player1TeamID: &tid1, Player2TeamID: &tid2},
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "Enemy", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CommonMatches[0].WereTeammates {
		t.Error("expected WereTeammates = false (different teams)")
	}
}

// TestExplorerService_GetCommonMatches_Pagination vérifie la pagination à 20 éléments.
func TestExplorerService_GetCommonMatches_Pagination(t *testing.T) {
	now := time.Now()
	tid1, tid2 := 0, 0
	// 25 matchs — page 1 = 20, page 2 = 5
	matches := make([]domain.CommonMatchRaw, 25)
	for i := range matches {
		matches[i] = domain.CommonMatchRaw{
			MatchID:       fmt.Sprintf("m%d", i),
			StartTime:     now,
			Player1TeamID: &tid1,
			Player2TeamID: &tid2,
		}
	}
	repo := &mockExplorerRepo{xuid: "other", matches: matches}
	svc := NewExplorerService(repo, "my-xuid")

	p1, err := svc.GetCommonMatches(context.Background(), "Player", "", 1)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(p1.CommonMatches) != 20 {
		t.Errorf("page 1 items = %d, want 20", len(p1.CommonMatches))
	}
	if p1.TotalCount != 25 {
		t.Errorf("TotalCount = %d, want 25", p1.TotalCount)
	}
	if p1.Page != 1 {
		t.Errorf("Page = %d, want 1", p1.Page)
	}

	p2, err := svc.GetCommonMatches(context.Background(), "Player", "", 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(p2.CommonMatches) != 5 {
		t.Errorf("page 2 items = %d, want 5", len(p2.CommonMatches))
	}
}

// TestExplorerService_GetCommonMatches_AllyPlusBadge vérifie que le badge ally_plus
// est émis quand le win rate dépasse le seuil.
func TestExplorerService_GetCommonMatches_AllyPlusBadge(t *testing.T) {
	now := time.Now()
	tid := 0
	// 4 matchs en équipe, 3 victoires → winrate 0.75 > 0.70
	matches := []domain.CommonMatchRaw{
		{MatchID: "a1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a2", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a3", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "a4", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 3},
	}
	repo := &mockExplorerRepo{xuid: "ally-xuid", matches: matches}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(context.Background(), "AllyPlayer", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var hasAllyPlus bool
	for _, b := range resp.Badges {
		if b.Kind == "ally_plus" {
			hasAllyPlus = true
		}
	}
	if !hasAllyPlus {
		t.Error("badge ally_plus attendu (winrate 75% > 70%)")
	}
}

// --- mocks pour les target_profile providers ---

// mockLocalIdentityResolver simule la résolution d'identité locale : renvoie
// `identity` (nil = cible non suivie). Aucun fetch live.
type mockLocalIdentityResolver struct {
	identity *domain.HomeSpartanIdentityRow
	called   bool
}

func (m *mockLocalIdentityResolver) LocalSpartanIdentity(_ context.Context, _ string) *domain.HomeSpartanIdentityRow {
	m.called = true
	return m.identity
}

type mockRemoteStatsProvider struct {
	stats  *domain.NormalizedPlayerStats
	medals []domain.RemoteMedalCount
	err    error
	called bool
}

func (m *mockRemoteStatsProvider) FetchServiceRecord(_ context.Context, _, _ string) (*domain.RemoteServiceRecord, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	if m.stats == nil {
		return nil, nil
	}
	return &domain.RemoteServiceRecord{Stats: *m.stats, Medals: m.medals}, nil
}

// mockLiveIdentity simule le fetch live d'identité (FetchLiveIdentity) d'un xuid
// tiers — cas Explorer d'une cible NON suivie localement (appearance via la vue
// publique). `called` vérifie que le fallback live a bien été emprunté.
type mockLiveIdentity struct {
	identity *domain.HomeSpartanIdentityRow
	err      error
	called   bool
}

func (m *mockLiveIdentity) FetchLiveIdentity(_ context.Context, _ string) (*domain.HomeSpartanIdentityRow, error) {
	m.called = true
	return m.identity, m.err
}

// ctxAuth construit un ctx avec ou sans tokens Halo.
func ctxAuth(hasAuth bool, userXUID string) context.Context {
	ctx := context.Background()
	if hasAuth {
		return ctxkeys.WithHaloAuth(ctx, &domain.HaloTokens{SpartanToken: "spartan-tok"}, userXUID)
	}
	return ctxkeys.WithHaloAuth(ctx, nil, userXUID)
}

// TestExplorerService_TargetProfile_LocalTargetAllSources couvre le cas heureux
// d'une cible LOCALE avec tokens : identité (locale) + carrière (live) + sample
// remplis. Aucune privacy (supprimée).
func TestExplorerService_TargetProfile_LocalTargetAllSources(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
		{MatchID: "m2", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 3},
	}
	repo := &mockExplorerRepo{
		xuid:    "target-xuid",
		matches: matches,
		participants: &domain.ParticipantStatsAggregate{
			Kills: 20, Deaths: 10, Assists: 8,
			Wins: 1, Losses: 1, Draws: 0,
			ShotsFired: 200, ShotsHit: 100,
			DamageDealt: 4500, DamageTaken: 3000,
		},
		medals: &domain.MedalCountsAggregate{Total: 42, Unique: 8},
	}
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{RankNumber: 76, CurrentXP: 47820},
	}
	remoteProv := &mockRemoteStatsProvider{
		stats: &domain.NormalizedPlayerStats{KDA: 1.5, WinRate: 0.6, Accuracy: 0.45},
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: idRes, RemoteStats: remoteProv, TitleSlug: "halo_infinite"})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "TargetPlayer", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	if !tp.AuthAvailable {
		t.Error("AuthAvailable doit être true avec tokens")
	}
	if tp.Identity == nil || tp.Identity.CareerRank == nil || tp.Identity.CareerRank.RankNumber != 76 {
		t.Errorf("Identity attendue career_rank.rank_number=76, got %+v", tp.Identity)
	}
	if !idRes.called {
		t.Error("LocalSpartanIdentity devait être appelé")
	}
	if tp.CareerStats == nil || tp.CareerStats.KDA != 1.5 {
		t.Errorf("CareerStats attendues KDA=1.5, got %+v", tp.CareerStats)
	}
	if tp.SampleStats == nil || tp.SampleStats.SampleSize != 2 {
		t.Errorf("SampleStats attendues SampleSize=2, got %+v", tp.SampleStats)
	}
	if tp.PrivacyWarning != nil {
		t.Errorf("PrivacyWarning doit toujours être nil (privacy supprimée), got %+v", tp.PrivacyWarning)
	}
}

// TestExplorerService_TargetProfile_NoTokens : sans tokens, l'identité LOCALE
// reste résolue (indépendante de l'auth — c'est la logique inversée), seule la
// carrière (live) est skip. Sample présent, privacy toujours nil.
func TestExplorerService_TargetProfile_NoTokens(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{
		xuid:    "target-xuid",
		matches: matches,
		participants: &domain.ParticipantStatsAggregate{
			Kills: 5, Deaths: 3,
		},
	}
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{RankNumber: 12},
	}
	remoteProv := &mockRemoteStatsProvider{
		stats: &domain.NormalizedPlayerStats{KDA: 1.5},
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: idRes, RemoteStats: remoteProv, TitleSlug: "halo_infinite"})

	resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "TargetPlayer", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	if tp.AuthAvailable {
		t.Error("AuthAvailable doit être false sans tokens")
	}
	// Identité local-first : résolue même sans tokens.
	if !idRes.called || tp.Identity == nil || tp.Identity.CareerRank == nil || tp.Identity.CareerRank.RankNumber != 12 {
		t.Errorf("Identity locale attendue career_rank.rank_number=12 même sans tokens, got %+v", tp.Identity)
	}
	if remoteProv.called {
		t.Error("FetchRemoteStats (carrière live) NE doit PAS être appelé sans tokens")
	}
	if tp.CareerStats != nil {
		t.Errorf("CareerStats attendues nil sans tokens, got %+v", tp.CareerStats)
	}
	if tp.PrivacyWarning != nil {
		t.Errorf("PrivacyWarning toujours nil, got %+v", tp.PrivacyWarning)
	}
	if tp.SampleStats == nil || tp.SampleStats.SampleSize != 1 {
		t.Errorf("SampleStats attendues SampleSize=1, got %+v", tp.SampleStats)
	}
}

// TestExplorerService_TargetProfile_CareerFetchError : si le fetch carrière
// échoue, la carrière reste nil mais l'identité (locale) + sample restent
// affichés. Pas de privacy.
func TestExplorerService_TargetProfile_CareerFetchError(t *testing.T) {
	repo := &mockExplorerRepo{xuid: "target", matches: nil}
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{RankNumber: 50},
	}
	remoteProv := &mockRemoteStatsProvider{
		stats: nil,
		err:   errors.New("waypoint 404"),
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: idRes, RemoteStats: remoteProv, TitleSlug: "halo_infinite"})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "SomePlayer", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil")
	}
	if tp.Identity == nil || tp.Identity.CareerRank == nil || tp.Identity.CareerRank.RankNumber != 50 {
		t.Errorf("Identity locale doit rester affichée, got %+v", tp.Identity)
	}
	if tp.CareerStats != nil {
		t.Errorf("CareerStats attendues nil quand FetchRemoteStats échoue, got %+v", tp.CareerStats)
	}
	if tp.PrivacyWarning != nil {
		t.Errorf("PrivacyWarning toujours nil, got %+v", tp.PrivacyWarning)
	}
}

// TestExplorerService_TargetProfile_NoProviders couvre le cas où aucune
// dépendance n'est wirée : TargetProfile reste non-nil mais seules les données
// locales (sample) sont actives.
func TestExplorerService_TargetProfile_NoProviders(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{
		xuid:    "target",
		matches: matches,
		participants: &domain.ParticipantStatsAggregate{
			Kills: 7, Deaths: 4,
		},
	}
	svc := NewExplorerService(repo, "my-xuid")

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Player", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil même sans providers (transport AuthAvailable)")
	}
	if tp.Identity != nil || tp.CareerStats != nil || tp.PrivacyWarning != nil {
		t.Errorf("identity/career/privacy attendus nil sans providers, got %+v", tp)
	}
	if tp.SampleStats == nil || tp.SampleStats.Kills != 7 {
		t.Errorf("SampleStats attendues kills=7 sans providers, got %+v", tp.SampleStats)
	}
}

// TestExplorerService_TargetProfile_OpponentLiveIdentity couvre le chemin #3 :
// cible NON suivie localement (LocalIdentity → nil) → fallback live
// FetchLiveIdentity (appearance servie via la vue publique). Vérifie l'ordre
// (local tenté d'abord), le passage en live, et le fallback bannière → backdrop
// (la vue publique n'expose pas de nameplate dédié).
func TestExplorerService_TargetProfile_OpponentLiveIdentity(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "opp-xuid", matches: matches}

	local := &mockLocalIdentityResolver{identity: nil} // cible non suivie localement
	sid := "MELG"
	backdrop := "/api/v1/assets/spartan/backdrop/halo_infinite/x.png"
	live := &mockLiveIdentity{
		identity: &domain.HomeSpartanIdentityRow{SpartanID: &sid, BackdropImageURL: &backdrop},
	}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			LocalIdentity: local, LiveIdentity: live, TitleSlug: "halo_infinite",
		})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil || tp.Identity == nil {
		t.Fatalf("identité live attendue pour un adversaire, got %+v", tp)
	}
	if !local.called {
		t.Error("LocalSpartanIdentity devait être tenté en premier")
	}
	if !live.called {
		t.Error("FetchLiveIdentity devait être appelé (cible non locale)")
	}
	if tp.Identity.SpartanID == nil || *tp.Identity.SpartanID != "MELG" {
		t.Errorf("SpartanID live attendu MELG, got %+v", tp.Identity.SpartanID)
	}
	if tp.Identity.BannerImageURL == nil || *tp.Identity.BannerImageURL != backdrop {
		t.Errorf("BannerImageURL doit retomber sur le backdrop (vue publique sans nameplate), got %+v",
			tp.Identity.BannerImageURL)
	}
}

// TestExplorerService_TargetProfile_LiveIdentityError : si le fetch live échoue,
// l'identité reste nil (best-effort) sans casser le reste du profil cible.
func TestExplorerService_TargetProfile_LiveIdentityError(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{
		xuid:         "opp-xuid",
		matches:      matches,
		participants: &domain.ParticipantStatsAggregate{Kills: 3, Deaths: 1},
	}
	local := &mockLocalIdentityResolver{identity: nil}
	live := &mockLiveIdentity{err: errors.New("halo down")}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			LocalIdentity: local, LiveIdentity: live, TitleSlug: "halo_infinite",
		})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches ne doit pas échouer sur erreur live: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil {
		t.Fatal("TargetProfile attendu non-nil malgré l'échec live")
	}
	if tp.Identity != nil {
		t.Errorf("Identity attendue nil sur échec live, got %+v", tp.Identity)
	}
	if tp.SampleStats == nil || tp.SampleStats.Kills != 3 {
		t.Errorf("SampleStats doivent rester calculées malgré l'échec live, got %+v", tp.SampleStats)
	}
}

// TestExplorerService_TargetProfile_CombatProfile vérifie que la source LOCALE du
// profil de combat (CombatProfileLocal) est peuplée quand le repo en fournit, et
// nil (best-effort) quand le repo échoue. (Sans provider live + sans auth, la
// source live CombatProfile reste vide.)
func TestExplorerService_TargetProfile_CombatProfile(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	rank := 1
	recent := []domain.ExplorerTargetRecentMatch{
		{MatchID: "r1", StartTime: now, MapUI: "Aquarius", ModeUI: "Slayer", Outcome: 2, Rank: &rank, Kills: 20, Deaths: 8, Assists: 5, KDA: 2.5, Score: 1800, DamageDealt: 4500, DamageTaken: 3000, MaxKillingSpree: 7, PerfectKills: 2},
		{MatchID: "r2", StartTime: now, MapUI: "Recharge", ModeUI: "Oddball", Outcome: 3, Kills: 10, Deaths: 12},
	}

	t.Run("peuplé quand le repo fournit", func(t *testing.T) {
		repo := &mockExplorerRepo{xuid: "target-xuid", matches: matches, recentMatches: recent}
		svc := NewExplorerService(repo, "my-xuid")
		resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "TargetPlayer", "", 1)
		if err != nil {
			t.Fatalf("GetCommonMatches: %v", err)
		}
		tp := resp.TargetProfile
		if tp == nil || len(tp.CombatProfileLocal) != 2 {
			t.Fatalf("CombatProfileLocal attendu 2 matchs, got %+v", tp)
		}
		if tp.CombatProfileLocal[0].MatchID != "r1" || tp.CombatProfileLocal[0].PerfectKills != 2 {
			t.Errorf("CombatProfileLocal[0] = %+v, want r1 perfect=2", tp.CombatProfileLocal[0])
		}
		if tp.CombatProfileLocal[0].Rank == nil || *tp.CombatProfileLocal[0].Rank != 1 {
			t.Errorf("CombatProfileLocal[0].Rank = %v, want 1", tp.CombatProfileLocal[0].Rank)
		}
		// Sans provider live ni auth, la source live reste vide.
		if len(tp.CombatProfile) != 0 {
			t.Errorf("CombatProfile (live) attendu vide sans provider/auth, got %d", len(tp.CombatProfile))
		}
	})

	t.Run("nil quand le repo échoue (best-effort)", func(t *testing.T) {
		repo := &mockExplorerRepo{xuid: "target-xuid", matches: matches, recentErr: errors.New("db down")}
		svc := NewExplorerService(repo, "my-xuid")
		resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "TargetPlayer", "", 1)
		if err != nil {
			t.Fatalf("GetCommonMatches ne doit pas échouer sur erreur combat profile: %v", err)
		}
		if resp.TargetProfile == nil || resp.TargetProfile.CombatProfileLocal != nil {
			t.Errorf("CombatProfileLocal attendu nil sur erreur repo, got %+v", resp.TargetProfile.CombatProfileLocal)
		}
	})
}

// TestExplorerService_TargetProfile_IdentitySerializesAsDTO est le test de
// non-régression du bug central : l'identité cible DOIT sérialiser en DTO
// snake_case avec career_rank imbriqué (comme la Home) — et NON en struct brute
// PascalCase (HomeSpartanIdentityRow à champs de rang plats) que le front,
// lisant identity.banner_image_url / career_rank.rank_title, ne voyait jamais.
// Couvre aussi le cas dégradé Ranks==nil (titre via RankName) et la propagation
// de l'adornment dans le DTO (parité visuelle Phase 3).
func TestExplorerService_TargetProfile_IdentitySerializesAsDTO(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "target-xuid", matches: matches}
	banner := "/api/v1/assets/spartan/banner/halo_infinite/x.png"
	emblem := "/api/v1/assets/spartan/emblem/halo_infinite/x.png"
	adornment := "/api/v1/assets/spartan/adornment/halo_infinite/x.png"
	rankName := "Hero"
	idRes := &mockLocalIdentityResolver{
		identity: &domain.HomeSpartanIdentityRow{
			RankNumber:        76,
			RankName:          &rankName, // Ranks==nil → ce libellé player-DB doit servir
			CurrentXP:         47820,
			XPForNextRank:     50000,
			BannerImageURL:    &banner,
			EmblemImageURL:    &emblem,
			AdornmentImageURL: &adornment,
		},
	}
	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{LocalIdentity: idRes, TitleSlug: "halo_infinite"})

	resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "TargetPlayer", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil || tp.Identity == nil {
		t.Fatalf("identité attendue non-nil, got %+v", tp)
	}
	// career_rank imbriqué (DTO) — plus de champ de rang plat.
	if tp.Identity.CareerRank == nil {
		t.Fatal("career_rank attendu non-nil (DTO imbriqué)")
	}
	if tp.Identity.CareerRank.RankNumber != 76 {
		t.Errorf("career_rank.rank_number = %d, want 76", tp.Identity.CareerRank.RankNumber)
	}
	// Ranks==nil → titre via RankName (fallback player DB).
	if tp.Identity.CareerRank.RankTitle != rankName {
		t.Errorf("rank_title = %q, want %q (fallback RankName quand Ranks nil)",
			tp.Identity.CareerRank.RankTitle, rankName)
	}
	// adornment propagé dans le DTO (Phase 3 parité visuelle Home/Explorer).
	if tp.Identity.CareerRank.AdornmentImageURL == nil ||
		*tp.Identity.CareerRank.AdornmentImageURL != adornment {
		t.Errorf("career_rank.adornment_image_url manquant, got %+v", tp.Identity.CareerRank.AdornmentImageURL)
	}

	// Sérialisation JSON : snake_case + career_rank imbriqué, jamais PascalCase.
	rawJSON, err := json.Marshal(tp.Identity)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	js := string(rawJSON)
	for _, want := range []string{
		`"banner_image_url"`, `"emblem_image_url"`, `"career_rank"`,
		`"rank_title"`, `"adornment_image_url"`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("JSON identité doit contenir %s — got %s", want, js)
		}
	}
	for _, bad := range []string{`"RankNumber"`, `"BannerImageURL"`, `"EmblemImageURL"`} {
		if strings.Contains(js, bad) {
			t.Errorf("JSON identité ne doit PAS contenir la clé PascalCase brute %s (régression) — got %s", bad, js)
		}
	}
}

// TestExplorerService_TargetProfile_DeterministicBannerFallback couvre la
// Phase 3.6 : une cible non-locale dont l'identité live n'a NI bannière NI
// backdrop reçoit une nameplate de repli piochée dans le pool local, de façon
// déterministe par xuid (même joueur → même bannière).
func TestExplorerService_TargetProfile_DeterministicBannerFallback(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "opp-xuid", matches: matches}
	local := &mockLocalIdentityResolver{identity: nil} // cible non suivie
	sid := "ZZZ"
	// Identité live SANS banner ni backdrop (rank pour que le DTO soit non-nil).
	live := &mockLiveIdentity{identity: &domain.HomeSpartanIdentityRow{RankNumber: 5, SpartanID: &sid}}
	pool := []string{"/b/0.png", "/b/1.png", "/b/2.png"}
	poolCalls := 0

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			LocalIdentity: local, LiveIdentity: live, TitleSlug: "halo_infinite",
			LocalBannerPool: func(_ context.Context) []string { poolCalls++; return pool },
		})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil || tp.Identity == nil || tp.Identity.BannerImageURL == nil {
		t.Fatalf("bannière de repli attendue, got %+v", tp)
	}
	want := pickDeterministicBanner("opp-xuid", pool)
	if want == "" {
		t.Fatal("pickDeterministicBanner ne doit pas renvoyer vide pour un pool non vide")
	}
	if *tp.Identity.BannerImageURL != want {
		t.Errorf("bannière = %q, want %q (déterministe par xuid)", *tp.Identity.BannerImageURL, want)
	}
	if poolCalls != 1 {
		t.Errorf("LocalBannerPool doit être appelé exactement 1 fois (lazy), got %d", poolCalls)
	}
}

// mockRecentMatches implémente port.RecentMatchesProvider pour les tests du repli
// live des graphes profil de combat.
type mockRecentMatches struct {
	rows  []domain.ExplorerTargetRecentMatch
	err   error
	calls int
}

func (m *mockRecentMatches) FetchRecentMatches(_ context.Context, _ string, _ int) ([]domain.ExplorerTargetRecentMatch, error) {
	m.calls++
	return m.rows, m.err
}

// TestExplorerService_CombatProfile_LiveIsDefault : CombatProfile (affiché par
// défaut) = le fetch LIVE, même quand le local est vide.
func TestExplorerService_CombatProfile_LiveIsDefault(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "opp-xuid", matches: matches} // recentMatches nil → local vide
	live := &mockRecentMatches{rows: []domain.ExplorerTargetRecentMatch{
		{MatchID: "r1", Kills: 20, Deaths: 5}, {MatchID: "r2", Kills: 8, Deaths: 9},
	}}
	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RecentMatches: live, TitleSlug: "halo_infinite"})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil || len(tp.CombatProfile) != 2 {
		t.Fatalf("CombatProfile (live, défaut) attendu 2, got %+v", tp)
	}
	if len(tp.CombatProfileLocal) != 0 {
		t.Errorf("CombatProfileLocal attendu vide (aucun match local), got %d", len(tp.CombatProfileLocal))
	}
	if live.calls != 1 {
		t.Errorf("RecentMatches (live) doit être appelé 1×, got %d", live.calls)
	}
}

// TestExplorerService_CombatProfile_BothSources : live ET local sont servis en
// parallèle — CombatProfile = live (défaut), CombatProfileLocal = base locale.
func TestExplorerService_CombatProfile_BothSources(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	recent := []domain.ExplorerTargetRecentMatch{{MatchID: "loc1", Kills: 5}}
	repo := &mockExplorerRepo{xuid: "tgt", matches: matches, recentMatches: recent}
	live := &mockRecentMatches{rows: []domain.ExplorerTargetRecentMatch{{MatchID: "r1"}}}
	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{RecentMatches: live, TitleSlug: "halo_infinite"})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Target", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil || len(tp.CombatProfile) != 1 || tp.CombatProfile[0].MatchID != "r1" {
		t.Fatalf("CombatProfile (défaut) doit venir du LIVE (r1), got %+v", tp)
	}
	if len(tp.CombatProfileLocal) != 1 || tp.CombatProfileLocal[0].MatchID != "loc1" {
		t.Fatalf("CombatProfileLocal doit venir de la base (loc1), got %+v", tp.CombatProfileLocal)
	}
	if live.calls != 1 {
		t.Errorf("live doit être appelé 1× (source par défaut), got %d", live.calls)
	}
}

// TestExplorerService_TargetProfile_BannerFallback_NoIdentity couvre le fix du
// point 4 : une cible NON-LOCALE sans identité exploitable (pas d'auth → aucun
// fetch live possible) reçoit malgré tout une nameplate de repli du pool, au
// lieu du placeholder "identité indisponible" sans bannière. C'est le cas qui
// laissait auparavant identityRaw==nil → Identity nil → aucune bannière.
func TestExplorerService_TargetProfile_BannerFallback_NoIdentity(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "opp-xuid", matches: matches}
	pool := []string{"/b/0.png", "/b/1.png", "/b/2.png"}

	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			LocalIdentity:   &mockLocalIdentityResolver{identity: nil}, // cible non suivie
			TitleSlug:       "halo_infinite",
			LocalBannerPool: func(_ context.Context) []string { return pool },
		})

	// hasAuth=false → aucun fetch live ; sans le fix, Identity serait nil.
	resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	if tp == nil || tp.Identity == nil || tp.Identity.BannerImageURL == nil {
		t.Fatalf("bannière de repli attendue même sans identité live, got %+v", tp)
	}
	want := pickDeterministicBanner("opp-xuid", pool)
	if *tp.Identity.BannerImageURL != want {
		t.Errorf("bannière = %q, want %q (déterministe par xuid)", *tp.Identity.BannerImageURL, want)
	}
	// Identité bannière-seule : aucune autre donnée (rang/emblem) injectée.
	if tp.Identity.CareerRank != nil {
		t.Errorf("career_rank attendu nil (identité bannière-seule), got %+v", tp.Identity.CareerRank)
	}
}

// TestExplorerService_TargetProfile_NoBanner_NoPool : sans pool câblé, une cible
// non-locale sans identité reste sans bannière (placeholder front) — pas de
// régression vers une identité fantôme.
func TestExplorerService_TargetProfile_NoBanner_NoPool(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "opp-xuid", matches: matches}
	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			LocalIdentity: &mockLocalIdentityResolver{identity: nil},
			TitleSlug:     "halo_infinite",
		})

	resp, err := svc.GetCommonMatches(ctxAuth(false, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	if tp := resp.TargetProfile; tp == nil || tp.Identity != nil {
		t.Errorf("sans pool, Identity attendue nil, got %+v", resp.TargetProfile)
	}
}

// TestExplorerService_TargetProfile_BannerPoolOverridesLiveNameplate : pour une
// cible NON-locale, la nameplate live (souvent un asset 404) est REMPLACÉE par une
// nameplate du pool local (preferPool), tout en conservant le rang carrière live.
func TestExplorerService_TargetProfile_BannerPoolOverridesLiveNameplate(t *testing.T) {
	now := time.Now()
	tid := 0
	matches := []domain.CommonMatchRaw{
		{MatchID: "m1", StartTime: now, Player1TeamID: &tid, Player2TeamID: &tid, Player1Outcome: 2},
	}
	repo := &mockExplorerRepo{xuid: "opp-xuid", matches: matches}
	brokenNameplate := "/api/v1/assets/spartan/banner/halo_infinite/broken-nameplate.png"
	sid := "ZZZ"
	live := &mockLiveIdentity{identity: &domain.HomeSpartanIdentityRow{
		RankNumber: 5, SpartanID: &sid, BannerImageURL: &brokenNameplate,
	}}
	pool := []string{"/b/0.png", "/b/1.png", "/b/2.png"}
	svc := NewExplorerService(repo, "my-xuid").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{
			LocalIdentity:   &mockLocalIdentityResolver{identity: nil}, // non suivi
			LiveIdentity:    live,
			TitleSlug:       "halo_infinite",
			LocalBannerPool: func(_ context.Context) []string { return pool },
		})

	resp, err := svc.GetCommonMatches(ctxAuth(true, "my-xuid"), "Opponent", "", 1)
	if err != nil {
		t.Fatalf("GetCommonMatches: %v", err)
	}
	tp := resp.TargetProfile
	want := pickDeterministicBanner("opp-xuid", pool)
	if tp == nil || tp.Identity == nil || tp.Identity.BannerImageURL == nil {
		t.Fatalf("bannière attendue (pool), got %+v", tp)
	}
	if *tp.Identity.BannerImageURL != want {
		t.Errorf("bannière = %q, want %q (pool, pas la nameplate live cassée)", *tp.Identity.BannerImageURL, want)
	}
	if tp.Identity.CareerRank == nil {
		t.Error("le rang carrière live doit être conservé malgré l'override de bannière")
	}
}

// TestPickDeterministicBanner vérifie le déterminisme et la borne du pool vide.
func TestPickDeterministicBanner(t *testing.T) {
	pool := []string{"a", "b", "c", "d"}
	first := pickDeterministicBanner("xuid-42", pool)
	if first != pickDeterministicBanner("xuid-42", pool) {
		t.Error("pickDeterministicBanner doit être stable pour un même xuid")
	}
	if pickDeterministicBanner("xuid-42", nil) != "" {
		t.Error("pool vide doit renvoyer la chaîne vide")
	}
}
