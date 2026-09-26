//go:build research

package grammar

// m4b_liste_research_test.go — LOT M4b : LA LISTE D UN PAQUET, RECORD PAR RECORD, ET SA VRAIE
// FRONTIERE. Mesure seule. Les instruments qui ont localise les defauts repares au lot :
//
//	TestM4bPaquetsDetailles    la marche de production d un paquet : bornes et composants
//	TestM4bChainesDeTete       les candidats de debut de liste et leur chaine d essai
//	                           (debut_de_liste.go)
//	TestM4bVraieSuite          les positions d ou une marche FERME la vue C : la vraie frontiere
//	                           du record qui suit une rupture
//	TestM4bDepartDeComposant   le depart d un composant d ou la boucle finit sur cette frontiere
//	                           (corps de mort : 8a485699 p692, bipede 542, i18 attendu a 8736)
//
// Rejouable : memes variables que `m4b_tir_continu_research_test.go`, plus M4B_PAQUETS
// (« chunk:paquet,... »), M4B_SLOT, M4B_DE et M4B_FIN.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestM4bPaquetsDetailles publie, pour les paquets M4B_PAQUETS (« chunk:paquet,... »), la marche
// de production record par record : bornes de bit, composants presents, desynchronisation.
func TestM4bPaquetsDetailles(t *testing.T) {
	cad := s3LireCadre(t)
	voulus := map[[2]int]bool{}
	for _, s := range strings.Split(os.Getenv("M4B_PAQUETS"), ",") {
		var c, p int
		if _, err := fmt.Sscanf(s, "%d:%d", &c, &p); err == nil {
			voulus[[2]int{c, p}] = true
		}
	}
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _, fin := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			if !voulus[[2]int{c, pk.Index}] {
				continue
			}
			t.Logf("== t%d c%d p%d : %d bits, debut %d, fin %d", cad.trame(pk.TimestampUS), c, pk.Index, len(pay)*8, debut, fin)
			for _, r := range recs {
				var sb strings.Builder
				for _, cr := range r.Trace.Comps {
					fmt.Fprintf(&sb, "i%d%s@%d ", cr.Index, map[bool]string{true: "", false: "!"}[cr.Ported], cr.StartBit)
				}
				t.Logf("   %s %d/%d @%d-%d desync %d : %s", map[int]string{1: "NEW", 2: "DEL", 3: "DLT"}[r.Type], r.Slot, r.TypeIndex, r.HeaderBit, r.Trace.EndBit, r.DesyncAt, sb.String())
			}
		}
	}
}

// TestM4bChainesDeTete publie, pour les paquets M4B_PAQUETS, chaque candidat de debut de liste
// (naissances et en-tetes NEW de la bande) et la chaine d essai qui en part, pas a pas.
func TestM4bChainesDeTete(t *testing.T) {
	voulus := map[[2]int]bool{}
	for _, s := range strings.Split(os.Getenv("M4B_PAQUETS"), ",") {
		var c, p int
		if _, err := fmt.Sscanf(s, "%d:%d", &c, &p); err == nil {
			voulus[[2]int{c, p}] = true
		}
	}
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if voulus[[2]int{c, pk.Index}] {
					loc := marchLocateStrict(pay, w, cfg)
					cands := candidatsDeTete(pay, loc, w)
					t.Logf("== c%d p%d : localise %d, %d candidats", c, pk.Index, loc, len(cands))
					for _, p := range cands {
						t.Logf("   candidat %d : %s", p, m4bTraceChaine(pay, p, cands, loc, w, cfg))
					}
				}
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					continue
				}
			}
			DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
		}
	}
}

// m4bTraceChaine rejoue la chaine d essai de `chaineDeNaissancesJusqua` en publiant chaque pas.
func m4bTraceChaine(pay []byte, p0 int, ancres []int, debut int, w *World, cfg FrameConfig) string {
	essai := cfg
	essai.Obs = nil
	pos := p0
	var sb strings.Builder
	for n := 0; n < plafondChaineDeTete && pos < debut; n++ {
		b2 := LecteurSur(pay)
		b2.SetBitPos(pos)
		typ := readRecordType(b2)
		id := readRecordID(b2, cfg.IDLowBits, cfg.IDBase)
		if typ == recNew {
			br := LecteurSur(pay)
			br.poserCadre(essai)
			br.SetBitPos(pos + woNewTypeBits + woNewSlotBits + woNewGenBits)
			tr := TraverseEntity(br, w.Reg, essai.NewDefaultStateBits)
			fmt.Fprintf(&sb, "N%d/%d@%d-%d(d%d) ", id&0x3fffffff, tr.TypeIndex, pos, tr.EndBit, tr.DesyncAt)
			if tr.DesyncAt != -1 || tr.EndBit <= pos {
				return sb.String() + "ARRET"
			}
			pos = tr.EndBit
			continue
		}
		rec, fin, ok := TryDeltaAt(pay, pos, w, essai)
		fmt.Fprintf(&sb, "t%d:%d/%d@%d-%d(ok %v) ", typ, id&0x3fffffff, rec.TypeIndex, pos, fin, ok)
		if !ok || fin <= pos {
			return sb.String() + "ARRET"
		}
		pos = fin
	}
	return sb.String() + fmt.Sprintf("FIN %d (debut %d)", pos, debut)
}

