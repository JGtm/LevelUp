// Package v2 — cycle_history_drain_test.go : un cycle dont l'attente de la file de persistance
// expire (DrainTimeoutError) reste un cycle réussi pour ce qui est prouvé rangé : post-sync des
// matchs rangés, compte juste, et aucun match encore en file n'est un échec de rattrapage (D-16).
package v2

import (
	"context"
	"sort"
	gosync "sync"
	"testing"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// recordingPostSync note les identifiants de matchs que le cycle remet au post-sync de chaque joueur.
type recordingPostSync struct {
	mu    gosync.Mutex
	calls int
	ids   []string
}

func (r *recordingPostSync) RunPostSync(_ context.Context, _ PlayerProfile, inserted []string) (PlayerPostSyncResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.ids = append(r.ids, inserted...)
	return PlayerPostSyncResult{}, nil
}

// cycleRecording joue un cycle pour alice avec un post-sync qui note ses entrées.
func (e *histEnv) cycleRecording(post *recordingPostSync) (PlayerOutcome, error) {
	e.t.Helper()
	e.client.starts = nil
	provider := NewMatchListProvider(func(string, string) HaloClient { return e.client }, "matchmaking", 25, 20)
	orch := NewCycleOrchestrator(e, provider, e.fetcher, &mockEnrichmentFetcher{}, e.persister, post, CycleConfig{}).
		WithHistoryWriter(e.writer)
	res, err := orch.Run(context.Background(), mkPlayers("alice"))
	return res.PerPlayer["alice"], err
}

// persistOnlyFirst : la persistance du cycle range les `n` premiers matchs du lot (par identifiant)
// puis son attente expire : les autres sont soumis mais encore dans la file.
func (e *histEnv) persistOnlyFirst(n int) {
	e.persister.hookBeforeReturn = func(_ context.Context, b CycleBatch) {
		ids := make([]string, 0, len(b.Matches))
		for id := range b.Matches {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		e.mu.Lock()
		defer e.mu.Unlock()
		for i, id := range ids {
			if i < n {
				e.known[id] = true
			}
		}
		e.persister.err = &DrainTimeoutError{Waited: time.Minute, Submitted: ids}
	}
}

// Attente de la file expirée avec 120 matchs rangés sur 200 : le cycle réussit, le post-sync reçoit
// exactement les 120 rangés, le compte est juste, les 80 en file ne sont pas des échecs et retiennent
// le marque-page ; la passe suivante (file finie) reprend sans rien perdre.
func TestCycleHistory_AttenteExpiree_CycleReussiPourLesMatchsRanges(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 3*budget)
	const persisted = 120
	e.persistOnlyFirst(persisted)
	post := &recordingPostSync{}
	title := ctxkeys.TitleSlug(context.Background())
	before := observability.LoadCounterT(title, counterPersistDrainTimeout)

	out, err := e.cycleRecording(post)
	if err != nil {
		t.Fatalf("cycle : %v (une attente expirée ne fait pas échouer le cycle)", err)
	}
	if out.Status != "ok" || !out.HistoryPending {
		t.Errorf("issue %+v, want ok avec l'historique en attente", out)
	}
	if out.MatchesInserted != persisted {
		t.Errorf("MatchesInserted = %d, want %d (les seuls matchs prouvés rangés)", out.MatchesInserted, persisted)
	}
	if len(out.Warnings) != 1 {
		t.Errorf("Warnings = %v, want un avertissement sur les matchs encore en file", out.Warnings)
	}
	if post.calls != 1 || len(post.ids) != persisted {
		t.Errorf("post-sync : %d appels, %d matchs, want 1 appel avec les %d matchs rangés", post.calls, len(post.ids), persisted)
	}
	for _, id := range post.ids {
		if !e.known[id] {
			t.Errorf("le post-sync a reçu %s, qui n'est pas rangé", id)
		}
	}
	if got := e.failures(); len(got) != 0 {
		t.Errorf("échecs rangés %v, want aucun (un match en file n'est pas un échec)", got)
	}
	if _, ok := e.cursor(); ok || e.complete() {
		t.Errorf("marque-page rangé %v / fin %v alors que des matchs sont encore en file", ok, e.complete())
	}
	if got := observability.LoadCounterT(title, counterPersistDrainTimeout) - before; got != 1 {
		t.Errorf("compteur d'attente expirée +%d, want +1", got)
	}

	// La file finit en arrière-plan : tous les matchs du lot sont rangés. La passe suivante repart
	// sans rien redemander et range son marque-page (la persistance habituelle est rétablie).
	e.mu.Lock()
	for _, id := range e.ids[:budget] {
		e.known[id] = true
	}
	e.mu.Unlock()
	e.persister.hookBeforeReturn = func(_ context.Context, b CycleBatch) {
		e.mu.Lock()
		defer e.mu.Unlock()
		for id := range b.Matches {
			e.known[id] = true
		}
		e.persister.err = nil
	}
	if _, err := e.cycle(); err != nil {
		t.Fatal(err)
	}
	if c, ok := e.cursor(); !ok || c.Offset != 2*budget || c.LastMatchID != e.ids[2*budget-1] {
		t.Errorf("passe suivante : marque-page %+v (%v), want offset %d", c, ok, 2*budget)
	}
}

// Attente expirée à chaque passe, rien de rangé (file bloquée au-delà du plafond) : les matchs en file
// ne coûtent aucune tentative, donc aucun n'est abandonné même au-delà de maxCatchUpAttempts passes ;
// le compte rangé est zéro, le post-sync ne reçoit rien.
func TestCycleHistory_AttenteExpiree_AucunFauxEchecD16(t *testing.T) {
	e := newHistEnv(t, 450)
	e.persistOnlyFirst(0)
	title := ctxkeys.TitleSlug(context.Background())
	abandonedBefore := observability.LoadCounterT(title, counterHistoryAbandoned)

	for pass := 1; pass <= maxCatchUpAttempts+2; pass++ {
		post := &recordingPostSync{}
		out, err := e.cycleRecording(post)
		if err != nil {
			t.Fatalf("passe %d : %v", pass, err)
		}
		if out.MatchesInserted != 0 || len(post.ids) != 0 || !out.HistoryPending {
			t.Fatalf("passe %d : %+v, post-sync %v, want zéro rangé, rattrapage en cours", pass, out, post.ids)
		}
		if got := e.failures(); len(got) != 0 {
			t.Fatalf("passe %d : échecs rangés %v, want aucun", pass, got)
		}
	}
	if got := observability.LoadCounterT(title, counterHistoryAbandoned) - abandonedBefore; got != 0 {
		t.Errorf("%d matchs abandonnés, want 0", got)
	}
	if _, ok := e.cursor(); ok || e.complete() {
		t.Errorf("marque-page %v / fin %v rangés sans aucun match prouvé", ok, e.complete())
	}
}

// Un vrai échec de récupération (500) pendant une passe à l'attente expirée compte une tentative ;
// les matchs en file, non.
func TestCycleHistory_AttenteExpiree_VraiEchecCompteLesMatchsEnFileNon(t *testing.T) {
	e := newHistEnv(t, 450)
	e.persistOnlyFirst(50)
	e.fetcher.perMatchErr[e.ids[7]] = serverErr()

	if _, err := e.cycleRecording(&recordingPostSync{}); err != nil {
		t.Fatal(err)
	}
	got := e.failures()
	if len(got) != 1 || got[0].MatchID != e.ids[7] || got[0].Attempts != 1 {
		t.Errorf("échecs rangés %v, want seulement %s à 1 tentative", got, e.ids[7])
	}
}

// Attente expirée mais tout le lot déjà rangé à la relecture : rien n'est en file, la passe se règle
// comme une persistance réussie (marque-page rangé).
func TestCycleHistory_AttenteExpiree_ToutRange_ReglageNormal(t *testing.T) {
	budget := historystate.PassBudget()
	e := newHistEnv(t, 450)
	e.persistOnlyFirst(budget)
	post := &recordingPostSync{}
	out, err := e.cycleRecording(post)
	if err != nil {
		t.Fatal(err)
	}
	if out.MatchesInserted != budget || len(post.ids) != budget || len(out.Warnings) != 0 {
		t.Errorf("issue %+v, post-sync %d, want %d rangés sans avertissement", out, len(post.ids), budget)
	}
	if c, ok := e.cursor(); !ok || c.Offset != budget {
		t.Errorf("marque-page %+v (%v), want offset %d", c, ok, budget)
	}
}
