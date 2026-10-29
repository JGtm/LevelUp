//go:build research

package grammar

// aj14f_faille_research_test.go — LOT F2 DU PLAN D AJUSTEMENTS DU 2026-10-14 : LES RECORDS ti=37
// DE LA MARCHE DE PRODUCTION AUTOUR D UNE POSE DE FAILLE.
//
// Un canal des trames de la marche unique ([Distribuer]) recueille, dans une fenetre de temps de
// film, chaque record d archetype 37 que la marche a lu : slot, generation, genre (NEW / DELTA /
// DEL), composants presents, et les valeurs a un bit des composants `item-at-rest-component` (i18)
// et `equipment-deployed-component` (i20), lues au bit de debut que la marche a trace ; la porte
// et la valeur d `equipment-activated-component` (i21). Aucune ecriture.
//
//	CGO_ENABLED=0 AJ14F_FILMS=<repo>/data/cache AJ14F_IDS='1b2d9e08=Dynasty' AJ14F_FENETRE=150000,190000 \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestAj14fRecordsTi37$' -count=1 -timeout 60m -v

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/filmcache"
	"github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"
	"github.com/JGtm/LevelUp/libs/film/internal/profile"
	"github.com/JGtm/LevelUp/libs/film/internal/source"
)

type aj14fCanalTi37 struct {
	t        *testing.T
	id       string
	origine  uint64
	lo, hi   float64
	m        *MarcheDistribuee
	slots    map[uint32]bool
	lignes   int
	nonLues  int
	horsMain int
}

func (c *aj14fCanalTi37) Interets() []Interet {
	var out []Interet
	for _, n := range []string{"item-at-rest-component", compEquipmentDeployed, compEquipmentActivated,
		compEquipmentCreator, compEquipmentEnergy, compEquipmentEnergyDelay, compEquipmentCharges} {
		out = append(out, Interet{Phase: PhaseImagesCles, TI: EquipmentTypeIndex, Composant: n})
	}
	for _, n := range []string{abilityEnergyName, abilityEnergyNameAlt} {
		out = append(out, Interet{Phase: PhaseImagesCles, TI: BipedTypeIndex, Composant: n})
	}
	return out
}

// ImageCle journalise l etat complet des objets ti=37 suivis : les champs d equipement par index.
func (c *aj14fCanalTi37) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	ms := float64(p.TS-c.origine) / 1000
	if ms < c.lo || ms > c.hi {
		return
	}
	larg := map[uint8]int{18: 1, 20: 1, 23: 6, 24: 14, 26: 10, 27: 8}
	for _, r := range p.Records {
		if r.TI == BipedTypeIndex && c.slots[r.Vie.Slot] {
			for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
				if k.Index == 56 && k.Bits > 0 {
					m := aj14fBits(p.Payload, int(k.Debut), 3)
					v := []uint64{}
					for i := 0; i < int(k.Bits-3)/7; i++ {
						v = append(v, aj14fBits(p.Payload, int(k.Debut)+3+7*i, 7))
					}
					c.t.Logf("%s %9.1f ms IMAGE-CLE BIPEDE slot %d i56 masque %03b valeurs %v (%d bits)", c.id, ms,
						r.Vie.Slot, m, v, k.Bits)
				}
			}
		}
		if r.TI != EquipmentTypeIndex || (len(c.slots) > 0 && !c.slots[r.Vie.Slot]) {
			continue
		}
		var b strings.Builder
		for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
			if k.Index == 10 && k.Bits > 0 {
				fmt.Fprintf(&b, " i10.attache=%d", aj14fBits(p.Payload, int(k.Debut), 1))
			}
			if k.Index == 0 && k.Bits > 0 {
				b.WriteString(" i0=" + aj14fPositionMonde(p.Payload, int(k.Debut)))
			}
			if w, ok := larg[k.Index]; ok && k.Bits > 0 {
				fmt.Fprintf(&b, " i%d=%d/%db", k.Index, aj14fBits(p.Payload, int(k.Debut), min(w, int(k.Bits))), k.Bits)
			}
		}
		c.t.Logf("%s %9.1f ms IMAGE-CLE slot %d desync %d%s", c.id, ms, r.Vie.Slot, r.Desync, b.String())
	}
}
func (c *aj14fCanalTi37) Clore(BilanDeMarche)                          {}
func (c *aj14fCanalTi37) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

