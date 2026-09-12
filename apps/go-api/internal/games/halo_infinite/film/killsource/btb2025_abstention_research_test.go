package killsource

// btb2025_abstention_research_test.go — BANC DE DIAGNOSTIC, garde par variable d environnement.
//
// POURQUOI IL EXISTE. Mesure du pilote (2026-09-12, vue `match_kill_events_latest`) : le taux de
// morts SANS source de degat sur les Big Team Battle vaut 0-7 % en 2023-2024, 83-91 % de mars a
// aout 2025, 75-80 % en octobre-novembre 2025, puis 0-14 % en 2026. La metrique de sante publiee
// (`sante.go`) dit QUE la couverture s effondre, elle ne dit pas OU. Ce banc ouvre la boite entre
// le roster et les deux voies de lecture : il compte, film par film, la population brute des
// dead-states de la marche et la RAISON de chaque refus du filtre de credibilite.
//
// IL NE TOURNE PAS EN CI ET N ASSERTE RIEN : c est un instrument de mesure, pas un garde-rail.
//
//	KS_BTB2025_FILMS=<racine>/<id8>[,<id8>...]   (ou KS_BTB2025_ROOT + KS_BTB2025_IDS)
//	go test ./internal/games/halo_infinite/film/killsource/ -run TestBTB2025Abstention -v

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

const (
	btb2025RootEnv = "KS_BTB2025_ROOT"
	btb2025IDsEnv  = "KS_BTB2025_IDS"
)

func TestBTB2025Abstention(t *testing.T) {
	root := os.Getenv(btb2025RootEnv)
	ids := strings.Split(os.Getenv(btb2025IDsEnv), ",")
	if root == "" || len(ids) == 0 || ids[0] == "" {
		t.Skipf("banc de diagnostic : %s et %s requis", btb2025RootEnv, btb2025IDsEnv)
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		t.Run(id, func(t *testing.T) { diagnostiquerFilm(t, filepath.Join(root, id)) })
	}
}

// diagnostiquerFilm : une passe `prepare` complete, puis le detail de ce que chaque etape voit.
func diagnostiquerFilm(t *testing.T, dir string) {
	t.Helper()
	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("chargement %s : %v", dir, err)
	}
	o := DefaultOptions()
	o.normalize()
	c := &decodeCtx{name: filepath.Base(dir), opts: o}
	if err := c.prepare(context.Background(), src); err != nil {
		t.Fatalf("prepare : %v", err)
	}

	t.Logf("VERSION  film_major_version=%d lue=%v", c.film.majorVersion, c.film.versionLue)
	t.Logf("ROSTER   humains_kill_feed=%d nPlay=%d noms=%d bots=%d non_epingles=%d",
		c.roster.nHumans, c.roster.nPlay, len(c.roster.names),
		len(c.roster.bots.Bots), len(c.roster.unpinned))
	t.Logf("FEED     kills=%d morts=%d couples=%d meme_instant=%d",
		c.feed.nKills, c.feed.nDeaths, len(c.feed.pairs), len(c.feed.real))
	t.Logf("CALIB    %s | marge_bijection=%d score=%d",
		c.calib.String(), bijectionMargin(c.roster, c.feed.pairs, c.scanCands, c.bijScore), c.bijScore)

	w := c.walkRes
	t.Logf("MARCHE   paquets_a_events=%d localises=%d morts_brutes=%d credibles=%d plage_bipede=[%d,%d]",
		w.withEv, w.located, len(w.deads), len(w.credible), w.bipLo, w.bipHi)

	var horsPlage, victHorsRoster, tueurHorsRoster, catHorsEnum int
	histA := map[int]int{}
	for _, d := range w.deads {
		if d.slot < w.bipLo || d.slot > w.bipHi {
			horsPlage++
			continue
		}
		histA[int(d.dead.EnumA)]++
		switch {
		case d.dead.EnumA < 0 || int(d.dead.EnumA) >= c.roster.nPlay:
			victHorsRoster++
		case d.dead.EnumB < 0 || int(d.dead.EnumB) >= c.roster.nPlay:
			tueurHorsRoster++
		case d.dead.Val0c > 9:
			catHorsEnum++
		}
	}
	t.Logf("REFUS    hors_plage_bipede=%d victime_hors_roster=%d tueur_hors_roster=%d categorie_hors_enum=%d",
		horsPlage, victHorsRoster, tueurHorsRoster, catHorsEnum)
	t.Logf("INDICES  victime (dans la plage bipede) : %s", histogramme(histA))
	t.Logf("SCAN     candidats=%d (porte T1/T2 : indice < nPlay=%d)", len(c.scanCands), c.roster.nPlay)
}

