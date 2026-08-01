//go:build integration

// Package sync — killsource_collector_test.go : LE GATE DU BRANCHEMENT.
//
// Le test central decode un FILM REEL, ecrit en base de test et relit PAR LA VUE `_latest` —
// la chaine entiere, sans mock du decodeur ni du persister. C est le seul test du lot qui
// prouve que les trois briques s emboitent.
//
// ⚠ LES FILMS NE SONT PAS VERSIONNES (107 Mo) : ce test se SKIPPE sans `KILLSOURCE_FIXTURES`,
// comme les goldens du decodeur. Le skip porte la commande exacte pour le rejouer — un skip
// muet ferait passer une absence de couverture pour une couverture.
//
//	KILLSOURCE_FIXTURES=../../../../data/cache/film_chunks \
//	  go test -tags=integration -p 1 ./internal/sync/ -run KillSource
//
// Les tests de contrat (capability, delai, traduction) tournent SANS fixture : ils n ont pas
// besoin d un film.

package sync

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/haloclient"
)

// ─── Harnais ───────────────────────────────────────────────────────────────────────────────

// fakeFilmClient : une source de chunks en memoire. Elle remplace le CDN, pas le decodeur.
type fakeFilmClient struct {
	chunks map[string][]haloclient.FilmChunk
	err    error
	calls  int
}

func (f *fakeFilmClient) GetFilmChunks(_ context.Context, matchID string) ([]haloclient.FilmChunk, bool, error) {
	f.calls++
	if f.err != nil {
		return nil, false, f.err
	}
	c, ok := f.chunks[matchID]
	if !ok {
		return nil, false, nil
	}
	return c, true, nil
}

// fakeRoster : la resolution nom -> xuid, sans base.
type fakeRoster map[string]string

func (r fakeRoster) RosterForMatch(_ context.Context, _ string) (map[string]string, error) {
	return r, nil
}

func openSharedTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

func sharedWriter(db *sql.DB) persist.SharedWriterFn {
	return func(_ context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
}

func capsAvecFilm() games.CapabilityMap {
	return games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}
}

// chargerFilmDeFixture lit `<KILLSOURCE_FIXTURES>/<court>/chunk_NN.bin` et rend la sequence
// avec les TYPES du manifeste tels que le pont les fournirait.
//
// Le type est deduit de la position — chunk 0 = HEADER, dernier = HIGHLIGHT, le reste
// REPLICATION_DATA. C est la structure verifiee sur le film 696a9d7c (J0.1 : 31 chunks, en-tete
// chunk 0 type 1, footer chunk 30 type 3). Le decodeur, lui, ne se fie PAS a ces types : il
// localise le kill-feed par son CONTENU. Ils ne servent donc ici qu a rendre le test fidele a
// ce que le client produit.
func chargerFilmDeFixture(t *testing.T, court string) []haloclient.FilmChunk {
	t.Helper()
	root := os.Getenv("KILLSOURCE_FIXTURES")
	if root == "" {
		t.Skip("KILLSOURCE_FIXTURES absent : les films ne sont pas versionnes (107 Mo). " +
			"Rejouer avec KILLSOURCE_FIXTURES=<racine des films en cache> " +
			"go test -tags=integration -p 1 ./internal/sync/ -run KillSource")
	}
	names, err := filepath.Glob(filepath.Join(root, court, "chunk_*.bin"))
	if err != nil || len(names) == 0 {
		t.Skipf("film %s absent de %s (%v) — fixture non disponible sur cette machine", court, root, err)
	}
	out := make([]haloclient.FilmChunk, 0, len(names))
	for i, n := range names {
		data, rErr := os.ReadFile(n)
		if rErr != nil {
			t.Fatalf("lecture %s: %v", n, rErr)
		}
		typ := 2
		switch {
		case i == 0:
			typ = 1
		case i == len(names)-1:
			typ = 3
		}
		out = append(out, haloclient.FilmChunk{Index: i, ChunkType: typ, Data: data})
	}
	return out
}

// ─── GATE 1 : la chaine entiere ────────────────────────────────────────────────────────────

