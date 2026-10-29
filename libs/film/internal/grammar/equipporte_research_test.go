//go:build research

package grammar

// equipporte_research_test.go — LOT « EQUIPEMENT PORTE » (plan d ajustements du 2026-10-14, G1) :
// CE QUE LA MARCHE DE PRODUCTION LIT DES OBJETS ti=37 PORTES, ET OU SA VUE B S ARRETE.
//
// Un canal des trames de la marche unique ([Distribuer]) suit chaque vie ti=37 : son NEW, son i10
// (porte « attache » et handle du parent, lus par [decodeObjectParentState] au bit de debut que la
// marche a trace), ses DELTA, son DEL. Pour chaque trame : le verdict de fermeture, la cause de la
// queue opaque (archetype et composant du record d arret), la sortie de la vue B et l eid rejete.
// Une ligne TSV par film dans EQUIPPORTE_OUT/<id>.tsv, et un journal des trames d arret. Aucune
// ecriture ailleurs.
//
//	CGO_ENABLED=0 EQUIPPORTE_FILMS=<repo>/data/cache/film_chunks EQUIPPORTE_ID=1cd3848a \
//	  EQUIPPORTE_OUT=<scratchpad>/g1 go test -tags=research \
//	  ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestEquipPorteG1$' -count=1 -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
	"github.com/JGtm/LevelUp/libs/film/types"
)

// epVie est une vie ti=37 suivie par le canal.
type epVie struct {
	gen          uint32
	neNS         uint64
	attacheAuNEW bool
	parent       uint32 // handle brut du parent (Quant16) au dernier i10 attache
	attache      bool
	deltasPorte  int
	deltasLibre  int
	i10Delta     int
	comps        map[uint8]int // composants vus en DELTA pendant le port
}

type epCanal struct {
	t        *testing.T
	id       string
	origine  uint64
	reg      *Registry
	vies     map[uint32]*epVie
	slots37  map[uint32]bool // tout slot ayant porte un ti=37 dans le film
	trames   int
	fermees  int
	porte    int // trames pendant lesquelles au moins une vie ti=37 attachee existe
	porteFer int
	causes   map[string]int // cause d arret, toutes trames non fermees
	causesP  map[string]int // cause d arret, trames de port
	deltas37 int
	news37   int
	newsAtt  int
	rejets37 int // rejets hors datum portant sur un slot ti=37 connu
	journal  *os.File
	lo, hi   float64
	suivis   map[uint32]bool
	tout37   bool
	// arretsAvant37 : trames de port dont la vue B s est arretee, ventilees par cause, quand une
	// vie attachee existe ET que la trame suivante fermee lit un delta d elle (indice faible).
	deltasParTrameFermee  int
	compsPorte, compsTous map[uint8]int
	deltasPorteTot        int
	trI28, trI28Fer       int
}

func (c *epCanal) Interets() []Interet                      { return nil }
func (c *epCanal) Clore(BilanDeMarche)                      {}
func (c *epCanal) Brancher(*Observation, *MarcheDistribuee) {}

func (c *epCanal) nomComposant(ti int16, idx int16) string {
	if c.reg == nil || ti < 0 || idx < 0 {
		return "?"
	}
	a, ok := c.reg.Archetype(int(ti))
	if !ok || int(idx) >= len(a.Components) {
		return "?"
	}
	return a.Components[idx]
}

func (c *epCanal) cause(p *lecture.Paquet) string {
	q := p.Fermeture.Queue
	switch q.Cause {
	case lecture.CauseComposantNonPorte:
		if q.Record >= 0 && int(q.Record) < len(p.Records) {
			r := p.Records[q.Record]
			g := map[lecture.Genre]string{lecture.GenreNeuf: "NEW", lecture.GenreDelta: "DELTA"}[r.Genre]
			return fmt.Sprintf("composant ti=%d i%d %s (%s)", r.TI, q.Composant, c.nomComposant(r.TI, q.Composant), g)
		}
		return "composant ?"
	case lecture.CauseAucune:
		return fmt.Sprintf("aucune/sortieB=%d", p.VueB.Sortie)
	}
	return fmt.Sprintf("cause=%d/sortieB=%d", q.Cause, p.VueB.Sortie)
}

