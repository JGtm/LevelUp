//go:build integration

// sync_v2_convergence_env_integration_test.go : banc de la convergence de bout en bout du rattrapage
// de l'historique Halo Infinite par le CHEMIN DE PRODUCTION de l'ordonnanceur.
//
// Ce qui est de production : l'orchestrateur V2 construit par buildSyncV2Orchestrator (mêmes
// découverte, récupération, persistance, post-sync, fin d'historique), la file de persistance et son
// worker ouverts par openPersistQueue, les migrations du démarrage (runMigrations, RunPlayerMigrations),
// le fournisseur B-swap de la base partagée, le client Halo poolé (sync.PooledHaloClient) et le client
// HTTP de la brique halo-api, l'ordonnanceur (RunOnceTrigger, boucle Run, cadence).
//
// Ce qui est simulé : le réseau. Le client de production fait de vraies requêtes HTTP vers les hôtes
// Halo ; le test remplace http.DefaultTransport (que le client de la brique utilise, faute de transport
// propre) par un faux qui répond en mémoire, sans ouvrir une seule connexion. Le faux client Halo est
// servi par cet enregistreur partagé (haloapitest.Enregistreur) : historique paginé de N matchs, stats,
// skill, film absent (404).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JGtm/halo-api/haloapitest"
	"golang.org/x/time/rate"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/auth/pool"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/historyrepo"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	settingsplatform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/historystate"
	syncv2 "levelup/go-api/internal/sync/v2"
)

const (
	convergenceGamertag = "ConvergenceJoueur"
	convergenceXUID     = "2533274800000001"
	convergencePageSize = 25
)

// convergenceMatchID rend l'identifiant (UUID valide) du n-ième match créé du faux monde.
func convergenceMatchID(n int) string {
	return fmt.Sprintf("c0de%04x-0000-4000-8000-%012d", n, n)
}

// fakeHalo est le faux service Halo : l'historique du joueur (plus récent d'abord), les pannes par match,
// et les compteurs de requêtes. [fakeHalo.transport] en fait le transport HTTP du banc.
type fakeHalo struct {
	mu sync.Mutex

	ids  []string // la liste de l'API, plus récent d'abord
	next int      // prochain numéro de match à créer

	statsStatus map[string]int // match -> statut HTTP durable des stats (500, 404...)

	historyStarts []int          // décalage de chaque page d'historique demandée depuis le dernier reset
	statsCalls    map[string]int // match -> requêtes de stats (essais compris)
	unexpected    []string       // requêtes hors du faux (hôte ou chemin non simulé)
}

func newFakeHalo(total int) *fakeHalo {
	f := &fakeHalo{statsStatus: map[string]int{}, statsCalls: map[string]int{}}
	f.addOnTop(total)
	return f
}

// addOnTop crée n matchs plus récents que tous les autres, en haut de la liste ; rend leurs identifiants,
// le plus récent d'abord.
func (f *fakeHalo) addOnTop(n int) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	fresh := make([]string, 0, n)
	for i := 0; i < n; i++ {
		fresh = append([]string{convergenceMatchID(f.next)}, fresh...)
		f.next++
	}
	f.ids = append(append([]string{}, fresh...), f.ids...)
	return fresh
}

func (f *fakeHalo) listLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.ids)
}

func (f *fakeHalo) idAt(pos int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ids[pos]
}

func (f *fakeHalo) setStatsStatus(id string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statsStatus[id] = status
}

// resetCounters remet à zéro les compteurs de lecture d'historique et de stats.
func (f *fakeHalo) resetCounters() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.historyStarts = nil
	f.statsCalls = map[string]int{}
}

func (f *fakeHalo) historyReads() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.historyStarts...)
}

func (f *fakeHalo) statsCallsFor(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.statsCalls[id]
}

func (f *fakeHalo) unexpectedRequests() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.unexpected...)
}

// transport rend le http.RoundTripper du faux service : l'enregistreur partagé des tests de la brique,
// qui sert [fakeHalo.serve].
func (f *fakeHalo) transport() http.RoundTripper {
	return &haloapitest.Enregistreur{Repondre: func(_ int, req *http.Request) *http.Response { return f.serve(req) }}
}

