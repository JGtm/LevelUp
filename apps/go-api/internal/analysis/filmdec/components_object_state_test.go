package filmdec

// components_object_state_test.go — LES LARGEURS D'i14 `object-dissolver-component` SOUS
// GARDE-RAIL (lot 6.10 bis, 2026-09-11).
//
// POURQUOI CE FICHIER EXISTE. La grammaire de `FUN_140dd9f9c` a ete relue instruction par
// instruction pour savoir si le composant portait un champ de FIN DE VIE (il n'en porte aucun
// de mesurable, cf. l'en-tete de `consumeObjectDissolver`). La relecture n'a change AUCUNE
// largeur — et c'est precisement ce qu'il faut verrouiller : sans test, la prochaine relecture
// n'aura aucun moyen de savoir que les quatre largeurs actuelles sont les bonnes.
//
// i14 est un composant d'objet du monde presente sur 67,1 % des creations `ti=42` : une largeur
// fausse ici decale tout ce qui suit dans le record.

import "testing"

// TestConsumeObjectDissolverLargeurs fige les DEUX chemins du composant.
func TestConsumeObjectDissolverLargeurs(t *testing.T) {
	// LES COMPTES SONT ECRITS EN CLAIR, JAMAIS A PARTIR DES CONSTANTES DU DECODEUR : un test
	// qui reutilise la constante qu'il verifie ne verifie rien. 113 = 4 (etat) + 96 (corps brut)
	// + 12 (duree dequantifiee) + 1 (drapeau).
	cas := []struct {
		nom  string
		etat uint64
		bits int
	}{
		{"etat neutre (13) : le corps est COUPE", 13, 4},
		{"etat 0 : corps complet", 0, 113},
		{"etat 8 : corps complet", 8, 113},
		{"etat 12 : corps complet", 12, 113},
		{"etat 14 : corps complet", 14, 113},
	}
	for _, c := range cas {
		w := &bitw{}
		w.put(c.etat, 4)
		for i := 0; i < 16; i++ {
			w.put(0xa5, 8)
		}
		br := NewBitReader(append(w.buf, make([]byte, 16)...))
		consumeObjectDissolver(br)
		if br.BitPos() != c.bits {
			t.Errorf("%s : %d bits consommes, %d attendus", c.nom, br.BitPos(), c.bits)
		}
	}
}

// TestConsumeObjectDissolverNeutreEstExclusif — LE TEMOIN, ecrit en RELATION plutot qu'en compte.
//
// Sur le MEME suffixe de flux, l'etat neutre consomme STRICTEMENT MOINS que tout autre etat.
// Deplacer la valeur neutre (13 -> autre chose) echange les deux et fait echouer l'assertion
// sans qu'aucun compte n'ait a etre recalcule a la main.
func TestConsumeObjectDissolverNeutreEstExclusif(t *testing.T) {
	cout := func(etat uint64) int {
		w := &bitw{}
		w.put(etat, 4)
		for i := 0; i < 16; i++ {
			w.put(0x5a, 8)
		}
		br := NewBitReader(append(w.buf, make([]byte, 16)...))
		consumeObjectDissolver(br)
		return br.BitPos()
	}
	neutre := cout(13)
	if neutre != 4 {
		t.Fatalf("etat neutre : %d bits consommes, 4 attendus — le corps est lu alors qu'il "+
			"devrait etre coupe (140dd9ffd : CMP R10D,0xd)", neutre)
	}
	for _, etat := range []uint64{0, 1, 7, 12, 14, 15} {
		if got := cout(etat); got <= neutre {
			t.Fatalf("etat %d : %d bits consommes, l'etat neutre en consomme %d — tout etat "+
				"non neutre doit STRICTEMENT en consommer plus", etat, got, neutre)
		}
	}
}
