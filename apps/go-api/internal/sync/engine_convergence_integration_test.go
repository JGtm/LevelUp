//go:build integration

// Package sync — engine_convergence_integration_test.go : convergence de bout en bout du moteur V1
// (chemin du suivi en direct : SyncEngine.RunDelta avec le MaxMatches serré du watcher) sur un
// faux historique de 1 000 matchs et de vraies bases DuckDB temporaires (base partagée sous le
// fournisseur B-swap, base joueur migrée).
//
// Contrat vérifié, le même que pour V2 et Halo 5 : un joueur dont l'historique n'est pas complet
// rattrape par petits pas (jamais plus que le plafond par passe, nouveaux matchs du dessus pris),
// finit par ranger la fin d'historique, puis redevient un delta (une page, aucun rattrapage).
package sync

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
)

const (
	convergenceV1Total  = 1000
	convergenceV1Budget = 25 // MaxMatches du suivi en direct : une page d'historique par passe
)

// convergenceID rend l'identifiant (UUID valide) du n-ième match créé du faux monde.
func convergenceID(n int) string {
	return fmt.Sprintf("c0de%04x-0000-4000-8000-%012d", n, n)
}

// convergenceWorld est le faux historique d'un joueur : la liste de l'API (plus récent d'abord) et
// les stats de chaque match, dont le joueur suivi est participant (condition pour que le match soit
// « connu » de la règle knownset une fois rangé).
type convergenceWorld struct {
	xuid string
	next int
	ids  []string // plus récent d'abord
}

// addOnTop crée n matchs plus récents que tous les autres et les place en haut de la liste.
func (w *convergenceWorld) addOnTop(c *convergenceClient, n int) []string {
	fresh := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := convergenceID(w.next)
		w.next++
		fresh = append([]string{id}, fresh...) // le dernier créé est le plus récent
		c.statsBody[id] = w.stats(id)
	}
	w.ids = append(append([]string{}, fresh...), w.ids...)
	c.history = makeHistory(w.ids...)
	return fresh
}

// stats rend le corps de stats d'un match : makeMatchJSON, dont le premier joueur est le joueur suivi.
func (w *convergenceWorld) stats(id string) map[string]any {
	body := makeMatchJSON(id, 2)
	players, _ := body["Players"].([]any)
	if first, ok := players[0].(map[string]any); ok {
		first["PlayerId"] = "xuid(" + w.xuid + ")"
	}
	return body
}

// convergenceClient est le faux client Halo : l'historique et les stats du mockHaloClient du paquet,
// plus des pannes par match (statut durable d'un match en échec).
type convergenceClient struct {
	*mockHaloClient
	failStats map[string]error
}

func (c *convergenceClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	if err, ok := c.failStats[matchID]; ok {
		return nil, err
	}
	return c.mockHaloClient.GetMatchStats(ctx, matchID)
}

// registryCount compte les matchs du registre partagé (vérité terrain : base DuckDB réelle).
func registryCount(t *testing.T, env *e2eTestEnv) int {
	t.Helper()
	db, release, err := env.provider.Get(context.Background())
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil {
		t.Fatalf("COUNT match_registry: %v", err)
	}
	return n
}

// registryHas dit si le match est au registre partagé.
func registryHas(t *testing.T, env *e2eTestEnv, id string) bool {
	t.Helper()
	db, release, err := env.provider.Get(context.Background())
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("lecture match_registry: %v", err)
	}
	return n > 0
}

func newConvergenceV1(t *testing.T) (*e2eTestEnv, *convergenceClient, *convergenceWorld) {
	t.Helper()
	env := newE2EEnv(t)
	client := &convergenceClient{
		mockHaloClient: &mockHaloClient{statsBody: map[string]map[string]any{}},
		failStats:      map[string]error{},
	}
	env.engine.SetCustomClient(client)
	w := &convergenceWorld{xuid: env.xuid}
	w.addOnTop(client, convergenceV1Total)
	return env, client, w
}

func convergenceV1Opts() domain.SyncOptions {
	return domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        convergenceV1Budget,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}
}

