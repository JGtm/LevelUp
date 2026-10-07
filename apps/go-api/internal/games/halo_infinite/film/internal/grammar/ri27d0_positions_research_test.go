//go:build research

package grammar

// ri27d0_positions_research_test.go — LA MESURE DE 2.7.d0, volet des positions (plan de l etape 2 de
// la representation intermediaire) : les positions bipedes que la cuisson lit aujourd hui (l ancrage
// d en-tete, [positionsDesAncres]) contre celles que donnerait la marche des trames d abord, l ancrage
// derriere elle, selon la regle du canal des lectures bipedes de 2.7.b ([rendParLAncrage]) : les
// records de la marche dont le composant i0 est absolu dans la region jouee (la grammaire d i0 de
// l ancrage), puis les records que l ancrage rend derriere. Les quanta sont compares record par
// record (chunk, paquet, slot). Aucun fichier de production n est touche.
//
// Sortie (RI27C_OUT) : `positions_d0.tsv`
//
//	S  film  classe  compte
//	D  film  classe  chunk  paquet  slot  detail   (au plus 30 par classe et par film)
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> [RI27C_CARTES=<id=Carte;...>] \
//	  go test -tags=research -count=1 -run '^TestRI27d0Positions$' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ri27d0Cle designe un record bipede : son chunk, son paquet et son slot.
type ri27d0Cle struct {
	chunk, paquet int
	slot          uint32
}

// ri27d0Position est une position lue : ses quanta et sa source.
type ri27d0Position struct {
	q        [3]uint32
	recupere bool
	i0       int
	gen      uint32
	ts       uint64
}

// ri27d0Positions tient les comptes d un film.
type ri27d0Positions struct {
	court   string
	classes map[string]int
	ech     map[string]int
	lignes  []string
}

func (m *ri27d0Positions) classer(classe string, k ri27d0Cle, detail string) {
	m.classes[classe]++
	if m.ech[classe] >= 30 {
		return
	}
	m.ech[classe]++
	m.lignes = append(m.lignes, fmt.Sprintf("D\t%s\t%s\t%d\t%d\t%d\t%s", m.court, classe, k.chunk, k.paquet, k.slot, detail))
}

// ri27d0Absolu dit si le composant i0 commencant au bit `i0` est absolu dans la region jouee : la
// grammaire d i0 de l ancrage ([matchBipedHeader]).
func ri27d0Absolu(pay []byte, i0 int, lay profile.I0Layout) bool {
	const preGate = profile.I0SpineBits + profile.I0UseDefaultBits
	if i0+lay.TotalBits() > len(pay)*8 {
		return false
	}
	if uint32(source.BitsStricts(pay, i0, preGate)) != 0 {
		return false
	}
	return uint32(source.BitsStricts(pay, i0+preGate, lay.GateBits-preGate)) == lay.Region
}

// ri27d0Payload rend le payload du paquet `index` du chunk `chunk`.
func ri27d0Payload(fc *FilmContext, chunk, index int) []byte {
	data, pks, ok := fc.ChunkAt(chunk)
	if !ok {
		return nil
	}
	for _, pk := range pks {
		if pk.Index == index {
			return pk.Payload(data)
		}
	}
	return nil
}

func TestRI27d0Positions(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		lay, err := fc.I0Layout()
		if err != nil {
			t.Fatalf("%s : decoupage i0 : %v", court, err)
		}
		m := &ri27d0Positions{court: court, classes: map[string]int{}, ech: map[string]int{}}
		// AUJOURD HUI : l ancrage du contexte.
		anciens := map[ri27d0Cle][]ri27d0Position{}
		var i0Anciens = map[ri27d0Cle]int{}
		genAnciens := map[ri27d0Cle]uint32{}
		c0 := map[ri27d0Cle]uint64{}
		fc.parcourirLesAncresBipedes(func(r deltaBipedRecord) {
			k := ri27d0Cle{r.Chunk, r.Packet.Index, r.Slot}
			i0Anciens[k], genAnciens[k], c0[k] = r.I0, r.Gen, r.Packet.TimestampUS
		})
		olds, _ := positionsDesAncres(fc, fc.ChunkNumbers(), lay, ScanFilmOptions{QuantaOnly: true})
		for _, p := range olds {
			k := ri27d0Cle{p.Chunk, p.PacketIndex, p.Slot}
			anciens[k] = append(anciens[k], ri27d0Position{q: p.Q, i0: i0Anciens[k], gen: genAnciens[k], ts: c0[k]})
		}
		// LA MARCHE D ABORD : le canal des lectures bipedes, i0 compris.
		c := nouveauCanalDesLecturesBipedes(fc)
		c.utiles |= 1
		desc := &ri27d0Descripteur{vus: map[ri27d0Cle]string{}}
		if err := Distribuer(fc, c, desc); err != nil {
			t.Fatalf("%s : distribution : %v", court, err)
		}
		neufs := map[ri27d0Cle][]ri27d0Position{}
		lusParLaMarche := map[ri27d0Cle]bool{}
		for i := range c.lu.records {
			rb := &c.lu.records[i]
			k := ri27d0Cle{rb.Chunk, rb.Packet.Index, rb.Slot}
			if !rb.Recupere {
				lusParLaMarche[k] = true
			}
			if rb.masque&1 == 0 {
				m.classes["neuf:sans_i0"]++
				continue
			}
			pay := ri27d0Payload(fc, rb.Chunk, rb.Packet.Index)
			if !ri27d0Absolu(pay, rb.I0, lay) {
				m.classes["neuf:i0_non_absolu"]++
				continue
			}
			var q [3]uint32
			for ax := range 3 {
				q[ax] = uint32(source.BitsStricts(pay, rb.I0+lay.AxisOffset(ax), int(lay.AxisW[ax])))
			}
			neufs[k] = append(neufs[k], ri27d0Position{q: q, recupere: rb.Recupere, i0: rb.I0})
		}
		ri27d0Desc = desc.vus
		ri27d0Comparer(m, c, anciens, neufs, lusParLaMarche)
		lignes = append(lignes, ri27d0Trier(court, "S", m.classes)...)
		lignes = append(lignes, m.lignes...)
		t.Logf("%s : anciennes %d, communes egales %d, anciennes seules %d, neuves seules %d", court,
			len(olds), m.classes["commune:egale"], m.classes["ancienne_seule"], m.classes["neuve_seule"])
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "positions_d0.tsv"), lignes)
}