// serve répond en mémoire aux requêtes du client Halo de production.
func (f *fakeHalo) serve(req *http.Request) *http.Response {
	path := req.URL.Path
	switch {
	case strings.Contains(path, "/players/") && strings.HasSuffix(path, "/matches"):
		return f.history(req)
	case strings.Contains(path, "/matches/") && strings.HasSuffix(path, "/stats"):
		return f.stats(req, between(path, "/matches/", "/stats"))
	case strings.Contains(path, "/matches/") && strings.HasSuffix(path, "/skill"):
		return f.skill(req, between(path, "/matches/", "/skill"))
	case strings.Contains(path, "/films/matches/"):
		return reply(req, http.StatusNotFound, "film absent") // « film absent » : cas normal d'un vieux match
	}
	f.mu.Lock()
	f.unexpected = append(f.unexpected, req.Method+" "+req.URL.Host+req.URL.RequestURI())
	f.mu.Unlock()
	return reply(req, http.StatusNotFound, "non simulé")
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	if j := strings.Index(s, b); j >= 0 {
		return s[:j]
	}
	return s
}

func reply(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     strconv.Itoa(status) + " " + http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func replyJSON(req *http.Request, v any) *http.Response {
	b, _ := json.Marshal(v)
	return reply(req, http.StatusOK, string(b))
}

// history sert une page de l'historique : start et count de la requête, au-delà de la liste une page vide.
func (f *fakeHalo) history(req *http.Request) *http.Response {
	q := req.URL.Query()
	start, _ := strconv.Atoi(q.Get("start"))
	count, _ := strconv.Atoi(q.Get("count"))
	f.mu.Lock()
	f.historyStarts = append(f.historyStarts, start)
	var page []map[string]any
	for i := start; i < len(f.ids) && i < start+count; i++ {
		started := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Add(-time.Duration(i) * time.Hour)
		page = append(page, map[string]any{"MatchId": f.ids[i], "MatchInfo": map[string]any{"StartTime": started.Format(time.RFC3339)}})
	}
	f.mu.Unlock()
	return replyJSON(req, map[string]any{"Results": page})
}

// stats sert les stats d'un match (ou le statut durable posé pour lui).
func (f *fakeHalo) stats(req *http.Request, id string) *http.Response {
	f.mu.Lock()
	f.statsCalls[id]++
	status := f.statsStatus[id]
	f.mu.Unlock()
	if status != 0 {
		return reply(req, status, "erreur simulée")
	}
	return replyJSON(req, convergenceMatchStats(id, f.positionOf(id)))
}

func (f *fakeHalo) positionOf(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, x := range f.ids {
		if x == id {
			return i
		}
	}
	return 0
}

// skill sert un skill plausible pour le joueur suivi.
func (f *fakeHalo) skill(req *http.Request, _ string) *http.Response {
	return replyJSON(req, map[string]any{"Value": []any{map[string]any{
		"Id": "xuid(" + convergenceXUID + ")", "ResultCode": 0,
		"Result": map[string]any{
			"TeamMmr": 1500.0, "TeamId": 0, "TeamMmrs": map[string]any{"0": 1500.0, "1": 1480.0},
			"StatPerformances": map[string]any{
				"Kills":  map[string]any{"Count": 12, "Expected": 10.0, "StdDev": 3.0},
				"Deaths": map[string]any{"Count": 8, "Expected": 9.0, "StdDev": 3.0},
			},
		},
	}}})
}

// convergenceMatchStats rend un corps de stats de match dont le premier participant est le joueur suivi
// (condition pour que le match soit « connu » une fois rangé), pour deux joueurs.
func convergenceMatchStats(id string, position int) map[string]any {
	players := make([]any, 0, 2)
	for i := 0; i < 2; i++ {
		xuid := convergenceXUID
		name := convergenceGamertag
		if i == 1 {
			xuid, name = "2533274800000002", "Adversaire"
		}
		core := map[string]any{
			"Kills": float64(10 + i), "Deaths": float64(5 + i), "Assists": float64(3 + i),
			"ShotsFired": 100.0, "ShotsHit": 50.0, "PersonalScore": float64(1000 + 100*i),
			"DamageDealt": 2500.0, "DamageTaken": 2000.0, "AverageLifeDuration": "PT30S",
			"Medals": []any{map[string]any{"NameId": float64(100 + i), "Count": 2.0}},
		}
		outcome := 2
		if i == 1 {
			outcome = 3
		}
		players = append(players, map[string]any{
			"PlayerId": "xuid(" + xuid + ")", "PlayerName": name, "LastTeamId": float64(i), "Outcome": float64(outcome),
			"Rank": float64(i + 1), "PlayerTeamStats": []any{map[string]any{"Stats": map[string]any{"CoreStats": core}}},
			"ParticipationInfo": map[string]any{"TimePlayed": "PT10M0S"},
		})
	}
	started := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).Add(-time.Duration(position) * time.Hour)
	return map[string]any{
		"MatchId": id,
		"MatchInfo": map[string]any{
			"StartTime": started.Format(time.RFC3339), "EndTime": started.Add(10 * time.Minute).Format(time.RFC3339),
			"MapVariant":          map[string]any{"AssetId": "map-asset-001", "PublicName": "Aquarius"},
			"GameVariantCategory": 9.0, "PlaylistExperience": "Arena:Slayer",
			"Playlist":       map[string]any{"AssetId": "playlist-001", "PublicName": "Ranked Arena"},
			"UgcGameVariant": map[string]any{"AssetId": "gv-001", "PublicName": "Slayer"},
			"Duration":       "PT10M0S", "PlayableDuration": "PT9M30S", "LifecycleMode": 3.0,
		},
		"Players": players,
	}
}