// Un historique de 1 000 matchs, un plafond de 25 par passe (suivi en direct), des matchs neufs qui
// arrivent en haut de la liste entre deux passes : le moteur V1 rattrape par petits pas, prend les
// nouveaux du dessus, range la fin d'historique, puis redevient un delta d'une page.
func TestConvergence_V1_SuiviEnDirect_RattrapeParPetitsPasRangeLaFinPuisDelta(t *testing.T) {
	env, client, w := newConvergenceV1(t)
	ctx := t.Context()

	// Passe n° -> nombre de matchs neufs qui apparaissent en haut AVANT cette passe.
	arrivals := map[int]int{3: 2, 12: 3, 30: 1}
	var (
		endPass       int
		totalCalls    int
		callsPerPass  []int64
		insertedTotal int
	)
	for pass := 1; pass <= 60 && endPass == 0; pass++ {
		var fresh []string
		if n := arrivals[pass]; n > 0 {
			fresh = w.addOnTop(client, n)
		}
		before := registryCount(t, env)
		missing := len(w.ids) - before
		callsBefore := client.callsGetHistory.Load()

		res, err := env.engine.RunDelta(ctx, convergenceV1Opts())
		if err != nil {
			t.Fatalf("passe %d : RunDelta: %v", pass, err)
		}
		calls := client.callsGetHistory.Load() - callsBefore
		callsPerPass = append(callsPerPass, calls)
		totalCalls += int(calls)
		insertedTotal += res.MatchesInserted

		// Plafond par passe respecté, et rien de moins que ce qui reste à prendre (petits pas pleins).
		want := min(convergenceV1Budget, missing)
		if res.MatchesInserted != want {
			t.Fatalf("passe %d : %d matchs insérés, attendu %d (reste %d)", pass, res.MatchesInserted, want, missing)
		}
		if got := registryCount(t, env); got != before+want {
			t.Fatalf("passe %d : registre à %d matchs, attendu %d", pass, got, before+want)
		}
		// Les matchs neufs du dessus sont pris dès cette passe (ils précèdent les inconnus plus anciens).
		for _, id := range fresh {
			if !registryHas(t, env, id) {
				t.Fatalf("passe %d : le match neuf du dessus %s n'a pas été pris", pass, id)
			}
		}
		// Bornes de lecture : le moteur V1 reparcourt la liste depuis le haut en sautant les connus
		// (pas de marque-page) : au plus une page par 25 matchs connus, plus la page de travail et la page
		// qui constate la fin.
		if max := int64(before/convergenceV1Budget + 3); calls > max {
			t.Fatalf("passe %d : %d lectures de pages d'historique, borne %d", pass, calls, max)
		}

		ranged := historyKeyRanged(t, env)
		if ranged == res.HistoryPending {
			t.Fatalf("passe %d : clé de fin rangée=%v mais HistoryPending=%v (doivent être opposés)", pass, ranged, res.HistoryPending)
		}
		if ranged {
			endPass = pass
			if !res.HistoryExhausted {
				t.Errorf("passe %d : la fin est rangée sans que la passe ait vu la page vide", pass)
			}
			continue
		}
		if !res.HistoryPending {
			t.Fatalf("passe %d : fin non rangée mais la passe ne se déclare pas en attente", pass)
		}
		if got := registryCount(t, env); got >= len(w.ids) && pass > len(w.ids)/convergenceV1Budget+3 {
			t.Fatalf("passe %d : tout est au registre mais la fin n'est toujours pas rangée", pass)
		}
	}
	if endPass == 0 {
		t.Fatalf("la fin d'historique n'a pas été rangée en 60 passes (registre %d / %d)", registryCount(t, env), len(w.ids))
	}
	if got, want := registryCount(t, env), len(w.ids); got != want {
		t.Fatalf("à la fin, registre à %d matchs, attendu %d", got, want)
	}
	t.Logf("V1 : %d passes jusqu'à la fin rangée, %d matchs insérés, %d lectures de pages au total ; par passe : %v",
		endPass, insertedTotal, totalCalls, callsPerPass)

	// Cycle suivant : un delta, une seule page, aucun rattrapage, rien d'en attente.
	callsBefore := client.callsGetHistory.Load()
	res, err := env.engine.RunDelta(ctx, convergenceV1Opts())
	if err != nil {
		t.Fatalf("passe après la fin : %v", err)
	}
	if calls := client.callsGetHistory.Load() - callsBefore; calls != 1 {
		t.Errorf("après la fin : %d lectures de pages, attendu 1 (delta)", calls)
	}
	if res.MatchesInserted != 0 || res.HistoryPending || res.HistoryExhausted {
		t.Errorf("après la fin : inséré=%d pending=%v exhausted=%v, attendu 0 / faux / faux",
			res.MatchesInserted, res.HistoryPending, res.HistoryExhausted)
	}

	// Un match neuf après la fin : pris par le delta, toujours une page.
	fresh := w.addOnTop(client, 1)
	callsBefore = client.callsGetHistory.Load()
	res, err = env.engine.RunDelta(ctx, convergenceV1Opts())
	if err != nil {
		t.Fatalf("delta après la fin : %v", err)
	}
	if calls := client.callsGetHistory.Load() - callsBefore; calls != 1 {
		t.Errorf("delta d'un match neuf : %d lectures de pages, attendu 1", calls)
	}
	if res.MatchesInserted != 1 || !registryHas(t, env, fresh[0]) {
		t.Errorf("delta d'un match neuf : %d insérés, présent=%v", res.MatchesInserted, registryHas(t, env, fresh[0]))
	}
	if !historyKeyRanged(t, env) {
		t.Error("la clé de fin d'historique ne doit pas disparaître après un delta")
	}
}
