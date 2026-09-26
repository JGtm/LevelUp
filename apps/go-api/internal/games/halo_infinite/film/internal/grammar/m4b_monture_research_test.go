//go:build research

package grammar

// m4b_monture_research_test.go — LOT M4b : CE QUE LE FILM ECRIT D UNE MONTURE, dans une fenetre.
// Mesure seule. Pour les slots de bipede M4B_SLOTS et les vehicules M4B_VEHICULES, sur les trames
// M4B_FENETRES : les lectures d `object-parent-state` (i10) de la marche des morts, les evenements
// d embarquement / de sortie en tete de liste, et les rafales de l index S3_INDEX.
//
// Rejouable : memes variables que `m4b_tir_continu_research_test.go`, plus M4B_SLOTS,
// M4B_VEHICULES, M4B_FENETRES.

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// m4bListe lit une liste d entiers separes par des virgules.
func m4bListe(nom string) map[uint32]bool {
	out := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv(nom), ",") {
		var v uint32
		if _, err := fmt.Sscanf(s, "%d", &v); err == nil {
			out[v] = true
		}
	}
	return out
}

// TestM4bMonture publie ce que le film ecrit de la monture (en-tete du fichier).
func TestM4bMonture(t *testing.T) {
	cad := s3LireCadre(t)
	slots, vehs := m4bListe("M4B_SLOTS"), m4bListe("M4B_VEHICULES")
	var fen [][2]int
	for _, s := range strings.Split(os.Getenv("M4B_FENETRES"), ",") {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			fen = append(fen, [2]int{a, b})
		}
	}
	tc := t516Cadre(t)
	mf, err := ScanMarchFacts(tc.fc)
	if err != nil {
		t.Fatalf("ScanMarchFacts : %v", err)
	}
	for _, o := range mf.Occupancy {
		tr := cad.trame(o.TimestampUS)
		if !m4bDans(tr, fen) || (!slots[o.Slot] && !vehs[o.ParentSlot]) {
			continue
		}
		t.Logf("i10 t%d slot %d attache %v parent %d siege %v/%d", tr, o.Slot, o.Attached,
			o.ParentSlot, o.HasSeat, o.Seat)
	}
	evs, err := ScanVehicleEvents(tc.fc)
	if err != nil {
		t.Fatalf("ScanVehicleEvents : %v", err)
	}
	for _, e := range evs {
		tr := cad.trame(e.TimestampUS)
		if !m4bDans(tr, fen) {
			continue
		}
		t.Logf("evenement t%d type %d occupant %d (present %v) vehicule %d (valide %v) siege %d",
			tr, e.Kind, e.OccupantSlot, e.OccupantPresent, e.VehicleSlot, e.VehicleSlotValid, e.Seat)
	}
	m, err := ScanMarcheDesTrames(tc.fc)
	if err != nil {
		t.Fatalf("ScanMarcheDesTrames : %v", err)
	}
	for _, r := range m.ContinuousFire {
		if r.FilmIndex != cad.index || !m4bDans(cad.trame(r.StartUS), fen) {
			continue
		}
		t.Logf("rafale t%d..%d arme %d trous %d", cad.trame(r.StartUS), cad.trame(r.EndUS), r.Weapon,
			len(r.Holes))
	}
}

// TestM4bMontureMarcheDesTrames cherche les lectures d `object-parent-state` des slots M4B_SLOTS
// dans la MARCHE DU FRAME-PROCESSEUR (celle des etats de mouvement et du tir continu), qui ne les
// recolte pas en production : la marche des morts les lit seule.
func TestM4bMontureMarcheDesTrames(t *testing.T) {
	cad := s3LireCadre(t)
	slots := m4bListe("M4B_SLOTS")
	var fen [][2]int
	for _, s := range strings.Split(os.Getenv("M4B_FENETRES"), ",") {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			fen = append(fen, [2]int{a, b})
		}
	}
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	obs := NouvelleObservation()
	cfg := tc.cfg
	cfg.Obs = obs
	marche := tc.fc.MarcheDImageCle()
	vus := 0
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
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _, _ := DecodeFrameViewsCurseur(pay, w, cfg, MovementStateViews, debut)
			for i := range recs {
				o, ok := occupancyFromRecord(&recs[i], pk.TimestampUS)
				if !ok {
					continue
				}
				vus++
				if slots[o.Slot] && m4bDans(cad.trame(pk.TimestampUS), fen) {
					t.Logf("i10 (marche des trames) t%d slot %d attache %v parent %d siege %v/%d",
						cad.trame(pk.TimestampUS), o.Slot, o.Attached, o.ParentSlot, o.HasSeat, o.Seat)
				}
			}
		}
	}
	t.Logf("lectures i10 de la bande bipede par la marche des trames : %d", vus)
}

// TestM4bMontureImagesCles lit, a chaque IMAGE-CLE de la fenetre, l etat complet des bipedes de
// M4B_SLOTS et en rend `object-parent-state` : l image-cle porte l etat entier, pas une transition.
func TestM4bMontureImagesCles(t *testing.T) {
	cad := s3LireCadre(t)
	slots := m4bListe("M4B_SLOTS")
	var fen [][2]int
	for _, s := range strings.Split(os.Getenv("M4B_FENETRES"), ",") {
		var a, b int
		if _, err := fmt.Sscanf(s, "%d-%d", &a, &b); err == nil {
			fen = append(fen, [2]int{a, b})
		}
	}
	tc := t516Cadre(t)
	marche := tc.fc.MarcheDImageCle()
	ctx := tc.fc.ContexteDeLecture()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			tr := cad.trame(pk.TimestampUS)
			if !m4bDans(tr, fen) {
				continue
			}
			pay := pk.Payload(data)
			recs := marche.Records(pay)
			t.Logf("image-cle c%d t%d : %d records", c, tr, len(recs))
			for _, r := range recs {
				if !slots[uint32(r.Slot)] { //nolint:gosec // slot borne
					continue
				}
				tra := WalkKeyframeFullState(pay, r.Bit, tc.reg, ctx)
				parent := "aucune lecture i10"
				for _, cr := range tra.Comps {
					if st, ok := cr.ParentOf(); ok {
						parent = fmt.Sprintf("attache %v quant %d (slot %d) queue6 %v/%d", st.Attached,
							st.Quant16&parentHandleValueMask,
							(st.Quant16&parentHandleValueMask)+parentHandleBase, st.HasTail6, st.Tail6)
					}
				}
				t.Logf("   slot %d ti %d gen %d desync %d : %s", r.Slot, r.TI, r.Gen, tra.DesyncAt, parent)
			}
		}
	}
}
