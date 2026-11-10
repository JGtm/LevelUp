//go:build research

package replaybuild

// fin_de_vie_presence_research_test.go — INSTRUMENT DE TEMOIN : LA FIN DE LA DERNIERE VIE D UN JOUEUR
// QUE L API DECLARE PRESENT A LA FIN, SUR LE REEL.
//
// « Avant » est le document reconstruit sans les drapeaux de presence du tableau (la fin d une vie y est
// sa mort ecrite ou son dernier point) ; « apres » est le meme document avec le tableau de la base. Un
// film a la fois, en memoire : le film est decode UNE fois (les faits du cache sont d un schema anterieur),
// puis les deux documents sont rejoues depuis les faits que ce decodage rend. RIEN N EST ECRIT : ni les
// faits, ni l artefact, ni la base (copie en lecture seule).
//
//	FIN_DEPOT=<checkout qui porte data/cache> FIN_BASE=<copie de shared_matches_v2.duckdb> \
//	  go test -tags research -count=1 -run '^TestFinDeViePresenceAvantApres$' -v ./internal/replaybuild/
//
// FIN_FILMS (liste de prefixes de match separes par des virgules) remplace l echantillon par defaut ; FIN_N
// borne le nombre de films de l echantillon (20 par defaut).

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

// temoinsPresence : les trois films nommes, et le gamertag dont la derniere vie doit finir avec le film.
var temoinsPresence = map[string]string{
	"008e1bba": "Joueur90000008", "1e26f641": "Joueur900d", "15d8d850": "JGtm",
}

// bilanPresence : ce que la regle change sur un film.
type bilanPresence struct {
	vies, changees, sansTableau, mortManquante, chevauchements, resumesDifferents int
	deltas                                                                        []int
}