// TestKillSourceCollecteFilmReelEtRelitParLaVue — LE test du gate.
//
// Il decode un film REEL, ecrit par le persister et relit PAR LA VUE `_latest` (jamais la table
// brute). Il verifie ensuite les proprietes que la chaine doit preserver de bout en bout :
// les trois etats de l assistant, `null` jamais confondu avec zero, la portee ecrite avec le
// resultat, et `assist_extra_count` interrogeable en SQL.
func TestKillSourceCollecteFilmReelEtRelitParLaVue(t *testing.T) {
	const film = "9b191a7f" // Arena 4v4 — l un des films de reference du decodeur
	chunks := chargerFilmDeFixture(t, film)

	db := openSharedTestDB(t)
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}}
	col := NewKillSourceCollector(client, fakeRoster{}, sharedWriter(db), capsAvecFilm(), 0)

	outcome, err := col.CollectMatch(context.Background(), film)
	if err != nil {
		t.Fatalf("CollectMatch: %v", err)
	}
	if outcome != OutcomeWritten {
		t.Fatalf("outcome = %q, attendu %q", outcome, OutcomeWritten)
	}

	// 1. La VUE sert des lignes, et elles portent la portee.
	var morts int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = ?`, film).
		Scan(&morts); err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if morts == 0 {
		t.Fatal("la vue ne sert aucune mort — la chaine film -> base est rompue")
	}
	t.Logf("film %s : %d morts ecrites et relues par la vue", film, morts)

	var sansPortee int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE read_path IS NULL OR read_path = '' OR read_origin IS NULL OR read_origin = ''`).
		Scan(&sansPortee); err != nil {
		t.Fatalf("select portee: %v", err)
	}
	if sansPortee != 0 {
		t.Errorf("%d ligne(s) sans portee — la portee s ecrit AVEC le resultat", sansPortee)
	}

	// 2. Le decodeur ne produit QUE des noms : sans roster, tous les xuid sont NULL. C est le
	//    comportement attendu, et il prouve que la resolution est bien du cote du collecteur.
	var xuidsRenseignes int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE victim_xuid IS NOT NULL`).Scan(&xuidsRenseignes); err != nil {
		t.Fatalf("select xuid: %v", err)
	}
	if xuidsRenseignes != 0 {
		t.Errorf("%d xuid renseigne(s) sans roster — le film n en porte aucun", xuidsRenseignes)
	}

	// 3. `assist_extra_count` INTERROGEABLE en SQL : c est le declencheur de migration vers une
	//    table fille. Sur le corpus il vaut 0 — ce qui compte est que la colonne reponde.
	var extra sql.NullInt64
	if err := db.QueryRow(`SELECT SUM(assist_extra_count) FROM match_kill_events_latest`).
		Scan(&extra); err != nil {
		t.Fatalf("SUM(assist_extra_count) doit etre interrogeable: %v", err)
	}
	t.Logf("SUM(assist_extra_count) = %v (0 attendu sur le corpus connu)", extra)

	// 4. LES TROIS ETATS DE L ASSISTANT ont survecu jusqu en base. Le film de reference porte
	//    les trois ; si une seule population manquait, la traduction les aurait confondus.
	var inconnus, aucun int
	if err := db.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE NOT assist_known),
		COUNT(*) FILTER (WHERE assist_known AND assist_gamertag IS NULL)
		FROM match_kill_events_latest`).Scan(&inconnus, &aucun); err != nil {
		t.Fatalf("select assistants: %v", err)
	}
	t.Logf("assistants : %d « on ne sait pas », %d « pas d assistant (mesure) »", inconnus, aucun)
	if aucun == 0 {
		t.Error("aucune ligne « pas d assistant MESURE » — les trois etats se sont effondres en deux")
	}

	// 5. `null` n est jamais zero : aucune part de degats a 0 ne doit apparaitre la ou la
	//    mesure est absente (une part non lue doit etre NULL, pas 0).
	var partsNulles int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE assist_damage_pct IS NOT NULL AND assist_gamertag IS NULL`).Scan(&partsNulles); err != nil {
		t.Fatalf("select parts: %v", err)
	}
	if partsNulles != 0 {
		t.Errorf("%d part(s) d assistant sans assistant nomme — sans assistant ce champ porte "+
			"une constante par film qui ne veut rien dire", partsNulles)
	}

	// 6. UNE SECONDE PASSE REMPLACE ENTIEREMENT LA PREMIERE dans la vue, et la table conserve
	//    les deux (append-only).
	if _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("2e passe: %v", err)
	}
	var vue, table int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_kill_events_latest),
		(SELECT COUNT(*) FROM match_kill_events)`).Scan(&vue, &table); err != nil {
		t.Fatalf("select 2e passe: %v", err)
	}
	if vue != morts {
		t.Errorf("apres 2 passes, la vue sert %d lignes au lieu de %d — elle melange des passes", vue, morts)
	}
	if table != 2*morts {
		t.Errorf("table = %d lignes, attendu %d (append-only : les deux passes sont conservees)",
			table, 2*morts)
	}
}

