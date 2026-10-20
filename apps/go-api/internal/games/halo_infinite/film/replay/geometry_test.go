package replay

// geometry_test.go — LA DISTANCE 3D DU PAQUET, ET LE GARDE-RAIL QUI L EMPECHE DE SE RECOPIER.
//
// D OU IL VIENT (correctif de revue du 2026-08-17). La formule euclidienne 3D etait ecrite
// QUATRE fois dans ce paquet : `gwPadsDist` (six parametres), `equipDist`, une copie en ligne
// dans `equipmentOrigin`, et `origineDist` dans un instrument. Les quatre mesuraient la meme
// chose ; deux d entre elles servaient le MEME seuil (`originDropMaxDist`), de sorte qu un
// correctif porte d un seul cote aurait classe differemment la meme pose selon le chemin.
//
// LA REGLE N°6 DU DEPOT : a la troisieme copie on centralise ET on pose un garde-rail, sans quoi
// la dette re-croit (lecon chiffree : predicat bot passe de 8 a 36 copies APRES centralisation).

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDist3MesureLaDistanceEuclidienne : la brique elle-meme, sur des cas ou la reponse est
// connue d avance — un garde-rail qui protege une fonction fausse ne vaut rien.
func TestDist3MesureLaDistanceEuclidienne(t *testing.T) {
	for _, c := range []struct {
		nom  string
		a, b [3]float32
		want float64
	}{
		{"points confondus", [3]float32{1, 2, 3}, [3]float32{1, 2, 3}, 0},
		{"triplet 3-4-5 dans le plan", [3]float32{0, 0, 0}, [3]float32{3, 4, 0}, 5},
		{"altitude seule", [3]float32{7, -2, 10}, [3]float32{7, -2, 4}, 6},
		{"symetrie", [3]float32{3, 4, 0}, [3]float32{0, 0, 0}, 5},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := dist3(c.a, c.b); math.Abs(got-c.want) > 1e-6 {
				t.Fatalf("dist3(%v, %v) = %v, attendu %v", c.a, c.b, got, c.want)
			}
		})
	}
}

// TestUneSeuleFormuleDeDistance3D : la formule ne s ecrit qu une fois, dans `geometry.go`.
//
// LE MOTIF VISE LA DISTANCE, PAS LE CARRE : `grapple_lines.go` compare des distances AU CARRE
// pour trouver un argmin (aucune racine, aucun seuil en metres) — ce n est pas la meme grandeur
// et ce n est pas une copie. Le test couvre aussi les `_test.go`, parce que la copie qui
// divergerait viendrait d un instrument : c est exactement d ou venait `origineDist`.
func TestUneSeuleFormuleDeDistance3D(t *testing.T) {
	const owner = "geometry.go"
	pattern := regexp.MustCompile(`math\.Sqrt\(\s*(float64\()?\s*dx\s*\*\s*dx\s*\+\s*dy\s*\*\s*dy\s*\+\s*dz\s*\*\s*dz`)
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	var offenders []string
	seenOwner := false
	for _, e := range entries {
		file := e.Name()
		if e.IsDir() || !strings.HasSuffix(file, ".go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatalf("lecture %s : %v", file, err)
		}
		if !pattern.Match(raw) {
			continue
		}
		if file == owner {
			seenOwner = true
			continue
		}
		offenders = append(offenders, file)
	}
	if !seenOwner {
		t.Fatalf("la formule a disparu de %s : le garde-rail ne verifie plus rien (dist3 renommee"+
			" ou deplacee ?)", owner)
	}
	if len(offenders) > 0 {
		t.Fatalf("la distance 3D est REECRITE hors de %s : %v. Le paquet n en a qu UNE ecriture"+
			" (`dist3`) ; un appelant qui a besoin d autres types ecrit un ADAPTATEUR d une ligne,"+
			" jamais la formule", owner, offenders)
	}
}
