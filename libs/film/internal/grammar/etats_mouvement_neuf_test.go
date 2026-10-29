package grammar

import "testing"

// etats_mouvement_neuf_test.go — UN RECORD NEW PUBLIE SES ETATS SOUS SA PROPRE VIE (GA1-3, lot J5.4
// du plan de suite de l'audit du decodeur, 2026-09-27 ; leve la note D13 de la note 5.3).

// composantVitesse : le composant de vitesse de translation, qui publie un etat de mouvement
// (`EtatVitesse`) a chaque lecture.
const composantVitesse = "object-translational-velocity-dynamic-precision-component"

// neufAvecVitesse ecrit un record NEW de l'archetype `ti` dont le masque clairseme annonce le
// composant 0, lu par le chemin delta « absent » (R(1)=0 puis porte R(1)=1 : deux bits).
func (w *bitWriter) neufAvecVitesse(slot uint32, tag, ti uint64) {
	w.bit(0)
	w.bits(recNew, 2)
	w.bits(uint64(slot), 11)
	w.bits(tag, 2)
	w.bits(ti, 6)
	w.bit(0)     // porte du record NEW
	w.bit(0)     // masque clairseme
	w.bits(1, 3) // un composant
	w.bits(0, 6) // d'index 0
	w.bit(0)     // vitesse : chemin delta
	w.bit(1)     // porte : composante absente
}

// deltaVide ecrit un record DELTA a masque vide, SELECTEUR DE BASE compris (`decodeDelta` le lit
// avant le masque ; `deltaEmpty` ne l ecrit pas, et ne vaut donc qu en fin de trame).
func (w *bitWriter) deltaVide(slot uint32) {
	w.bit(1)
	w.bits(uint64(slot), 11)
	w.bits(0, 2)
	w.bit(0)     // selecteur de base
	w.bit(0)     // masque clairseme
	w.bits(0, 3) // aucun composant
}

// TestEtatsDeMouvement_RecordNEWPublieSousSaPropreVie : la boucle d'inference (`decodeInferLoop`,
// celle des marches de production) pose le slot de capture de CHAQUE record, NEW compris.
//
// ROUGE AVANT : seul le delta posait le slot (dans `decodeDelta`) ; la lecture du NEW du slot 300
// etait publiee sous le slot 124 du record precedent (D13 de la note 5.3 : 3 035 des 7 947
// lectures d'`i29` de `bfecd02b` tombaient ainsi sur le slot 0).
//
// MUTATION : retirer `br.poserSlotDeCapture(slot)` de la boucle -> la lecture tombe sous 124, rouge.
func TestEtatsDeMouvement_RecordNEWPublieSousSaPropreVie(t *testing.T) {
	reg := &Registry{Archetypes: []Archetype{{Index: 0}, {Index: 1},
		{Index: 2, Components: []string{composantVitesse}}, {Index: 3}, {Index: 4}}}
	w := NewWorld(reg)
	w.BindImageCle(1, 124, 4) // le record precedent : un delta d'un slot vivant

	var bw bitWriter
	bw.deltaVide(124)
	bw.neufAvecVitesse(300, 1, 2)
	bw.end()

	var slots []uint32
	prev := emptyCfg.Obs
	emptyCfg.Obs = NouvelleObservation()
	emptyCfg.Obs.EtatMouvementHook = func(c EtatMouvementComposant, slot uint32, _ []uint64) {
		if c == EtatVitesse {
			slots = append(slots, slot)
		}
	}
	defer func() { emptyCfg.Obs = prev }()
	withChain(false, func() {
		recs, _ := DecodeFrameInfer(bw.buf, w, emptyCfg)
		if len(recs) != 2 || recs[1].DesyncAt != -1 {
			t.Fatalf("records %+v : attendu le delta et le NEW, lus proprement", recs)
		}
	})
	if len(slots) != 1 || slots[0] != 300 {
		t.Fatalf("lectures de vitesse publiees sous %v, attendu [300] — le NEW publie sous le slot "+
			"du record precedent", slots)
	}
}