func (c *aj14fCanalTi37) Trame(p *lecture.Paquet) {
	ms := float64(p.TS-c.origine) / 1000
	if ms < c.lo || ms > c.hi {
		return
	}
	recs, lus := c.m.recordsDeLaTrame()
	if !lus {
		c.nonLues++
		return
	}
	for i := range recs {
		r := &recs[i]
		if r.TypeIndex == BipedTypeIndex && c.slots[r.Slot] && os.Getenv("AJ14F_BIPEDES") != "" {
			var b strings.Builder
			for _, k := range r.Trace.Comps {
				fmt.Fprintf(&b, " i%d", k.Index)
				if k.Index >= 56 && k.Index <= 59 {
					fmt.Fprintf(&b, "@%x", aj14fBits(p.Payload, k.StartBit, 16))
				}
			}
			c.t.Logf("%s %9.1f ms BIPEDE slot %d desync %d comps%s", c.id, ms, r.Slot, r.DesyncAt, b.String())
		}
		if r.TypeIndex != EquipmentTypeIndex {
			continue
		}
		if len(c.slots) > 0 && !c.slots[r.Slot] {
			continue
		}
		var b strings.Builder
		for _, k := range r.Trace.Comps {
			fmt.Fprintf(&b, " i%d", k.Index)
			switch k.Name {
			case "item-at-rest-component", compEquipmentDeployed:
				fmt.Fprintf(&b, "=%d", int(aj14fBits(p.Payload, k.StartBit, 1)))
			case compEquipmentActivated:
				if int(aj14fBits(p.Payload, k.StartBit, 1)) == 0 {
					fmt.Fprintf(&b, "=v%d", aj14fBits(p.Payload, k.StartBit+1, 3))
				} else {
					b.WriteString("=porte1")
				}
			case compEquipmentCreator:
				if int(aj14fBits(p.Payload, k.StartBit, 1)) == 0 {
					fmt.Fprintf(&b, "=c%d", aj14fBits(p.Payload, k.StartBit+1, 5))
				}
			case compEquipmentEnergy:
				fmt.Fprintf(&b, "=e%d", aj14fBits(p.Payload, k.StartBit, 14))
			case compEquipmentEnergyDelay:
				fmt.Fprintf(&b, "=d%d", aj14fBits(p.Payload, k.StartBit, 10))
			case compEquipmentCharges:
				fmt.Fprintf(&b, "=n%d", aj14fBits(p.Payload, k.StartBit, 8))
			}
			if !k.Ported {
				b.WriteString("(non porte)")
			}
		}
		genre := map[int]string{recNew: "NEW", recDel: "DEL", recDelta: "DELTA"}[r.Type]
		c.t.Logf("%s %9.1f ms slot %d gen %d %-5s desync %d comps%s", c.id, ms, r.Slot, r.ID>>30, genre,
			r.DesyncAt, b.String())
		c.lignes++
	}
}

