package killcollector

// emprise_v0_passe_research_test.go — LOT V0 DU PLAN `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`,
// VOLET COLLECTEUR : la passe ACTUELLE chronometree, et la mesure V0.1 (le joueur en vehicule
// n'est pas situe).
//
// # CE QUE CE FICHIER MESURE, ET POURQUOI ICI
//
// Deux tests, joues DANS L'ORDRE par le protocole du lot (voir le journal V0 du plan) :
//
//	TestEmpriseV0Identites  ce que la BASE dit de chaque match (noms de carte par le
//	                        resolveur de production, faits de match) — pose en JSON pour
//	                        l'instrument de cuisson (`replaybuild`), qui n'ouvre aucune base.
//	TestEmpriseV0Passe      la passe du collecteur telle que la production l'enchaine
//	                        (`CollectMatch` : decodage, roster, fusion + ecriture, tirs,
//	                        positions, faits d'isolement), plusieurs tours, mediane et pic
//	                        memoire ; puis UNE passe instrumentee qui garde le registre et les
//	                        positions du collecteur pour la mesure V0.1.
//
// V0.1 LIT LA VISIBILITE PAR LA FONCTION DE PRODUCTION, SANS LA RECOPIER. `visibleA` est privee
// a `film/replay` ; `replay.ContextesDesMorts` rend un contexte pour chaque entree du journal
// dont la VICTIME est visible (position de moins d'une seconde, `visibleA`), et seulement pour
// elle. Chaque instant a bord devient donc une « mort » sondee de l'occupant, seul de son camp :
// le contexte sort si et seulement si `visibleA` repond oui. Meme pont slot -> xuid (le registre
// de la passe), meme horloge (film µs / 1000 − `DeathOffsetMS()` du registre).
//
// # CE QU'IL NE FAIT PAS
//
// Aucune ecriture dans `data/` : la base reelle est une COPIE ouverte en lecture seule, les
// films sont lus dans le cache, et les ecritures de la passe vont dans une base TEMPORAIRE
// migree (`t.TempDir()`). Les sorties JSON vont dans le repertoire du lot (scratchpad).
//
// SANS LES VARIABLES, IL SE SAUTE (les films et la base ne sont pas versionnes) :
//
//	EMPRISE_V0_FILMS=<match_id,...> EMPRISE_V0_DIR=<scratch> EMPRISE_V0_DB=<copie shared>
//	EMPRISE_V0_CACHE=<racine data/cache> [EMPRISE_V0_TOURS=3] \
//	  go test ./internal/sync/killcollector/ -run '^TestEmpriseV0(Identites|Passe)$' -v -count=1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/replayidentity"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/testutil"
)

// v0Env porte les entrees du protocole, lues dans l'environnement.
type v0Env struct {
	films        []string
	dir, db, cac string
	tours        int
}

// v0Garde lit l'environnement et SAUTE proprement quand une entree manque.
func v0Garde(t *testing.T) v0Env {
	t.Helper()
	e := v0Env{
		dir: os.Getenv("EMPRISE_V0_DIR"), db: os.Getenv("EMPRISE_V0_DB"),
		cac: os.Getenv("EMPRISE_V0_CACHE"), tours: 3,
	}
	for _, f := range strings.Split(os.Getenv("EMPRISE_V0_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			e.films = append(e.films, f)
		}
	}
	if len(e.films) == 0 || e.dir == "" || e.db == "" || e.cac == "" {
		t.Skip("EMPRISE_V0_FILMS, EMPRISE_V0_DIR, EMPRISE_V0_DB et EMPRISE_V0_CACHE requis : instrument saute")
	}
	if v, err := strconv.Atoi(os.Getenv("EMPRISE_V0_TOURS")); err == nil && v > 0 {
		e.tours = v
	}
	return e
}