// fakeTokenPool est le pool de jetons du test : un seul jeton, sans limite de débit (le limiteur du
// créneau est illimité, sinon le plancher de 5 requêtes par seconde du client poolé rendrait le test
// interminable).
type fakeTokenPool struct{}

func (fakeTokenPool) Acquire(context.Context, pool.AcquirePolicy, string) (*pool.Lease, error) {
	return &pool.Lease{
		Tokens:   &domain.HaloTokens{SpartanToken: "jeton-spartan-test", ClearanceToken: "jeton-clearance-test"},
		Gamertag: convergenceGamertag,
		Release:  func() {},
		Limiter:  rate.NewLimiter(rate.Inf, 1),
	}, nil
}
func (fakeTokenPool) Size() int                                                      { return 1 }
func (fakeTokenPool) HasPlayer(gamertag string) bool                                 { return gamertag == convergenceGamertag }
func (fakeTokenPool) IsWithheld(string) bool                                         { return false }
func (fakeTokenPool) SetShared(string, bool) bool                                    { return true }
func (fakeTokenPool) MarkUnhealthy(string, error)                                    {}
func (fakeTokenPool) OnHTTPError(int, time.Duration)                                 {}
func (fakeTokenPool) On429ForToken(string, time.Duration)                            {}
func (fakeTokenPool) AddOrUpdateSource(context.Context, pool.CredentialSource) error { return nil }
func (fakeTokenPool) Close()                                                         {}

// recordingOrchestrator enveloppe l'orchestrateur de production et garde le résultat de chaque cycle
// (l'ordonnanceur ne rend pas HistoryPending par joueur ; l'orchestrateur, si).
type recordingOrchestrator struct {
	inner syncv2.CycleOrchestrator
	mu    sync.Mutex
	runs  []syncv2.CycleResult
	errs  []error
}

func (r *recordingOrchestrator) Run(ctx context.Context, players []syncv2.PlayerProfile) (syncv2.CycleResult, error) {
	res, err := r.inner.Run(ctx, players)
	r.mu.Lock()
	r.runs = append(r.runs, res)
	r.errs = append(r.errs, err)
	r.mu.Unlock()
	return res, err
}

func (r *recordingOrchestrator) last(t *testing.T) (syncv2.PlayerOutcome, syncv2.CycleResult) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.runs) == 0 {
		t.Fatal("l'orchestrateur n'a été appelé par aucun cycle")
	}
	if err := r.errs[len(r.errs)-1]; err != nil {
		t.Fatalf("le cycle de l'orchestrateur a échoué : %v", err)
	}
	res := r.runs[len(r.runs)-1]
	out, ok := res.PerPlayer[convergenceGamertag]
	if !ok {
		t.Fatalf("le joueur suivi est absent de l'issue du cycle : %+v", res.PerPlayer)
	}
	return out, res
}

// convergenceEnv est le banc complet d'un joueur Halo Infinite.
type convergenceEnv struct {
	t         *testing.T
	ctx       context.Context
	cfg       *config.AppConfig
	fake      *fakeHalo
	sched     *scheduler.AutoSyncScheduler
	rec       *recordingOrchestrator
	provider  sharedprovider.Provider
	playerDB  string
	intervalH int
}

