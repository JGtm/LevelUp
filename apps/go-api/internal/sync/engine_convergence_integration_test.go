//go:build integration

// Package sync — engine_convergence_integration_test.go : le moteur V1 (chemin du suivi en direct, de
// la synchronisation manuelle et de la synchronisation initiale : SyncEngine.RunDelta) est un delta
// pur sur un faux historique de 1 000 matchs et de vraies bases DuckDB temporaires (base partagée sous
// le fournisseur B-swap, base joueur migrée).
//
// Contrat vérifié (D-18 du plan 8.2.2) : le moteur V1 ne rattrape pas l'historique. Il ne lit ni
// n'écrit les clés de l'état de l'historique (historystate), s'arrête après domain.DeltaKnownRun matchs
// connus d'affilée (D-23), ne remplit pas HistoryPending, et sans match connu (première synchronisation) récupère jusqu'au
// plafond demandé. Le rattrapage est fait par le cycle de l'ordonnanceur : sa convergence est
// testée côté V2 (cmd/server) et Halo 5 (livesync).
package sync

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/historystate"
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

// historyStateKeys rend les clés de l'état de l'historique rangées dans la base joueur du moteur
// (clé de fin, marque-page, compte d'échecs : toute ligne `history%` de sync_meta) et dit si la fin
// est vue selon historystate.Complete.
func historyStateKeys(t *testing.T, env *e2eTestEnv) (keys []string, complete bool) {
	t.Helper()
	handle, err := OpenPlayerDB(env.playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer handle.Close()
	rows, err := handle.SQLDb().QueryContext(t.Context(), `SELECT key FROM sync_meta WHERE key LIKE 'history%' ORDER BY key`)
	if err != nil {
		t.Fatalf("lecture sync_meta: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			t.Fatalf("scan sync_meta: %v", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("itération sync_meta: %v", err)
	}
	return keys, historystate.Complete(t.Context(), handle.SQLDb())
}

// requireHistoryUntouched échoue si le moteur a touché à l'état de l'historique du joueur.
func requireHistoryUntouched(t *testing.T, env *e2eTestEnv, when string) {
	t.Helper()
	keys, complete := historyStateKeys(t, env)
	if len(keys) != 0 || complete {
		t.Fatalf("%s : le moteur V1 ne doit jamais toucher à l'état de l'historique (clés=%v, fin vue=%v)", when, keys, complete)
	}
}

// Un historique incomplet dont un match échoue de façon durable (statut 500 à chaque lecture) : le
// moteur V1 reste un delta. Il s'arrête après 25 matchs connus d'affilée (deux pages lues par passe :
// le match en échec, retenté à chaque passe, remet la suite à zéro dans la première page), ne
// rattrape ni le match en échec ni rien derrière la zone connue, ne remplit pas HistoryPending et ne touche
// jamais aux clés de l'état de l'historique. C'est le cycle de l'ordonnanceur qui rattrape (et
// abandonne un match durable, D-16).
func TestConvergence_V1_DeltaPur_HistoriqueIncompletMatchEnEchec_NeTouchePasAuxClesEtSArreteApresUneSuiteDeConnus(t *testing.T) {
	env, client, w := newConvergenceV1(t)
	ctx := t.Context()
	failing := w.ids[10]
	client.failStats[failing] = fmt.Errorf("GetMatchStats(%s): HTTP 500", failing)

	// Première passe : un historique vide côté joueur, le delta prend ce que le plafond permet.
	res, err := env.engine.RunDelta(ctx, convergenceV1Opts())
	if err != nil {
		t.Fatalf("passe 1 : %v", err)
	}
	if res.MatchesInserted == 0 {
		t.Fatalf("passe 1 : aucun match inséré, résultat %+v", res)
	}
	if registryHas(t, env, failing) {
		t.Fatal("le match en échec ne peut pas être au registre : le scénario est faux")
	}
	requireHistoryUntouched(t, env, "après la passe 1")
	known := registryCount(t, env)
	if known >= convergenceV1Total {
		t.Fatalf("le scénario suppose un historique resté incomplet, registre à %d", known)
	}

	// Passes suivantes, avec des matchs neufs en haut de la liste : deux pages lues, aucun rattrapage.
	for pass := 2; pass <= 6; pass++ {
		var fresh []string
		if pass%2 == 0 {
			fresh = w.addOnTop(client, 2)
		}
		callsBefore := client.callsGetHistory.Load()
		registryBefore := registryCount(t, env)

		res, err := env.engine.RunDelta(ctx, convergenceV1Opts())
		if err != nil {
			t.Fatalf("passe %d : %v", pass, err)
		}
		if calls := client.callsGetHistory.Load() - callsBefore; calls != 2 {
			t.Errorf("passe %d : %d lectures de pages d'historique, attendu 2 (suite de 25 connus complétée sur la page 2 ; avant : 1)", pass, calls)
		}
		if res.MatchesInserted != len(fresh) {
			t.Errorf("passe %d : %d matchs insérés, attendu %d (les matchs neufs du dessus, rien d'ancien)", pass, res.MatchesInserted, len(fresh))
		}
		if got := registryCount(t, env); got != registryBefore+len(fresh) {
			t.Errorf("passe %d : registre à %d matchs, attendu %d", pass, got, registryBefore+len(fresh))
		}
		for _, id := range fresh {
			if !registryHas(t, env, id) {
				t.Errorf("passe %d : le match neuf du dessus %s n'a pas été pris", pass, id)
			}
		}
		if registryHas(t, env, failing) {
			t.Errorf("passe %d : le delta ne doit pas rattraper le match en échec", pass)
		}
		if res.HistoryPending {
			t.Errorf("passe %d : le moteur V1 ne remplit pas HistoryPending", pass)
		}
		requireHistoryUntouched(t, env, fmt.Sprintf("après la passe %d", pass))
	}
}

// Première synchronisation d'un joueur neuf (aucun match connu) : le delta récupère jusqu'au plafond
// demandé, comme le faisait le mode full tant que rien n'est connu (synchronisation initiale :
// plafond de 200 par défaut ou initial_max_matches du profil). Et l'état de l'historique reste intact.
func TestConvergence_V1_PremiereSynchronisation_RecupereJusquAuPlafond(t *testing.T) {
	for _, plafond := range []int{200, 50} {
		t.Run(fmt.Sprintf("plafond_%d", plafond), func(t *testing.T) {
			env, client, _ := newConvergenceV1(t)
			opts := convergenceV1Opts()
			opts.MaxMatches = plafond

			res, err := env.engine.RunDelta(t.Context(), opts)
			if err != nil {
				t.Fatalf("RunDelta: %v", err)
			}
			if res.MatchesInserted != plafond || len(res.Errors) != 0 {
				t.Fatalf("%d matchs insérés (erreurs %v), attendu %d", res.MatchesInserted, res.Errors, plafond)
			}
			if got := registryCount(t, env); got != plafond {
				t.Errorf("registre à %d matchs, attendu %d", got, plafond)
			}
			if calls, want := client.callsGetHistory.Load(), int64(plafond/convergenceV1Budget); calls != want {
				t.Errorf("%d lectures de pages d'historique, attendu %d (une par page de 25)", calls, want)
			}
			if res.HistoryPending {
				t.Error("le moteur V1 ne remplit pas HistoryPending")
			}
			requireHistoryUntouched(t, env, "après la première synchronisation")
		})
	}
}

// Un historique que l'API épuise (page vide) : ni le delta ni RunFull ne rangent la fin d'historique,
// seul le cycle de l'ordonnanceur le fait. Un RunFull garde son comportement de parcours complet.
func TestConvergence_V1_FinDHistoriqueVue_NeRangeJamaisLaFin(t *testing.T) {
	run := map[string]func(*SyncEngine, context.Context, domain.SyncOptions) (domain.SyncResult, error){
		"RunDelta": (*SyncEngine).RunDelta,
		"RunFull":  (*SyncEngine).RunFull,
	}
	for name, fn := range run {
		t.Run(name, func(t *testing.T) {
			env := newE2EEnv(t)
			env.seedMockMatches("fin-vue-"+name, 3)
			opts := convergenceV1Opts()
			opts.MaxMatches = 50

			res, err := fn(env.engine, t.Context(), opts)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if res.MatchesInserted != 3 || res.HistoryPending {
				t.Fatalf("%s : inséré=%d pending=%v, attendu 3 / faux", name, res.MatchesInserted, res.HistoryPending)
			}
			requireHistoryUntouched(t, env, "après "+name)
		})
	}
}

// Le joueur joue A seul, puis B avec un coéquipier ; le coéquipier range B en premier : B est connu du
// joueur, A non. La liste est [B, A, ...] : le delta ne s'arrête plus sur B et prend A (D-23). Une
// passe sans nouveau match, ensuite, ne lit toujours qu'une page.
func TestConvergence_V1_Delta_ConnuRangeParUnCoequipier_LInconnuEnDessousEstPris(t *testing.T) {
	env, client, w := newConvergenceV1(t)
	ctx := t.Context()
	if _, err := env.engine.RunDelta(ctx, convergenceV1Opts()); err != nil {
		t.Fatalf("passe 1 : %v", err)
	}

	b := w.addOnTop(client, 1)[0]
	if _, err := env.engine.RunDelta(ctx, convergenceV1Opts()); err != nil {
		t.Fatalf("passe B : %v", err)
	}
	if !registryHas(t, env, b) {
		t.Fatal("B devrait être rangé : le scénario est faux")
	}

	// A, joué avant B, que la synchronisation n'a jamais vu : placé juste sous B.
	a := convergenceID(w.next)
	w.next++
	client.statsBody[a] = w.stats(a)
	w.ids = append([]string{w.ids[0], a}, w.ids[1:]...)
	client.history = makeHistory(w.ids...)

	if _, err := env.engine.RunDelta(ctx, convergenceV1Opts()); err != nil {
		t.Fatalf("passe A : %v", err)
	}
	if !registryHas(t, env, a) {
		t.Fatal("A, joué seul sous le match B déjà connu, n'a pas été pris par le delta")
	}
	requireHistoryUntouched(t, env, "après la passe A")

	callsBefore := client.callsGetHistory.Load()
	if _, err := env.engine.RunDelta(ctx, convergenceV1Opts()); err != nil {
		t.Fatalf("passe sans nouveau match : %v", err)
	}
	if calls := client.callsGetHistory.Load() - callsBefore; calls != 1 {
		t.Errorf("sans nouveau match : %d lectures de pages d'historique, attendu 1 (comme avant)", calls)
	}
}
