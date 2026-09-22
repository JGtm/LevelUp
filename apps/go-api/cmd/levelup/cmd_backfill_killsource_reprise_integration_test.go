//go:build integration

package main

// cmd_backfill_killsource_reprise_integration_test.go — LA REPRISE, PROUVEE (lot 5.24.4).
//
// # CE QUE CE TEST ETABLIT, ET POURQUOI IL LE FAIT SUR DE VRAIS FILMS
//
// L en-tete de la commande affirme depuis des mois qu elle est « reprenable, et la cle est
// `decoder_rev` ». C etait vrai, et ce n etait pas verifie : aucun test n interrompait une passe
// pour la relancer. Ce fichier le fait, de bout en bout, sur des films REELS du cache :
//
//	1. une passe est lancee, puis ANNULEE au milieu (annulation du contexte d ARRET, exactement
//	   ce qu un SIGINT declenche) ;
//	2. la SELECTION de production (`filmsACollecter` -> `matchsAJour`, les vues `_latest`) est
//	   rejouee : les films deja ecrits doivent en etre SORTIS, et leur compte doit valoir
//	   exactement ce que la premiere passe a ecrit ;
//	3. la passe est relancee et va au bout ;
//	4. le resultat final est compare, LIGNE A LIGNE, a celui d une passe qui n a jamais ete
//	   interrompue.
//
// ET IL ETABLIT AUSSI CE QUE L ARRET NE FAIT PAS : les films EN VOL au moment de l annulation
// vont au bout et sont ECRITS. Sans cela, N ouvriers perdraient N decodages entames a chaque
// Ctrl-C, pour rien — la reprise se decidant de toute facon en base.
//
// ⚠ SANS `KILLSOURCE_FIXTURES`, IL SE SAUTE.
//
// ⚠ PAS DE CAPTURE DE POSITIONS ICI, ET C EST DELIBERE : elle exige une base metadata, que ce
// test n a pas de raison de fabriquer. Les tables comparees sont donc le journal des morts et la
// ventilation des tirs. L egalite des CINQ vues (positions et faits d isolement compris) est
// etablie par `TestOuvriers_MemesLignesQuUnSeulOuvrier` cote `killcollector`.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/killcollector"
)

// filmsDeLaReprise : des films du bas du cout QUI ECRIVENT (les moins chers du cache ne
// produisent rien neuf fois sur dix — D1 (5.24)). Six suffisent : on en coupe apres deux.
var filmsDeLaReprise = []string{
	"ee90570b", "c0a82e88", "e157a672", "1a37bcc8", "30d3c047", "cf040013",
}

// filmsCoupesA : apres combien de films l annulation tombe.
const filmsCoupesA = 2

// vuesDeLaReprise : ce que la passe sans capture de positions peuple.
var vuesDeLaReprise = []string{"match_kill_events_latest", "match_weapon_shots_latest"}

// baseDeReprise ouvre une base de test migree et y inscrit les films.
func baseDeReprise(t *testing.T, cacheRoot string, films []string) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_ = migration.All()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	cache := haloclient.NewLocalFilmCache(cacheRoot)
	if cache == nil {
		t.Skipf("cache de films introuvable sous %s", cacheRoot)
	}
	for _, film := range films {
		inscrireAuRegistreDeTest(t, db, cache, film)
	}
	return db
}

