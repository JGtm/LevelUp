//go:build research

package grammar

// champ_fin_research_test.go — LA FIN DU CHAMP DE REPARATION EST-ELLE ECRITE DANS LE FILM ?
// (demande du user, 2026-10-15 : « si on a l evenement, on rebranche le son de fin »).
//
// Une marche de production porte deux canaux : celui de l equipement porte (production) et un canal
// qui note la naissance et la suppression de toute vie. Pour chaque objet que nomme une pile i28, une
// ligne TSV : son archetype, s il nait dans le paquet de la pile, sa duree de vie. Lecture seule, un
// film par processus (mesure de `grammar-2026-10-15`, `rev_chronique.go`).
//
//	CGO_ENABLED=0 EQUIPPORTE_FILMS=<repo>/data/cache/film_chunks EQUIPPORTE_ID=5676a9ba \
//	  EQUIPPORTE_CARTE=Insolence EQUIPPORTE_OUT=<scratchpad>/suivis go test -tags=research \
//	  ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestChampParc$' -count=1 -v

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

type cfVie struct {
	pos       [3]float32
	aPos      bool
	ti        int16
	slot, gen uint32
	ne, fin   uint64
	attache   bool
	porteur   uint32
}

type cfCanal struct {
	fc      *FilmContext
	posTI   int16
	wr      profile.Vec3Range
	vies    []*cfVie
	actives map[uint32]*cfVie
}

func (c *cfCanal) Interets() []Interet                      { return nil }
func (c *cfCanal) Clore(BilanDeMarche)                      {}
func (c *cfCanal) Brancher(*Observation, *MarcheDistribuee) {}

func (c *cfCanal) Trame(p *lecture.Paquet) {
	for _, r := range p.Records {
		switch r.Genre {
		case lecture.GenreNeuf:
			v := &cfVie{ti: r.TI, slot: r.Vie.Slot, gen: r.Vie.Gen, ne: p.TS}
			if c.fc != nil && r.TI == c.posTI {
				v.pos, v.aPos = c.positionALaCreation(p, r)
			}
			if r.TI == int16(EquipmentTypeIndex) {
				for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
					if k.Index == 10 && k.Bits > 0 {
						br := LecteurSur(p.Payload)
						br.SetBitPos(int(k.Debut))
						st := decodeObjectParentState(br, 3, uint32(EquipmentTypeIndex))
						v.attache, v.porteur = st.Attached, epPorteur(st.Quant16)
					}
				}
			}
			c.vies = append(c.vies, v)
			c.actives[r.Vie.Slot] = v
		case lecture.GenreSuppression:
			if v := c.actives[r.Vie.Slot]; v != nil {
				v.fin = p.TS
				delete(c.actives, r.Vie.Slot)
			}
		}
	}
}

