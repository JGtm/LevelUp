//go:build research

package grammar

// mouvement_5_22_differentielle_research_test.go — LA DIFFERENTIELLE DU TEMOIN (lot 5.22.1).
//
// Ce que le lot 5.11 ne pouvait pas faire et que ce temoin permet : comparer, SUR LA MEME VIE et
// SUR L HORLOGE DU FILM, ce qui se replique AU DECOLLAGE et ce qui se replique le reste du temps.
// Madina97294 (slot 523) est repliquee a chaque tick de 80,0 s a 160,0 s ; ses sept sauts sont
// dates par la mesure (`TestMouvement522Ancres`) et verifies dans Theater par l utilisateur.
//
// LA FENETRE EST UNE FENETRE DE DECOLLAGE, pas de vol : l entree precede la physique, donc
// `[t0 - 250 ms ; t0 + 100 ms]` — la meme forme qu au 5.11.2, bornee ici par la mesure.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// m522AvantDecollageUS et m522ApresDecollageUS bornent la fenetre de decollage.
const (
	m522AvantDecollageUS = 250_000
	m522ApresDecollageUS = 100_000
)

// m522Record est UN record du temoin, avec son masque et ses bits.
type m522Record struct {
	ts    uint64
	rel   float64
	comps []int
	bits  map[int]string
	// nonPortes liste les composants que le port a saute.
	nonPortes []int
	desync    int
}

// m522Trace est ce que la marche a retenu : les records du slot, les episodes de montee, et les
// evenements de tete dates.
type m522Trace struct {
	recs    []m522Record
	eps     []jumpEpisode
	origine uint64
	// tetes : instant relatif -> type d evenement de tete du paquet.
	tetes map[float64]int
}

// m522Deroule fait UNE passe de decodage du film et rend tout ce dont les mesures du lot ont
// besoin. Une passe, parce que la consigne de quota du lot est « la differentielle est UNE
// mesure » : les trois tests de ce fichier la partagent.
func m522Deroule(t *testing.T, slot uint32) m522Trace {
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
	out := m522Trace{tetes: map[float64]int{}}
	var ts uint64
	var vs []jumpVelSample
	obs := NouvelleObservation()
	obs.EtatMouvementHook = func(c EtatMouvementComposant, s uint32, v []uint64) {
		if c != EtatVitesse || s != slot || len(v) < 4 || v[0] != 0 || v[1] != 0 {
			return
		}
		vec := DecodeVelocity(v[2], v[3])
		vs = append(vs, jumpVelSample{ts: ts, vz: float64(vec[2])})
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
			rel := float64(ts-out.origine) / 1e6
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if typ, present := PacketHeadEventType(pay); present {
				out.tetes[rel] = typ
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, MovementStateViews, debut)
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex || r.Slot != slot {
					continue
				}
				out.recs = append(out.recs, m522RecordDe(pay, r, ts, rel))
			}
		}
	}
	sort.SliceStable(out.recs, func(i, j int) bool { return out.recs[i].ts < out.recs[j].ts })
	sort.SliceStable(vs, func(i, j int) bool { return vs[i].ts < vs[j].ts })
	out.eps = episodesDeMontee(slot, vs)
	return out
}

// m522RecordDe extrait d un record decode son masque, ses largeurs et ses bits bruts.
func m522RecordDe(pay []byte, r FrameRecord, ts uint64, rel float64) m522Record {
	out := m522Record{ts: ts, rel: rel, bits: map[int]string{}, desync: r.DesyncAt}
	comps := r.Trace.Comps
	for i, c := range comps {
		fin := r.Trace.EndBit
		if i+1 < len(comps) {
			fin = comps[i+1].StartBit
		}
		out.comps = append(out.comps, c.Index)
		out.bits[c.Index] = m511Bits(pay, c.StartBit, min(fin-c.StartBit, 128))
		if !c.Ported {
			out.nonPortes = append(out.nonPortes, c.Index)
		}
	}
	return out
}

// m522DansDecollage dit si un instant tombe dans la fenetre de decollage d un des episodes.
func m522DansDecollage(ts uint64, eps []jumpEpisode, retenusSeuls bool) bool {
	for _, e := range eps {
		if retenusSeuls && !hauteurDeSaut(e.haut) {
			continue
		}
		if ts+m522AvantDecollageUS >= e.t0 && ts <= e.t0+m522ApresDecollageUS {
			return true
		}
	}
	return false
}

