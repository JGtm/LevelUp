//go:build integration

// Package sync — known_set_v1_integration_test.go : la règle « connu » (internal/sync/knownset)
// vue depuis le moteur V1 — pagination delta et full réelles, persistance réelle (DuckDB en
// mémoire, chemin batch INSERT-only), client Halo simulé.
//
//   - un enrichissement joueur dont le match manque au registre partagé est re-récupéré, en
//     delta comme en full, puis devient connu (convergence) ;
//   - un orphelin plus ancien que le premier connu est récupéré par son match_id (passe dédiée
//     de syncHistory, alimentée par run()) ;
//   - en régime normal (chaque enrichissement a son match au registre), seuls les nouveaux
//     matchs sont récupérés ;
//   - base partagée illisible : RunDelta s'arrête avec l'erreur typée, sans appel API ni écriture.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	stdsync "sync"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync/knownset"
)

const (
	ksKnown1  = "aabbccdd-0000-4000-8000-0000000000a1"
	ksKnown2  = "aabbccdd-0000-4000-8000-0000000000a2"
	ksOrphan  = "aabbccdd-0000-4000-8000-0000000000b1"
	ksNew     = "aabbccdd-0000-4000-8000-0000000000c1"
	ksXUID    = "0000000000000000" // Player0 de makeMatchJSON
	ksGamertg = "Player0"
)

// pagedHistoryClient : mockHaloClient dont l'historique respecte start/count (la pagination
// full du moteur avance par pages) et qui enregistre les match_id récupérés.
type pagedHistoryClient struct {
	*mockHaloClient
	full    []MatchHistoryEntry
	mu      stdsync.Mutex
	fetched []string
}

func newPagedHistoryClient(history ...string) *pagedHistoryClient {
	body := map[string]map[string]any{}
	for _, id := range []string{ksKnown1, ksKnown2, ksOrphan, ksNew} {
		body[id] = makeMatchJSON(id, 2)
	}
	return &pagedHistoryClient{
		mockHaloClient: &mockHaloClient{statsBody: body},
		full:           makeHistory(history...),
	}
}

func (c *pagedHistoryClient) GetMatchHistory(_ context.Context, _, _ string, start, count int) ([]MatchHistoryEntry, error) {
	c.callsGetHistory.Add(1)
	if start >= len(c.full) {
		return nil, nil
	}
	return c.full[start:min(start+count, len(c.full))], nil
}

func (c *pagedHistoryClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	c.mu.Lock()
	c.fetched = append(c.fetched, matchID)
	c.mu.Unlock()
	return c.mockHaloClient.GetMatchStats(ctx, matchID)
}

func (c *pagedHistoryClient) fetchedSorted() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Sorted(slices.Values(c.fetched))
}

// knownSetV1Env : bases en mémoire au schéma de persistance, moteur du joueur Player0.
type knownSetV1Env struct {
	e        *SyncEngine
	playerDB *sql.DB
	sharedDB *sql.DB
	opts     domain.SyncOptions
}

func newKnownSetV1Env(t *testing.T) *knownSetV1Env {
	t.Helper()
	playerDB, sharedDB := newInMemoryDBs(t)
	patchSharedSchemaForBatch(t, sharedDB)
	return &knownSetV1Env{
		e:        &SyncEngine{gamertag: ksGamertg, xuid: ksXUID, titleSlug: "halo_infinite"},
		playerDB: playerDB,
		sharedDB: sharedDB,
		opts: domain.SyncOptions{
			MatchType: "matchmaking", MaxMatches: 50, WithParticipants: true, WithMedals: true,
		},
	}
}

// seedSynced persiste des matchs comme une sync complète : registre, participants, enrichissement.
func (env *knownSetV1Env) seedSynced(t *testing.T, ids ...string) {
	t.Helper()
	seed := newPagedHistoryClient()
	for _, id := range ids {
		fm, err := env.e.fetchMatchData(t.Context(), seed, id, env.opts)
		if err != nil || fm == nil {
			t.Fatalf("fetchMatchData(%s) = %v, %v", id, fm, err)
		}
		if err := env.e.persistFetchedMatch(t.Context(), env.sharedDB, env.playerDB, &domain.SyncResult{}, fm); err != nil {
			t.Fatalf("persistFetchedMatch(%s): %v", id, err)
		}
	}
}

// seedOrphan : enrichissement joueur seul — la base partagée (restaurée plus ancienne) n'a
// pas le match.
func (env *knownSetV1Env) seedOrphan(t *testing.T, id string) {
	t.Helper()
	if err := UpsertPlayerEnrichment(t.Context(), env.playerDB, id, ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment(%s): %v", id, err)
	}
}

