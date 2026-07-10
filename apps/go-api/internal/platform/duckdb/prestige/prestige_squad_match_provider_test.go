package prestige

import "testing"

// TestTopNByFreq couvre la sélection des labels dominants de l'indice escouade :
// fréquence décroissante, départage alphabétique déterministe, bornage n.
func TestTopNByFreq(t *testing.T) {
	counts := map[string]int{"Ranked": 5, "BTB": 3, "Fiesta": 5, "Slayer": 1}
	got := topNByFreq(counts, 2)
	// Top 2 par fréquence (Fiesta=5, Ranked=5) ; tie cassé par ordre alpha.
	if len(got) != 2 || got[0] != "Fiesta" || got[1] != "Ranked" {
		t.Fatalf("topNByFreq = %v, want [Fiesta Ranked]", got)
	}

	// n > len → toutes les clés.
	all := topNByFreq(map[string]int{"a": 2}, 5)
	if len(all) != 1 || all[0] != "a" {
		t.Errorf("topNByFreq(n>len) = %v, want [a]", all)
	}

	// Map vide → slice vide.
	if empty := topNByFreq(map[string]int{}, 3); len(empty) != 0 {
		t.Errorf("topNByFreq(empty) = %v, want []", empty)
	}
}

// TestUUIDLabelRE vérifie que l'indice écarte les labels non résolus (UUID brut)
// mais garde les libellés normaux (y compris accentués).
func TestUUIDLabelRE(t *testing.T) {
	if !uuidLabelRE.MatchString("0123abcd-0123-0123-0123-0123456789ab") {
		t.Error("UUID non détecté (devrait être écarté de l'indice)")
	}
	if uuidLabelRE.MatchString("Classé") || uuidLabelRE.MatchString("Big Team Battle") {
		t.Error("libellé normal détecté à tort comme UUID")
	}
}