// inscrireAuRegistreDeTest pose le match et ses participants, tires DU FILM (le fil des morts) :
// les xuids que la passe resoudra sont alors les vrais.
func inscrireAuRegistreDeTest(t *testing.T, db *sql.DB, cache *haloclient.LocalFilmCache, film string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, map_name) VALUES (?, ?)`,
		film, "streets"); err != nil {
		t.Fatalf("registre %s: %v", film, err)
	}
	chunks, ok, err := killcollector.NewLocalCacheFilms(cache).GetFilmChunks(context.Background(), film)
	if err != nil || !ok {
		t.Skipf("film %s absent du cache (%v)", film, err)
	}
	decode, err := killcollector.FilmOf(chunks)
	if err != nil {
		t.Skipf("film %s illisible: %v", film, err)
	}
	deaths, err := replay.ScanDeaths(decode)
	if err != nil {
		return
	}
	vus := map[uint64]bool{}
	n := 0
	for _, d := range deaths {
		if d.XUID == 0 || vus[d.XUID] {
			continue
		}
		vus[d.XUID] = true
		x := strconv.FormatUint(d.XUID, 10)
		if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag, team_id)
			VALUES (?, ?, ?, ?)`, film, x, d.Gamertag, n%2); err != nil {
			t.Fatalf("participants %s: %v", film, err)
		}
		if _, err := db.Exec(`INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
			VALUES (?, ?, ?, ?, ?)`, film, x, d.Gamertag, x, d.Gamertag); err != nil {
			t.Fatalf("couples %s: %v", film, err)
		}
		n++
	}
}

// collecteurDeReprise construit la passe comme la commande, capture de positions exceptee.
func collecteurDeReprise(
	db *sql.DB, cacheRoot string, porte *killcollector.PorteDeLaBase,
	suivi *suiviDeLaPasse, arret context.Context,
) *killcollector.KillSourceCollector {
	return killcollector.NewKillSourceCollector(
		killcollector.NewLocalCacheFilms(haloclient.NewLocalFilmCache(cacheRoot)),
		porte.GarderLeRoster(killcollector.NewSharedRoster(db).AvecAnnuaireDePasse()),
		porte.GarderLeWriter(writerDeja(db)),
		games.CapabilityMap{
			games.CapFilmKillSource:  games.CapSupported,
			games.CapFilmWeaponShots: games.CapSupported,
		}, 0).
		AvecObservateur(suivi).AvecArretDoux(arret)
}

// racineDuCache : la racine des films, deduite de `KILLSOURCE_FIXTURES` (qui pointe sur
// `film_chunks`) — le cache attendu par `LocalFilmCache` est son PARENT.
func racineDuCache(t *testing.T) string {
	t.Helper()
	chunks := os.Getenv("KILLSOURCE_FIXTURES")
	if chunks == "" {
		t.Skip("KILLSOURCE_FIXTURES absent : les films ne sont pas versionnes. " +
			"KILLSOURCE_FIXTURES=<racine>/data/cache/film_chunks go test -tags=integration -p 1 ...")
	}
	return filepath.Dir(chunks)
}

