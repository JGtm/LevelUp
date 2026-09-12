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
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmsource"
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
		b.WriteString(itoa(k) + ":" + itoa(h[k]))
	}
	return b.String()
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [24]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TestBTB2025KillFeedNoms — LE MEME BANC, MAIS SUR LE SEUL CHUNK HIGHLIGHT.
//
// Il isole l etape amont : combien de XUID distincts, combien de gamertags non vides, et ce que
// le MEME chunk rend sous l autre implantation d event (`filmMajorVersion` 39-40, dont le
// gamertag vit a `b[12:44]` et non `b[0:32]`). `killsource.loadKillFeed` passe 0 en dur.
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
		for _, ver := range []int{41, 39} {
			best, nk := meilleurHighlight(f, ver)
			xu, gt, vides := statsNoms(best)
			t.Logf("%-8s version=%2d kills=%3d events=%4d xuid_distincts=%2d gamertags_distincts=%2d gamertags_vides=%4d",
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
// la couverture a 5-17 %. Un retour a la lecture du gamertag << en tete >> sur ces films le
// ferait echouer immediatement.
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
		t.Errorf("roster humain = %d, want >= 20 — le decoupage du gamertag des films de "+
			"version 39-40 n est plus resolu (cf. analysis.scanEvents)", res.Roster.Humans)
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

// TestBTB2025ParcDecoupage — LE PARC ENTIER DU CACHE, pour chiffrer le redecodage.
//
// Il ne charge PAS les films : il lit le seul chunk HIGHLIGHT (le dernier du repertoire, type 3
// au manifeste) et compare les deux decoupages du gamertag. Un film dont le decoupage decale
// rend plus de noms distincts est un film de version 39-40, donc un film dont les lignes
// `match_kill_events` ont ete produites avec un roster effondre.
//
//	KS_BTB2025_PARC=<cache>/film_chunks go test ... -run TestBTB2025ParcDecoupage -v
func TestBTB2025ParcDecoupage(t *testing.T) {
	racine := os.Getenv("KS_BTB2025_PARC")
	if racine == "" {
		t.Skip("banc de diagnostic : KS_BTB2025_PARC requis")
	}
	entrees, err := os.ReadDir(racine)
	if err != nil {
		t.Fatalf("lecture %s : %v", racine, err)
	}
	var decale, enTete, illisible []string
	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		brut, err := dernierChunk(filepath.Join(racine, e.Name()))
		if err != nil {
			illisible = append(illisible, e.Name())
			continue
		}
		evs, err := analysis.ParseHighlightEvents(brut, versionGamertagEnTeteTest)
		if err != nil || len(evs) == 0 {
			illisible = append(illisible, e.Name())
			continue
		}
		alt, err := analysis.ParseHighlightEvents(brut, versionGamertagDecaleTest)
		if err != nil {
			illisible = append(illisible, e.Name())
			continue
		}
		_, a, _ := statsNoms(evs)
		_, b, _ := statsNoms(alt)
		switch {
		case b > a:
			decale = append(decale, e.Name())
		default:
			enTete = append(enTete, e.Name())
		}
	}
	t.Logf("PARC     films=%d decoupage_decale_39_40=%d decoupage_en_tete=%d sans_highlight_lisible=%d",
		len(entrees), len(decale), len(enTete), len(illisible))
	t.Logf("DECALE   %s", strings.Join(decale, " "))
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