// v0OuvrirCopie ouvre la COPIE de la base en lecture seule.
func v0OuvrirCopie(t *testing.T, chemin string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", chemin+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("copie de base %s : %v", chemin, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// v0Lecteur adapte un handle au port de lecture partagee du resolveur de carte.
type v0Lecteur struct{ db *sql.DB }

func (l v0Lecteur) Get(context.Context) (*sql.DB, func(), error) { return l.db, func() {}, nil }

// v0Identite : ce que la base dit d'un match, pose pour l'instrument de cuisson.
type v0Identite struct {
	MatchID string          `json:"matchId"`
	Noms    []string        `json:"noms"`
	Faits   port.MatchFacts `json:"faits"`
}

// v0Ecrire pose un JSON dans le repertoire du lot.
func v0Ecrire(t *testing.T, dir, sous, court string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, sous), 0o750); err != nil {
		t.Fatalf("repertoire %s : %v", sous, err)
	}
	blob, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("serialisation %s/%s : %v", sous, court, err)
	}
	if err := os.WriteFile(filepath.Join(dir, sous, court+".json"), blob, 0o600); err != nil {
		t.Fatalf("ecriture %s/%s : %v", sous, court, err)
	}
}

// v0Lire relit un JSON du repertoire du lot ; faux quand il n'existe pas.
func v0Lire(t *testing.T, dir, sous, court string, v any) bool {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(dir, sous, court+".json"))
	if err != nil {
		return false
	}
	if err := json.Unmarshal(blob, v); err != nil {
		t.Fatalf("relecture %s/%s : %v", sous, court, err)
	}
	return true
}

func TestEmpriseV0Identites(t *testing.T) {
	e := v0Garde(t)
	db := v0OuvrirCopie(t, e.db)
	ctx := context.Background()
	cartes := duckdb.NewReplayMapRepo(v0Lecteur{db}, nil)
	faits := duckdb.NewReplayFactsRepo(db)
	for _, id := range e.films {
		keys, err := cartes.MapKeysForMatch(ctx, id)
		if err != nil {
			t.Fatalf("%s : carte : %v", id, err)
		}
		f, err := faits.FactsForMatch(ctx, id)
		if err != nil {
			t.Fatalf("%s : faits : %v", id, err)
		}
		court := title.FilmShortMatchID(id)
		v0Ecrire(t, e.dir, "k1", court, v0Identite{MatchID: id, Noms: keys.Names, Faits: f})
		t.Logf("%s  %-28s joueurs=%d cartes=%v", court, f.GameVariantName, len(f.Players), keys.Names)
	}
}

// v0Echantillon tient le PIC d'empreinte (meme mesure que la sentinelle du depot,
// `filmproc.Footprint`) pendant une phase, echantillonne toutes les 10 ms. Au-dela de 10 Gio il
// ARRETE le processus : une mesure ne doit jamais faire suffoquer la machine de travail.
type v0Echantillon struct {
	pic  atomic.Uint64
	stop chan struct{}
	fini chan struct{}
}

const v0PlafondDur = 10 << 30

func v0Echantillonner() *v0Echantillon {
	runtime.GC()
	debug.FreeOSMemory()
	s := &v0Echantillon{stop: make(chan struct{}), fini: make(chan struct{})}
	s.noter(filmproc.Footprint())
	go func() {
		defer close(s.fini)
		tk := time.NewTicker(10 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-tk.C:
				v := filmproc.Footprint()
				s.noter(v)
				if v > v0PlafondDur {
					fmt.Fprintf(os.Stderr, "EMPRISE V0 : empreinte %.2f Gio > plafond — arret\n", float64(v)/(1<<30))
					os.Exit(3)
				}
			}
		}
	}()
	return s
}

func (s *v0Echantillon) noter(v uint64) {
	for {
		old := s.pic.Load()
		if v <= old || s.pic.CompareAndSwap(old, v) {
			return
		}
	}
}