// TestReprise_UnePasseInterrompueRepartAuDernierEtat — LE GATE DE 5.24.4.
func TestReprise_UnePasseInterrompueRepartAuDernierEtat(t *testing.T) {
	cacheRoot := racineDuCache(t)
	films := filmsPresentsDuCache(t, cacheRoot, filmsDeLaReprise)
	if len(films) <= filmsCoupesA {
		t.Skipf("il faut plus de %d films du cache pour couper au milieu (%d trouve(s))",
			filmsCoupesA, len(films))
	}
	o := killsourceOptions{titleSlug: "halo_infinite", cacheDir: cacheRoot, workers: 2}

	// ── LA PASSE TEMOIN : jamais interrompue ────────────────────────────────────────────────
	dbTemoin := baseDeReprise(t, cacheRoot, films)
	temoin, bilanTemoin := passeDeReprise(t, dbTemoin, cacheRoot, o, nil)
	if bilanTemoin.DejaAJour != 0 {
		t.Fatalf("la passe temoin part avec %d films deja a jour, attendu 0", bilanTemoin.DejaAJour)
	}
	if temoin.Written == 0 {
		t.Fatal("la passe temoin n a rien ecrit : le test ne compare rien")
	}

	// ── LA PASSE INTERROMPUE ────────────────────────────────────────────────────────────────
	db := baseDeReprise(t, cacheRoot, films)
	arret, annuler := context.WithCancel(context.Background())
	chemin := filepath.Join(t.TempDir(), "etat.json")
	// L ANNULATION EST DECLENCHEE PAR LE FICHIER D ETAT LUI-MEME : c est la preuve, au passage,
	// qu il est lisible et a jour pendant que la passe tourne.
	go annulerApresKFilms(chemin, filmsCoupesA, annuler)

	sum1, _ := passeDeRepriseAvec(t, db, cacheRoot, o, arret, chemin)
	if sum1.Written < filmsCoupesA {
		t.Fatalf("la passe interrompue n a ecrit que %d films, attendu au moins %d (les films en "+
			"vol doivent aller au bout : un arret doux ne jette pas un decodage entame)",
			sum1.Written, filmsCoupesA)
	}
	if sum1.Written >= len(films) {
		t.Skipf("la passe est allee au bout (%d/%d) avant que l annulation ne porte — machine "+
			"trop rapide pour ce corpus ; rejouer avec plus de films", sum1.Written, len(films))
	}
	t.Logf("passe interrompue : %d films ecrits sur %d (coupure demandee apres %d)",
		sum1.Written, len(films), filmsCoupesA)
	etat := lireEtatDeReprise(t, chemin)
	if etat.Phase != phaseTerminee && etat.Films.Traites != sum1.Written {
		t.Errorf("le fichier d etat annonce %d films traites, la passe en a ecrit %d",
			etat.Films.Traites, sum1.Written)
	}

	// ── LA SELECTION SAIT CE QUI EST DEJA FAIT ──────────────────────────────────────────────
	restants, bilan2, err := filmsACollecter(context.Background(), db, cacheRoot, o)
	if err != nil {
		t.Fatalf("selection de reprise: %v", err)
	}
	if bilan2.DejaAJour != sum1.Written {
		t.Errorf("la reprise saute %d films, la premiere passe en a ecrit %d — la selection et "+
			"la base ne disent pas la meme chose (`matchsAJour`, vues `_latest`)",
			bilan2.DejaAJour, sum1.Written)
	}
	if len(restants) != len(films)-sum1.Written {
		t.Errorf("%d films restants, attendu %d", len(restants), len(films)-sum1.Written)
	}

	// ── LA RELANCE VA AU BOUT, ET LE RESULTAT EST LE MEME ───────────────────────────────────
	sum2, bilan3 := passeDeReprise(t, db, cacheRoot, o, nil)
	t.Logf("relance : %d films deja a jour (sautes), %d ecrits ; temoin d une traite : %d",
		bilan3.DejaAJour, sum2.Written, temoin.Written)
	if bilan3.DejaAJour != sum1.Written {
		t.Errorf("la relance annonce %d films deja a jour, attendu %d", bilan3.DejaAJour, sum1.Written)
	}
	if total := sum1.Written + sum2.Written; total != temoin.Written {
		t.Errorf("%d films ecrits en deux temps contre %d d une traite", total, temoin.Written)
	}
	for _, vue := range vuesDeLaReprise {
		comparerVueDeReprise(t, dbTemoin, db, vue)
	}
}

// passeDeReprise joue une passe complete (sans arret) et rend sa synthese et son bilan.
func passeDeReprise(
	t *testing.T, db *sql.DB, cacheRoot string, o killsourceOptions, arret context.Context,
) (killcollector.KillSourceSummary, bilanDeSelection) {
	t.Helper()
	if arret == nil {
		arret = context.Background()
	}
	return passeDeRepriseAvec(t, db, cacheRoot, o, arret, filepath.Join(t.TempDir(), "etat.json"))
}

// passeDeRepriseAvec joue une passe sur le chemin d etat donne.
func passeDeRepriseAvec(
	t *testing.T, db *sql.DB, cacheRoot string, o killsourceOptions,
	arret context.Context, chemin string,
) (killcollector.KillSourceSummary, bilanDeSelection) {
	t.Helper()
	ctx := context.Background()
	candidats, bilan, err := filmsACollecter(ctx, db, cacheRoot, o)
	if err != nil {
		t.Fatalf("selection: %v", err)
	}
	if len(candidats) == 0 {
		return killcollector.KillSourceSummary{}, bilan
	}
	suivi := nouveauSuivi(chemin, o.titleSlug, candidats, bilan, o)
	porte := killcollector.NouvellePorteDeLaBase()
	col := collecteurDeReprise(db, cacheRoot, porte, suivi, arret)
	ids := make([]string, 0, len(candidats))
	for _, c := range candidats {
		ids = append(ids, c.matchID)
	}
	sum := col.CollectMatchesOuvriers(ctx, ids, o.workers)
	suivi.Terminee(causeDArret(arret))
	return sum, bilan
}