// newConvergenceEnv monte le banc : racine temporaire (rien sous data/ du dépôt), réglages, profil
// suivi, bases migrées comme au démarrage, file de persistance et worker, orchestrateur de production,
// ordonnanceur avec sa boucle Run.
func newConvergenceEnv(t *testing.T, total int) *convergenceEnv {
	t.Helper()
	duckdbpkg.CloseAll()
	t.Cleanup(duckdbpkg.CloseAll)

	root := t.TempDir()
	pr := titlePkg.NewPathResolver(root)
	slug := titlePkg.DefaultSlug
	const intervalHours = 6
	settingsPath := filepath.Join(root, "app_settings.json")
	profilesPath := filepath.Join(root, "db_profiles.json")
	writeTestFile(t, settingsPath, fmt.Sprintf(`{"spnkr_auto_sync_enabled": true, "spnkr_auto_sync_interval_hours": %d}`, intervalHours))
	writeTestFile(t, profilesPath, fmt.Sprintf(`{"version":"3.0","admin":%q,"profiles":{%q:{%q:{"db_path":%q,"xuid":%q,"waypoint_player":%q}}}}`,
		convergenceGamertag, slug, convergenceGamertag, "data/titles/"+slug+"/players/"+convergenceGamertag+"/stats.duckdb", convergenceXUID, convergenceGamertag))
	cfg := &config.AppConfig{
		RepoRoot: root, DBProfilesPath: profilesPath, AppSettingsPath: settingsPath,
		PersistBatchAsync: true,
	}

	// Bases migrées comme au démarrage du serveur (mêmes fonctions), puis base joueur créée à l'onboarding
	// (OpenPlayerDB) et migrée (RunPlayerMigrations).
	if err := ensureWarehouseDir(pr, slug); err != nil {
		t.Fatalf("ensureWarehouseDir: %v", err)
	}
	if err := runMigrations(pr.MetadataDBPath(slug), pr.SharedDBPath(slug), pr.SharedSocialDBPath(slug), pr.SharedPVEDBPath(slug), ""); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}
	playerPath := pr.PlayerDBPath(slug, convergenceGamertag)
	if err := os.MkdirAll(filepath.Dir(playerPath), 0o755); err != nil {
		t.Fatalf("mkdir joueur: %v", err)
	}
	created, err := syncpkg.OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	_ = created.Close()
	if err := RunPlayerMigrations(playerPath, slug); err != nil {
		t.Fatalf("RunPlayerMigrations: %v", err)
	}

	// Fournisseur B-swap de la base partagée et base metadata partagée, comme openSharedReader / openMetadataDB.
	mgr := sharedprovider.NewManager()
	provider, err := mgr.For(pr.SharedDBPath(slug))
	if err != nil {
		t.Fatalf("sharedprovider: %v", err)
	}
	cfg.SharedProvider, cfg.SharedManager = provider, mgr
	t.Cleanup(func() { _ = mgr.Close() })
	metaDB, err := duckdbpkg.OpenReadWriteShared(pr.MetadataDBPath(slug))
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	t.Cleanup(func() { _ = metaDB.Close() })

	// Faux réseau : toutes les requêtes HTTP du client de production passent par fakeHalo.
	fake := newFakeHalo(total)
	previous := http.DefaultTransport
	http.DefaultTransport = fake.transport()
	t.Cleanup(func() { http.DefaultTransport = previous })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	settingsStore := settingsplatform.NewStore(settingsPath)
	sched := scheduler.New(cfg, settingsStore, nil, fakeTokenPool{}).
		WithHistoryReader(historyrepo.New(func(titleSlug, gamertag string) string {
			return config.PlayerDBPath(cfg, titleSlug, gamertag)
		}, nil))

	// File de persistance et worker : la fonction de démarrage elle-même (openPersistQueue).
	st := &bootState{
		ctx: ctx, cfg: cfg, pr: pr, titleSlug: slug, autoScheduler: sched,
		dbPaths: bootDBPaths{shared: pr.SharedDBPath(slug), metadata: pr.MetadataDBPath(slug)},
	}
	st.boot = newBootTasks(config.DeploymentDev, &st.schedulerWG)
	openPersistQueue(st)
	if st.autoBatchQueue == nil {
		t.Fatal("openPersistQueue n'a pas ouvert la file de persistance")
	}
	t.Cleanup(func() {
		drainCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_ = st.autoBatchQueue.Drain(drainCtx)
		_ = st.autoBatchQueue.Close()
		st.workerWG.Wait()
	})

	// Orchestrateur de production. Écart assumé : Settings, Friends, PostSyncRunner, PrestigeHook et
	// ReplayEnqueue restent nuls. Settings câblerait l'indexation des médias (elle lit les dossiers de
	// captures de la machine) et la construction des rejeux (film, ffmpeg) ; PostSyncRunner,
	// PrestigeHook et Friends servent des notifications et des défis. Aucun ne participe à la
	// découverte, à la récupération, à la persistance ni à la fin d'historique, que ce banc éprouve.
	inner := buildSyncV2Orchestrator(SyncV2WiringDeps{
		Cfg: cfg, PathResolver: pr, TitleSlug: slug, TokenPool: fakeTokenPool{},
		BatchQueue: st.autoBatchQueue, MetaDB: metaDB.SQLDb(),
	})
	if inner == nil {
		t.Fatal("buildSyncV2Orchestrator a rendu nil")
	}
	rec := &recordingOrchestrator{inner: inner}
	sched.WithCycleOrchestrator(rec)

	// Boucle Run de l'ordonnanceur : la cadence se recale à la fin de chaque cycle demandé hors tick.
	go sched.Run(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for sched.NextCycleAt().IsZero() {
		if time.Now().After(deadline) {
			t.Fatal("la boucle Run de l'ordonnanceur n'a pas démarré")
		}
		time.Sleep(10 * time.Millisecond)
	}
	return &convergenceEnv{t: t, ctx: ctx, cfg: cfg, fake: fake, sched: sched, rec: rec, provider: provider, playerDB: playerPath, intervalH: intervalHours}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("écriture %s: %v", path, err)
	}
}

