//go:build research

package killsource

// ri27c_research_test.go — LES MESURES QUI OUVRENT 2.7.c (plan de l etape 2 de la representation
// intermediaire, item 2.7.c0), cote killsource. Elles confrontent ce que killsource lit lui-meme a
// ce que la marche unique de la grammaire lit des memes faits, rendu en TSV par
// `grammar/ri27c_killsource_research_test.go` (lancer d abord `TestRI27cMorts`, `TestRI27cCalibration`
// et `TestRI27cVueA` sur les memes films, meme dossier de sortie). Aucun fichier de production n est
// touche.
//
// Par film :
//
//	W  les dead-states `Mort` de la marche de killsource contre ceux de la marche des trames
//	   (decoupage du format), par (horodatage, slot) : communs (valeurs egales ou non), propres a
//	   chaque cote, ranges par classe de la trame cote grammaire ; records propres et a queue rompue ;
//	   bande de killsource contre bande de la phase des images-cles ;
//	R  le resultat de killsource sous chaque variante (A0 telle quelle ; A1 marche des trames, filtre
//	   de bande ; A2 filtre d archetype ; A3 decoupage declare ; A4 la marche de killsource sous le
//	   decoupage declare ; A5 decoupage declare, queues rompues apres le dead-state acceptees) : lignes
//	   publiees, couvertes, candidats ; D les lignes gagnees, perdues ou changees contre A0 ;
//	C  les scores de la calibration de killsource sous sa timeline (la grammaire rend ceux du monde
//	   des preliminaires), et sa calibration retenue ;
//	E  les kill-events de killsource contre les messages de genre 85 de la vue A unique.
//
// Les lignes C, CK, E et la variante A4 se construisent dans `ri27c_calibration_research_test.go`.
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier de la grammaire> \
//	  [RI27C_CARTES=<id=Carte;...>] \
//	  go test -tags=research -count=1 -run '^TestRI27cKillsource$' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/facts/killsource/

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/constat"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27cMort est un dead-state `Mort` lu par l une des deux marches.
type ri27cMort struct {
	ts                 uint64
	chunk, pidx, slot  int
	gen, ti            uint32
	bit, desync, idxDS int
	classe             string
	dead               types.DeadState
}

// ri27cCle identifie un dead-state et sa valeur.
type ri27cCle struct {
	ts                 uint64
	slot               int
	enumA, enumB, val0 int
	tag                uint32
}

func (m ri27cMort) cle() ri27cCle {
	return ri27cCle{m.ts, m.slot, int(m.dead.EnumA), int(m.dead.EnumB), int(m.dead.Val0c), m.dead.SrcTag0}
}

// ri27cGrammaire est ce que l instrument de la grammaire a rendu pour un film.
type ri27cGrammaire struct {
	morts       map[string][]ri27cMort // par decoupage
	classes     map[string]map[uint64]string
	bande       map[string][2]int
	compteurs   map[string][]string
	calib       map[string]int
	kills       map[[2]uint64][]string // (ts, bit du genre) -> champs
	paquetsVA   map[uint64][]string    // ts -> P
	variante    []string
	calibVus    string
	calibTaille string
}

// ri27cLireGrammaire relit les TSV de la grammaire pour un film.
func ri27cLireGrammaire(t *testing.T, dossier, court string) ri27cGrammaire {
	t.Helper()
	g := ri27cGrammaire{morts: map[string][]ri27cMort{}, classes: map[string]map[uint64]string{},
		bande: map[string][2]int{}, compteurs: map[string][]string{}, calib: map[string]int{},
		kills: map[[2]uint64][]string{}, paquetsVA: map[uint64][]string{}}
	for _, nom := range []string{"morts_" + court + ".tsv", "vuea_" + court + ".tsv", "calibration_grammaire.tsv"} {
		for _, l := range ri27cLignes(t, filepath.Join(dossier, nom)) {
			ri27cRangerLigne(&g, court, l)
		}
	}
	return g
}