func (c *epCanal) Trame(p *lecture.Paquet) {
	c.trames++
	ferme := p.Fermeture.Verdict == lecture.VerdictFerme
	if ferme {
		c.fermees++
	}
	for _, r := range p.Records {
		if r.TI != int16(EquipmentTypeIndex) {
			continue
		}
		avec := false
		for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
			avec = avec || k.Index == 28
		}
		if avec {
			c.trI28++
			if ferme {
				c.trI28Fer++
			}
			break
		}
	}
	enPort := false
	for _, v := range c.vies {
		if v.attache {
			enPort = true
			break
		}
	}
	if enPort {
		c.porte++
		if ferme {
			c.porteFer++
		}
	}
	ms := float64(p.TS-c.origine) / 1000
	if c.journal != nil && ms >= c.lo && ms <= c.hi {
		etat := "FERMEE"
		if !ferme {
			etat = "ARRET " + c.cause(p)
		}
		fmt.Fprintf(c.journal, "%s\t%.1f\tTRAME\t%s records %d\n", c.id, ms, etat, len(p.Records))
	}
	for _, r := range p.Records {
		tout37 := c.tout37 && r.TI == int16(EquipmentTypeIndex) && ms >= c.lo && ms <= c.hi
		if (c.suivis[r.Vie.Slot] || tout37) && c.journal != nil {
			var b strings.Builder
			for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
				fmt.Fprintf(&b, " i%d/%d", k.Index, k.Bits)
				if k.Bits > 0 && k.Bits <= 32 {
					fmt.Fprintf(&b, "=%x", epBits(p.Payload, int(k.Debut), int(k.Bits)))
				}
			}
			fmt.Fprintf(c.journal, "%s\t%.1f\tSUIVI\tti=%d slot %d gen %d genre %d desync %d%s\n", c.id, ms, r.TI,
				r.Vie.Slot, r.Vie.Gen, r.Genre, r.Desync, b.String())
		}
	}
	if !ferme {
		k := c.cause(p)
		if p.VueB.Sortie == lecture.SortieRejetHorsDatum {
			if c.slots37[p.VueB.EIDRejete&0x3fffffff] {
				c.rejets37++
				k += " [slot ti=37 connu]"
			}
		}
		c.causes[k]++
		if enPort {
			c.causesP[k]++
		}
	}
	for _, r := range p.Records {
		if r.TI != int16(EquipmentTypeIndex) {
			continue
		}
		slot := r.Vie.Slot
		comps := p.Comps[r.Comps[0]:r.Comps[1]]
		switch r.Genre {
		case lecture.GenreNeuf:
			c.news37++
			c.slots37[slot] = true
			v := &epVie{gen: r.Vie.Gen, neNS: p.TS, comps: map[uint8]int{}}
			if st, ok := c.i10(p, comps); ok {
				v.attacheAuNEW, v.attache, v.parent = st.Attached, st.Attached, st.Quant16
				if st.Attached {
					c.newsAtt++
				}
			}
			c.vies[slot] = v
			if v.attache && c.journal != nil {
				var nb strings.Builder
				for _, k := range comps {
					if k.Bits <= 64 {
						fmt.Fprintf(&nb, " i%d/%d=%s", k.Index, k.Bits, epHex(p.Payload, int(k.Debut), int(k.Bits)))
					}
				}
				fmt.Fprintf(c.journal, "%s\t%.1f\tNEW_ATTACHE\tslot %d gen %d porteur %d%s\n", c.id,
					float64(p.TS-c.origine)/1000, slot, r.Vie.Gen, epPorteur(v.parent), nb.String())
			}
		case lecture.GenreDelta:
			c.deltas37++
			for _, k := range comps {
				c.compsTous[k.Index]++
			}
			v := c.vies[slot]
			if v == nil {
				continue
			}
			if st, ok := c.i10(p, comps); ok {
				v.i10Delta++
				if c.journal != nil && v.attache != st.Attached {
					fmt.Fprintf(c.journal, "%s\t%.1f\tI10\tslot %d attache %v -> %v parent %d\n", c.id,
						float64(p.TS-c.origine)/1000, slot, v.attache, st.Attached, st.Quant16&0x1fff)
				}
				if v.attache {
					v.deltasPorte++
				}
				v.attache = st.Attached
				if st.Attached {
					v.parent = st.Quant16
				}
			} else if v.attache {
				v.deltasPorte++
			} else {
				v.deltasLibre++
			}
			if v.attache {
				for _, k := range comps {
					v.comps[k.Index]++
					c.compsPorte[k.Index]++
				}
				if c.journal != nil {
					var b strings.Builder
					for _, k := range comps {
						fmt.Fprintf(&b, " i%d/%d=%s", k.Index, k.Bits, epHex(p.Payload, int(k.Debut), int(k.Bits)))
					}
					fmt.Fprintf(c.journal, "%s\t%.1f\tDELTA_PORTE\tslot %d porteur %d%s\n", c.id, float64(p.TS-c.origine)/1000,
						slot, epPorteur(v.parent), b.String())
				}
			}
		case lecture.GenreSuppression:
			delete(c.vies, slot)
		}
	}
}

