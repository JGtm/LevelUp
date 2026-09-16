package sync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/auth/pool"
)

// engine_history_retry_test.go — rejeu borné d'une page d'historique (D1, plan robustesse
// 2026-09-16). Aucun test ne dort : `historyRetrySleep` est remplacé par un enregistreur de
// durées (seam de paquet), et les durées DEMANDÉES sont assertées.
//
// CHACUN de ces tests rougit si la correction est retirée : sans fetchHistoryPage, un 429
// sur une page remonte tel quel (a, b, c) et la passe se déclarait `success` (c).

// historyResponse : une réponse scriptée de GetMatchHistory.
type historyResponse struct {
	entries []MatchHistoryEntry
	err     error
}

// scriptedHistoryClient rend les réponses dans l'ordre du script. Le reste de l'interface
// HaloClient vient du mock déterministe du paquet (halo_client_mock_test.go).
type scriptedHistoryClient struct {
	*mockHaloClient
	script []historyResponse
	calls  int
}

func (c *scriptedHistoryClient) GetMatchHistory(
	_ context.Context,
	player, _ string,
	_, _ int,
) ([]MatchHistoryEntry, error) {
	c.calls++
	c.lastHistoryPlayer.Store(player)
	if c.calls > len(c.script) {
		return nil, fmt.Errorf("script épuisé après %d appels", len(c.script))
	}
	r := c.script[c.calls-1]
	return r.entries, r.err
}

func newScriptedClient(script ...historyResponse) *scriptedHistoryClient {
	return &scriptedHistoryClient{mockHaloClient: &mockHaloClient{}, script: script}
}

// captureHistorySleeps remplace le seam d'attente et rend le pointeur vers les durées
// demandées. Restauré par t.Cleanup.
func captureHistorySleeps(t *testing.T) *[]time.Duration {
	t.Helper()
	waits := make([]time.Duration, 0, 4)
	original := historyRetrySleep
	historyRetrySleep = func(d time.Duration) { waits = append(waits, d) }
	t.Cleanup(func() { historyRetrySleep = original })
	return &waits
}

func testHistoryEngine() (*SyncEngine, historyPaginationInputs) {
	e := &SyncEngine{gamertag: "Nuzzles", xuid: "2533274873954866"}
	in := historyPaginationInputs{opts: domain.SyncOptions{MaxMatches: 25, MatchType: "matchmaking"}}
	return e, in
}

func httpErr(status int) error {
	return &HTTPError{StatusCode: status, URL: "https://halostats/matches", Err: fmt.Errorf("HTTP %d", status)}
}

// (a) 429 puis 200 : la page est obtenue au rejeu IMMÉDIAT (le pool sert un autre slot),
// sans aucune attente.
func TestFetchHistoryPage_429PuisSucces_RejoueSansAttendre(t *testing.T) {
	waits := captureHistorySleeps(t)
	e, in := testHistoryEngine()
	client := newScriptedClient(
		historyResponse{err: httpErr(429)},
		historyResponse{entries: []MatchHistoryEntry{{MatchID: "m1"}}},
	)
	in.client = client

	entries, err := e.fetchHistoryPage(context.Background(), &in, 225)
	if err != nil {
		t.Fatalf("attendu succès au rejeu, obtenu %v", err)
	}
	if len(entries) != 1 || entries[0].MatchID != "m1" {
		t.Fatalf("page inattendue: %+v", entries)
	}
	if client.calls != 2 {
		t.Errorf("appels GetMatchHistory = %d, attendu 2", client.calls)
	}
	if len(*waits) != 0 {
		t.Errorf("un 429 ne doit RIEN attendre, durées demandées: %v", *waits)
	}
	if got := client.lastHistoryPlayer.Load().(string); got != "xuid(2533274873954866)" {
		t.Errorf("format joueur = %q, attendu xuid(NNN)", got)
	}
}

