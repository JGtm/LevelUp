package main

import "testing"

// TestFNV1VecteursDeReference — sans ce test, une fonction de hachage FAUSSE et une liste
// de candidats INCOMPLETE produisent le meme symptome (« aucun nom retrouve ») et on ne
// saurait pas laquelle corriger. Vecteurs standards FNV-1 32 bits (pas FNV-1a).
func TestFNV1VecteursDeReference(t *testing.T) {
	cas := []struct {
		entree string
		attend uint32
	}{
		{"", 0x811c9dc5},
		{"a", 0x050c5d7e},
		{"foobar", 0x31f0b262},
	}
	for _, c := range cas {
		if got := fnv1(c.entree); got != c.attend {
			t.Errorf("fnv1(%q) = %#08x, attendu %#08x", c.entree, got, c.attend)
		}
	}
}

// TestFNV1MetEnMinuscules — Wwise hache le nom en minuscules ; « Play_X » et « play_x »
// doivent donner le meme identifiant, sinon la moitie des candidats generes est perdue.
func TestFNV1MetEnMinuscules(t *testing.T) {
	if fnv1("Play_Wea_Fire") != fnv1("play_wea_fire") {
		t.Fatal("fnv1 n'est pas insensible a la casse")
	}
}

// TestCandidatsCouvreLeTir — garde-fou de la liste : la forme la plus attendue pour le tir
// doit etre generee pour une arme donnee.
func TestCandidatsCouvreLeTir(t *testing.T) {
	cands := candidatsNoms("un_assaultrifle")
	for _, attendu := range []string{
		"play_wea_un_assaultrifle_fire",
		"wea_un_assaultrifle_fire",
		"play_wea_assaultrifle_fire",
	} {
		if cands[fnv1(attendu)] == "" {
			t.Errorf("candidat absent de la generation : %q", attendu)
		}
	}
}
