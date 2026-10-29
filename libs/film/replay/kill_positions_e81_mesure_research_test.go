//go:build research

package replay

// kill_positions_e81_mesure_research_test.go — E8.1 (f) : MESURE AVANT / APRES DE `kill_positions`.
//
// Instrument de LECTURE : faits cuits (8 fixtures versionnes et, avec L1_DEPOT, le cache de faits du
// poste), couples du JOURNAL exportes d'une copie a froid de la base (KP_KILLS, TSV film, time_ms,
// tueur, victime) et lignes `kill_positions_latest` de la meme copie (KP_BASE, TSV film, killer_xuid,
// time_ms, killer_x/y/z, victim_x/y/z). AUCUN octet de film n'est decode, rien n'est ecrit.
//
//	avant   la base (ce que le collecteur a ecrit, couples du fil du film resolus par gamertag) ;
//	apres_a les couples du journal (E8.1 a), placement inchange ;
//	apres   apres_a + la regle du tueur posthume (E8.1 d), morts du journal = victimes du TSV et
//	        morts du fil du film a xuid.
//
// Un cote PERDU est un cote place par la reference et nil apres ; l instrument les compte contre la
// base et contre apres_a (ce dernier doit etre nul par construction).
//
//	L1_DEPOT=<checkout> KP_KILLS=<tsv> KP_BASE=<tsv> REG_COLLECTEUR=1 \
//	  go test -tags research -count=1 -run '^TestMesureE81KillPositions$' -v ./internal/games/halo_infinite/film/replay/

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/profile"
)

type ligneBase struct {
	killer, victim *Vec3
}

type cleKP struct {
	film   string
	killer uint64
	tMS    int64
}

func lireBaseKP(t *testing.T) map[cleKP]ligneBase {
	t.Helper()
	out := map[cleKP]ligneBase{}
	p := os.Getenv("KP_BASE")
	if p == "" {
		return out
	}
	f, err := os.Open(p) //nolint:gosec // mesure
	if err != nil {
		t.Fatalf("KP_BASE : %v", err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	vec := func(c []string) *Vec3 {
		if c[0] == "NULL" {
			return nil
		}
		x, _ := strconv.ParseFloat(c[0], 64)
		y, _ := strconv.ParseFloat(c[1], 64)
		z, _ := strconv.ParseFloat(c[2], 64)
		return &Vec3{X: x, Y: y, Z: z}
	}
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) != 9 {
			continue
		}
		k, e1 := strconv.ParseUint(c[1], 10, 64)
		tm, e2 := strconv.ParseInt(c[2], 10, 64)
		if e1 != nil || e2 != nil {
			continue
		}
		out[cleKP{c[0], k, tm}] = ligneBase{killer: vec(c[3:6]), victim: vec(c[6:9])}
	}
	return out
}

// e81Bilan : les comptes d'un corpus.
type e81Bilan struct {
	films, kills                                int
	tueurAbs, victimeAbs, aucune                [3]int // avant, apres_a, apres
	posthumes, posthumesSansLieu                int
	perdusContreBase, perdusContreA, ecartsBase [2]int // tueur, victime
	ecartMaxBase                                float64
	filmsSansBase                               int
	causesTueur, causesVictime                  map[string]int
}

func (b *e81Bilan) mesurer(c *contexteFilm, kills []tsvKill, base map[cleKP]ligneBase, short string) {
	if len(kills) == 0 {
		return
	}
	b.films++
	refs := make([]KillRef, 0, len(kills))
	vus := map[[2]int64]bool{}
	var morts []MortDuJournal
	for _, k := range kills {
		refs = append(refs, KillRef{KillerXUID: k.killer, VictimXUID: k.victim, TimeMS: k.timeMS})
		cle := [2]int64{int64(k.victim), k.timeMS} //nolint:gosec // xuid : mesure
		if !vus[cle] {
			vus[cle] = true
			morts = append(morts, MortDuJournal{VictimeXUID: k.victim, TempsMS: k.timeMS})
		}
	}
	for _, d := range c.g.Deaths {
		cle := [2]int64{int64(d.XUID), d.TimeMS} //nolint:gosec // xuid : mesure
		if d.XUID != 0 && !vus[cle] {
			vus[cle] = true
			morts = append(morts, MortDuJournal{VictimeXUID: d.XUID, TempsMS: d.TimeMS})
		}
	}
	pos := c.g.Positions
	e := EntreeDesPositions{Positions: pos, Registre: c.reg, Kills: refs, OrigineUS: c.origine}
	posA, _ := BuildKillPositions(e)
	e.Morts = morts
	posD, rep := BuildKillPositions(e)
	b.posthumes += rep.Posthumous
	b.posthumesSansLieu += rep.PosthumousUnplaced
	indexer := func(l []KillPosition) map[KillRef]KillPosition {
		m := map[KillRef]KillPosition{}
		for _, p := range l {
			m[p.KillRef] = p
		}
		return m
	}
	ia, id := indexer(posA), indexer(posD)
	aBase := false
	for _, r := range refs {
		if _, ok := base[cleKP{short, r.KillerXUID, r.TimeMS}]; ok {
			aBase = true
			break
		}
	}
	if !aBase {
		b.filmsSansBase++
	}
	for _, r := range refs {
		b.kills++
		lb := base[cleKP{short, r.KillerXUID, r.TimeMS}]
		la, ld := ia[r], id[r]
		b.compter(0, lb.killer, lb.victim)
		b.compter(1, la.Killer, la.Victim)
		b.compter(2, ld.Killer, ld.Victim)
		b.perte(0, lb.killer, ld.Killer, &b.perdusContreBase)
		b.perte(1, lb.victim, ld.Victim, &b.perdusContreBase)
		b.perte(0, la.Killer, ld.Killer, &b.perdusContreA)
		b.perte(1, la.Victim, ld.Victim, &b.perdusContreA)
		b.ecart(0, lb.killer, ld.Killer)
		b.ecart(1, lb.victim, ld.Victim)
		tUS := uint64(r.TimeMS*1000 + c.origine) //nolint:gosec // mesure
		if ld.Killer == nil {
			b.causesTueur[strings.Fields(c.diagnostiquer(r.KillerXUID, tUS, true, r.TimeMS).cause)[0]]++
		}
		if ld.Victim == nil {
			b.causesVictime[strings.Fields(c.diagnostiquer(r.VictimXUID, tUS, false, r.TimeMS).cause)[0]]++
		}
	}
}

