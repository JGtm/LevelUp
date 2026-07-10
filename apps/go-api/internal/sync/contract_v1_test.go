package sync

// contract_v1_test.go — invariants contractuels testés contre le SyncEngine V1
// (RunDelta), package sync pour accéder à NewSyncEngine + SetCustomClient + le
// mockHaloClient interne. Complète le scaffold black-box de contract_test.go
// (package sync_test, encore skippé pour la suite V2).

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// TestContract_HaloAPIURLFormatXUID_V1 verrouille l'anti-régression du format
// d'URL Halo : GetMatchHistory doit recevoir `xuid(NNN)`, jamais le gamertag brut.
// L'envoi du gamertag brut a causé un incident de 14 jours sans insert (mai 2026 ;
// l'API /matches renvoie alors une réponse stale figée, pas un 404).
func TestContract_HaloAPIURLFormatXUID_V1(t *testing.T) {
	repoRoot := t.TempDir()
	const gamertag = "Chocoboflor"
	const xuid = "2533274823110022"
	tokens := &domain.HaloTokens{SpartanToken: "t", ClearanceToken: "c"}
	engine := NewSyncEngine(repoRoot, gamertag, xuid, tokens, nil)

	// Historique vide : ce contrat ne teste QUE le format d'URL passé à
	// GetMatchHistory (xuid(NNN)), pas la persistance. GetMatchHistory est appelé
	// AVANT tout persist ; un historique vide suffit à capturer l'argument et
	// évite le chemin batch persister (qui exige un SharedProvider câblé, non
	// pertinent pour ce contrat d'URL).
	mock := &mockHaloClient{history: makeHistory()}
	engine.SetCustomClient(mock)

	opts := domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 5,
		WithParticipants: true, WithMedals: true, RequestsPerSecond: 100,
	}
	if _, err := engine.RunDelta(context.Background(), opts); err != nil {
		t.Fatalf("RunDelta: %v", err)
	}

	arg, _ := mock.lastHistoryPlayer.Load().(string)
	if arg == "" {
		t.Fatal("GetMatchHistory jamais appelé (arg player vide)")
	}
	if !strings.HasPrefix(arg, "xuid(") || !strings.HasSuffix(arg, ")") {
		t.Errorf("arg GetMatchHistory = %q, attendu xuid(NNN) (anti-régression mai 2026)", arg)
	}
	if !strings.Contains(arg, xuid) {
		t.Errorf("arg = %q doit contenir le xuid %s", arg, xuid)
	}
	if arg == gamertag {
		t.Error("arg = gamertag brut → incident mai 2026 reproduit")
	}
}