// annulerApresKFilms surveille le fichier d etat et annule des que `k` films sont traites.
func annulerApresKFilms(chemin string, k int, annuler context.CancelFunc) {
	for i := 0; i < 3000; i++ {
		raw, err := os.ReadFile(chemin)
		if err == nil {
			var e etatDeLaPasse
			if json.Unmarshal(raw, &e) == nil && e.Films.Traites >= k {
				annuler()
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	annuler()
}

// lireEtatDeReprise relit le fichier d etat.
func lireEtatDeReprise(t *testing.T, chemin string) etatDeLaPasse {
	t.Helper()
	raw, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("etat illisible: %v", err)
	}
	var e etatDeLaPasse
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("etat non deserialisable: %v", err)
	}
	return e
}

// filmsPresentsDuCache ne garde que les films reellement sur cette machine.
func filmsPresentsDuCache(t *testing.T, cacheRoot string, ids []string) []string {
	t.Helper()
	var out []string
	for _, id := range ids {
		if _, ok := compterChunks(cacheRoot, id); ok {
			out = append(out, id)
		}
	}
	return out
}

// comparerVueDeReprise : les memes lignes des deux cotes, identite technique exclue.
//
// `id` (sequence), `decode_pass` (tirage aleatoire) et `written_at` (l horloge) sont HORS
// COMPARAISON : une passe en deux temps numerote autrement, et c est sans consequence — ce sont
// les FAITS qui doivent etre identiques.
func comparerVueDeReprise(t *testing.T, temoin, reprise *sql.DB, vue string) {
	t.Helper()
	a := lignesDeLaVueDeReprise(t, temoin, vue)
	b := lignesDeLaVueDeReprise(t, reprise, vue)
	if len(a) != len(b) {
		t.Errorf("%s : %d lignes d une traite, %d en deux temps", vue, len(a), len(b))
		return
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("%s : ligne %d differente\n  d une traite %s\n  en deux temps %s", vue, i, a[i], b[i])
			return
		}
	}
	t.Logf("%s : %d lignes IDENTIQUES entre la passe d une traite et la passe reprise", vue, len(a))
}

func lignesDeLaVueDeReprise(t *testing.T, db *sql.DB, vue string) []string {
	t.Helper()
	colonnes := colonnesPorteusesDeFait(t, db, vue)
	rows, err := db.Query(fmt.Sprintf(`SELECT %s FROM %s ORDER BY ALL`,
		strings.Join(colonnes, ", "), vue))
	if err != nil {
		t.Fatalf("lecture de %s: %v", vue, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		vals := make([]any, len(colonnes))
		ptrs := make([]any, len(colonnes))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("scan de %s: %v", vue, err)
		}
		parts := make([]string, len(vals))
		for i, v := range vals {
			parts[i] = fmt.Sprintf("%v", v)
		}
		out = append(out, strings.Join(parts, "|"))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("lignes de %s: %v", vue, err)
	}
	sort.Strings(out)
	return out
}

// colonnesPorteusesDeFait lit le schema et retire l identite technique.
func colonnesPorteusesDeFait(t *testing.T, db *sql.DB, vue string) []string {
	t.Helper()
	rows, err := db.Query("DESCRIBE " + vue)
	if err != nil {
		t.Fatalf("DESCRIBE %s: %v", vue, err)
	}
	defer func() { _ = rows.Close() }()
	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("DESCRIBE %s (colonnes): %v", vue, err)
	}
	horsFait := map[string]bool{"id": true, "decode_pass": true, "written_at": true}
	var out []string
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatalf("DESCRIBE %s (scan): %v", vue, err)
		}
		if nom := fmt.Sprintf("%v", vals[0]); !horsFait[nom] {
			out = append(out, `"`+nom+`"`)
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s : aucune colonne porteuse de fait", vue)
	}
	return out
}