// TestChampParc : pour chaque objet engendre que nomme une pile i28 (canal de production), une ligne
// TSV dans EQUIPPORTE_OUT/<id>.suivis.tsv : film, rang i48 de l objet porte, ti de la vie nommee, ne
// dans le paquet de la pile (1/0), duree de vie en s (-1 : fin non lue), fin lue apres la
// suppression de l objet porte (1/0).
func TestChampParc(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	cf := &cfCanal{actives: map[uint32]*cfVie{}}
	prod := nouveauCanalDeLEquipementPorte(fc)
	if err := Distribuer(fc, cf, prod); err != nil {
		t.Fatalf("%s : marche : %v", id, err)
	}
	fc2, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	ch, _, err := ScanEquipmentChanges(fc2, nil)
	if err != nil {
		t.Fatalf("%s : i48 : %v", id, err)
	}
	rangs := map[[2]uint64]int{}
	for _, e := range ch {
		if e.Kind == types.EquipmentTaken || e.Kind == types.EquipmentSpawned {
			rangs[[2]uint64{e.TimestampUS, uint64(e.Slot)}] = e.Rank
		}
	}
	type suivi struct {
		rang       int
		ne, fin    uint64
		finPorte   uint64
		vieParPile *cfVie
	}
	rangObjet := map[types.LifeKey]int{}
	finObjet := map[types.LifeKey]uint64{}
	suivis := map[types.LifeKey]*suivi{}
	var ordre []types.LifeKey
	for _, l := range prod.lu.lectures {
		switch l.Kind {
		case types.CarriedNew:
			r, ok := rangs[[2]uint64{l.TimestampUS, uint64(l.Carrier)}]
			if !ok {
				r = -1
			}
			rangObjet[l.Object] = r
		case types.CarriedDel:
			finObjet[l.Object] = l.TimestampUS
		case types.CarriedTrackedEnd:
			if s := suivis[l.Tracked[0]]; s != nil {
				s.fin, s.finPorte = l.TimestampUS, finObjet[l.Object]
			}
		}
		for _, k := range l.Tracked {
			if _, ok := suivis[k]; ok || l.Kind == types.CarriedTrackedEnd {
				continue
			}
			s := &suivi{rang: rangObjet[l.Object], ne: l.TimestampUS}
			for _, v := range cf.vies {
				if v.slot == k.Slot && v.gen == k.Gen && v.ne <= l.TimestampUS && (v.fin == 0 || v.fin >= l.TimestampUS) {
					s.vieParPile = v
				}
			}
			suivis[k] = s
			ordre = append(ordre, k)
		}
	}
	var b strings.Builder
	for _, k := range ordre {
		s := suivis[k]
		ti, meme := int16(-1), 0
		if s.vieParPile != nil {
			ti = s.vieParPile.ti
			if s.vieParPile.ne == s.ne {
				meme = 1
			}
		}
		duree := -1.0
		if s.fin > 0 {
			duree = float64(s.fin-s.ne) / 1e6
		}
		apres := 0
		if s.fin > 0 && s.finPorte > 0 && s.finPorte < s.fin {
			apres = 1
		}
		fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%.2f\t%d\n", id, s.rang, ti, meme, duree, apres)
	}
	if err := os.WriteFile(filepath.Join(out, id+".suivis.tsv"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("ecriture : %v", err)
	}
	t.Logf("%s : %d objets engendres, stats %+v", id, len(ordre), prod.lu.st)
}

// TestSuivisDetail : pour chaque vie nommee par une pile i28, la lecture qui la nomme (objet porte,
// porteur, instant) et TOUTES les vies du meme slot connues de la marche (archetype, generation,
// naissance, suppression). EQUIPPORTE_FILMS, EQUIPPORTE_ID, EQUIPPORTE_CARTE.
func TestSuivisDetail(t *testing.T) {
	racine, id := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID")
	if racine == "" || id == "" {
		t.Skip("EQUIPPORTE_FILMS ou EQUIPPORTE_ID absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	cf := &cfCanal{actives: map[uint32]*cfVie{}}
	prod := nouveauCanalDeLEquipementPorte(fc)
	if err := Distribuer(fc, cf, prod); err != nil {
		t.Fatalf("%s : marche : %v", id, err)
	}
	vus := map[types.LifeKey]bool{}
	ms := func(ts uint64) float64 { return float64(int64(ts)-int64(origine)) / 1000 }
	for _, l := range prod.lu.lectures {
		for _, k := range l.Tracked {
			if vus[k] || l.Kind == types.CarriedTrackedEnd {
				continue
			}
			vus[k] = true
			var b strings.Builder
			for _, v := range cf.vies {
				if v.slot == k.Slot {
					fmt.Fprintf(&b, " [ti=%d g%d ne %.1f fin %.1f]", v.ti, v.gen, ms(v.ne), ms(v.fin))
				}
			}
			t.Logf("%s %9.1f objet %d porteur %d suivi slot %d g%d :%s", id, ms(l.TimestampUS), l.Object.Slot,
				l.Carrier, k.Slot, k.Gen, b.String())
		}
	}
}

// TestArchetype : les composants de l archetype EQUIPPORTE_TI du registre du film.
func TestArchetype(t *testing.T) {
	racine, id := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID")
	if racine == "" || id == "" {
		t.Skip("EQUIPPORTE_FILMS ou EQUIPPORTE_ID absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	for _, s := range strings.Split(os.Getenv("EQUIPPORTE_TI"), ",") {
		var ti int
		fmt.Sscan(s, &ti)
		a, ok := reg.Archetype(ti)
		if !ok {
			continue
		}
		for i, c := range a.Components {
			t.Logf("ti=%d i%d %s", ti, i, c)
		}
	}
}

// TestFailleObjet : pour chaque vie nommee par une pile i28 d un objet porte, sa naissance (vie de la
// marche) et sa position A LA CREATION si elle est de l archetype EQUIPPORTE_TI (38 pour la faille).
// Une ligne TSV par vie dans EQUIPPORTE_OUT/<id>.objets.tsv : film, objet porte, porteur, slot, gen,
// ti, naissance (ms), x, y, z (vides sans position).
func TestFailleObjet(t *testing.T) {
	racine, id, out := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID"), os.Getenv("EQUIPPORTE_OUT")
	if racine == "" || id == "" || out == "" {
		t.Skip("EQUIPPORTE_FILMS, EQUIPPORTE_ID ou EQUIPPORTE_OUT absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	var ti int
	fmt.Sscan(os.Getenv("EQUIPPORTE_TI"), &ti)
	cf := &cfCanal{actives: map[uint32]*cfVie{}, fc: fc, posTI: int16(ti), wr: bornesDeLaCarte(t)}
	prod := nouveauCanalDeLEquipementPorte(fc)
	if err := Distribuer(fc, cf, prod); err != nil {
		t.Fatalf("%s : marche : %v", id, err)
	}
	ms := func(ts uint64) float64 { return float64(int64(ts)-int64(origine)) / 1000 }
	vus := map[types.LifeKey]bool{}
	var b strings.Builder
	for _, l := range prod.lu.lectures {
		for _, k := range l.Tracked {
			if vus[k] || l.Kind == types.CarriedTrackedEnd {
				continue
			}
			vus[k] = true
			viti, ne, pos := int16(-1), 0.0, "\t\t"
			for _, v := range cf.vies {
				if v.slot == k.Slot && v.gen == k.Gen && v.ne <= l.TimestampUS {
					viti, ne, pos = v.ti, ms(v.ne), "\t\t"
					if v.aPos {
						pos = fmt.Sprintf("%.2f\t%.2f\t%.2f", v.pos[0], v.pos[1], v.pos[2])
					}
				}
			}
			fmt.Fprintf(&b, "%s\t%d\t%d\t%d\t%d\t%d\t%.1f\t%s\n", id, l.Object.Slot, l.Carrier, k.Slot, k.Gen,
				viti, ne, pos)
		}
	}
	if err := os.WriteFile(filepath.Join(out, id+".objets.tsv"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Log(b.String())
}

// positionALaCreation rend la position (i0) d un record NEW par le decodeur des pistes
// ([decodeWorldObjectPos] : chemin dominant, region de la carte, quanta satures rejetes), aux bornes
// de la carte EQUIPPORTE_CARTE.
func (c *cfCanal) positionALaCreation(p *lecture.Paquet, r lecture.Record) ([3]float32, bool) {
	for _, k := range p.Comps[r.Comps[0]:r.Comps[1]] {
		if k.Index != 0 || k.Bits == 0 {
			continue
		}
		return decodeWorldObjectPos(p.Payload, int(k.Debut), &c.wr, c.fc.ProfilDeBalayage().LargeursObjetDuMonde())
	}
	return [3]float32{}, false
}

// bornesDeLaCarte rend les bornes de la carte EQUIPPORTE_CARTE.
func bornesDeLaCarte(t *testing.T) profile.Vec3Range {
	cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range strings.Split(os.Getenv("EQUIPPORTE_CARTE"), "|") {
		if e, err := cat.Lookup(c); err == nil {
			return e.Range()
		}
	}
	t.Fatal("carte absente du catalogue")
	return profile.Vec3Range{}
}

// TestNaissancesDuCanal journalise les creations que le canal de production joint aux piles i28
// (EQUIPPORTE_FILMS, EQUIPPORTE_ID, EQUIPPORTE_CARTE) : la verification du canal contre
// [TestFailleObjet].
func TestNaissancesDuCanal(t *testing.T) {
	racine, id := os.Getenv("EQUIPPORTE_FILMS"), os.Getenv("EQUIPPORTE_ID")
	if racine == "" || id == "" {
		t.Skip("EQUIPPORTE_FILMS ou EQUIPPORTE_ID absent : instrument saute")
	}
	fc, err := epContexte(filepath.Join(racine, id))
	if err != nil {
		t.Fatalf("%s : %v", id, err)
	}
	origine, _ := ScanClockOrigin(fc.Film())
	wr := bornesDeLaCarte(t)
	ls, st, err := ScanEquipementPorte(fc, &wr)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range ls {
		for _, n := range l.Spawned {
			t.Logf("%s porteur %d objet %d : ne %.1f ti=%d slot %d g%d pos %v (%.2f, %.2f, %.2f)", id, l.Carrier,
				l.Object.Slot, float64(n.TimestampUS-origine)/1000, n.TypeIndex, n.Life.Slot, n.Life.Gen,
				n.HasPosition, n.X, n.Y, n.Z)
		}
	}
	t.Logf("stats %+v", st)
}
