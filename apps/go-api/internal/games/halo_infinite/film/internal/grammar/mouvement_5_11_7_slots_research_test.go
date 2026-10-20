//go:build research

package grammar

import "testing"

// mouvement_5_11_7_slots_research_test.go — LE CONTROLE DE LA DECOUPE DU PIED (lot 5.11.7-c).
//
// Les deux en-tetes de rejet du pied se decoupent `[prefixe 1][idLow][tag 2]`. SI LA DECOUPE EST
// JUSTE, les slots qu ils nomment doivent exister quelque part. Ce test les confronte au monde
// amorce par les IMAGES-CLES, et le resultat separe les deux en-tetes :
//
//	VUE 2 — ses slots sont LIES (26 et 19 sur `dad793c7`, tous deux `ti=6` statborg). C est un
//	  vrai en-tete de record, et la decoupe est confirmee.
//	VUE 3 — ses slots (7136, 7140, 3968, 4161, 2194, 8191) ne sont JAMAIS lies. Ce n est pas une
//	  refutation : le monde hors ligne attribue TOUTES les liaisons d image-cle a la vue 0
//	  (lot 5.11.7-b), donc il ne modelise AUCUNE entite de vue 3. L identite de l entite que la
//	  vue 3 nomme reste hors de portee tant que les images-cles ne sont pas ventilees par vue.
//
// C est la limite exacte de la transcription actuelle, et elle se mesure ici.
func TestMouvement5117SlotsDuPied(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	w := NewWorld(reg)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
	}
	t.Logf("MONDE AMORCE PAR LES IMAGES-CLES : %d slots lies", w.Bound())
	for _, s := range []uint32{26, 19, 7136, 7140, 3968, 4161, 2194, 8191, 123, 512} {
		ti, ok := w.ArchetypeForSlot(s)
		if ok {
			t.Logf("  slot %5d : LIE, archetype ti=%d", s, ti)
		} else {
			t.Logf("  slot %5d : INCONNU du monde", s)
		}
	}
}
