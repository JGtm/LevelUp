// Package sync — engine_mock_test.go : tests unitaires du moteur de sync avec mockHaloClient.
//
// Sprint 52 A6 : ces tests remplacent les dépendances réseau par mockHaloClient.
// Aucun accès internet n'est nécessaire.
// Les tests d'intégration avec DB réelle restent dans engine_test.go (//go:build integration).
package sync

import (
	"context"
	"testing"
)

// ── Tests validation GetMatchHistory ─────────────────────────────────────────

func TestHaloClient_GetMatchHistory_GamertagVide(t *testing.T) {
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	_, err := client.GetMatchHistory(context.Background(), "", "all", 0, 10)
	if err == nil {
		t.Fatal("attendu une erreur pour gamertag vide, got nil")
	}
}

func TestHaloClient_GetMatchHistory_MatchTypeInvalide(t *testing.T) {
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	_, err := client.GetMatchHistory(context.Background(), "Player1", "invalid_type", 0, 10)
	if err == nil {
		t.Fatal("attendu une erreur pour matchType invalide, got nil")
	}
}

func TestHaloClient_GetMatchHistory_CountHorsLimite(t *testing.T) {
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	for _, count := range []int{0, 26, -1} {
		_, err := client.GetMatchHistory(context.Background(), "Player1", "all", 0, count)
		if err == nil {
			t.Errorf("attendu une erreur pour count=%d, got nil", count)
		}
	}
}

func TestHaloClient_GetMatchHistory_StartNegatif(t *testing.T) {
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	_, err := client.GetMatchHistory(context.Background(), "Player1", "all", -1, 10)
	if err == nil {
		t.Fatal("attendu une erreur pour start=-1, got nil")
	}
}

func TestHaloClient_GetMatchHistory_ParamsValides(t *testing.T) {
	// Tous les paramètres sont valides côté validation locale ; l'appel réseau
	// qui suit doit échouer mais SURTOUT pas avec un message de validation.
	//
	// Hermétique (aucun appel réseau réel) : un contexte DÉJÀ ANNULÉ laisse passer
	// la validation locale (params valides) puis fait échouer IMMÉDIATEMENT la
	// couche HTTP sur ctx annulé. Évite la dépendance internet, la latence des
	// retries (HTTP timeout × 4 + backoff) et le flake sous charge parallèle qui
	// faisait échouer le package `internal/sync` en `go test ./...`. Même pattern
	// que TestPooledHaloClientGetCareerRank_PinnedToken.
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetMatchHistory(ctx, "TestPlayer", "matchmaking", 0, 5)
	if err == nil {
		t.Fatal("attendu une erreur réseau (context annulé), got nil")
	}
	errStr := err.Error()
	for _, badMsg := range []string{"gamertag vide", "matchType invalide", "count doit", "start doit"} {
		if contains(errStr, badMsg) {
			t.Errorf("erreur de validation inattendue (les params sont valides) : %v", err)
		}
	}
}

// ── Tests validation GetMatchStats ───────────────────────────────────────────

func TestHaloClient_GetMatchStats_UUIDInvalide(t *testing.T) {
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	_, err := client.GetMatchStats(context.Background(), "pas-un-uuid")
	if err == nil {
		t.Fatal("attendu une erreur pour UUID invalide, got nil")
	}
}

func TestHaloClient_GetMatchFilm_UUIDInvalide(t *testing.T) {
	client := NewHaloAPIClient("spartan-tok", "clearance-tok", 10)
	_, _, err := client.GetMatchFilm(context.Background(), "pas-un-uuid")
	if err == nil {
		t.Fatal("attendu une erreur pour UUID invalide, got nil")
	}
}

// ── Tests mock : mockHaloClient ───────────────────────────────────────────────

// TestMockHaloClient_GetHistory_Fixtures vérifie que le mock retourne les fixtures.
func TestMockHaloClient_GetHistory_Fixtures(t *testing.T) {
	mock := &mockHaloClient{
		history: []MatchHistoryEntry{
			{MatchID: "aabbccdd-0000-0000-0000-000000000001", StartTime: "2024-01-01T00:00:00Z"},
			{MatchID: "aabbccdd-0000-0000-0000-000000000002", StartTime: "2024-01-02T00:00:00Z"},
		},
	}
	entries, err := mock.GetMatchHistory(context.Background(), "Player1", "all", 0, 25)
	if err != nil {
		t.Fatalf("GetMatchHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("attendu 2 entrées, obtenu %d", len(entries))
	}
	if entries[0].MatchID != "aabbccdd-0000-0000-0000-000000000001" {
		t.Errorf("MatchID inattendu : %s", entries[0].MatchID)
	}
}

// TestMockHaloClient_GetHistory_Error vérifie le comportement en cas d'erreur simulée.
func TestMockHaloClient_GetHistory_Error(t *testing.T) {
	mock := &mockHaloClient{getHistoryErr: errNetworkExpected}
	_, err := mock.GetMatchHistory(context.Background(), "Player1", "all", 0, 25)
	if err == nil {
		t.Fatal("attendu une erreur, got nil")
	}
}

// TestMockHaloClient_GetStats_MinimalBody vérifie le corps JSON minimal.
func TestMockHaloClient_GetStats_MinimalBody(t *testing.T) {
	mock := &mockHaloClient{}
	body, err := mock.GetMatchStats(context.Background(), "aabbccdd-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatalf("GetMatchStats: %v", err)
	}
	if _, ok := body["MatchInfo"]; !ok {
		t.Error("corps minimal attendu avec clé MatchInfo")
	}
}

// TestMockHaloClient_GetFilm_RetourneAbsent vérifie que le mock retourne (nil, false, nil).
func TestMockHaloClient_GetFilm_RetourneAbsent(t *testing.T) {
	mock := &mockHaloClient{}
	chunks, ok, err := mock.GetMatchFilm(context.Background(), "aabbccdd-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatalf("GetMatchFilm: %v", err)
	}
	if ok || chunks != nil {
		t.Errorf("attendu (nil, false, nil) pour film absent, got (%v, %v, nil)", chunks, ok)
	}
}

// TestMockHaloClient_GetCareerRank_Data vérifie le retour de la carrière.
func TestMockHaloClient_GetCareerRank_Data(t *testing.T) {
	mock := &mockHaloClient{
		careerData: &CareerRankData{
			XUID:            "1234567890123456",
			CurrentRank:     42,
			CurrentRankName: "Onyx",
			CurrentXP:       1000,
		},
	}
	rank, err := mock.GetCareerRank(context.Background(), "1234567890123456")
	if err != nil {
		t.Fatalf("GetCareerRank: %v", err)
	}
	if rank.CurrentRank != 42 {
		t.Errorf("CurrentRank attendu 42, obtenu %d", rank.CurrentRank)
	}
}

// TestMockHaloClient_CallCount vérifie le comptage des appels.
func TestMockHaloClient_CallCount(t *testing.T) {
	mock := &mockHaloClient{
		history: []MatchHistoryEntry{{MatchID: "aabbccdd-0000-4000-8000-000000000001"}},
	}
	for i := 0; i < 3; i++ {
		_, _ = mock.GetMatchHistory(context.Background(), "Player1", "all", 0, 25)
	}
	if mock.callsGetHistory.Load() != 3 {
		t.Errorf("callsGetHistory = %d, attendu 3", mock.callsGetHistory.Load())
	}
}