// cycle lance un cycle PAR L'ORDONNANCEUR (RunOnceTrigger), compteurs de requêtes remis à zéro avant.
func (e *convergenceEnv) cycle() (syncv2.PlayerOutcome, []int) {
	e.t.Helper()
	e.fake.resetCounters()
	res := e.sched.RunOnceTrigger(e.ctx, "manual")
	if res == nil {
		e.t.Fatal("RunOnceTrigger a rendu nil")
	}
	out, _ := e.rec.last(e.t)
	return out, e.fake.historyReads()
}

// registryIDs rend les matchs du registre partagé (vérité terrain : base DuckDB réelle).
func (e *convergenceEnv) registryIDs() map[string]bool {
	e.t.Helper()
	db, release, err := e.provider.Get(e.ctx)
	if err != nil {
		e.t.Fatalf("provider.Get: %v", err)
	}
	defer release()
	rows, err := db.Query(`SELECT match_id FROM match_registry`)
	if err != nil {
		e.t.Fatalf("lecture match_registry: %v", err)
	}
	defer rows.Close()
	ids := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			e.t.Fatalf("scan: %v", err)
		}
		ids[id] = true
	}
	return ids
}

// historyState lit la clé de fin et le marque-page dans la base joueur réelle.
func (e *convergenceEnv) historyState() (complete bool, cursor *historystate.Cursor) {
	e.t.Helper()
	db, release, err := duckdbpkg.OpenPlayerReadForQuery(e.playerDB)
	if err != nil {
		e.t.Fatalf("lecture base joueur: %v", err)
	}
	defer release()
	complete, err = historystate.Complete(e.ctx, db)
	if err != nil {
		e.t.Fatalf("état de l'historique: %v", err)
	}
	if cur, ok := historystate.LoadCursor(e.ctx, db); ok {
		cursor = &cur
	}
	return complete, cursor
}

// waitCadence attend que la boucle Run ait recalé le prochain cycle à `want` (à deux minutes près) et
// rend le délai constaté. La cadence se lit sur NextCycleAt de l'ordonnanceur, recalée par sa boucle à la
// fin de chaque cycle demandé hors tick.
func (e *convergenceEnv) waitCadence(want time.Duration) time.Duration {
	e.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var got time.Duration
	for time.Now().Before(deadline) {
		got = time.Until(e.sched.NextCycleAt())
		if got > want-2*time.Minute && got < want+2*time.Minute {
			return got
		}
		time.Sleep(20 * time.Millisecond)
	}
	e.t.Fatalf("cadence : prochain cycle dans %v, attendu environ %v", got.Round(time.Second), want)
	return got
}