// histogramme : rendu trie et compact d une distribution d indices.
func histogramme(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(strconv.Itoa(k) + ":" + strconv.Itoa(h[k]))
	}
	return b.String()
}

// TestBTB2025KillFeedNoms — LE MEME BANC, MAIS SUR LE SEUL CHUNK HIGHLIGHT.
//
// Il isole l etape amont : la version LUE dans l en-tete du registre, puis, pour chacune des deux
// implantations du bloc d event, combien de XUID distincts et combien de gamertags non vides. La
// version lue doit designer celle des deux qui rend un nom par joueur — verification croisee de
// l indicateur.
func TestBTB2025KillFeedNoms(t *testing.T) {
	root := os.Getenv(btb2025RootEnv)
	ids := strings.Split(os.Getenv(btb2025IDsEnv), ",")
	if root == "" || len(ids) == 0 || ids[0] == "" {
		t.Skipf("banc de diagnostic : %s et %s requis", btb2025RootEnv, btb2025IDsEnv)
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		src, err := filmsource.LoadDir(filepath.Join(root, id), nil)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		f, err := loadFilm(src)
		if err != nil {
			t.Fatalf("%s : %v", id, err)
		}
		t.Logf("%-8s film_major_version=%d (lue=%v)", id, f.majorVersion, f.versionLue)
		for _, ver := range []int{versionGamertagEnTeteTest, versionGamertagDecaleTest} {
			best, nk := meilleurHighlight(f, ver)
			xu, gt, vides := statsNoms(best)
			t.Logf("%-8s decoupage=%2d kills=%3d events=%4d xuid_distincts=%2d gamertags_distincts=%2d gamertags_vides=%4d",
				id, ver, nk, len(best), xu, gt, vides)
		}
	}
}

// meilleurHighlight : le chunk qui produit le plus de kills, sous la version demandee.
func meilleurHighlight(f *film, ver int) ([]analysis.HighlightEvent, int) {
	var best []analysis.HighlightEvent
	bestN := 0
	for ch := range f.chunks {
		evs, err := analysis.ParseHighlightEvents(f.chunks[ch], ver)
		if err != nil {
			continue
		}
		nk := 0
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				nk++
			}
		}
		if nk > bestN {
			bestN, best = nk, evs
		}
	}
	return best, bestN
}

// statsNoms : XUID distincts, gamertags distincts non vides, events sans gamertag.
func statsNoms(evs []analysis.HighlightEvent) (xuids, tags, vides int) {
	sx, st := map[uint64]bool{}, map[string]bool{}
	for _, e := range evs {
		sx[e.XUID] = true
		if e.Gamertag == "" {
			vides++
			continue
		}
		st[e.Gamertag] = true
	}
	return len(sx), len(st), vides
}

// TestBTB2025NonRegression — LE GARDE-RAIL DU CORRECTIF, garde par la meme variable.
//
// Il ASSERTE, lui, et il est le seul de ce fichier a le faire. Sur un film Big Team Battle de
// 2025 (version de film 39-40), le roster humain doit compter au moins 20 joueurs et la
// couverture depasser 80 % — avant le correctif du 2026-09-12 le roster tombait a 10-12 noms et
// la couverture a 5-17 %. Debrancher la lecture de la version (retour au 0 en dur, donc au
// decoupage << gamertag en tete >>) le ferait echouer immediatement.
//
//	KS_BTB2025_ROOT=<cache>/film_chunks KS_BTB2025_NONREG=111fa685 \
//	  go test ./internal/games/halo_infinite/film/killsource/ -run TestBTB2025NonRegression
func TestBTB2025NonRegression(t *testing.T) {
	root, id := os.Getenv(btb2025RootEnv), os.Getenv("KS_BTB2025_NONREG")
	if root == "" || id == "" {
		t.Skipf("banc garde : %s et KS_BTB2025_NONREG requis", btb2025RootEnv)
	}
	src, err := filmsource.LoadDir(filepath.Join(root, id), nil)
	if err != nil {
		t.Fatalf("chargement %s : %v", id, err)
	}
	res, err := Decode(context.Background(), id, src, nil)
	if err != nil {
		t.Fatalf("Decode : %v", err)
	}
	if res.Roster.Humans < 20 {
		t.Errorf("roster humain = %d, want >= 20 — la version du film n est plus LUE dans "+
			"l en-tete de son registre (cf. filmdec.FilmMajorVersion, killsource.loadFilm)",
			res.Roster.Humans)
	}
	if res.Coverage.RealPairs == 0 {
		t.Fatal("aucun couple reel : le kill-feed n a pas ete lu")
	}
	taux := float64(res.Coverage.Covered) / float64(res.Coverage.RealPairs)
	if taux < 0.80 {
		t.Errorf("couverture = %.1f %% (%d/%d), want >= 80 %%", taux*100,
			res.Coverage.Covered, res.Coverage.RealPairs)
	}
}