// ri27cRangerLigne range une ligne TSV de la grammaire.
func ri27cRangerLigne(g *ri27cGrammaire, court string, l []string) {
	if len(l) < 2 || l[1] != court {
		return
	}
	switch l[0] {
	case "M":
		m := ri27cMort{ts: ri27cU(l[5]), chunk: ri27cI(l[3]), pidx: ri27cI(l[4]), slot: ri27cI(l[6]),
			gen: uint32(ri27cU(l[7])), ti: uint32(ri27cU(l[8])), bit: ri27cI(l[9]), desync: ri27cI(l[10]),
			idxDS: ri27cI(l[11]), classe: l[12]}
		m.dead = types.DeadState{Mort: true, EnumA: int32(ri27cI(l[15])), EnumB: int32(ri27cI(l[16])),
			Val0c: uint8(ri27cU(l[17])), SrcTag0: uint32(ri27cU(l[18])), GlobalID: uint32(ri27cU(l[19])),
			Val0e: uint8(ri27cU(l[20]))}
		g.morts[l[2]] = append(g.morts[l[2]], m)
	case "T":
		if g.classes[l[2]] == nil {
			g.classes[l[2]] = map[uint64]string{}
		}
		g.classes[l[2]][ri27cU(l[5])] = l[6] + "/v" + l[7] + "/d" + l[8]
	case "B":
		g.bande[l[2]] = [2]int{ri27cI(l[3]), ri27cI(l[4])}
		g.compteurs[l[2]] = l[5:]
	case "C":
		g.calib[l[3]] = ri27cI(l[4])
		g.calibVus, g.calibTaille = l[5], l[6]
	case "V":
		g.variante = l[2:]
	case "P":
		g.paquetsVA[ri27cU(l[4])] = l[5:]
	case "K":
		g.kills[[2]uint64{ri27cU(l[4]), ri27cU(l[5])}] = l[6:]
	}
}

func ri27cLignes(t *testing.T, chemin string) [][]string {
	t.Helper()
	f, err := os.Open(chemin)
	if err != nil {
		t.Fatalf("TSV de la grammaire : %v", err)
	}
	defer f.Close()
	var out [][]string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if s := sc.Text(); s != "" {
			out = append(out, strings.Split(s, "\t"))
		}
	}
	return out
}

func ri27cI(s string) int    { n, _ := strconv.Atoi(s); return n }
func ri27cU(s string) uint64 { n, _ := strconv.ParseUint(s, 10, 64); return n }

// ri27cMarcheDeKillsource est [runWalk], tous les dead-states `Mort` gardes avec leur archetype et
// leur desynchronisation.
func ri27cMarcheDeKillsource(f *film, tl *timeline, views int, cal *calibration) []ri27cMort {
	tl.rewind()
	cfg := grammar.DefaultFrameConfig()
	cfg.Profil = cal.Profil
	var out []ri27cMort
	for i := range f.t0 {
		p := &f.t0[i]
		w := tl.advanceTo(p.ts)
		start := 2
		if hasEvents(p) {
			s, _ := grammar.DebutDeLaVueB(p.payload, w, cfg, cal.VueA)
			if s < 0 {
				continue
			}
			start = s
		}
		snap := w.Snapshot()
		recs := walkFrom(p.payload, w, cfg, start, views)
		w.Restore(snap)
		for k := range recs {
			r := &recs[k]
			if r.Trace.Dead == nil || !r.Trace.Dead.Mort {
				continue
			}
			out = append(out, ri27cMort{ts: p.ts, chunk: p.chunk, pidx: p.idx, slot: int(r.Slot), gen: r.ID >> 30,
				ti: r.TypeIndex, bit: deadStateBit(r), desync: r.DesyncAt, dead: *r.Trace.Dead})
		}
	}
	return out
}

// ri27cDecoder decode un film comme `Decode`, en gardant l etat de la passe.
func ri27cDecoder(t *testing.T, racine, court string) (*decodeCtx, *source.Film) {
	t.Helper()
	src, err := source.LoadDir(filepath.Join(racine, court), nil)
	if err != nil {
		t.Fatalf("%s : %v", court, err)
	}
	o := DefaultOptions()
	carte := carteDuCatalogue(t, ri27cCarte(t, court))
	o.Carte = &carte
	o.normalize()
	c := &decodeCtx{name: court, opts: o}
	if err := c.prepare(t.Context(), src); err != nil {
		t.Fatalf("%s : prepare : %v", court, err)
	}
	return c, src
}