// arreter rend le pic de la phase, instant present compris.
func (s *v0Echantillon) arreter() uint64 {
	s.noter(filmproc.Footprint())
	close(s.stop)
	<-s.fini
	return s.pic.Load()
}

// v0BaseTemporaire : la base ou la passe ECRIT — temporaire, migree comme la base partagee.
func v0BaseTemporaire(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("base temporaire : %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migration de la base temporaire : %v", err)
	}
	return db
}

// v0Collecteur cable le collecteur comme la production : les trois capabilities du titre
// (`capabilities.toml` : film.kill_source, film.weapon_shots, film.kill_positions), le cache
// disque comme source de films, le roster par defaut (jointure par match) sur la copie, la
// capture de positions par `CaptureDepuisCatalogue` et le resolveur de carte de production.
// `ConfigureFilmAccuracy` n'est PAS appele : aucun appelant de production ne l'appelle.
func v0Collecteur(t *testing.T, e v0Env, lecture, ecriture *sql.DB) *KillSourceCollector {
	t.Helper()
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	deps, err := CaptureDepuisCatalogue(repoRoot, title.DefaultSlug,
		duckdb.NewReplayMapRepo(v0Lecteur{lecture}, nil))
	if err != nil {
		t.Fatalf("capture : %v", err)
	}
	caps := games.CapabilityMap{
		games.CapFilmKillSource:    games.CapSupported,
		games.CapFilmWeaponShots:   games.CapSupported,
		games.CapFilmKillPositions: games.CapSupported,
	}
	return NewKillSourceCollector(NewLocalCacheFilms(haloclient.NewLocalFilmCache(e.cac)),
		NewSharedRoster(lecture), func(context.Context) (*sql.DB, func(), error) {
			return ecriture, func() {}, nil
		}, caps, 0).AvecCapture(deps)
}

// v0Mediane rend la mediane d'une serie (copie triee).
func v0Mediane(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	if len(c)%2 == 1 {
		return c[len(c)/2]
	}
	return (c[len(c)/2-1] + c[len(c)/2]) / 2
}

// v0Passe : ce que le volet collecteur pose pour le volet porteurs et le rapport.
type v0Passe struct {
	MatchID      string          `json:"matchId"`
	Variante     string          `json:"variante"`
	PasseMS      []float64       `json:"passeMs"`
	PasseMedMS   float64         `json:"passeMedianeMs"`
	PasseMorts   int             `json:"passeMorts"`
	PicPasse     []uint64        `json:"picPasse"`
	RosterXUIDs  []string        `json:"rosterXuids"`
	Participants json.RawMessage `json:"participants"`
	Equipes      map[string]int  `json:"equipes"`
	Bots         json.RawMessage `json:"bots"`
	Calage       int64           `json:"calageMs"`
	CalageApp    int             `json:"calageAppariements"`
	Publiable    bool            `json:"pontPubliable"`
	Vies         [][3]int64      `json:"vies"`
	OrigineUS    uint64          `json:"origineUs"`
	V01          *v0Vehicules    `json:"v01,omitempty"`
	V01Refus     string          `json:"v01Refus,omitempty"`
}

func TestEmpriseV0Passe(t *testing.T) {
	e := v0Garde(t)
	lecture := v0OuvrirCopie(t, e.db)
	ctx := context.Background()
	for _, id := range e.films {
		court := title.FilmShortMatchID(id)
		var ident v0Identite
		if !v0Lire(t, e.dir, "k1", court, &ident) {
			t.Fatalf("%s : k1 absent — jouer TestEmpriseV0Identites d'abord", court)
		}
		p := v0Passe{MatchID: id, Variante: ident.Faits.GameVariantName}
		col := v0Collecteur(t, e, lecture, v0BaseTemporaire(t))
		for i := 0; i < e.tours; i++ {
			ech := v0Echantillonner()
			debut := time.Now()
			out, morts, err := col.CollectMatch(ctx, id)
			d := time.Since(debut)
			pic := ech.arreter()
			if err != nil || out != OutcomeWritten {
				t.Fatalf("%s : passe %d : %s %v", court, i, out, err)
			}
			p.PasseMS = append(p.PasseMS, float64(d.Microseconds())/1000)
			p.PicPasse = append(p.PicPasse, pic)
			p.PasseMorts = morts
		}
		p.PasseMedMS = v0Mediane(p.PasseMS)
		v0Instrumenter(t, ctx, col, &p)
		v0Ecrire(t, e.dir, "col", court, p)
		t.Logf("%s  %-26s passe mediane %7.0f ms  tours %v  pic %s", court, p.Variante,
			p.PasseMedMS, p.PasseMS, v0Gio(p.PicPasse))
		v0JournalV01(t, court, p)
	}
}

