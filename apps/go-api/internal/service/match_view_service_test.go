package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// --- mock ---

type mockMatchViewRepo struct {
	meta      *domain.MatchMetaRaw
	metaErr   error
	stats     *domain.PlayerMatchStatsRaw
	statsErr  error
	enrich    *domain.MatchEnrichmentRaw
	enrichErr error
	board     []domain.ScoreboardRaw
	boardErr  error
	medals    []domain.MedalRaw
	medalsErr error
	events    []domain.EventRaw
	eventsErr error
	kvPairs   []domain.KVPairRaw
	// killSources : sources de dégât par mort (Q21b), pour l'arme du kill feed.
	killSources []domain.KillSourceRaw
	kvErr       error
	// notParticipant : si true, IsParticipant renvoie false (gating ADR 0029).
	// Défaut false → "a participé" → comportement inchangé pour les tests existants.
	notParticipant bool
	participantErr error
}

func (m *mockMatchViewRepo) GetMatchMeta(_ context.Context, _ string) (*domain.MatchMetaRaw, error) {
	return m.meta, m.metaErr
}
func (m *mockMatchViewRepo) GetPlayerMatchStats(_ context.Context, _, _ string) (*domain.PlayerMatchStatsRaw, error) {
	return m.stats, m.statsErr
}
func (m *mockMatchViewRepo) IsParticipant(_ context.Context, _, _ string) (bool, error) {
	return !m.notParticipant, m.participantErr
}
func (m *mockMatchViewRepo) GetMatchEnrichment(_ context.Context, _ string) (*domain.MatchEnrichmentRaw, error) {
	return m.enrich, m.enrichErr
}
func (m *mockMatchViewRepo) GetMatchScoreboard(_ context.Context, _ string) ([]domain.ScoreboardRaw, error) {
	return m.board, m.boardErr
}
func (m *mockMatchViewRepo) GetMatchObjectiveScore(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockMatchViewRepo) GetMatchMedals(_ context.Context, _, _ string) ([]domain.MedalRaw, error) {
	return m.medals, m.medalsErr
}
func (m *mockMatchViewRepo) GetMatchEvents(_ context.Context, _ string) ([]domain.EventRaw, error) {
	return m.events, m.eventsErr
}
func (m *mockMatchViewRepo) GetMatchKillSources(_ context.Context, _ string) ([]domain.KillSourceRaw, error) {
	return m.killSources, nil
}
func (m *mockMatchViewRepo) GetMatchKVPairs(_ context.Context, _ string) ([]domain.KVPairRaw, error) {
	return m.kvPairs, m.kvErr
}
func (m *mockMatchViewRepo) GetMatchNeighbors(_ context.Context, _, _ string) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchNeighborsFiltered(_ context.Context, _, _ string, _ *domain.MatchFilterSpec) (*domain.MatchNeighbors, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchEncounters(_ context.Context, _, _ string) ([]domain.EncounterRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchEncounterStats(_ context.Context, _, _ string) ([]domain.EncounterStatsRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchSkillRank(_ context.Context, _ string) (*domain.SkillRankRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchMedia(_ context.Context, _ string) ([]domain.MediaAssocRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchExpectedStats(_ context.Context, _, _ string) (*domain.ExpectedStatsRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchBulkMedals(_ context.Context, _ string) ([]domain.BulkMedalRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchBulkWeaponKills(_ context.Context, _ string) ([]domain.BulkWeaponKillRaw, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetHistoryForAvg(_ context.Context, _ string) ([]domain.MatchHistAvgRow, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetHistoryForAvgBulk(_ context.Context, _ []string) (map[string][]domain.MatchHistAvgRow, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetPlayerAssistsModel(_ context.Context, _ string) (*domain.PlayerAssistsModel, error) {
	return nil, nil
}
func (m *mockMatchViewRepo) GetMatchSharedCSRs(_ context.Context, _ string) (map[string]*domain.SkillRankRaw, error) {
	return nil, nil
}

// --- tests localisation FR ---

// TestMatchViewService_LocalisationFR_ModeMapPlaylistTraduits : cas nominal complet.
// Le repo a résolu ModeNameFR / MapNameFR / PlaylistNameFR via mode_name_tr +
// asset_translations. Le service doit les retransmettre tels quels dans
// header.ModeUI / MapUI / PlaylistLabel sans les tronquer ni les remplacer.
// C'est le test de non-régression principal pour le bug "titre = Forbidden uniquement"
// (2026-05-09) où mode_ui était nil faute de lookup mode_name_tr.
func TestMatchViewService_LocalisationFR_ModeMapPlaylistTraduits(t *testing.T) {
	modeFR, mapFR, plFR := "Capture du drapeau", "Forbidden", "Partie rapide"
	repo := &mockMatchViewRepo{
		meta: &domain.MatchMetaRaw{
			MatchID:        "m-fr",
			ModeNameFR:     &modeFR,
			MapNameFR:      &mapFR,
			PlaylistNameFR: &plFR,
		},
	}
	resp, err := NewMatchViewService(repo, "x").GetMatchView(context.Background(), "m-fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.ModeUI != "Capture du drapeau" {
		t.Errorf("ModeUI = %q, want 'Capture du drapeau' (mode_name_tr lookup)", resp.Header.ModeUI)
	}
	if resp.Header.MapUI != "Forbidden" {
		t.Errorf("MapUI = %q, want 'Forbidden' (asset_translations)", resp.Header.MapUI)
	}
	if resp.Header.PlaylistLabel != "Partie rapide" {
		t.Errorf("PlaylistLabel = %q, want 'Partie rapide' (asset_translations)", resp.Header.PlaylistLabel)
	}
}

// TestMatchViewService_LocalisationFR_ModeNilFallbackNonVide : défense en profondeur.
// Quand ModeNameFR est nil (repo n'a pas pu résoudre), le service ne doit pas
// produire un ModeUI vide — ce qui causerait un titre frontend = "Forbidden" seul.
// Le fallback ResolveModeUI extrait au moins le sous-mode EN du pair_name brut.
func TestMatchViewService_LocalisationFR_ModeNilFallbackNonVide(t *testing.T) {
	pairN := "Arena:CTF on Forbidden"
	repo := &mockMatchViewRepo{
		meta: &domain.MatchMetaRaw{
			MatchID:    "m-fallback",
			PairName:   &pairN,
			ModeNameFR: nil,
		},
	}
	resp, err := NewMatchViewService(repo, "x").GetMatchView(context.Background(), "m-fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.ModeUI == "" {
		t.Errorf("ModeUI vide alors que PairName=%q — frontend afficherait titre = carte seule", pairN)
	}
}

// TestMatchViewService_LocalisationFR_ENFallbackSiPasDeFR : MapNameFR absent →
// MapUI repris depuis MapName (EN) plutôt que chaîne vide.
func TestMatchViewService_LocalisationFR_ENFallbackSiPasDeFR(t *testing.T) {
	mapEN := "Aquarius"
	repo := &mockMatchViewRepo{
		meta: &domain.MatchMetaRaw{
			MatchID: "m-en-fb",
			MapName: &mapEN,
		},
	}
	resp, err := NewMatchViewService(repo, "x").GetMatchView(context.Background(), "m-en-fb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.MapUI != "Aquarius" {
		t.Errorf("MapUI = %q, want 'Aquarius' (fallback EN quand MapNameFR absent)", resp.Header.MapUI)
	}
}

// TestMatchViewService_LocalisationFR_FRPrioritaireSurEN : quand MapNameFR et
// MapName sont tous les deux renseignés, la version FR est utilisée.
func TestMatchViewService_LocalisationFR_FRPrioritaireSurEN(t *testing.T) {
	mapEN, mapFR := "Aquarius", "Verseau"
	repo := &mockMatchViewRepo{
		meta: &domain.MatchMetaRaw{
			MatchID:   "m-fr-pref",
			MapName:   &mapEN,
			MapNameFR: &mapFR,
		},
	}
	resp, err := NewMatchViewService(repo, "x").GetMatchView(context.Background(), "m-fr-pref")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.MapUI != "Verseau" {
		t.Errorf("MapUI = %q, want 'Verseau' (FR prioritaire sur EN)", resp.Header.MapUI)
	}
}

// --- tests ---

func TestMatchViewService_GetMatchView_OK(t *testing.T) {
	now := time.Now()
	mapN, pairN, plN := aquariusMap, slayerMode, sessionCategoryRanked
	dur := 600.0
	repo := &mockMatchViewRepo{
		meta: &domain.MatchMetaRaw{
			MatchID:         "m1",
			StartTime:       &now,
			MapName:         &mapN,
			PairName:        &pairN,
			PlaylistName:    &plN,
			DurationSeconds: &dur,
		},
		stats: &domain.PlayerMatchStatsRaw{
			OutcomeCode: 2, Kills: 15, Deaths: 5, Assists: 3,
		},
		enrich: &domain.MatchEnrichmentRaw{IsWithFriends: true},
		board: []domain.ScoreboardRaw{
			{XUID: "xuid1", Gamertag: "Player1", Kills: 15, Deaths: 5, Assists: 3},
		},
		medals:  []domain.MedalRaw{{MedalID: 1, Count: 2, Label: "Double Kill"}},
		events:  []domain.EventRaw{{EventType: "kill"}},
		kvPairs: []domain.KVPairRaw{{KillerXUID: "x1", VictimXUID: "x2", KillCount: 3}},
	}
	svc := NewMatchViewService(repo, "xuid1")

	resp, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.MatchID != "m1" {
		t.Errorf("MatchID = %q, want m1", resp.Header.MatchID)
	}
}

// TestMatchViewService_GetMatchView_MetaError : un match absent du substrat local
// (jamais synchronisé, ou pas encore) renvoie un APIError 404 typé "not_found"
// (le handler le traduit en code "match_not_found", cf. handlers/match_view.go),
// PAS un fetch live vers l'API du titre — décision user 2026-07-19 (BACKLOG
// "Retirer le fallback LIVE du Match view"). MatchViewService n'expose plus aucun
// hook DataAdapter/viewer gamertag : il est structurellement impossible qu'un
// appel API live parte de ce service.
func TestMatchViewService_GetMatchView_MetaError(t *testing.T) {
	repo := &mockMatchViewRepo{metaErr: errors.New("no rows in result set")}
	svc := NewMatchViewService(repo, "xuid1")

	_, err := svc.GetMatchView(context.Background(), "m1")
	if err == nil {
		t.Fatal("expected error when meta fails")
	}
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("attendu un *domain.APIError typé, obtenu %T: %v", err, err)
	}
	if apiErr.Code != "not_found" {
		t.Errorf("Code = %q, want 'not_found' (traduit en 'match_not_found' par le handler)", apiErr.Code)
	}
	if !strings.Contains(apiErr.Message, "m1") {
		t.Errorf("Message = %q, doit citer le match_id demandé", apiErr.Message)
	}
}

// ADR 0029 Couche B : un joueur non-participant reçoit match_not_participant,
// même si le match existe (meta OK) — fail-fast avant les chargements parallèles.
func TestMatchViewService_GetMatchView_NotParticipant(t *testing.T) {
	now := time.Now()
	repo := &mockMatchViewRepo{
		meta:           &domain.MatchMetaRaw{MatchID: "m1", StartTime: &now},
		notParticipant: true,
	}
	svc := NewMatchViewService(repo, "xuid-absent")

	_, err := svc.GetMatchView(context.Background(), "m1")
	if err == nil {
		t.Fatal("attendu une erreur match_not_participant, obtenu nil")
	}
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "match_not_participant" {
		t.Fatalf("attendu APIError match_not_participant, obtenu %v", err)
	}
}

// Une erreur DB sur la vérification de participation ne bloque pas (best-effort).
func TestMatchViewService_GetMatchView_ParticipationErrorIsBestEffort(t *testing.T) {
	now := time.Now()
	mapN := aquariusMap
	repo := &mockMatchViewRepo{
		meta:           &domain.MatchMetaRaw{MatchID: "m1", StartTime: &now, MapName: &mapN},
		participantErr: errors.New("db down"),
	}
	svc := NewMatchViewService(repo, "xuid1")

	if _, err := svc.GetMatchView(context.Background(), "m1"); err != nil {
		t.Fatalf("erreur de participation devrait être best-effort, obtenu %v", err)
	}
}

func TestMatchViewService_GetMatchView_GracefulDegradation(t *testing.T) {
	now := time.Now()
	repo := &mockMatchViewRepo{
		meta:      &domain.MatchMetaRaw{MatchID: "m1", StartTime: &now},
		statsErr:  errors.New("no stats"),
		enrichErr: errors.New("no enrich"),
		boardErr:  errors.New("no board"),
		medalsErr: errors.New("no medals"),
		eventsErr: errors.New("no events"),
		kvErr:     errors.New("no kv"),
	}
	svc := NewMatchViewService(repo, "xuid1")

	resp, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}
	if resp.Header.MatchID != "m1" {
		t.Errorf("MatchID = %q, want m1", resp.Header.MatchID)
	}
}

func TestMatchViewService_GetMatchView_NilMeta(t *testing.T) {
	repo := &mockMatchViewRepo{meta: nil}
	svc := NewMatchViewService(repo, "xuid1")

	// nil meta (no error) should still return a response
	resp, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = resp
}

// --- Tests Phase 2b : GetMatchNeighborsFiltered ---

// mockNeighborsRepo : capture le spec passé pour vérifier la propagation
// jusqu'au repo. Retourne des neighbors fixes.
type mockNeighborsRepo struct {
	mockMatchViewRepo
	receivedSpec  *domain.MatchFilterSpec
	receivedXUID  string
	receivedMatch string
	returnNil     bool
}

func (m *mockNeighborsRepo) GetMatchNeighborsFiltered(_ context.Context, xuid, matchID string, spec *domain.MatchFilterSpec) (*domain.MatchNeighbors, error) {
	m.receivedSpec = spec
	m.receivedXUID = xuid
	m.receivedMatch = matchID
	if m.returnNil {
		return nil, nil
	}
	prev := "p"
	next := "n"
	return &domain.MatchNeighbors{
		PreviousMatchID: &prev,
		NextMatchID:     &next,
		CurrentIndex:    1,
		TotalMatches:    3,
	}, nil
}

func TestMatchViewService_GetMatchNeighborsFiltered_PropagateSpec(t *testing.T) {
	repo := &mockNeighborsRepo{}
	svc := NewMatchViewService(repo, "xuid-me")

	out := "win"
	spec := &domain.MatchFilterSpec{PlaylistNames: []string{"Ranked Arena"}, Outcome: &out}
	resp, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.receivedXUID != "xuid-me" {
		t.Errorf("xuid = %q, want xuid-me", repo.receivedXUID)
	}
	if repo.receivedMatch != "m1" {
		t.Errorf("match = %q, want m1", repo.receivedMatch)
	}
	if repo.receivedSpec == nil || len(repo.receivedSpec.PlaylistNames) != 1 ||
		repo.receivedSpec.PlaylistNames[0] != "Ranked Arena" {
		t.Errorf("spec not propagated correctly : %+v", repo.receivedSpec)
	}
	// AppliedFilters echo doit être présent quand spec non vide
	if resp.AppliedFilters == nil || len(resp.AppliedFilters.PlaylistNames) == 0 {
		t.Errorf("AppliedFilters echo missing : %+v", resp.AppliedFilters)
	}
	if resp.CurrentIndex != 1 || resp.TotalMatches != 3 {
		t.Errorf("neighbors mal renvoyés : %+v", resp)
	}
}

func TestMatchViewService_GetMatchNeighborsFiltered_EmptySpec_NoAppliedFilters(t *testing.T) {
	repo := &mockNeighborsRepo{}
	svc := NewMatchViewService(repo, "xuid-me")

	resp, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", &domain.MatchFilterSpec{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AppliedFilters != nil {
		t.Errorf("AppliedFilters doit être nil pour spec vide, got %+v", resp.AppliedFilters)
	}
}

func TestMatchViewService_GetMatchNeighborsFiltered_NilSpec(t *testing.T) {
	repo := &mockNeighborsRepo{}
	svc := NewMatchViewService(repo, "xuid-me")

	_, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Le spec nil est passé tel quel au repo (qui doit le déléguer à GetMatchNeighbors)
	if repo.receivedSpec != nil {
		t.Errorf("spec nil doit être propagé tel quel, got %+v", repo.receivedSpec)
	}
}

func TestMatchViewService_GetMatchNeighborsFiltered_NilResult(t *testing.T) {
	repo := &mockNeighborsRepo{returnNil: true}
	svc := NewMatchViewService(repo, "xuid-me")

	resp, err := svc.GetMatchNeighborsFiltered(context.Background(), "m1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PreviousMatchID != nil || resp.NextMatchID != nil {
		t.Errorf("résultat nil doit retourner MatchNeighbors zéro, got %+v", resp)
	}
}