// (b) pool sans slot sain puis 200 : UNE attente, égale à min(cooldown, 60 s).
func TestFetchHistoryPage_PoolSansSlotSain_AttendPuisRejoue(t *testing.T) {
	waits := captureHistorySleeps(t)
	e, in := testHistoryEngine()
	client := newScriptedClient(
		historyResponse{err: fmt.Errorf("pooled: Acquire failed: %w", pool.ErrNoHealthySlot)},
		historyResponse{entries: []MatchHistoryEntry{{MatchID: "m2"}}},
	)
	in.client = client

	entries, err := e.fetchHistoryPage(context.Background(), &in, 0)
	if err != nil {
		t.Fatalf("attendu succès au rejeu, obtenu %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("page inattendue: %+v", entries)
	}
	want := min(historyNoSlotCooldown, historyNoSlotWaitCap)
	if len(*waits) != 1 || (*waits)[0] != want {
		t.Errorf("attentes = %v, attendu exactement [%v]", *waits, want)
	}
}

// (c) 429 sur les trois tentatives : la pagination s'arrête, l'échec est compté en
// ERREUR (plus en avertissement) et le statut devient `failure` faute d'insertion.
func TestPaginateAndPersistHistory_429Persistant_ErreurEtStatutFailure(t *testing.T) {
	waits := captureHistorySleeps(t)
	e, in := testHistoryEngine()
	client := newScriptedClient(
		historyResponse{err: httpErr(429)},
		historyResponse{err: httpErr(429)},
		historyResponse{err: httpErr(429)},
	)
	in.client = client
	result := &domain.SyncResult{}

	e.paginateAndPersistHistory(context.Background(), in, result)

	if client.calls != historyFetchAttempts {
		t.Errorf("appels GetMatchHistory = %d, attendu %d", client.calls, historyFetchAttempts)
	}
	if len(*waits) != 0 {
		t.Errorf("trois 429 n'attendent pas, durées: %v", *waits)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("erreurs = %v, attendu exactement une", result.Errors)
	}
	if !strings.Contains(result.Errors[0], "historique interrompu à start=0") {
		t.Errorf("message = %q, attendu « historique interrompu à start=0 »", result.Errors[0])
	}
	if len(result.Warnings) != 0 {
		t.Errorf("une page perdue n'est plus un avertissement: %v", result.Warnings)
	}
	if got := result.Status(); got != "failure" {
		t.Errorf("statut = %q, attendu failure (0 match inséré)", got)
	}
}

// (d) 500 : erreur remontée immédiatement, aucun rejeu, aucune attente.
func TestFetchHistoryPage_Erreur500_RemonteSansRejeu(t *testing.T) {
	waits := captureHistorySleeps(t)
	e, in := testHistoryEngine()
	client := newScriptedClient(
		historyResponse{err: httpErr(500)},
		historyResponse{entries: []MatchHistoryEntry{{MatchID: "jamais"}}},
	)
	in.client = client

	if _, err := e.fetchHistoryPage(context.Background(), &in, 50); err == nil {
		t.Fatal("attendu une erreur, obtenu nil")
	}
	if client.calls != 1 {
		t.Errorf("appels = %d, attendu 1 (aucun rejeu sur 500)", client.calls)
	}
	if len(*waits) != 0 {
		t.Errorf("aucune attente attendue, durées: %v", *waits)
	}
}

// (e) 503 : DÉJÀ retenté par le client HTTP (doGet) — pas de second étage de rejeu ici.
func TestFetchHistoryPage_Erreur503_NonRejoueeIci(t *testing.T) {
	waits := captureHistorySleeps(t)
	e, in := testHistoryEngine()
	client := newScriptedClient(
		historyResponse{err: httpErr(503)},
		historyResponse{entries: []MatchHistoryEntry{{MatchID: "jamais"}}},
	)
	in.client = client

	_, err := e.fetchHistoryPage(context.Background(), &in, 75)
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != 503 {
		t.Fatalf("attendu le 503 tel quel, obtenu %v", err)
	}
	if client.calls != 1 {
		t.Errorf("appels = %d, attendu 1 (le 503 est retenté dans doGet, pas ici)", client.calls)
	}
	if len(*waits) != 0 {
		t.Errorf("aucune attente attendue, durées: %v", *waits)
	}
}