// v0Gio formate des pics en Gio.
func v0Gio(pics []uint64) string {
	parts := make([]string, 0, len(pics))
	for _, v := range pics {
		parts = append(parts, fmt.Sprintf("%.2f", float64(v)/(1<<30)))
	}
	return strings.Join(parts, "/") + " Gio"
}

// v0Instrumenter rejoue la passe PIECE A PIECE, avec les fonctions de production du collecteur,
// pour garder ce que `CollectMatch` ne rend pas : les identites du roster, le registre et les
// positions. Aucune ecriture.
func v0Instrumenter(t *testing.T, ctx context.Context, col *KillSourceCollector, p *v0Passe) {
	t.Helper()
	chunks, _, err := FilmChunksForMatch(ctx, col.client, p.MatchID)
	if err != nil {
		t.Fatalf("%s : chunks : %v", p.MatchID, err)
	}
	film, err := FilmOf(chunks)
	if err != nil {
		t.Fatalf("%s : film : %v", p.MatchID, err)
	}
	// MEMES OPTIONS QUE `decodeFilmForMatch` : la configuration gelee plus la carte du match.
	opts := decfilm.DefaultOptions()
	carte, err := col.carteDuMatch(ctx, p.MatchID)
	if err != nil {
		t.Fatalf("%s : carte : %v", p.MatchID, err)
	}
	opts.Carte = carte
	res, err := decfilm.Decode(ctx, p.MatchID, film, &opts)
	if err != nil {
		t.Fatalf("%s : decodage : %v", p.MatchID, err)
	}
	ids, err := col.roster.IdentitiesForMatch(ctx, p.MatchID)
	if err != nil {
		t.Fatalf("%s : roster : %v", p.MatchID, err)
	}
	batch := BuildKillSourceBatch(ctx, p.MatchID, res, ids)
	entry, err := col.resolveMapBounds(ctx, p.MatchID)
	if err != nil {
		t.Fatalf("%s : carte : %v", p.MatchID, err)
	}
	_, mat, err := buildPositionRows(ctx, film, res, entry, ids, killRefsFromDeaths(batch.Deaths), p.MatchID)
	if err != nil {
		t.Fatalf("%s : positions : %v", p.MatchID, err)
	}
	p.RosterXUIDs, p.Equipes = ids.XUIDs, ids.Equipes
	p.Participants = v0Brut(t, ids.Participants)
	p.Bots = v0Brut(t, replayidentity.BotIdentities(res))
	reg := mat.registre
	p.Calage, p.CalageApp, p.Publiable = reg.DeathOffsetMS(), reg.DeathOffsetMatches(), reg.PontPubliable()
	for _, v := range reg.ViesNommees() {
		p.Vies = append(p.Vies, [3]int64{int64(v.XUID), v.DebutMS, v.FinMS})
	}
	p.OrigineUS = v0PremierePosition(mat)
	v0MesurerV01(t, p, mat)
}

// v0Brut serialise une valeur en JSON brut (champ opaque pour ce volet).
func v0Brut(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("serialisation : %v", err)
	}
	return b
}
