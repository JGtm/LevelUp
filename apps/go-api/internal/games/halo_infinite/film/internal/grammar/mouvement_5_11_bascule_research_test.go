//go:build research

package grammar

// mouvement_5_11_bascule_research_test.go — L ORACLE DE LA BASCULE (lot 5.11).
//
// Scission par DEPLACEMENT PUR de `mouvement_5_11_temoin_research_test.go` (seuil de 500 lignes,
// piege 5 de la passation 5.3.3). Aucune ligne n est reecrite ici.
//
// DEUX TESTS, ET C EST LE COEUR DU LOT : `TestMouvement511Bascule` croise TOUT champ replique du
// film temoin contre la fenetre du saut ; `TestMouvement511Corpus` generalise le meme croisement
// a un film de MATCH, par les episodes de la PRODUCTION.

import (
	"fmt"
	"sort"
	"testing"
)

type m511Cle struct {
	ti   uint32
	idx  int
	nom  string
	slot uint32
}

// TestMouvement511Bascule est LE TEST DE L ORACLE : pour CHAQUE champ replique du film (tout
// archetype, tout composant, tout slot), la liste des instants ou il est DECLARE. Un declencheur
// de saut est un champ dont tous les instants tombent dans la fenetre du saut.
//
// C est le dispositif que le film temoin rend possible et qu un film de match interdit : ici il
// n y a qu un bipede, une action, un instant connu.
func TestMouvement511Bascule(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	sautA, sautB := m511Bornes()
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	vus := map[m511Cle][]float64{}
	evs := map[int][]float64{}
	var t0 uint64
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			rel := float64(pk.TimestampUS-t0) / 1e6
			pay := pk.Payload(data)
			debut := 2
			if typ, present := PacketHeadEventType(pay); present {
				evs[typ] = append(evs[typ], rel)
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			for _, r := range recs {
				for _, cp := range r.Trace.Comps {
					k := m511Cle{ti: r.TypeIndex, idx: cp.Index, nom: cp.Name, slot: r.Slot}
					vus[k] = append(vus[k], rel)
				}
			}
		}
	}
	m511Verdict(t, vus, evs, sautA, sautB)
}

// m511Verdict trie les champs en trois familles : DEDANS SEULEMENT (les candidats), DEDANS ET
// DEHORS (des correlations), DEHORS SEULEMENT (les temoins).
func m511Verdict(t *testing.T, vus map[m511Cle][]float64, evs map[int][]float64,
	a, b float64) {
	t.Helper()
	type bilan struct {
		k             m511Cle
		dedans, total int
		premier       float64
	}
	var dans, mixtes, hors []bilan
	for k, ts := range vus {
		bl := bilan{k: k, total: len(ts), premier: ts[0]}
		for _, x := range ts {
			if x >= a && x <= b {
				bl.dedans++
			}
		}
		switch {
		case bl.dedans == bl.total:
			dans = append(dans, bl)
		case bl.dedans > 0:
			mixtes = append(mixtes, bl)
		default:
			hors = append(hors, bl)
		}
	}
	tri := func(v []bilan) {
		sort.Slice(v, func(i, j int) bool {
			if v[i].k.ti != v[j].k.ti {
				return v[i].k.ti < v[j].k.ti
			}
			return v[i].k.idx < v[j].k.idx
		})
	}
	tri(dans)
	tri(mixtes)
	tri(hors)
	t.Logf("FENETRE DU SAUT : [%.3f ; %.3f] s (temps du film, depuis le premier paquet delta)",
		a, b)
	t.Logf("CANDIDATS — champs dont TOUS les instants sont DANS la fenetre (%d) :", len(dans))
	for _, x := range dans {
		t.Logf("    ti=%2d i%-2d slot %5d %-52s %4d instants, premier a %.3f s",
			x.k.ti, x.k.idx, x.k.slot, x.k.nom, x.total, x.premier)
	}
	t.Logf("CORRELATIONS — champs presents DANS et DEHORS (%d) :", len(mixtes))
	for _, x := range mixtes {
		t.Logf("    ti=%2d i%-2d slot %5d %-52s %4d instants dont %d dedans",
			x.k.ti, x.k.idx, x.k.slot, x.k.nom, x.total, x.dedans)
	}
	t.Logf("HORS FENETRE SEULEMENT (%d) :", len(hors))
	for _, x := range hors {
		t.Logf("    ti=%2d i%-2d slot %5d %-52s %4d instants, premier a %.3f s",
			x.k.ti, x.k.idx, x.k.slot, x.k.nom, x.total, x.premier)
	}
	types := make([]int, 0, len(evs))
	for ty := range evs {
		types = append(types, ty)
	}
	sort.Ints(types)
	for _, ty := range types {
		var dedans int
		for _, x := range evs[ty] {
			if x >= a && x <= b {
				dedans++
			}
		}
		t.Logf("EVENEMENT DE TETE type %d : %d paquets, dont %d dans la fenetre",
			ty, len(evs[ty]), dedans)
	}
}

