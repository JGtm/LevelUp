package powerpos

import (
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

func cel(col, lig int) tactical.Cellule { return tactical.Cellule{Col: col, Lig: lig} }

func TestComposantesVideEtSingleton(t *testing.T) {
	if got := Composantes(nil); len(got) != 0 {
		t.Errorf("Composantes(nil) = %v, attendu vide", got)
	}
	got := Composantes([]tactical.Cellule{cel(4, 4)})
	if len(got) != 1 || len(got[0]) != 1 || got[0][0] != cel(4, 4) {
		t.Errorf("Composantes(un singleton) = %v", got)
	}
}

// TestComposantesSeparentLesDiagonales : c'est LA difference avec le 8-voisinage de
// `tactical.GrappesDeSpawn`, et elle est voulue (cf. composantes.go).
func TestComposantesSeparentLesDiagonales(t *testing.T) {
	got := Composantes([]tactical.Cellule{cel(0, 0), cel(1, 1)})
	if len(got) != 2 {
		t.Fatalf("composantes = %d, attendu 2 (deux cellules qui ne se touchent que par un coin)", len(got))
	}
}

func TestComposantesRegroupentParArete(t *testing.T) {
	entree := []tactical.Cellule{
		cel(0, 0), cel(1, 0), cel(1, 1), cel(1, 2), // un L connexe par aretes
		cel(10, 10), cel(10, 11), // une seconde composante, loin
		cel(20, 20), // un isole
	}
	got := Composantes(entree)
	if len(got) != 3 {
		t.Fatalf("composantes = %d, attendu 3 : %v", len(got), got)
	}
	// Tri par taille decroissante.
	if len(got[0]) != 4 || len(got[1]) != 2 || len(got[2]) != 1 {
		t.Fatalf("tailles = %d/%d/%d, attendu 4/2/1", len(got[0]), len(got[1]), len(got[2]))
	}
	// Chaque composante est triee par colonne puis ligne.
	for _, comp := range got {
		for i := 1; i < len(comp); i++ {
			if !avantCellule(comp[i-1], comp[i]) {
				t.Fatalf("composante non triee : %v", comp)
			}
		}
	}
}

// TestComposantesDeterministes : deux ordres d'entree differents rendent exactement le
// meme decoupage, sans quoi un catalogue cuit deux fois ne serait pas le meme fichier.
func TestComposantesDeterministes(t *testing.T) {
	a := []tactical.Cellule{cel(0, 0), cel(0, 1), cel(5, 5), cel(5, 6), cel(5, 7)}
	b := []tactical.Cellule{cel(5, 7), cel(0, 1), cel(5, 5), cel(0, 0), cel(5, 6)}
	ga, gb := Composantes(a), Composantes(b)
	if len(ga) != len(gb) {
		t.Fatalf("nombre de composantes instable : %d puis %d", len(ga), len(gb))
	}
	for i := range ga {
		if len(ga[i]) != len(gb[i]) {
			t.Fatalf("composante %d de taille %d puis %d", i, len(ga[i]), len(gb[i]))
		}
		for j := range ga[i] {
			if ga[i][j] != gb[i][j] {
				t.Fatalf("composante %d, cellule %d : %v puis %v", i, j, ga[i][j], gb[i][j])
			}
		}
	}
}

// TestComposantesDedoublonnentLEntree : une cellule presente deux fois n'appartient qu'a
// une composante et ne s'y compte qu'une fois.
func TestComposantesDedoublonnentLEntree(t *testing.T) {
	got := Composantes([]tactical.Cellule{cel(2, 2), cel(2, 2), cel(2, 3)})
	if len(got) != 1 {
		t.Fatalf("composantes = %d, attendu 1", len(got))
	}
	if len(got[0]) != 2 {
		t.Errorf("taille = %d, attendu 2", len(got[0]))
	}
}