// ri27cCarte rend la carte d un film : RI27C_CARTES d abord, les faits d equivalence ensuite.
func ri27cCarte(t *testing.T, court string) string {
	t.Helper()
	for _, kv := range strings.Split(os.Getenv("RI27C_CARTES"), ";") {
		if k, v, ok := strings.Cut(kv, "="); ok && k == court {
			return v
		}
	}
	brut, err := os.ReadFile(filepath.Join("..", "..", "..", "replay", "testdata", "equivalence", court+".facts.json"))
	if err != nil {
		t.Fatalf("faits %s : %v", court, err)
	}
	var f struct {
		MapNames []string `json:"mapNames"`
	}
	if err := json.Unmarshal(brut, &f); err != nil || len(f.MapNames) == 0 {
		t.Fatalf("carte de %s introuvable (%v)", court, err)
	}
	return f.MapNames[0]
}

// ri27cMarcheDesTrames convertit les dead-states de la grammaire en marche de killsource : records
// propres (et, `queues`, a queue rompue apres le dead-state), filtre d archetype ou de bande.
func ri27cMarcheDesTrames(c *decodeCtx, morts []ri27cMort, bande [2]int, parArchetype, queues bool) *walkResult {
	pos := map[uint64][2]int{}
	for i := range c.film.t0 {
		p := &c.film.t0[i]
		if _, deja := pos[p.ts]; !deja {
			pos[p.ts] = [2]int{p.chunk, p.idx}
		}
	}
	res := &walkResult{bipLo: bande[0], bipHi: bande[1]}
	for _, m := range morts {
		propre := m.desync == -1
		queue := !propre && m.idxDS >= 0 && m.desync > m.idxDS
		if !propre && !(queues && queue) {
			res.desync++
			continue
		}
		if parArchetype && m.ti != grammar.BipedTypeIndex {
			continue
		}
		cp := pos[m.ts]
		res.deads = append(res.deads, deadRecord{ms: int((m.ts - c.film.tsBase) / 1000), chunk: cp[0], pidx: cp[1],
			slot: m.slot, bit: m.bit, dead: m.dead})
	}
	if parArchetype {
		res.bipLo, res.bipHi = 0, 1<<30
	}
	trierMortsDeLaMarche(res.deads)
	res.selectCredible(c.roster)
	return res
}

// ri27cJSON rend une ligne publiee, encodee.
func ri27cJSON(k Kill) string {
	b, _ := json.Marshal(k)
	return string(b)
}

// ri27cComparer rend les lignes R et D d une variante contre la reference.
func ri27cComparer(court, alt string, ref, res *Result) []string {
	lignes := []string{fmt.Sprintf("R\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d", court, alt, len(res.Kills),
		res.Coverage.Covered, res.Coverage.RealPairs, res.Health.Candidates, res.Stats.Walk.Published,
		res.Stats.Scan.Published)}
	a, b := map[int]string{}, map[int]string{}
	for _, k := range ref.Kills {
		a[k.TimeMS] = ri27cJSON(k)
	}
	for _, k := range res.Kills {
		b[k.TimeMS] = ri27cJSON(k)
	}
	var cles []int
	for k := range a {
		cles = append(cles, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			cles = append(cles, k)
		}
	}
	sort.Ints(cles)
	for _, k := range cles {
		va, oka := a[k]
		vb, okb := b[k]
		switch {
		case oka && !okb:
			lignes = append(lignes, fmt.Sprintf("D\t%s\t%s\t%d\tperdue\t%s\t", court, alt, k, va))
		case !oka && okb:
			lignes = append(lignes, fmt.Sprintf("D\t%s\t%s\t%d\tgagnee\t\t%s", court, alt, k, vb))
		case va != vb:
			lignes = append(lignes, fmt.Sprintf("D\t%s\t%s\t%d\tchangee\t%s\t%s", court, alt, k, va, vb))
		}
	}
	return lignes
}