// ri27d0Comparer range chaque record des deux lectures.
func ri27d0Comparer(m *ri27d0Positions, c *canalDesLecturesBipedes, anciens, neufs map[ri27d0Cle][]ri27d0Position,
	lusParLaMarche map[ri27d0Cle]bool) {
	for k, as := range anciens {
		ns, ok := neufs[k]
		if !ok {
			m.classes["ancienne_seule"]++
			t, vu := c.trames[paquetDuFlux{k.chunk, k.paquet}]
			switch {
			case !vu:
				m.classer("ancienne_seule:trame_non_rendue", k, "")
			case slices.Contains(t.slots, k.slot) && lusParLaMarche[k]:
				m.classer("ancienne_seule:marche_lit_le_record_sans_i0_absolu", k, fmt.Sprintf("q=%v", as[0].q))
			case slices.Contains(t.slots, k.slot):
				m.classer("ancienne_seule:marche_lit_le_slot_record_ecarte:"+ri27d0Ecart(c, k, as[0]), k, ri27d0Desc[k]+" "+
					fmt.Sprintf("q=%v gen=%d", as[0].q, as[0].gen))
			case int64(as[0].i0) >= int64(t.prouveeDes):
				m.classer("ancienne_seule:prouve_par_la_fermeture", k, fmt.Sprintf("q=%v i0=%d prouvee_des=%d", as[0].q, as[0].i0, t.prouveeDes))
			default:
				m.classer("ancienne_seule:rendue_puis_ecartee:"+ri27d0Ecart(c, k, as[0]), k, fmt.Sprintf("q=%v gen=%d", as[0].q, as[0].gen))
			}
			continue
		}
		if ns[0].q == as[0].q {
			src := "marche"
			if ns[0].recupere {
				src = "recuperee"
			}
			m.classes["commune:egale"]++
			m.classes["commune:egale:"+src]++
		} else {
			m.classer("commune:differente", k, fmt.Sprintf("ancien=%v neuf=%v recupere=%v i0=%d/%d", as[0].q, ns[0].q,
				ns[0].recupere, as[0].i0, ns[0].i0))
		}
	}
	for k, ns := range neufs {
		if _, ok := anciens[k]; ok {
			continue
		}
		m.classes["neuve_seule"]++
		t := c.trames[paquetDuFlux{k.chunk, k.paquet}]
		etat := "trame_fermee"
		if t.prouveeDes == rienDeProuve {
			etat = "trame_non_prouvee"
		}
		src := "marche"
		if ns[0].recupere {
			src = "recuperee"
		}
		m.classer("neuve_seule:"+src+":"+etat, k, fmt.Sprintf("q=%v i0=%d", ns[0].q, ns[0].i0))
	}
}

// ri27d0Ecart dit pourquoi le canal ecarte la position `p` du slot de `k` : le corps est mort a cet
// instant (le dead-state lu par la marche le precede), ou la garde des generations vivantes datees le
// refuse.
func ri27d0Ecart(c *canalDesLecturesBipedes, k ri27d0Cle, p ri27d0Position) string {
	if mort, connue := c.mortA[types.LifeKey{Slot: k.slot, Gen: p.gen}]; connue && p.ts >= mort {
		return "corps_mort"
	}
	if !c.fc.GenerationsVivantesA(p.ts).Accepte(types.LifeKey{Slot: k.slot, Gen: p.gen}) {
		return "generation_refusee"
	}
	return "autre"
}

// ri27d0Descripteur est un canal des trames qui decrit ce que la marche lit de chaque slot de chaque
// paquet (genre, archetype, generation, en-tete, i0).
type ri27d0Descripteur struct {
	m   *MarcheDistribuee
	vus map[ri27d0Cle]string
}

func (*ri27d0Descripteur) Interets() []Interet { return nil }
func (*ri27d0Descripteur) Clore(BilanDeMarche) {}

func (d *ri27d0Descripteur) Brancher(_ *Observation, m *MarcheDistribuee) { d.m = m }

func (d *ri27d0Descripteur) Trame(p *lecture.Paquet) {
	recs, lus := d.m.recordsDeLaTrame()
	if !lus {
		return
	}
	for i := range recs {
		r := &recs[i]
		i0 := "sans_i0"
		for _, co := range r.Trace.Comps {
			if co.Index == 0 {
				i0 = fmt.Sprintf("i0@%d_porte=%v", co.StartBit, co.Ported)
			}
		}
		d.vus[ri27d0Cle{p.Chunk, p.Index, r.Slot}] += fmt.Sprintf("[%s/ti%d/gen%d/@%d/%s/mort=%v]",
			nomDeTypeDeRecord(r.Type), r.TypeIndex, r.ID>>30, r.HeaderBit, i0, r.Trace.Dead != nil)
	}
}

// ri27d0Desc : la description de la marche du film en cours (instrument sequentiel).
var ri27d0Desc map[ri27d0Cle]string