// TestMouvement511Corpus GENERALISE l oracle du film temoin a un film de MATCH : pour chaque
// composant du bipede, la part des records qui le declarent DANS une montee de saut (fenetre
// [t0 - 250 ms ; t1]) contre la part hors de toute montee. Un declencheur de saut serait un
// composant enrichi d un facteur franc DANS la fenetre.
//
// LES EPISODES SONT CEUX DE LA PRODUCTION (`episodesDeMontee` + `hauteurDeSaut`,
// `movement_states_jump.go`) : l instrument ne re-derive rien.
func TestMouvement511Corpus(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	var vits = map[uint32][]jumpVelSample{}
	var recs []m511RecRef
	var ti35, desync int
	etalon := map[int]int{}
	cfg := fc.CadreDeBalayage()
	var ts uint64
	obs := NouvelleObservation()
	obs.EtatMouvementHook = func(c EtatMouvementComposant, slot uint32, v []uint64) {
		if c != EtatVitesse || len(v) < 4 || v[0] != 0 || v[1] != 0 {
			return
		}
		vec := DecodeVelocity(v[2], v[3])
		vits[slot] = append(vits[slot], jumpVelSample{ts: ts, vz: float64(vec[2])})
	}
	cfg.Obs = obs
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			ts = pk.TimestampUS
			out, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			for _, r := range out {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				ti35++
				if r.DesyncAt >= 0 {
					desync++
				}
				for _, b := range []int{0, 1, 21, 25} {
					if r.Trace.Mask&(1<<uint(b)) != 0 {
						etalon[b]++
					}
				}
				noms := make([]string, 0, len(r.Trace.Comps))
				for _, cp := range r.Trace.Comps {
					noms = append(noms, fmt.Sprintf("i%d %s", cp.Index, cp.Name))
				}
				recs = append(recs, m511RecRef{slot: r.Slot, ts: pk.TimestampUS, comps: noms})
			}
		}
	}
	t.Logf("ORACLE DE CONTENU : %d records ti=35 (%d desynchronises) · i0 %.1f %% · i1 %.1f %% · "+
		"i21 %.1f %% · i25 %.1f %%", ti35, desync, m533bPart(etalon[0], ti35),
		m533bPart(etalon[1], ti35), m533bPart(etalon[21], ti35), m533bPart(etalon[25], ti35))
	if m533bPart(etalon[21], ti35) < 50 {
		t.Fatalf("ETALON REFUSE : i21 sous 50 %% — trame non cadree, aucune conclusion")
	}
	m511CorpusVerdict(t, vits, recs, ti35)
}

// m511RecRef est UN record de bipede reduit a ce que le verdict croise : sa vie, son instant, et
// les composants qu il DECLARE.
type m511RecRef struct {
	slot  uint32
	ts    uint64
	comps []string
}

// m511FenetreSaut est la marge AVANT le premier instant de la montee : le declencheur d un saut
// precede son premier echantillon de vitesse positive, et 250 ms couvrent quinze ticks.
const m511FenetreSaut = 250000

// m511CorpusVerdict publie le tableau dedans/dehors par composant.
func m511CorpusVerdict(t *testing.T, vits map[uint32][]jumpVelSample,
	recs []m511RecRef, ti35 int) {
	t.Helper()
	type borne struct{ a, b uint64 }
	fenetres := map[uint32][]borne{}
	var sauts int
	for s, vs := range vits {
		for _, e := range episodesDeMontee(s, vs) {
			if !hauteurDeSaut(e.haut) {
				continue
			}
			sauts++
			a := e.t0
			if a > m511FenetreSaut {
				a -= m511FenetreSaut
			}
			fenetres[s] = append(fenetres[s], borne{a: a, b: e.t1})
		}
	}
	dedansRec, horsRec := 0, 0
	dedans, hors := map[string]int{}, map[string]int{}
	for _, r := range recs {
		var in bool
		for _, f := range fenetres[r.slot] {
			if r.ts >= f.a && r.ts <= f.b {
				in = true
				break
			}
		}
		if in {
			dedansRec++
		} else {
			horsRec++
		}
		for _, c := range r.comps {
			if in {
				dedans[c]++
			} else {
				hors[c]++
			}
		}
	}
	noms := make([]string, 0, len(dedans)+len(hors))
	vu := map[string]bool{}
	for _, m := range []map[string]int{dedans, hors} {
		for c := range m {
			if !vu[c] {
				vu[c] = true
				noms = append(noms, c)
			}
		}
	}
	sort.Strings(noms)
	t.Logf("%d sauts derives · %d records de bipede DANS une fenetre de saut, %d hors "+
		"(sur %d ti=35)", sauts, dedansRec, horsRec, ti35)
	t.Logf("%-58s %8s %8s %8s %8s %8s", "composant", "dedans", "part %", "hors", "part %",
		"facteur")
	for _, c := range noms {
		pd := m533bPart(dedans[c], dedansRec)
		ph := m533bPart(hors[c], horsRec)
		f := 0.0
		if ph > 0 {
			f = pd / ph
		}
		t.Logf("%-58s %8d %7.2f %8d %7.2f %8.2f", c, dedans[c], pd, hors[c], ph, f)
	}
}
