// Package v2 — e2e_known_set_test.go : la règle « connu » (internal/sync/knownset) vue depuis le
// cycle V2 complet (harnais de e2e_test.go) — enrichissement sans registre re-récupéré en delta,
// base partagée illisible = cycle arrêté sans rien récupérer ni écrire. V2 n'a pas de mode full.
package v2

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/knownset"
)

// TestE2E_V2_EnrichissementSansRegistreReRecupere : la base joueur garde l'enrichissement de
// m_orphelin mais la base partagée (restaurée plus ancienne) ne l'a pas. Le delta ne s'arrête
// PAS sur lui : il est re-récupéré avec m_new, et l'arrêt se fait sur m_old (vraiment connu).
func TestE2E_V2_EnrichissementSansRegistreReRecupere(t *testing.T) {
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
	}
	env := setupE2EEnv(t, []string{"alice"})
	seedKnown(t, env, players[0], "m_old")
	if _, err := env.playerDBs["alice"].SQLDb().Exec(
		"INSERT INTO player_match_enrichment (match_id) VALUES ('m_orphelin')",
	); err != nil {
		t.Fatalf("seed m_orphelin: %v", err)
	}

	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m_new", "m_orphelin", "m_old"),
		},
		statsByMatch: map[string]map[string]any{
			"m_new":      {"placeholder": 1},
			"m_orphelin": {"placeholder": 2},
		},
	}

	orch, _ := buildE2EOrchestrator(t, env, client, players)
	res, err := orch.Run(context.Background(), players)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if res.UniqueMatches != 2 {
		t.Errorf("UniqueMatches = %d, want 2 (m_new + m_orphelin re-récupéré)", res.UniqueMatches)
	}
	if got := client.statsCallCount.Load(); got != 2 {
		t.Errorf("GetMatchStats calls = %d, want 2 (m_old connu non re-récupéré)", got)
	}
}

// TestE2E_V2_BasePartageeIllisibleArreteLeCycle : la connexion partagée n'est pas disponible
// au chargement de l'ensemble connu → le cycle s'arrête avec l'erreur typée, sans pagination
// d'historique, sans fetch, sans persistance, sans post-sync.
func TestE2E_V2_BasePartageeIllisibleArreteLeCycle(t *testing.T) {
	players := []PlayerProfile{
		{Gamertag: "alice", XUID: "1000000000000001", PlayerSlug: "alice"},
		{Gamertag: "bob", XUID: "1000000000000002", PlayerSlug: "bob"},
	}
	env := setupE2EEnv(t, []string{"alice", "bob"})
	seedKnown(t, env, players[0], "m_old")
	env.getShared = func() *sql.DB { return nil }

	client := &mockNarrowClient{
		historyByArg: map[string][]syncpkg.MatchHistoryEntry{
			"xuid(1000000000000001)": histList("m_new", "m_old"),
			"xuid(1000000000000002)": histList("m_new"),
		},
		statsByMatch: map[string]map[string]any{"m_new": {"placeholder": 1}},
	}

	orch, postSyncMock := buildE2EOrchestrator(t, env, client, players)
	res, err := orch.Run(context.Background(), players)
	if !errors.Is(err, knownset.ErrSharedUnreadable) {
		t.Fatalf("err = %v, want knownset.ErrSharedUnreadable", err)
	}
	client.historyArgSeenMu.Lock()
	historyCalls := len(client.historyArgSeen)
	client.historyArgSeenMu.Unlock()
	if historyCalls != 0 {
		t.Errorf("GetMatchHistory calls = %d, want 0", historyCalls)
	}
	if got := client.statsCallCount.Load(); got != 0 {
		t.Errorf("GetMatchStats calls = %d, want 0", got)
	}
	if got, qerr := env.queue.PendingCount(); qerr != nil || got != 0 {
		t.Errorf("batches en attente = %d (err %v), want 0", got, qerr)
	}
	if got := env.acked.Load(); got != 0 {
		t.Errorf("batches persistés = %d, want 0", got)
	}
	if got := postSyncMock.totalCalls.Load(); got != 0 {
		t.Errorf("PostSync calls = %d, want 0", got)
	}
	for _, p := range players {
		if out := res.PerPlayer[p.PlayerSlug]; out.Status != "failed" {
			t.Errorf("PerPlayer[%s].Status = %q, want failed", p.PlayerSlug, out.Status)
		}
	}
}