// TestBTB2025ParcVersions — LE PARC ENTIER DU CACHE, CLASSE PAR LA VERSION LUE.
//
// Il ne charge PAS les films : il lit les quatre premiers octets de `chunk_00.bin` (le
// FilmMajorVersion) et compte les films par version. Les versions 39-40 sont celles dont le
// gamertag vit a `b[12:44]` : ce sont les films dont les lignes `match_kill_events` ont ete
// produites avec un roster effondre, donc ceux a redecoder.
//
// LE CROISEMENT EST FAIT AU PASSAGE, sur demande (`KS_BTB2025_PARC_CROISE=1`) : le chunk
// HIGHLIGHT est relu sous les deux decoupages, et le test signale tout film ou le decoupage qui
// rend le plus de noms distincts CONTREDIT la version lue. Zero contradiction = l indicateur est
// le bon, mesure sur tout le parc.
//
//	KS_BTB2025_PARC=<cache>/film_chunks go test ... -run TestBTB2025ParcVersions -v
func TestBTB2025ParcVersions(t *testing.T) {
	racine := os.Getenv("KS_BTB2025_PARC")
	if racine == "" {
		t.Skip("banc de diagnostic : KS_BTB2025_PARC requis")
	}
	croise := os.Getenv("KS_BTB2025_PARC_CROISE") != ""
	entrees, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture %s : %v", racine, err)
	}
	parVersion := map[int]int{}
	var sansRegistre, contradictions []string
	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		registre, err := os.ReadFile(filepath.Join(racine, e.Name(), "chunk_00.bin"))
		if err != nil {
			sansRegistre = append(sansRegistre, e.Name())
			continue
		}
		version, ok := filmdec.FilmMajorVersionFromHeader(filmsource.Inflate(registre))
		if !ok {
			sansRegistre = append(sansRegistre, e.Name())
			continue
		}
		parVersion[version]++
		if !croise {
			continue
		}
		mesure, mesurable := decoupageMesure(filepath.Join(racine, e.Name()))
		if mesurable && mesure != decoupageDeLaVersion(version) {
			contradictions = append(contradictions, e.Name()+":v"+strconv.Itoa(version))
		}
	}
	t.Logf("PARC     films=%d sans_registre_lisible=%d", len(entrees), len(sansRegistre))
	t.Logf("VERSIONS %s", histogramme(parVersion))
	if croise {
		t.Logf("CROISE   contradictions=%d %s", len(contradictions), strings.Join(contradictions, " "))
	}
}

// decoupageDeLaVersion : l implantation du gamertag qu une version DECLARE.
func decoupageDeLaVersion(version int) int {
	if version <= 38 || version >= 41 {
		return versionGamertagEnTeteTest
	}
	return versionGamertagDecaleTest
}

// decoupageMesure : l implantation qui rend le plus de gamertags distincts sur le chunk HIGHLIGHT.
// C EST UNE MESURE DE CONTROLE, jamais un chemin de production : le decodeur LIT la version, il
// ne la devine pas (decision du 2026-09-12).
func decoupageMesure(dir string) (int, bool) {
	brut, err := dernierChunk(dir)
	if err != nil {
		return 0, false
	}
	evs, err := analysis.ParseHighlightEvents(brut, versionGamertagEnTeteTest)
	if err != nil || len(evs) == 0 {
		return 0, false
	}
	alt, err := analysis.ParseHighlightEvents(brut, versionGamertagDecaleTest)
	if err != nil {
		return 0, false
	}
	_, enTete, _ := statsNoms(evs)
	_, decale, _ := statsNoms(alt)
	if decale > enTete {
		return versionGamertagDecaleTest, true
	}
	return versionGamertagEnTeteTest, true
}

// versionGamertagEnTeteTest / versionGamertagDecaleTest : les deux implantations, nommees ici
// parce que les constantes du paquet `analysis` sont privees.
const (
	versionGamertagEnTeteTest = 41
	versionGamertagDecaleTest = 39
)

// dernierChunk : le chunk de plus haut indice d un repertoire de film — le HIGHLIGHT.
func dernierChunk(dir string) ([]byte, error) {
	noms, err := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	if err != nil || len(noms) == 0 {
		return nil, os.ErrNotExist
	}
	sort.Strings(noms)
	return os.ReadFile(noms[len(noms)-1])
}