// paginate charge l'ensemble connu par la règle unique puis déroule l'historique du moteur
// comme run() : pagination, puis orphelins par match_id (syncHistory).
func (env *knownSetV1Env) paginate(t *testing.T, client HaloClient, isDelta bool) {
	t.Helper()
	set, err := knownset.Load(t.Context(), env.playerDB, env.sharedDB, env.e.xuid)
	if err != nil {
		t.Fatalf("knownset.Load: %v", err)
	}
	result := domain.SyncResult{}
	env.e.syncHistory(t.Context(), historyPaginationInputs{
		client: client, opts: env.opts, known: set.Known,
		sharedDB: env.sharedDB, playerDB: env.playerDB, isDelta: isDelta,
		orphans: set.Recover,
	}, &result)
	if len(result.Errors) > 0 {
		t.Fatalf("pagination en erreur : %v", result.Errors)
	}
}

func (env *knownSetV1Env) inRegistry(t *testing.T, id string) bool {
	t.Helper()
	var n int
	if err := env.sharedDB.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("lecture registre: %v", err)
	}
	return n == 1
}

func assertFetched(t *testing.T, c *pagedHistoryClient, want ...string) {
	t.Helper()
	slices.Sort(want)
	if got := c.fetchedSorted(); !slices.Equal(got, want) {
		t.Errorf("matchs récupérés = %v, attendu %v", got, want)
	}
}

// TestKnownSetV1_DeltaReRecupereLEnrichissementSansRegistre : historique [nouveau, orphelin,
// connu, connu]. Le delta ne s'arrête pas sur l'orphelin : il le récupère avec le nouveau,
// s'arrête sur le premier vrai connu ; l'orphelin rejoint le registre et le delta suivant ne
// récupère plus rien.
func TestKnownSetV1_DeltaReRecupereLEnrichissementSansRegistre(t *testing.T) {
	env := newKnownSetV1Env(t)
	env.seedSynced(t, ksKnown1, ksKnown2)
	env.seedOrphan(t, ksOrphan)

	client := newPagedHistoryClient(ksNew, ksOrphan, ksKnown1, ksKnown2)
	env.paginate(t, client, true)
	assertFetched(t, client, ksNew, ksOrphan)
	if !env.inRegistry(t, ksOrphan) {
		t.Fatal("l'orphelin n'a pas rejoint match_registry après le delta")
	}

	again := newPagedHistoryClient(ksNew, ksOrphan, ksKnown1, ksKnown2)
	env.paginate(t, again, true)
	assertFetched(t, again)
}

// TestKnownSetV1_FullReRecupereLEnrichissementSansRegistre : en full, un connu est sauté sans
// arrêt ; l'orphelin placé entre deux connus est récupéré, et seulement lui.
func TestKnownSetV1_FullReRecupereLEnrichissementSansRegistre(t *testing.T) {
	env := newKnownSetV1Env(t)
	env.seedSynced(t, ksKnown1, ksKnown2)
	env.seedOrphan(t, ksOrphan)

	client := newPagedHistoryClient(ksKnown1, ksOrphan, ksKnown2)
	env.paginate(t, client, false)
	assertFetched(t, client, ksOrphan)
	if !env.inRegistry(t, ksOrphan) {
		t.Fatal("l'orphelin n'a pas rejoint match_registry après le full")
	}
}

// TestKnownSetV1_RegimeNormalInchange : chaque enrichissement a son match au registre — delta
// et full ne récupèrent que le nouveau match, exactement comme avec l'ancienne union.
func TestKnownSetV1_RegimeNormalInchange(t *testing.T) {
	for _, mode := range []struct {
		nom     string
		isDelta bool
	}{{"delta", true}, {"full", false}} {
		t.Run(mode.nom, func(t *testing.T) {
			env := newKnownSetV1Env(t)
			env.seedSynced(t, ksKnown1, ksKnown2)

			client := newPagedHistoryClient(ksNew, ksKnown1, ksKnown2)
			env.paginate(t, client, mode.isDelta)
			assertFetched(t, client, ksNew)
		})
	}
}