func TestFinDeViePresenceAvantApres(t *testing.T) {
	depot, base := os.Getenv("FIN_DEPOT"), os.Getenv("FIN_BASE")
	if depot == "" || base == "" {
		t.Skip("instrument de temoin : FIN_DEPOT et FIN_BASE requis")
	}
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewBuilder(context.Background(), repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", base+"?access_mode=read_only")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repo := duckdb.NewReplayFactsRepo(db)

	var total bilanPresence
	var films int
	for _, id := range filmsDeLEchantillon(t, depot, db) {
		bil, ok := temoinDUnFilm(t, b, repo, db, depot, id)
		if !ok {
			continue
		}
		films++
		total.vies += bil.vies
		total.changees += bil.changees
		total.sansTableau += bil.sansTableau
		total.mortManquante += bil.mortManquante
		total.chevauchements += bil.chevauchements
		total.resumesDifferents += bil.resumesDifferents
		total.deltas = append(total.deltas, bil.deltas...)
	}
	sort.Ints(total.deltas)
	t.Logf("TOTAL %d films : %d vies publiees, %d vies changees (%d%%), %d cas sans tableau, %d cas mort manquante, "+
		"%d chevauchements, %d resumes d usage differents ; delta de fin (frames) : %s",
		films, total.vies, total.changees, pourcent(total.changees, total.vies), total.sansTableau,
		total.mortManquante, total.chevauchements, total.resumesDifferents, quantiles(total.deltas))
	if total.chevauchements != 0 {
		t.Errorf("%d chevauchement(s) de vies : la fin d une vie franchit l ouverture de la suivante", total.chevauchements)
	}
}

func pourcent(a, b int) int {
	if b == 0 {
		return 0
	}
	return a * 100 / b
}

func quantiles(d []int) string {
	if len(d) == 0 {
		return "aucun"
	}
	return fmt.Sprintf("min %d, mediane %d, p90 %d, max %d", d[0], d[len(d)/2], d[len(d)*9/10], d[len(d)-1])
}

// filmsDeLEchantillon rend les prefixes de match a rejouer : FIN_FILMS, sinon les trois temoins puis un
// pas regulier dans les faits persistes dont le match est en base.
func filmsDeLEchantillon(t *testing.T, depot string, db *sql.DB) []string {
	if l := os.Getenv("FIN_FILMS"); l != "" {
		return strings.Split(l, ",")
	}
	n := 20
	if v, err := strconv.Atoi(os.Getenv("FIN_N")); err == nil && v > 0 {
		n = v
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var tous []string
	for _, e := range entrees {
		if id, ok := strings.CutSuffix(e.Name(), ".filmfacts.bin"); ok {
			tous = append(tous, id)
		}
	}
	sort.Strings(tous)
	out := []string{"008e1bba", "1e26f641", "15d8d850"}
	pas := max(1, len(tous)/(n*2))
	for i := 0; i < len(tous) && len(out) < n; i += pas {
		if _, temoin := temoinsPresence[tous[i]]; !temoin {
			out = append(out, tous[i])
		}
	}
	return out
}

// temoinDUnFilm rejoue un film avec et sans la presence du tableau et rend le bilan.
func temoinDUnFilm(t *testing.T, b *Builder, repo *duckdb.ReplayFactsRepo, db *sql.DB,
	depot, id string) (bilanPresence, bool) {
	ctx := context.Background()
	var matchID string
	if err := db.QueryRowContext(ctx, `SELECT match_id FROM match_registry WHERE match_id LIKE ? LIMIT 1`,
		id+"%").Scan(&matchID); err != nil {
		t.Logf("%s : match absent de la base (%v)", id, err)
		return bilanPresence{}, false
	}
	facts, err := repo.FactsForMatch(ctx, matchID)
	if err != nil {
		t.Fatalf("%s : faits : %v", id, err)
	}
	apres, avant, ok := rejouerLeFilm(t, b, depot, matchID, facts, nomsDeCarte(ctx, db, matchID))
	if !ok {
		return bilanPresence{}, false
	}
	if len(apres.Tracks) != len(avant.Tracks) {
		t.Fatalf("%s : %d vies apres, %d avant : la regle ne change que les fins", id, len(apres.Tracks), len(avant.Tracks))
	}
	bil := bilanPresence{vies: len(apres.Tracks)}
	for i, tr := range apres.Tracks {
		av := avant.Tracks[i]
		if tr.Slot != av.Slot || tr.StartFrame != av.StartFrame || len(tr.Points) != len(av.Points) {
			t.Fatalf("%s : la vie %d change autrement que par sa fin", id, i)
		}
		if tr.EndFrame != av.EndFrame {
			bil.changees++
			bil.deltas = append(bil.deltas, tr.EndFrame-av.EndFrame)
		}
	}
	bil.chevauchements = chevauchements(apres.Tracks)
	for _, f := range apres.Coverage.Fallbacks {
		switch f.Name {
		case "repli_fin_de_vie_sans_tableau_api":
			bil.sansTableau = f.Hits
		case "repli_fin_de_vie_mort_manquante":
			bil.mortManquante = f.Hits
		}
	}
	bil.resumesDifferents = resumesDifferents(t, &avant, &apres)
	t.Logf("%s (%d frames) : %d vies, %d changees, sansTableau=%d mortManquante=%d chevauchements=%d resumesDifferents=%d",
		id, apres.FrameCount, bil.vies, bil.changees, bil.sansTableau, bil.mortManquante, bil.chevauchements,
		bil.resumesDifferents)
	if nom, ok := temoinsPresence[id]; ok {
		verifierLeTemoinNomme(t, id, nom, apres, avant)
	}
	return bil, true
}

// nomsDeCarte rend le nom brut de la carte du registre : la candidate du catalogue de bornes.
func nomsDeCarte(ctx context.Context, db *sql.DB, matchID string) []string {
	var nom sql.NullString
	_ = db.QueryRowContext(ctx, `SELECT map_name FROM match_registry WHERE match_id = ?`, matchID).Scan(&nom)
	return []string{strings.TrimSpace(nom.String)}
}

// rejouerLeFilm decode le film par les etapes de BuildBytes (sans serialiser ni ranger quoi que ce soit),
// puis rejoue le document depuis les faits rendus, avec puis sans la presence du tableau.
func rejouerLeFilm(t *testing.T, b *Builder, depot, matchID string, facts port.MatchFacts, noms []string,
) (apres, avant replay.ReplayDocument, ok bool) {
	ctx := context.Background()
	entry, err := b.ResolveMapEntry(noms)
	if err != nil {
		t.Logf("%s : carte hors catalogue %v", matchID, noms)
		return apres, avant, false
	}
	filmDir := filmcache.ChunkDir(title.NewPathResolver(depot).CacheRootDir(), matchID[:8])
	src, err := b.entreesDeLaCuisson(ctx, matchID, filmDir, entry)
	if err != nil {
		t.Logf("%s : film non decodable : %v", matchID, err)
		return apres, avant, false
	}
	stats := assemblerFilmStats(ctx, matchID, src.statborg, facts, src.deaths)
	b.poserLeReglementDuScore(stats.score, facts.GameVariantName)
	cat := b.collecterEntreesCatalogue(ctx, matchID, noms, facts, &stats, src)
	cuit, err := b.documentDeLaCuisson(ctx, matchID, filmDir, b.buildReplayOptions(ctx, entry, facts, cat, &stats), src)
	if err != nil || cuit.aPersister == nil {
		t.Logf("%s : pas de faits rendus par le decodage (%v)", matchID, err)
		return apres, avant, false
	}
	avec := b.buildReplayOptions(ctx, entry, facts, cat, &stats)
	sans := b.buildReplayOptions(ctx, entry, sansPresence(facts), cat, &stats)
	apres = replay.BuildFromFacts(ctx, matchID, b.titleSlug, cuit.aPersister, avec)
	avant = replay.BuildFromFacts(ctx, matchID, b.titleSlug, cuit.aPersister, sans)
	runtime.GC()
	return apres, avant, true
}

// sansPresence rend les faits sans aucun drapeau de presence ni nombre de morts : la cuisson d avant la regle.
func sansPresence(f port.MatchFacts) port.MatchFacts {
	joueurs := make([]port.MatchPlayerFact, len(f.Players))
	copy(joueurs, f.Players)
	for i := range joueurs {
		joueurs[i].PresentAtCompletion, joueurs[i].LeftInProgress, joueurs[i].DeathsDeclared = nil, nil, false
	}
	f.Players = joueurs
	return f
}

// verifierLeTemoinNomme : la derniere vie du joueur finit a la derniere frame du film.
func verifierLeTemoinNomme(t *testing.T, id, nom string, apres, avant replay.ReplayDocument) {
	var xuid string
	for _, r := range apres.Roster {
		if r.Name == nom {
			xuid = r.XUID
		}
	}
	if xuid == "" {
		t.Errorf("%s : %s absent du roster", id, nom)
		return
	}
	derniere := -1
	for i, tr := range apres.Tracks {
		if tr.XUID == xuid && (derniere < 0 || tr.StartFrame > apres.Tracks[derniere].StartFrame) {
			derniere = i
		}
	}
	if derniere < 0 {
		t.Errorf("%s : aucune vie publiee pour %s", id, nom)
		return
	}
	tr := apres.Tracks[derniere]
	pts := tr.Points
	t.Logf("%s %s : dernier point %d, fin avant %d, fin apres %d, derniere frame du film %d", id, nom,
		pts[len(pts)-1].T, avant.Tracks[derniere].EndFrame, tr.EndFrame, apres.FrameCount-1)
	if tr.EndFrame != apres.FrameCount-1 {
		t.Errorf("%s %s : EndFrame = %d, attendu la derniere frame du film %d", id, nom, tr.EndFrame, apres.FrameCount-1)
	}
}

// chevauchements compte les paires de vies d un meme slot, et d un meme joueur, dont l une finit a ou apres
// l ouverture de la suivante.
func chevauchements(tracks []replay.Track) int {
	n := 0
	for _, cle := range []func(replay.Track) string{
		func(t replay.Track) string { return "s" + strconv.Itoa(int(t.Slot)) },
		func(t replay.Track) string {
			if t.XUID == "" {
				return ""
			}
			return "x" + t.XUID
		},
	} {
		groupes := map[string][]replay.Track{}
		for _, tr := range tracks {
			if k := cle(tr); k != "" {
				groupes[k] = append(groupes[k], tr)
			}
		}
		for _, g := range groupes {
			sort.Slice(g, func(i, j int) bool { return g[i].StartFrame < g[j].StartFrame })
			for i := 1; i < len(g); i++ {
				if g[i].StartFrame <= g[i-1].EndFrame {
					n++
				}
			}
		}
	}
	return n
}

// resumesDifferents compte les joueurs dont le resume d usage differe entre les deux documents.
func resumesDifferents(t *testing.T, avant, apres *replay.ReplayDocument) int {
	a, b := replay.BuildUsageSummary(avant), replay.BuildUsageSummary(apres)
	n := 0
	if len(a.Players) != len(b.Players) {
		return len(a.Players) + len(b.Players)
	}
	for i := range a.Players {
		if !reflect.DeepEqual(a.Players[i], b.Players[i]) {
			n++
			va, vb := reflect.ValueOf(a.Players[i]), reflect.ValueOf(b.Players[i])
			for k := 0; k < va.NumField(); k++ {
				if !reflect.DeepEqual(va.Field(k).Interface(), vb.Field(k).Interface()) {
					t.Logf("   resume de %s : %s %v -> %v", a.Players[i].XUID, va.Type().Field(k).Name, va.Field(k).Interface(), vb.Field(k).Interface())
				}
			}
		}
	}
	return n
}