// TestM4bVraieSuite cherche, dans les paquets M4B_PAQUETS, la position de bit d ou une marche de
// production (monde restaure a chaque essai) FERME la vue C : la vraie frontiere du record qui
// suit le point de rupture. Elle publie aussi les records de la marche du paquet.
func TestM4bVraieSuite(t *testing.T) {
	voulus := map[[2]int]bool{}
	for _, s := range strings.Split(os.Getenv("M4B_PAQUETS"), ",") {
		var c, p int
		if _, err := fmt.Sscanf(s, "%d:%d", &c, &p); err == nil {
			voulus[[2]int{c, p}] = true
		}
	}
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	var derniere LectureVueC
	obs.VueControleHook = func(l LectureVueC) { derniere = l }
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					continue
				}
			}
			if voulus[[2]int{c, pk.Index}] {
				snap := w.Snapshot()
				for p := debut; p < len(pay)*8-24; p++ {
					derniere = LectureVueC{}
					recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, p)
					w.Restore(snap)
					if derniere.Fermee && len(recs) > 0 {
						var sb strings.Builder
						for _, r := range recs {
							fmt.Fprintf(&sb, "%s%d/%d ", map[int]string{1: "N", 2: "D", 3: ""}[r.Type], r.Slot, r.TypeIndex)
						}
						t.Logf("c%d p%d : depart %d FERME : %s", c, pk.Index, p, sb.String())
					}
				}
			}
			DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
		}
	}
}

// TestM4bDepartDeComposant cherche, pour le record du slot M4B_SLOT dans le paquet M4B_PAQUETS
// (un seul), la position de depart du composant M4B_DE d ou la boucle de composants finit
// EXACTEMENT sur M4B_FIN (la vraie frontiere du record, lue par TestM4bVraieSuite), et publie les
// comptes de grenades lus (4 attendus : `unit_weaponstate.go`).
func TestM4bDepartDeComposant(t *testing.T) {
	var c0, p0 int
	fmt.Sscanf(os.Getenv("M4B_PAQUETS"), "%d:%d", &c0, &p0) //nolint:errcheck // instrument
	slot, _ := strconv.Atoi(os.Getenv("M4B_SLOT"))
	de, _ := strconv.Atoi(os.Getenv("M4B_DE"))
	finVraie, _ := strconv.Atoi(os.Getenv("M4B_FIN"))
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		w.PoserChunkCourant(c)
		lierLeChunkAuMonde(w, marche, data, pks, obs)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = m4bDebut(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			if c != c0 || pk.Index != p0 {
				continue
			}
			for _, r := range recs {
				if int(r.Slot) != slot {
					continue
				}
				arch, _ := w.Reg.Archetype(int(r.TypeIndex))
				lu := -1
				for _, cr := range r.Trace.Comps {
					if cr.Index == de {
						lu = cr.StartBit
					}
				}
				t.Logf("record %d/%d masque %x : composant %d lu a %d", r.Slot, r.TypeIndex, r.Trace.Mask, de, lu)
				for q := lu - 120; q <= lu+120; q++ {
					var comptes []uint64
					o2 := NouvelleObservation()
					o2.GrenadeCountsHook = func(n uint64, _ []uint64) { comptes = append(comptes, n) }
					br := LecteurSur(pay)
					br.poserCadre(cfg)
					br.PoserObservation(o2)
					br.SetBitPos(q)
					tr := EntityTrace{Mask: r.Trace.Mask, DesyncAt: -1}
					traverseComponentLoopFrom(br, arch, &tr, de)
					if br.BitPos() == finVraie {
						t.Logf("   depart %d (ecart %+d) : fin %d desync %d comptes %v", q, q-lu, br.BitPos(), tr.DesyncAt, comptes)
					}
				}
			}
		}
	}
}