// TestKnownSetV1_BasePartageeIllisibleArreteLaSync : la base partagée servie au moteur n'a pas
// de tables (illisible pour la règle) — RunDelta rend knownset.ErrSharedUnreadable avant tout
// appel d'historique, toute récupération et toute écriture.
func TestKnownSetV1_BasePartageeIllisibleArreteLaSync(t *testing.T) {
	empty := openMemDB(t)
	engine := NewSyncEngine(t.TempDir(), ksGamertg, ksXUID, &domain.HaloTokens{SpartanToken: "t", ClearanceToken: "c"}, nil).
		WithSharedProvider(sharedprovider.FromInMemoryDB(empty, ":memory:"))
	client := newPagedHistoryClient(ksNew, ksKnown1)
	engine.SetCustomClient(client)

	_, err := engine.RunDelta(context.Background(), domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 5, WithParticipants: true, WithMedals: true, RequestsPerSecond: 100,
	})
	if !errors.Is(err, knownset.ErrSharedUnreadable) {
		t.Fatalf("RunDelta err = %v, attendu knownset.ErrSharedUnreadable", err)
	}
	if n := client.callsGetHistory.Load(); n != 0 {
		t.Errorf("GetMatchHistory appelé %d fois, attendu 0", n)
	}
	assertFetched(t, client)

	var tables int
	if err := empty.QueryRow(`SELECT COUNT(*) FROM information_schema.tables`).Scan(&tables); err != nil {
		t.Fatalf("lecture catalogue partagé: %v", err)
	}
	if tables != 0 {
		t.Errorf("base partagée : %d tables créées, attendu 0 (aucune écriture)", tables)
	}
	playerDB, release, err := duckdbpkg.OpenReadForQuery(engine.playerDBPath)
	if err != nil {
		t.Fatalf("ouverture base joueur: %v", err)
	}
	defer release()
	var enrich int
	if err := playerDB.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&enrich); err != nil {
		t.Fatalf("lecture enrichissements: %v", err)
	}
	if enrich != 0 {
		t.Errorf("enrichissements écrits = %d, attendu 0", enrich)
	}
}

// TestKnownSetV1_OrphelinPlusAncienQueLeConnuRecupereParIdentifiant : historique [nouveau,
// connu, connu, orphelin] — l'orphelin est PLUS ANCIEN que le premier connu, le delta s'arrête
// avant lui. Il est récupéré quand même, par son match_id, dans le même run (sans sync
// complète) ; il rejoint le registre et le delta suivant ne récupère plus rien.
func TestKnownSetV1_OrphelinPlusAncienQueLeConnuRecupereParIdentifiant(t *testing.T) {
	env := newKnownSetV1Env(t)
	env.seedSynced(t, ksKnown1, ksKnown2)
	env.seedOrphan(t, ksOrphan)

	client := newPagedHistoryClient(ksNew, ksKnown1, ksKnown2, ksOrphan)
	env.paginate(t, client, true)
	assertFetched(t, client, ksNew, ksOrphan)
	if !env.inRegistry(t, ksOrphan) {
		t.Fatal("l'orphelin plus ancien que le connu n'a pas rejoint match_registry")
	}

	again := newPagedHistoryClient(ksNew, ksKnown1, ksKnown2, ksOrphan)
	env.paginate(t, again, true)
	assertFetched(t, again)
}

// TestKnownSetV1_RecuperationParIdentifiantBornee : knownset.OrphanRecoveryPerCycle+3
// orphelins absents de l'historique. Un run en récupère exactement OrphanRecoveryPerCycle (les
// premiers en ordre lexicographique), le suivant les 3 restants, le troisième aucun.
func TestKnownSetV1_RecuperationParIdentifiantBornee(t *testing.T) {
	env := newKnownSetV1Env(t)
	env.seedSynced(t, ksKnown1)
	const n = knownset.OrphanRecoveryPerCycle + 3
	orphelins := make([]string, n)
	for i := range n {
		orphelins[i] = fmt.Sprintf("aabbccdd-0000-4000-8000-%012d", i)
		env.seedOrphan(t, orphelins[i])
	}
	nouveauClient := func() *pagedHistoryClient {
		c := newPagedHistoryClient(ksKnown1)
		for _, id := range orphelins {
			c.statsBody[id] = makeMatchJSON(id, 2)
		}
		return c
	}

	premier := nouveauClient()
	env.paginate(t, premier, true)
	assertFetched(t, premier, orphelins[:knownset.OrphanRecoveryPerCycle]...)

	second := nouveauClient()
	env.paginate(t, second, true)
	assertFetched(t, second, orphelins[knownset.OrphanRecoveryPerCycle:]...)

	troisieme := nouveauClient()
	env.paginate(t, troisieme, true)
	assertFetched(t, troisieme)
	for _, id := range orphelins {
		if !env.inRegistry(t, id) {
			t.Fatalf("orphelin %s absent du registre après trois runs", id)
		}
	}
}