func TestAj14fRecordsTi37(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	lo, hi, ok := aj14fFenetre()
	if racine == "" || liste == "" || !ok {
		t.Skip("AJ14F_FILMS, AJ14F_IDS ou AJ14F_FENETRE absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	slots := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv("AJ14F_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscan(s, &v); err == nil {
			slots[v] = true
		}
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		origine, err := ScanClockOrigin(film)
		if err != nil {
			t.Fatalf("%s : origine : %v", id, err)
		}
		aj14fEntree = &entree
		c := &aj14fCanalTi37{t: t, id: id, origine: origine, lo: lo, hi: hi, slots: slots}
		if err := Distribuer(aj14fContexte(film, &entree), c); err != nil {
			t.Fatalf("%s : marche : %v", id, err)
		}
		t.Logf("%s : %d records ti=37 dans la fenetre ; %d trames non lues", id, c.lignes, c.nonLues)
	}
}

// TestAj14fPosesTi37 rend les poses (vies ti=37 creees) et les creations du film dans la fenetre,
// avec le GlobalID de leur `eqip` et leur position.
func TestAj14fPosesTi37(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	lo, hi, ok := aj14fFenetre()
	if racine == "" || liste == "" || !ok {
		t.Skip("AJ14F_FILMS, AJ14F_IDS ou AJ14F_FENETRE absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	for _, couple := range strings.Split(liste, ";") {
		id, carte, _ := strings.Cut(couple, "=")
		entree, err := cat.Lookup(carte)
		if err != nil {
			t.Fatalf("%s : carte %q : %v", id, carte, err)
		}
		film, ok, err := filmcache.LoadFilm(racine, id)
		if err != nil || !ok {
			t.Fatalf("film %s : %v (present %v)", id, err, ok)
		}
		origine, _ := ScanClockOrigin(film)
		fc := aj14fContexte(film, &entree)
		wr := entree.Range()
		pl, st, err := ScanEquipmentPlacements(fc, &wr)
		t.Logf("%s : poses %d (err %v) stats %+v", id, len(pl), err, st.Lives)
		for _, p := range pl {
			ms := float64(p.T0US-origine) / 1000
			if ms >= lo && ms <= hi {
				t.Logf("%s POSE %9.1f ms -> %9.1f ms vie %d/%d id 0x%08x (%.2f, %.2f, %.2f) points %d", id, ms,
					float64(p.T1US-origine)/1000, p.Life.Slot, p.Life.Gen, p.GlobalID, p.X, p.Y, p.Z, p.Points)
			}
		}
		cr, _, err := ScanEquipmentCreations(aj14fContexte(film, &entree), &wr)
		t.Logf("%s : creations %d (err %v)", id, len(cr), err)
		for _, c := range cr {
			ms := float64(c.TimestampUS-origine) / 1000
			if ms >= lo && ms <= hi {
				t.Logf("%s CREATION %9.1f ms vie %d/%d mpp %v %x (%.2f, %.2f, %.2f) masque %v", id, ms, c.Slot,
					c.Gen, c.MPPPresent, c.MPPVal, c.X, c.Y, c.Z, c.Mask)
			}
		}
	}
}

// aj14fBits lit `n` bits a `at`, ou 255 hors du payload.
func aj14fBits(pay []byte, at, n int) uint64 {
	if at < 0 || at+n > len(pay)*8 {
		return 255
	}
	return source.BitsStricts(pay, at, n)
}

// TestAj14fComposants imprime les composants des archetypes bipede (35) et equipement (37) du film.
func TestAj14fComposants(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	if racine == "" || liste == "" {
		t.Skip("AJ14F_FILMS ou AJ14F_IDS absent : instrument saute")
	}
	id, _, _ := strings.Cut(liste, "=")
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film %s : %v", id, err)
	}
	fc := NewFilmContext(film)
	for _, ti := range []int{BipedTypeIndex, EquipmentTypeIndex} {
		a, _, ok, err := fc.archetype(ti)
		if err != nil || !ok {
			t.Fatalf("ti %d : %v", ti, err)
		}
		for i, n := range a.Components {
			t.Logf("ti=%d i%d %s", ti, i, n)
		}
	}
}

// TestAj14fEquipementDUnite imprime les emissions d i26 `unit-equipment-component` des slots suivis.
func TestAj14fEquipementDUnite(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	if racine == "" || liste == "" {
		t.Skip("AJ14F_FILMS ou AJ14F_IDS absent : instrument saute")
	}
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	id, carte, _ := strings.Cut(liste, "=")
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte : %v", err)
	}
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film %s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(film)
	em, err := ScanUnitEquipment(aj14fContexte(film, &entree))
	if err != nil {
		t.Fatalf("i26 : %v", err)
	}
	slots := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv("AJ14F_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscan(s, &v); err == nil {
			slots[v] = true
		}
	}
	for _, e := range em {
		if len(slots) == 0 || slots[e.Slot] {
			t.Logf("%s %9.1f ms i26 slot %d tete %d entrees %+v", id, float64(e.TimestampUS-origine)/1000, e.Slot,
				e.Read.Head, e.Read.Entries)
		}
	}
}

// TestAj14fEtatEquipement rend les echantillons d etat (i20, i21, i23, i24, i26, i27) du balayage
// [ScanEquipmentState] pour les slots suivis, et le recensement des masques ti=37 du film.
func TestAj14fEtatEquipement(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	if racine == "" || liste == "" {
		t.Skip("AJ14F_FILMS ou AJ14F_IDS absent : instrument saute")
	}
	id, carte, _ := strings.Cut(liste, "=")
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue : %v", err)
	}
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte : %v", err)
	}
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film %s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(film)
	ech, st, err := ScanEquipmentState(aj14fContexte(film, &entree))
	if err != nil {
		t.Fatalf("etat : %v", err)
	}
	t.Logf("%s : records %d, avec champ %d, marches %d, cassees %d, recensement %v", id, st.Records, st.WithAny,
		st.Walked, st.Broken, st.MaskCensus)
	slots := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv("AJ14F_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscan(s, &v); err == nil {
			slots[v] = true
		}
	}
	for _, e := range ech {
		if len(slots) > 0 && !slots[e.Slot] {
			continue
		}
		var b strings.Builder
		for f := range EquipmentFieldCount {
			if e.Seen[f] {
				fmt.Fprintf(&b, " %s=%v/%d", EquipmentField(f), e.Present[f], e.Val[f])
			}
		}
		t.Logf("%s %9.1f ms ETAT slot %d gen %d%s", id, float64(e.TimestampUS-origine)/1000, e.Slot, e.Gen, b.String())
	}
}