// TestMouvement522Fenetre publie, record par record, tout ce que le temoin replique dans une
// fenetre de temps (`MOUV511_T0` / `MOUV511_T1`, en secondes d horloge film) : masque, largeurs,
// bits bruts, et les instants de decollage en marge.
func TestMouvement522Fenetre(t *testing.T) {
	slot := m522Slot()
	tr := m522Deroule(t, slot)
	t0, t1 := c514Fenetre()
	t.Logf("SLOT %d — %d records sur le film · fenetre [%.3f ; %.3f] s", slot, len(tr.recs), t0, t1)
	decollages := map[uint64]bool{}
	for _, e := range tr.eps {
		if hauteurDeSaut(e.haut) {
			decollages[e.t0] = true
		}
	}
	detaille := os.Getenv("MOUV522_BITS") != ""
	var n int
	for _, r := range tr.recs {
		if r.rel < t0 || r.rel > t1 {
			continue
		}
		n++
		marque := " "
		if decollages[r.ts] {
			marque = "*"
		} else if m522DansDecollage(r.ts, tr.eps, true) {
			marque = "+"
		}
		var noms []string
		for _, i := range r.comps {
			noms = append(noms, fmt.Sprintf("i%d", i))
		}
		tete := ""
		if typ, ok := tr.tetes[r.rel]; ok {
			tete = fmt.Sprintf(" · tete type %d", typ)
		}
		t.Logf("%s t=%8.3f s · %2d comps : %-28s · desync %3d%s", marque, r.rel,
			len(r.comps), strings.Join(noms, " "), r.desync, tete)
		if !detaille {
			continue
		}
		for _, i := range r.comps {
			t.Logf("      i%-2d %s", i, r.bits[i])
		}
	}
	t.Logf("TOTAL FENETRE : %d records", n)
}

// TestMouvement522Bascule EST LA MESURE DU LOT : pour chaque composant du bipede, la part des
// records du temoin qui le declarent DANS une fenetre de decollage et HORS, sur la vie entiere.
//
// Un declencheur du saut est un composant dont la part dedans est ecrasante et la part dehors
// nulle. Le 5.11.2 a mesure ce tableau sur la POPULATION de tous les joueurs ; ici il porte sur
// UNE vie continue, ou la replication ne commence pas avec le saut.
func TestMouvement522Bascule(t *testing.T) {
	slot := m522Slot()
	tr := m522Deroule(t, slot)
	if len(tr.recs) == 0 {
		t.Fatalf("slot %d : aucun record", slot)
	}
	var retenus, rejetes int
	for _, e := range tr.eps {
		if hauteurDeSaut(e.haut) {
			retenus++
		} else {
			rejetes++
		}
	}
	dedans, dehors := map[int]int{}, map[int]int{}
	var nDedans, nDehors int
	valeurs := map[int]map[string]int{}
	for _, r := range tr.recs {
		in := m522DansDecollage(r.ts, tr.eps, true)
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
			if i == 0 || i == 1 {
				continue // position et vitesse : le derive lui-meme, pas un candidat
			}
			if valeurs[i] == nil {
				valeurs[i] = map[string]int{}
			}
			valeurs[i][r.bits[i]]++
		}
	}
	t.Logf("SLOT %d · %d records · %d episodes (%d retenus, %d rejetes) · fenetre de decollage "+
		"[-%d ms ; +%d ms]", slot, len(tr.recs), len(tr.eps), retenus, rejetes,
		m522AvantDecollageUS/1000, m522ApresDecollageUS/1000)
	t.Logf("RECORDS : %d dans une fenetre de decollage, %d hors", nDedans, nDehors)
	idx := make([]int, 0, len(dedans)+len(dehors))
	vus := map[int]bool{}
	for i := range dedans {
		if !vus[i] {
			idx, vus[i] = append(idx, i), true
		}
	}
	for i := range dehors {
		if !vus[i] {
			idx, vus[i] = append(idx, i), true
		}
	}
	sort.Ints(idx)
	t.Logf("%-6s %8s %8s %8s %8s %8s", "comp", "dedans", "part", "dehors", "part", "facteur")
	for _, i := range idx {
		pd := float64(dedans[i]) / float64(max(nDedans, 1)) * 100
		ph := float64(dehors[i]) / float64(max(nDehors, 1)) * 100
		f := 0.0
		if ph > 0 {
			f = pd / ph
		}
		t.Logf("i%-5d %8d %7.2f%% %8d %7.2f%% %8.2f", i, dedans[i], pd, dehors[i], ph, f)
	}
	t.Logf("VALEURS DISTINCTES PAR COMPOSANT (hors i0/i1) :")
	for _, i := range idx {
		if valeurs[i] == nil {
			continue
		}
		t.Logf("  i%-2d : %d valeurs distinctes sur %d lectures", i, len(valeurs[i]),
			dedans[i]+dehors[i])
	}
}