// TestKnownSetV1_RunDelta_EnrichiAuRegistreSansParticipantResteConnu : par le point d'entrée
// réel (RunDelta → run), base partagée servie par un provider réel. Le match R est au registre
// (inséré par un autre joueur, sans ligne de participant pour le xuid de user0) et enrichi dans
// la base de user0. R est CONNU — seul l'enrichissement le borne au joueur — : le delta s'arrête
// sur lui sans le récupérer. Si la base joueur n'atteignait pas knownset.Load, R serait
// inconnu et re-récupéré à chaque cycle (le fetch serait jeté : registre déjà écrit).
func TestKnownSetV1_RunDelta_EnrichiAuRegistreSansParticipantResteConnu(t *testing.T) {
	env := newMultiUserEnv(t, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opts := domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 5, WithParticipants: true, WithMedals: true, RequestsPerSecond: 100,
	}
	const matchR = "c0000000-0000-4000-8000-000000000003"
	body := map[string]map[string]any{matchR: makeMatchJSON(matchR, 2)} // participants Player0/Player1

	// user1 insère R : registre + participants (xuids de makeMatchJSON, ni user0 ni user1).
	env.users[1].mock.history = makeHistory(matchR)
	env.users[1].mock.statsBody = body
	if _, err := env.users[1].engine.RunDelta(ctx, opts); err != nil {
		t.Fatalf("RunDelta user1: %v", err)
	}

	// user0 : R enrichi dans sa base joueur.
	u := env.users[0]
	ph, err := OpenPlayerDB(u.engine.playerDBPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB user0: %v", err)
	}
	if err := UpsertPlayerEnrichment(ctx, ph.SQLDb(), matchR, ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}
	_ = ph.Close()

	u.mock.history = makeHistory(matchR)
	u.mock.statsBody = body
	// Le post-sync peut interroger l'API pour ses propres convergences : on lit les compteurs
	// de la pagination (R sauté comme connu, rien inséré), pas le nombre d'appels du client.
	res, err := u.engine.RunDelta(ctx, opts)
	if err != nil {
		t.Fatalf("RunDelta user0: %v", err)
	}
	if res.MatchesSkipped != 1 || res.MatchesInserted != 0 {
		t.Errorf("sauté = %d, inséré = %d, attendu 1 et 0 (R connu : au registre et enrichi, arrêt delta sur lui)",
			res.MatchesSkipped, res.MatchesInserted)
	}
}

// TestKnownSetV1_RunDelta_OrphelinPlusAncienQueLeConnuRecupere : par le point d'entrée réel
// (RunDelta → run), provider réel. K est connu (au registre, enrichi) ; l'orphelin O (enrichi,
// absent du registre) est PLUS ANCIEN que K dans l'historique : le delta s'arrête sur K avant
// de le voir. run() transmet Set.Recover à syncHistory : O est récupéré par son match_id dans
// le même run et rejoint le registre.
func TestKnownSetV1_RunDelta_OrphelinPlusAncienQueLeConnuRecupere(t *testing.T) {
	env := newMultiUserEnv(t, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opts := domain.SyncOptions{
		MatchType: "matchmaking", MaxMatches: 5, WithParticipants: true, WithMedals: true, RequestsPerSecond: 100,
	}
	const (
		matchK = "c0000000-0000-4000-8000-0000000000a1"
		matchO = "c0000000-0000-4000-8000-0000000000b1"
	)
	u := env.users[0]
	u.mock.statsBody = map[string]map[string]any{
		matchK: makeMatchJSON(matchK, 2),
		matchO: makeMatchJSON(matchO, 2),
	}

	// K : synchronisé (registre, participants, enrichissement de user0).
	u.mock.history = makeHistory(matchK)
	if _, err := u.engine.RunDelta(ctx, opts); err != nil {
		t.Fatalf("RunDelta K: %v", err)
	}
	// O : enrichissement seul (base partagée restaurée plus ancienne).
	ph, err := OpenPlayerDB(u.engine.playerDBPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB user0: %v", err)
	}
	if err := UpsertPlayerEnrichment(ctx, ph.SQLDb(), matchO, ""); err != nil {
		t.Fatalf("UpsertPlayerEnrichment: %v", err)
	}
	_ = ph.Close()
	if inSharedRegistry(ctx, t, env, matchO) {
		t.Fatal("pré-condition : O déjà au registre")
	}

	u.mock.history = makeHistory(matchK, matchO)
	res, err := u.engine.RunDelta(ctx, opts)
	if err != nil {
		t.Fatalf("RunDelta K, O: %v", err)
	}
	if !inSharedRegistry(ctx, t, env, matchO) {
		t.Fatalf("orphelin plus ancien que le connu non récupéré par son match_id (inséré = %d, sauté = %d)",
			res.MatchesInserted, res.MatchesSkipped)
	}
	if res.MatchesInserted != 1 {
		t.Errorf("inséré = %d, attendu 1 (O seul : le delta s'arrête sur K)", res.MatchesInserted)
	}
}

// inSharedRegistry : le match a sa ligne match_registry dans la base partagée (lecture par le
// provider).
func inSharedRegistry(ctx context.Context, t *testing.T, env *multiUserEnv, id string) bool {
	t.Helper()
	db, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_registry WHERE match_id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("lecture registre: %v", err)
	}
	return n == 1
}