// aj14fCanalIR lit les records de la STRUCTURE (lecture.Paquet.Records) des trames : archetype,
// slot, genre, composants (index, etat, debut, longueur) pour les slots suivis.
type aj14fCanalIR struct {
	t       *testing.T
	id      string
	origine uint64
	lo, hi  float64
	slots   map[uint32]bool
	tis     map[int16]bool
}

func (c *aj14fCanalIR) Interets() []Interet                      { return nil }
func (c *aj14fCanalIR) Clore(BilanDeMarche)                      {}
func (c *aj14fCanalIR) Brancher(*Observation, *MarcheDistribuee) {}
func (c *aj14fCanalIR) Trame(p *lecture.Paquet) {
	ms := float64(p.TS-c.origine) / 1000
	if ms < c.lo || ms > c.hi {
		return
	}
	for _, r := range p.Records {
		if !c.tis[r.TI] || (len(c.slots) > 0 && !c.slots[r.Vie.Slot]) {
			continue
		}
		var b strings.Builder
		for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
			fmt.Fprintf(&b, " i%d", k.Index)
			if k.Bits > 0 && k.Bits <= 32 {
				fmt.Fprintf(&b, "=%x/%d", aj14fBits(p.Payload, int(k.Debut), int(k.Bits)), k.Bits)
			} else if k.Bits > 32 {
				fmt.Fprintf(&b, "=(%d)", k.Bits)
			}
		}
		c.t.Logf("%s %9.1f ms IR ti=%d slot %d genre %d desync %d%s", c.id, ms, r.TI, r.Vie.Slot, r.Genre, r.Desync,
			b.String())
	}
}

// TestAj14fStructure journalise les records de la structure des trames (AJ14F_TIS, AJ14F_SLOTS).
func TestAj14fStructure(t *testing.T) {
	racine, liste := os.Getenv("AJ14F_FILMS"), os.Getenv("AJ14F_IDS")
	lo, hi, ok := aj14fFenetre()
	if racine == "" || liste == "" || !ok {
		t.Skip("instrument saute")
	}
	cat, _ := profile.LoadMapQuantCatalog(e191bCatalogue())
	id, carte, _ := strings.Cut(liste, "=")
	entree, err := cat.Lookup(carte)
	if err != nil {
		t.Fatalf("carte : %v", err)
	}
	film, ok, err := filmcache.LoadFilm(racine, id)
	if err != nil || !ok {
		t.Fatalf("film : %v", err)
	}
	origine, _ := ScanClockOrigin(film)
	c := &aj14fCanalIR{t: t, id: id, origine: origine, lo: lo, hi: hi, slots: map[uint32]bool{}, tis: map[int16]bool{}}
	for _, s := range strings.Split(os.Getenv("AJ14F_SLOTS"), ",") {
		var v uint32
		if _, err := fmt.Sscan(s, &v); err == nil {
			c.slots[v] = true
		}
	}
	for _, s := range strings.Split(os.Getenv("AJ14F_TIS"), ",") {
		var v int16
		if _, err := fmt.Sscan(s, &v); err == nil {
			c.tis[v] = true
		}
	}
	if err := Distribuer(aj14fContexte(film, &entree), c); err != nil {
		t.Fatalf("marche : %v", err)
	}
}

// aj14fPositionMonde decode une position d objet du monde (consumeObjectPositionMonde) sous l entree
// de catalogue du film : precHigh, porte d index, trois axes. « ? » hors de ce que l entree sait lire.
func aj14fPositionMonde(pay []byte, at int) string {
	e := aj14fEntree
	if e == nil || aj14fBits(pay, at, 1) != 0 {
		return "?"
	}
	at++
	axes := e.AxisWidths
	if aj14fBits(pay, at, 1) == 0 {
		at++
		idx := aj14fBits(pay, at, int(e.EffectiveRegionIndexBits()))
		at += int(e.EffectiveRegionIndexBits())
		if uint32(idx) != e.Region {
			return fmt.Sprintf("region%d", idx)
		}
	} else {
		return "defaut"
	}
	var out [3]float32
	for a := range 3 {
		q := aj14fBits(pay, at, int(axes[a]))
		at += int(axes[a])
		out[a] = DequantBipedAxis(uint32(q), a, profile.I0Layout{AxisW: axes}, e.Range())
	}
	return fmt.Sprintf("(%.2f,%.2f,%.2f)", out[0], out[1], out[2])
}