// i10 relit l occurrence i10 d un record ti=37 a son bit de debut.
func (c *epCanal) i10(p *lecture.Paquet, comps []lecture.Composant) (ObjectParentState, bool) {
	for _, k := range comps {
		if k.Index != 10 || k.Bits == 0 {
			continue
		}
		br := LecteurSur(p.Payload)
		br.SetBitPos(int(k.Debut))
		st := decodeObjectParentState(br, 3, uint32(EquipmentTypeIndex))
		return st, true
	}
	return ObjectParentState{}, false
}

func TestEquipPorteG1(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil && fc == nil {
		t.Fatalf("%s : %v", id, err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("%s : registre : %v", id, err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	c := &epCanal{t: t, id: id, origine: origine, reg: reg, vies: map[uint32]*epVie{}, slots37: map[uint32]bool{},
		causes: map[string]int{}, causesP: map[string]int{}, compsPorte: map[uint8]int{}, compsTous: map[uint8]int{}}
	epEnvFenetre(c)
	if os.Getenv("EQUIPPORTE_JOURNAL") != "" {
		f, err := os.Create(filepath.Join(out, id+".journal.tsv"))
		if err == nil {
			c.journal = f
			defer f.Close()
		}
	}
	if err := Distribuer(fc, c); err != nil {
		t.Fatalf("%s : marche : %v", id, err)
	}
	if c.journal != nil {
		fc2, _ := epContexte(filepath.Join(racine, id))
		ch, _, err := ScanEquipmentChanges(fc2, nil)
		if err != nil {
			t.Logf("%s : i48 : %v", id, err)
		}
		for _, e := range ch {
			fmt.Fprintf(c.journal, "%s\t%.1f\tI48\tslot %d kind %s rang %d avant %d\n", id,
				float64(e.TimestampUS-origine)/1000, e.Slot, e.Kind, e.Rank, e.Previous)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "film\t%s\ttrames\t%d\tfermees\t%d\tport\t%d\tport_fermees\t%d\tnew37\t%d\tnew37_attache\t%d\tdelta37\t%d\trejets_slot37\t%d\ttrames_i28\t%d\ttrames_i28_fermees\t%d\n",
		id, c.trames, c.fermees, c.porte, c.porteFer, c.news37, c.newsAtt, c.deltas37, c.rejets37, c.trI28, c.trI28Fer)
	type kv struct {
		k string
		n int
	}
	trier := func(m map[string]int) []kv {
		var l []kv
		for k, n := range m {
			l = append(l, kv{k, n})
		}
		sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n || (l[i].n == l[j].n && l[i].k < l[j].k) })
		return l
	}
	for i := 0; i < 32; i++ {
		if c.compsTous[uint8(i)] > 0 || c.compsPorte[uint8(i)] > 0 {
			fmt.Fprintf(&b, "comp\ti%d\t%d\t%d\n", i, c.compsTous[uint8(i)], c.compsPorte[uint8(i)])
		}
	}
	for _, e := range trier(c.causes) {
		fmt.Fprintf(&b, "cause\t%s\t%d\t%d\n", e.k, e.n, c.causesP[e.k])
	}
	if err := os.WriteFile(filepath.Join(out, id+".tsv"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	t.Log(b.String())
}

// epBits lit `n` bits a `at`, ou 1<<63 hors du payload.
func epBits(pay []byte, at, n int) uint64 {
	if at < 0 || at+n > len(pay)*8 {
		return 1 << 63
	}
	return source.BitsStricts(pay, at, n)
}

// epEnvFenetre lit EQUIPPORTE_FENETRE (« lo,hi » en ms de film) et EQUIPPORTE_SLOTS.
func epEnvFenetre(c *epCanal) {
	c.lo, c.hi = -1, -1
	if f := os.Getenv("EQUIPPORTE_FENETRE"); f != "" {
		fmt.Sscanf(f, "%f,%f", &c.lo, &c.hi)
	}
	c.suivis = map[uint32]bool{}
	c.tout37 = os.Getenv("EQUIPPORTE_TOUT37") != ""
	for _, s := range strings.Split(os.Getenv("EQUIPPORTE_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscan(s, &v); err == nil {
			c.suivis[v] = true
		}
	}
}

// epHex rend les `n` bits a `at` en binaire MSB d abord (n <= 64), « ? » hors du payload.
func epHex(pay []byte, at, n int) string {
	if n <= 0 || n > 64 || at < 0 || at+n > len(pay)*8 {
		return "?"
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte('0' + byte(source.BitsStricts(pay, at+i, 1)))
	}
	return b.String()
}

// epPorteur rend le slot du bipede parent d un handle i10 lu sous la sonde (valeur de 9 bits : la
// bande des bipedes commence a 512), ou la valeur brute de 13 bits.
func epPorteur(parent uint32) uint32 {
	v := parent & 0x1fff
	if v < 512 {
		return 512 + v
	}
	return v
}

// epContexte ouvre le film comme la cuisson quand EQUIPPORTE_CARTE nomme sa carte : entree de
// catalogue, puis carte et decoupage poses (`replay/build_from_film.go`). Sans carte : le contexte
// des instruments ([ContexteDeFilm]), qui n est PAS celui de la cuisson.
func epContexte(dir string) (*FilmContext, error) {
	carte := os.Getenv("EQUIPPORTE_CARTE")
	if carte == "" {
		fc, _, err := ContexteDeFilm(dir)
		if fc == nil {
			return nil, err
		}
		return fc, nil
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		return nil, err
	}
	// Les candidats s essaient dans l ordre, comme `replaybuild.ResolveMapEntry`.
	var entree profile.MapQuantEntry
	for _, c := range strings.Split(carte, "|") {
		if entree, err = cat.Lookup(c); err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	fc := NewFilmContextForMap(film, &entree, DefaultScanFilmOptions().Layout)
	fc.PoserLaCarteEtLeDecoupage()
	return fc, nil
}

// TestEquipPorteScan journalise [ScanEquipementPorte] (le canal de production) sur un film :
// EQUIPPORTE_FILMS, EQUIPPORTE_ID, EQUIPPORTE_CARTE.
func TestEquipPorteScan(t *testing.T) {
	racine, id := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID")
	if racine == "" || id == "" {
		t.Skip("EQUIPPORTE_FILMS ou EQUIPPORTE_ID absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	ls, st, err := ScanEquipementPorte(fc, nil)
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	t.Logf("%s : %d lectures, %+v", id, len(ls), st)
	for _, l := range ls {
		if l.Kind == types.CarriedNew && !l.HasEnergy {
			continue
		}
		t.Logf("%s %9.1f %-5s objet %d porteur %d energie %v/%d activation %v signal %v/%d poignees %v/%d", id,
			float64(l.TimestampUS-origine)/1000, l.Kind, l.Object.Slot, l.Carrier, l.HasEnergy, l.Energy,
			l.HasActivation, l.HasSignal, l.Signal, l.HasHandles, l.Handles)
	}
}