func (b *e81Bilan) compter(i int, k, v *Vec3) {
	if k == nil {
		b.tueurAbs[i]++
	}
	if v == nil {
		b.victimeAbs[i]++
	}
	if k == nil && v == nil {
		b.aucune[i]++
	}
}

func (b *e81Bilan) perte(cote int, ref, apres *Vec3, dst *[2]int) {
	if ref != nil && apres == nil {
		dst[cote]++
	}
}

func (b *e81Bilan) ecart(cote int, ref, apres *Vec3) {
	if ref == nil || apres == nil {
		return
	}
	f32 := func(v *Vec3) [3]float32 { return [3]float32{float32(v.X), float32(v.Y), float32(v.Z)} }
	d := dist3(f32(ref), f32(apres))
	if d > 1e-3 {
		b.ecartsBase[cote]++
	}
	b.ecartMaxBase = math.Max(b.ecartMaxBase, d)
}

func (b *e81Bilan) rendre(t *testing.T, titre string) {
	t.Logf("==== %s : %d films (%d sans aucune ligne en base), %d frags du journal", titre, b.films, b.filmsSansBase, b.kills)
	t.Logf("| | avant (base) | apres (a) journal | apres (a)+(d) posthume |")
	t.Logf("| tueur sans position | %d | %d | %d |", b.tueurAbs[0], b.tueurAbs[1], b.tueurAbs[2])
	t.Logf("| victime sans position | %d | %d | %d |", b.victimeAbs[0], b.victimeAbs[1], b.victimeAbs[2])
	t.Logf("| ligne absente (aucun des deux) | %d | %d | %d |", b.aucune[0], b.aucune[1], b.aucune[2])
	t.Logf("tueurs posthumes poses=%d, reconnus morts sans lieu=%d", b.posthumes, b.posthumesSansLieu)
	t.Logf("PERDUS contre la base : tueur=%d victime=%d ; contre apres (a) : tueur=%d victime=%d",
		b.perdusContreBase[0], b.perdusContreBase[1], b.perdusContreA[0], b.perdusContreA[1])
	t.Logf("ecarts > 1 mm contre la base (meme cote place des deux cotes) : tueur=%d victime=%d, max=%.3f m",
		b.ecartsBase[0], b.ecartsBase[1], b.ecartMaxBase)
	rendreCauses(t, "tueur", b.causesTueur)
	rendreCauses(t, "victime", b.causesVictime)
}

func rendreCauses(t *testing.T, qui string, m map[string]int) {
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	var parts []string
	for _, k := range cles {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	t.Logf("  causes restantes %s : %s", qui, strings.Join(parts, " "))
}

func TestMesureE81KillPositions(t *testing.T) {
	kills, base := lireKillsTSV(t), lireBaseKP(t)
	nouveau := func() *e81Bilan { return &e81Bilan{causesTueur: map[string]int{}, causesVictime: map[string]int{}} }
	fix := nouveau()
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		c, ok := preparerFilm(b.Short8, b.Build, g)
		if !ok {
			continue
		}
		fix.mesurer(c, kills[b.Short8], base, b.Short8)
	}
	fix.rendre(t, "CORPUS VERSIONNE")
	depot := os.Getenv("L1_DEPOT")
	if depot == "" {
		return
	}
	loc := nouveau()
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(depot, "data", "titles", "halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	dir := filepath.Join(depot, "data", "cache", "film_facts", "halo_infinite")
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cache de faits : %v", err)
	}
	for _, e := range entrees {
		if !strings.HasSuffix(e.Name(), ".filmfacts.bin") {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // mesure
		if err != nil {
			continue
		}
		head, err := DecodeFilmFactsEntete(blob)
		if err != nil {
			continue
		}
		var entry profile.MapQuantEntry
		for _, m := range cat.Maps {
			if m.Module == head.MapModule {
				entry = m
				break
			}
		}
		ff, err := DecodeFilmFactsFile(blob, entry)
		if err != nil {
			continue
		}
		nom := strings.TrimSuffix(e.Name(), ".filmfacts.bin")
		c, ok := preparerFilm(nom, "", &ff.Facts)
		if !ok {
			continue
		}
		loc.mesurer(c, kills[nom], base, nom)
	}
	loc.rendre(t, "FAITS LOCAUX")
}