// ri27cComparerLesMorts rend les lignes W : les dead-states des deux marches, par (horodatage, slot).
func ri27cComparerLesMorts(court string, propres []ri27cMort, g ri27cGrammaire, bande [2]int) []string {
	type bilan struct{ communsEgaux, communsDiff, killsourceSeul, grammaireSeule int }
	parClasse := map[string]*bilan{}
	get := func(c string) *bilan {
		if parClasse[c] == nil {
			parClasse[c] = &bilan{}
		}
		return parClasse[c]
	}
	classeDe := func(ts uint64) string {
		if c, ok := g.classes["format"][ts]; ok {
			return c
		}
		return "sans_evenement"
	}
	cleSlot := func(m ri27cMort) [2]uint64 { return [2]uint64{m.ts, uint64(m.slot)} }
	ks, gr := map[[2]uint64][]ri27cMort{}, map[[2]uint64][]ri27cMort{}
	for _, m := range propres {
		if m.desync == -1 {
			ks[cleSlot(m)] = append(ks[cleSlot(m)], m)
		}
	}
	for _, m := range g.morts["format"] {
		if m.desync == -1 {
			gr[cleSlot(m)] = append(gr[cleSlot(m)], m)
		}
	}
	for k, ms := range ks {
		b := get(classeDe(k[0]))
		if gs, ok := gr[k]; ok {
			if ms[0].cle() == gs[0].cle() {
				b.communsEgaux++
			} else {
				b.communsDiff++
			}
			continue
		}
		b.killsourceSeul++
	}
	for k := range gr {
		if _, ok := ks[k]; !ok {
			get(classeDe(k[0])).grammaireSeule++
		}
	}
	var lignes []string
	for c, b := range parClasse {
		lignes = append(lignes, fmt.Sprintf("W\t%s\t%s\t%d\t%d\t%d\t%d", court, c, b.communsEgaux, b.communsDiff,
			b.killsourceSeul, b.grammaireSeule))
	}
	sort.Strings(lignes)
	gb := g.bande["format"]
	return append(lignes, fmt.Sprintf("WB\t%s\tkillsource=[%d,%d]\tgrammaire=[%d,%d]\t%s", court, bande[0], bande[1],
		gb[0], gb[1], strings.Join(g.compteurs["format"], "\t")))
}

// ri27cFilm mesure un film et rend ses lignes.
func ri27cFilm(t *testing.T, racine, dossier, court string) []string {
	t.Helper()
	c, src := ri27cDecoder(t, racine, court)
	var diag constat.Diagnostics
	tl, err := newTimeline(c.film, &diag)
	if err != nil {
		t.Fatalf("%s : timeline : %v", court, err)
	}
	g := ri27cLireGrammaire(t, dossier, court)
	bande := [2]int{c.walkRes.bipLo, c.walkRes.bipHi}
	propres := ri27cMarcheDeKillsource(c.film, tl, c.opts.Views, &c.calib)
	lignes := ri27cComparerLesMorts(court, propres, g, bande)
	lignes = append(lignes, ri27cCalibration(court, c, tl, g)...)
	lignes = append(lignes, ri27cKillEvents(court, c, g)...)
	ref := c.finish()
	lignes = append(lignes, ri27cComparer(court, "A0", ref, ref)...)
	propre := c.walkRes
	variantes := []struct {
		nom               string
		dec               string
		archetype, queues bool
	}{{"A1", "format", false, false}, {"A2", "format", true, false}, {"A3", "declare", true, false},
		{"A5", "declare", true, true}}
	for _, v := range variantes {
		c.walkRes = ri27cMarcheDesTrames(c, g.morts[v.dec], bande, v.archetype, v.queues)
		lignes = append(lignes, ri27cComparer(court, v.nom, ref, c.finish())...)
	}
	if w4, cal4 := ri27cSousDecoupageDeclare(c, src, tl); w4 != nil {
		calib := c.calib
		c.walkRes, c.calib = w4, *cal4
		lignes = append(lignes, ri27cComparer(court, "A4", ref, c.finish())...)
		lignes = append(lignes, fmt.Sprintf("CK4\t%s\t%s", court, cal4.String()))
		c.calib = calib
	}
	c.walkRes = propre
	return lignes
}

func TestRI27cKillsource(t *testing.T) {
	films, racine, dossier := os.Getenv("RI27C_FILMS"), os.Getenv("RI27C_RACINE"), os.Getenv("RI27C_OUT")
	if films == "" || racine == "" || dossier == "" {
		t.Skip("instrument : RI27C_FILMS, RI27C_RACINE et RI27C_OUT requis")
	}
	var lignes []string
	for _, court := range strings.Split(films, ",") {
		l := ri27cFilm(t, racine, dossier, court)
		lignes = append(lignes, l...)
		t.Logf("%s : %d lignes", court, len(l))
		if err := os.WriteFile(filepath.Join(dossier, "killsource.tsv"), []byte(strings.Join(lignes, "\n")+"\n"),
			0o600); err != nil {
			t.Fatalf("sortie : %v", err)
		}
		runtime.GC()
	}
}
