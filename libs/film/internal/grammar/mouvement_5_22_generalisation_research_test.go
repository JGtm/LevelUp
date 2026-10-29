//go:build research

package grammar

// mouvement_5_22_generalisation_research_test.go — LA DIFFERENTIELLE SUR TOUTE LA POPULATION
// (lot 5.22.3).
//
// La case 5.22.1 mesure UNE vie, celle du temoin que l utilisateur a verifie dans Theater. Cette
// case-ci refait la meme croix sur TOUTES les vies de bipede du film, et sur un second film :
// c est le controle de sortie du lot, et c est ce qui distingue « le temoin ne porte rien » de
// « le film ne porte rien ».
//
// Elle publie AUSSI, et d abord, l etalon de contenu de la marche (records `ti=35`, desyncs,
// vies) : sans lui, « aucun candidat » ne dit pas sur quelle part du film il porte.

import (
	"sort"
	"testing"
)

// m522PopRecord est UN record de bipede, quel que soit son slot.
type m522PopRecord struct {
	slot  uint32
	ts    uint64
	comps []int
}

// m522Population est ce que la marche a retenu du film entier.
type m522Population struct {
	recs    []m522PopRecord
	vit     map[uint32][]jumpVelSample
	desyncs int
	origine uint64
}

// m522DeroulerTout fait UNE passe et retient tous les records de bipede et toutes les vitesses.
func m522DeroulerTout(t *testing.T) m522Population {
	t.Helper()
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
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	out := m522Population{vit: map[uint32][]jumpVelSample{}}
	var ts uint64
	obs := NouvelleObservation()
	obs.EtatMouvementHook = func(c EtatMouvementComposant, s uint32, v []uint64) {
		if c != EtatVitesse || len(v) < 4 || v[0] != 0 || v[1] != 0 {
			return
		}
		if ti, ok := w.ArchetypeForSlot(s); !ok || ti != BipedTypeIndex {
			return
		}
		vec := DecodeVelocity(v[2], v[3])
		out.vit[s] = append(out.vit[s], jumpVelSample{ts: ts, vz: float64(vec[2])})
	}
	cfg.Obs = obs
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
			if out.origine == 0 {
				out.origine = pk.TimestampUS
			}
			ts = pk.TimestampUS
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, MovementStateViews, debut)
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				if r.DesyncAt >= 0 {
					out.desyncs++
				}
				pr := m522PopRecord{slot: r.Slot, ts: ts}
				for _, cp := range r.Trace.Comps {
					pr.comps = append(pr.comps, cp.Index)
				}
				out.recs = append(out.recs, pr)
			}
		}
	}
	return out
}

// TestMouvement522Generalisation croise TOUS les composants de TOUTES les vies de bipede contre
// les fenetres de decollage de LEUR PROPRE vie.
func TestMouvement522Generalisation(t *testing.T) {
	pop := m522DeroulerTout(t)
	if len(pop.recs) == 0 {
		t.Fatalf("aucun record de bipede")
	}
	eps := map[uint32][]jumpEpisode{}
	var retenus, fermes int
	slots := make([]uint32, 0, len(pop.vit))
	for s := range pop.vit {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		vs := vitessesOrdonnees(pop.vit[s])
		for _, e := range episodesDeMontee(s, vs) {
			fermes++
			if !hauteurDeSaut(e.haut) {
				continue
			}
			retenus++
			eps[s] = append(eps[s], e)
		}
	}
	t.Logf("ETALON DE LA MARCHE : %d records `ti=35` · %d desyncs · %d vies de bipede · "+
		"%d episodes fermes dont %d retenus (%d vies sautent)", len(pop.recs), pop.desyncs,
		len(pop.vit), fermes, retenus, len(eps))

	dedans, dehors := map[int]int{}, map[int]int{}
	var nDedans, nDehors int
	// exclusifs : les composants dont AUCUNE declaration ne tombe hors d une fenetre.
	for _, r := range pop.recs {
		in := m522DansDecollage(r.ts, eps[r.slot], true)
		if in {
			nDedans++
		} else {
			nDehors++
		}
		for _, i := range r.comps {
			if in {
				dedans[i]++
			} else {
				dehors[i]++
			}
		}
	}
	t.Logf("RECORDS : %d dans une fenetre de decollage [-%d ms ; +%d ms], %d hors", nDedans,
		m522AvantDecollageUS/1000, m522ApresDecollageUS/1000, nDehors)
	idx := make([]int, 0, len(dedans))
	vus := map[int]bool{}
	for _, m := range []map[int]int{dedans, dehors} {
		for i := range m {
			if !vus[i] {
				idx, vus[i] = append(idx, i), true
			}
		}
	}
	sort.Ints(idx)
	var exclusifs []int
	t.Logf("%-6s %9s %8s %9s %8s %8s", "comp", "dedans", "part", "dehors", "part", "facteur")
	for _, i := range idx {
		pd := float64(dedans[i]) / float64(max(nDedans, 1)) * 100
		ph := float64(dehors[i]) / float64(max(nDehors, 1)) * 100
		f := 0.0
		if ph > 0 {
			f = pd / ph
		}
		if dehors[i] == 0 && dedans[i] > 0 {
			exclusifs = append(exclusifs, i)
		}
		t.Logf("i%-5d %9d %7.2f%% %9d %7.2f%% %8.2f", i, dedans[i], pd, dehors[i], ph, f)
	}
	t.Logf("COMPOSANTS EXCLUSIFS AUX FENETRES DE DECOLLAGE : %v (attendu : aucun)", exclusifs)
}
