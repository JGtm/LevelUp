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
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

type cfVie struct {
	ti        int16
	slot, gen uint32
	ne, fin   uint64
	attache   bool
	porteur   uint32
}

type cfCanal struct {
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
