package sync

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/historyretry"
)

// engine_history_retry_test.go — le VERDICT de la passe quand l'historique s'interrompt.
//
// Le rejeu lui-même (429, parc en cooldown, 500, 503, durées d'attente) est testé dans
// internal/sync/historyretry. Ici on vérifie ce que la pagination en FAIT : une page perdue
// compte en ERREUR, plus en avertissement, et le statut cesse de mentir (D1, 2026-09-16).
// Aucun sommeil réel : le seam historyretry.Sleep est remplacé.

// clientHistoriqueEnEchec : HaloClient dont GetMatchHistory échoue toujours, avec un compteur.
type clientHistoriqueEnEchec struct {
	*mockHaloClient
	appels int
	err    error
}

func (c *clientHistoriqueEnEchec) GetMatchHistory(
	_ context.Context,
	player, _ string,
	_, _ int,
) ([]MatchHistoryEntry, error) {
	c.appels++
	c.lastHistoryPlayer.Store(player)
	return nil, c.err
}

func TestPaginateAndPersistHistory_429Persistant_ErreurEtStatutFailure(t *testing.T) {
	originalSleep := historyretry.Sleep
	attentes := 0
	historyretry.Sleep = func(context.Context, time.Duration) error { attentes++; return nil }
	t.Cleanup(func() { historyretry.Sleep = originalSleep })

	client := &clientHistoriqueEnEchec{
		mockHaloClient: &mockHaloClient{},
		err: &HTTPError{
			StatusCode: 429,
			URL:        "https://halostats/matches",
			Err:        fmt.Errorf("HTTP 429"),
		},
	}
	e := &SyncEngine{gamertag: "Nuzzles", xuid: "2533274873954866"}
	in := historyPaginationInputs{
		client: client,
		opts:   domain.SyncOptions{MaxMatches: 25, MatchType: "matchmaking"},
	}
	result := &domain.SyncResult{}

	e.paginateAndPersistHistory(context.Background(), in, result)

	if client.appels != historyretry.Attempts {
		t.Errorf("appels GetMatchHistory = %d, attendu %d (rejeu borné)", client.appels, historyretry.Attempts)
	}
	if attentes != 0 {
		t.Errorf("trois 429 n'attendent pas, attentes = %d", attentes)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("erreurs = %v, attendu exactement une", result.Errors)
	}
	if !strings.Contains(result.Errors[0], "historique interrompu à start=0") {
		t.Errorf("message = %q, attendu « historique interrompu à start=0 »", result.Errors[0])
	}
	if len(result.Warnings) != 0 {
		t.Errorf("une page perdue n'est plus un avertissement : %v", result.Warnings)
	}
	if got := result.Status(); got != "failure" {
		t.Errorf("statut = %q, attendu failure (0 match inséré)", got)
	}
	if got, _ := client.lastHistoryPlayer.Load().(string); got != "xuid(2533274873954866)" {
		t.Errorf("format joueur = %q, attendu xuid(NNN)", got)
	}
}
