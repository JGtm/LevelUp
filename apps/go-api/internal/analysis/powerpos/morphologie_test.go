package powerpos

import (
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

func cellulesDe(paires ...[2]int) []tactical.Cellule {
	out := make([]tactical.Cellule, 0, len(paires))
	for _, p := range paires {
		out = append(out, tactical.Cellule{Col: p[0], Lig: p[1]})
	}
	return out
}

func contientToutes(ensemble []tactical.Cellule, voulues []tactical.Cellule) bool {
	index := map[tactical.Cellule]bool{}
	for _, c := range ensemble {
		index[c] = true
	}
	for _, v := range voulues {
		if !index[v] {
			return false
		}
	}
	return true
}

// TestFermetureRayonNulEstIdentite : sans rayon, l'ensemble est rendu tel quel (trie).
func TestFermetureRayonNulEstIdentite(t *testing.T) {
	in := cellulesDe([2]int{5, 5}, [2]int{1, 1}, [2]int{3, 3})
	got := Fermeture(in, 0)
	if len(got) != 3 || got[0] != (tactical.Cellule{Col: 1, Lig: 1}) || got[2] != (tactical.Cellule{Col: 5, Lig: 5}) {
		t.Errorf("fermeture de rayon 0 = %v, attendu l'entree triee", got)
	}
	if in[0] != (tactical.Cellule{Col: 5, Lig: 5}) {
		t.Error("l'entree a ete reordonnee : effet de bord cache")
	}
}

// TestFermetureContientLEntree : la fermeture ne fait qu'AJOUTER.
func TestFermetureContientLEntree(t *testing.T) {
	in := cellulesDe([2]int{0, 0}, [2]int{2, 0}, [2]int{0, 2}, [2]int{7, 7})
	got := Fermeture(in, 1)
	if !contientToutes(got, in) {
		t.Errorf("la fermeture a perdu une cellule d'entree : %v", got)
	}
}

// TestFermetureRempliUnTrou : un semis en damier (une cellule sur deux) redevient un
// pave plein — c'est le defaut mesure au verdict v1 (amas de 2 a 10 cellules pour un lieu
// de quelques metres carres).
func TestFermetureRempliUnTrou(t *testing.T) {
	var damier []tactical.Cellule
	for c := 0; c < 6; c++ {
		for l := 0; l < 6; l++ {
			if (c+l)%2 == 0 {
				damier = append(damier, tactical.Cellule{Col: c, Lig: l})
			}
		}
	}
	got := Fermeture(damier, 1)
	// Le pave interieur [1,4] x [1,4] doit etre entierement rempli.
	var interieur []tactical.Cellule
	for c := 1; c <= 4; c++ {
		for l := 1; l <= 4; l++ {
			interieur = append(interieur, tactical.Cellule{Col: c, Lig: l})
		}
	}
	if !contientToutes(got, interieur) {
		t.Errorf("le damier n'est pas referme : %d cellules, interieur incomplet", len(got))
	}
	comps := Composantes(got)
	if len(comps) != 1 {
		t.Errorf("composantes 4-connexes apres fermeture = %d, attendu 1", len(comps))
	}
}

// TestFermetureGardeDistinctsDeuxAmasEloignes : deux paves separes de plus de 2r cellules
// restent deux composantes.
func TestFermetureGardeDistinctsDeuxAmasEloignes(t *testing.T) {
	var in []tactical.Cellule
	for c := 0; c < 3; c++ {
		for l := 0; l < 3; l++ {
			in = append(in, tactical.Cellule{Col: c, Lig: l}, tactical.Cellule{Col: c + 6, Lig: l})
		}
	}
	got := Fermeture(in, 1)
	if comps := ComposantesConnexite(got, true); len(comps) != 2 {
		t.Errorf("composantes 8-connexes apres fermeture = %d, attendu 2 (ecart de 3 cellules > 2r = 2)", len(comps))
	}
}

// TestFermetureSoudeDeuxAmasProches : deux paves separes d'une cellule sont soudes a
// rayon 1 — c'est le comportement voulu (un trou d'une cellule n'est pas un mur).
func TestFermetureSoudeDeuxAmasProches(t *testing.T) {
	var in []tactical.Cellule
	for c := 0; c < 3; c++ {
		for l := 0; l < 3; l++ {
			in = append(in, tactical.Cellule{Col: c, Lig: l}, tactical.Cellule{Col: c + 4, Lig: l})
		}
	}
	got := Fermeture(in, 1)
	if comps := Composantes(got); len(comps) != 1 {
		t.Errorf("composantes apres fermeture = %d, attendu 1 (trou d'une cellule refermé)", len(comps))
	}
}

// TestFermetureVide : rien n'entre, rien ne sort, aucune panique.
func TestFermetureVide(t *testing.T) {
	if got := Fermeture(nil, 2); len(got) != 0 {
		t.Errorf("fermeture du vide = %v", got)
	}
}
