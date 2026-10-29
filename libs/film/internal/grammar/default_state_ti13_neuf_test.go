package grammar

// default_state_ti13_neuf_test.go — l exception datee du lot J6-bis (2026-09-28) : le record NEW
// de ti=13 lit son etat par defaut SANS la charge des variants, l image-cle AVEC.

import "testing"

// registreTI13 rend un registre dont l archetype 13 porte ses deux premiers composants.
func registreTI13() *Registry {
	archs := make([]Archetype, archetypeProprieteGeree+1)
	for i := range archs {
		archs[i] = Archetype{Index: i}
	}
	archs[archetypeProprieteGeree].Components = []string{"managed-object-property-name-component",
		compManagedObjectProperty}
	return &Registry{Archetypes: archs}
}

// TestLeRecordNeufDeTI13LitLEtatParDefautDAvant : `fb1a1a72` chunk 34 paquet 568 et `60ae07c4`
// chunks 10 et 33 ne se ferment que sous cette lecture — l etiquette 3 d un variant en mode A,
// SANS ses 24 bits, puis la porte et le masque. Lue comme le jeu (etiquette + R(24)), la
// traversee consomme la porte, le masque et le marqueur : ROUGE.
func TestLeRecordNeufDeTI13LitLEtatParDefautDAvant(t *testing.T) {
	w := &bitw{}
	w.put(archetypeProprieteGeree, 6)
	w.put(0, 1)           // version absente
	w.put(0x20010f22, 32) // "propertyName"
	w.put(0, 1)           // g = 0 : un variant, mode A
	w.put(ManagedPropertyTagQuant, 4)
	w.put(0, 1) // porte du record NEW
	w.put(0, 1) // masque clairseme
	w.put(0, 3) // aucun composant
	fin := 6 + 1 + 32 + 1 + 4 + 1 + 1 + 3
	w.put(0x2a, 8)
	br := lecteurDInstrument(append(w.buf, make([]byte, 16)...))
	tr := TraverseEntity(br, registreTI13(), 0)
	if tr.DesyncAt != -1 || tr.EndBit != fin {
		t.Fatalf("record NEW ti=13 : fin %d (desync %d), %d attendu", tr.EndBit, tr.DesyncAt, fin)
	}
	if m := br.ReadBits(8); m != 0x2a {
		t.Fatalf("marqueur %#x, 0x2a attendu", m)
	}
}

// TestLImageCleDeTI13GardeLaLectureDuJeu : l exception ne touche que le record NEW ; l etat par
// defaut que lit l image-cle ([defaultStateDeserByTI], `keyframe_record_walk.go`) garde la charge
// (GA2-4, 0 % -> ~100 % de fermeture d image-cle).
func TestLImageCleDeTI13GardeLaLectureDuJeu(t *testing.T) {
	fn, ok := defaultStateDeserByTI[archetypeProprieteGeree]
	if !ok {
		t.Fatal("ti=13 n a plus d etat par defaut d image-cle")
	}
	w := &bitw{}
	w.put(0, 1)
	w.put(0x20010f22, 32)
	w.put(0, 1)
	w.put(ManagedPropertyTagQuant, 4)
	w.put(0x5a5a5a, 24)
	br := lecteurDInstrument(append(w.buf, make([]byte, 16)...))
	fn(br)
	if got := br.BitPos(); got != 1+32+1+4+24 {
		t.Fatalf("etat par defaut ti=13 d image-cle : %d bits lus, %d attendus", got, 1+32+1+4+24)
	}
}