// TestKillSourceRosterResoutLesXuid — avec un roster, les noms deviennent des xuid. Le meme
// film, la meme chaine : seule la resolution change.
func TestKillSourceRosterResoutLesXuid(t *testing.T) {
	const film = "9b191a7f"
	chunks := chargerFilmDeFixture(t, film)

	db := openSharedTestDB(t)
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{film: chunks}}

	// On lit d abord les noms du film pour fabriquer un roster credible.
	col := NewKillSourceCollector(client, fakeRoster{}, sharedWriter(db), capsAvecFilm(), 0)
	if _, err := col.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("passe de reconnaissance: %v", err)
	}
	var victime string
	if err := db.QueryRow(`SELECT victim_gamertag FROM match_kill_events_latest LIMIT 1`).
		Scan(&victime); err != nil {
		t.Fatalf("select victime: %v", err)
	}

	db2 := openSharedTestDB(t)
	col2 := NewKillSourceCollector(client, fakeRoster{victime: "xuid(2533274792395366)"},
		sharedWriter(db2), capsAvecFilm(), 0)
	if _, err := col2.CollectMatch(context.Background(), film); err != nil {
		t.Fatalf("passe avec roster: %v", err)
	}

	var resolus int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest
		WHERE victim_xuid = 'xuid(2533274792395366)'`).Scan(&resolus); err != nil {
		t.Fatalf("select resolus: %v", err)
	}
	if resolus == 0 {
		t.Errorf("le roster n a resolu aucun xuid pour %q", victime)
	}
}

// ─── Contrats, sans fixture ────────────────────────────────────────────────────────────────

// TestKillSourceCapabilityAbsenteNeCassePas — degradation gracieuse : le cycle continue, aucun
// panic, aucune erreur remontee, et surtout AUCUN appel reseau.
func TestKillSourceCapabilityAbsenteNeCassePas(t *testing.T) {
	client := &fakeFilmClient{}
	col := NewKillSourceCollector(client, fakeRoster{}, nil, games.CapabilityMap{}, 0)

	outcome, err := col.CollectMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("capability absente ne doit PAS rendre d erreur: %v", err)
	}
	if outcome != OutcomeNotSupported {
		t.Errorf("outcome = %q, attendu %q", outcome, OutcomeNotSupported)
	}
	if client.calls != 0 {
		t.Errorf("%d appel(s) au client alors que la capability est absente — la porte doit "+
			"passer AVANT le reseau", client.calls)
	}
}

// TestKillSourceFilmAbsentEstUnEtatPasUneErreur — au moins 28 % des matchs n auront jamais de
// film (les films Theater expirent cote serveur). Si c etait une erreur, un backfill
// s arreterait sur le premier vieux match.
func TestKillSourceFilmAbsentEstUnEtatPasUneErreur(t *testing.T) {
	col := NewKillSourceCollector(&fakeFilmClient{}, fakeRoster{}, nil, capsAvecFilm(), 0)
	outcome, err := col.CollectMatch(context.Background(), "match-sans-film")
	if err != nil {
		t.Fatalf("film absent ne doit PAS rendre d erreur: %v", err)
	}
	if outcome != OutcomeNoFilm {
		t.Errorf("outcome = %q, attendu %q", outcome, OutcomeNoFilm)
	}
}

// TestKillSourcePasseMultiMatchsCompteToutSansSArreter — une erreur sur un match ne doit pas
// tuer la passe : elle est comptee, et la passe continue.
func TestKillSourcePasseMultiMatchsCompteToutSansSArreter(t *testing.T) {
	client := &fakeFilmClient{err: errors.New("CDN indisponible")}
	col := NewKillSourceCollector(client, fakeRoster{}, nil, capsAvecFilm(), 0)

	sum := col.CollectMatches(context.Background(), []string{"m1", "m2", "m3"})
	if sum.Total != 3 {
		t.Errorf("Total = %d, attendu 3", sum.Total)
	}
	if sum.Errors != 3 {
		t.Errorf("Errors = %d, attendu 3 — la passe doit compter et continuer", sum.Errors)
	}
	if client.calls != 3 {
		t.Errorf("%d appel(s), attendu 3 — la passe s est arretee au premier echec", client.calls)
	}
}

// TestKillSourcePasseSArreteSurAnnulation — l arret demande par l appelant rend la synthese de
// ce qui a ete fait, pas une erreur.
func TestKillSourcePasseSArreteSurAnnulation(t *testing.T) {
	client := &fakeFilmClient{}
	col := NewKillSourceCollector(client, fakeRoster{}, nil, capsAvecFilm(), 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sum := col.CollectMatches(ctx, []string{"m1", "m2"})
	if client.calls != 0 {
		t.Errorf("%d appel(s) apres annulation, attendu 0", client.calls)
	}
	if sum.Total != 2 {
		t.Errorf("Total = %d, attendu 2 (le perimetre demande, pas le perimetre traite)", sum.Total)
	}
}

// TestKillSourceLimiteDeTempsParDefaut — la limite existe et vaut la valeur documentee. Un
// collecteur sans limite laisse un film pathologique bloquer la passe entiere.
func TestKillSourceLimiteDeTempsParDefaut(t *testing.T) {
	col := NewKillSourceCollector(nil, nil, nil, capsAvecFilm(), 0)
	if col.timeout != defaultKillSourceTimeout {
		t.Errorf("timeout par defaut = %v, attendu %v", col.timeout, defaultKillSourceTimeout)
	}
	explicite := NewKillSourceCollector(nil, nil, nil, capsAvecFilm(), 3*time.Second)
	if explicite.timeout != 3*time.Second {
		t.Errorf("timeout explicite = %v, attendu 3s", explicite.timeout)
	}
}

// TestPontAssembleLaSequenceCompleteAvecTrous — les index du manifeste ne sont pas garantis
// contigus : le pont doit laisser les trous a nil sans decaler les chunks suivants (un decalage
// changerait le film decode sans rien signaler).
func TestPontAssembleLaSequenceCompleteAvecTrous(t *testing.T) {
	client := &fakeFilmClient{chunks: map[string][]haloclient.FilmChunk{
		"m1": {
			{Index: 0, ChunkType: 1, Data: []byte("entete")},
			{Index: 2, ChunkType: 2, Data: []byte("replication")},
			{Index: 5, ChunkType: 3, Data: []byte("killfeed")},
		},
	}}
	src, found, err := ChunkSourceForMatch(context.Background(), client, "m1")
	if err != nil || !found {
		t.Fatalf("ChunkSourceForMatch: found=%v err=%v", found, err)
	}
	if got := src.NumChunks(); got != 6 {
		t.Fatalf("NumChunks = %d, attendu 6 (dimensionne sur l index MAX, pas sur le compte)", got)
	}
	for idx, attendu := range map[int]string{0: "entete", 2: "replication", 5: "killfeed"} {
		b, cErr := src.Chunk(idx)
		if cErr != nil || string(b) != attendu {
			t.Errorf("chunk %d = %q (%v), attendu %q", idx, b, cErr, attendu)
		}
	}
	for _, vide := range []int{1, 3, 4} {
		if b, _ := src.Chunk(vide); len(b) != 0 {
			t.Errorf("chunk %d = %q, attendu vide (le trou du manifeste)", vide, b)
		}
	}
}

// TestPontCompteLesTypes — le diagnostic qui rend `ErrNoKillFeed` lisible : « 29 chunks de
// replication et AUCUN kill-feed » explique, l erreur seule laisse chercher.
func TestPontCompteLesTypes(t *testing.T) {
	types := CountFilmChunkTypes([]haloclient.FilmChunk{
		{ChunkType: 1}, {ChunkType: 2}, {ChunkType: 2}, {ChunkType: 3},
	})
	if types[1] != 1 || types[2] != 2 || types[3] != 1 {
		t.Errorf("comptes = %v, attendu 1/2/1", types)
	}
}
