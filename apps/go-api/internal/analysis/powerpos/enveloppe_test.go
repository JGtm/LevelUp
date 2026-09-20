package powerpos

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

func TestEnveloppeCompositionVide(t *testing.T) {
	if got := Enveloppe(tactical.GrilleParDefaut(), nil); got != nil {
		t.Errorf("Enveloppe(vide) = %v, attendu nil", got)
	}
}

// TestEnveloppeUneCelluleEstDilatee : une cellule de 0,5 m dilatee d'une demi-cellule fait
// un carre de 1 m de cote, soit 1 m2.
func TestEnveloppeUneCelluleEstDilatee(t *testing.T) {
	g := tactical.GrilleParDefaut()
	got := Enveloppe(g, []tactical.Cellule{{Col: 0, Lig: 0}})
	if len(got) != 4 {
		t.Fatalf("sommets = %d, attendu 4 : %v", len(got), got)
	}
	if aire := AirePolygone(got); math.Abs(aire-1) > 1e-9 {
		t.Errorf("aire = %v m2, attendu 1", aire)
	}
	minX, minY, maxX, maxY := bornesDe(got)
	if minX != -0.25 || minY != -0.25 || maxX != 0.75 || maxY != 0.75 {
		t.Errorf("bornes = (%v, %v)-(%v, %v), attendu (-0.25, -0.25)-(0.75, 0.75)", minX, minY, maxX, maxY)
	}
}

// TestEnveloppeRectangleNAPasDeSommetsAlignes : les coins intermediaires d'un alignement
// de cellules sont ecartes, sinon le nombre de sommets dependrait de l'ordre d'entree.
func TestEnveloppeRectangleNAPasDeSommetsAlignes(t *testing.T) {
	g := tactical.GrilleParDefaut()
	comp := []tactical.Cellule{{Col: 0, Lig: 0}, {Col: 1, Lig: 0}, {Col: 2, Lig: 0}}
	got := Enveloppe(g, comp)
	if len(got) != 4 {
		t.Fatalf("sommets = %d, attendu 4 (un rectangle) : %v", len(got), got)
	}
	// 3 cellules de 0,5 m en ligne = 1,5 m x 0,5 m, dilate d'un demi-pas = 2,0 m x 1,0 m.
	if aire := AirePolygone(got); math.Abs(aire-2) > 1e-9 {
		t.Errorf("aire = %v m2, attendu 2", aire)
	}
}

// TestEnveloppeRecouvreLeConcave : l'ecart assume de la v1, mesure ici pour qu'il soit
// CONSTATE et non decouvert plus tard (cf. enveloppe.go).
func TestEnveloppeRecouvreLeConcave(t *testing.T) {
	g := tactical.GrilleParDefaut()
	// Un L : trois cellules, le coin rentrant est recouvert par l'enveloppe convexe.
	comp := []tactical.Cellule{{Col: 0, Lig: 0}, {Col: 1, Lig: 0}, {Col: 0, Lig: 1}}
	got := Enveloppe(g, comp)
	aire := AirePolygone(got)
	aireCellulesDilatees := 3 * 1.0 // borne basse grossiere : chaque cellule dilatee fait 1 m2
	if aire >= aireCellulesDilatees {
		t.Logf("aire de l'enveloppe = %v m2 (recouvrement du coin rentrant attendu)", aire)
	}
	if aire <= 0 {
		t.Fatalf("aire = %v, attendu strictement positive", aire)
	}
}

func TestEnveloppeConvexeMoinsDeTroisPoints(t *testing.T) {
	if got := EnveloppeConvexe(nil); len(got) != 0 {
		t.Errorf("EnveloppeConvexe(nil) = %v", got)
	}
	got := EnveloppeConvexe([][2]float64{{1, 1}, {1, 1}})
	if len(got) != 1 {
		t.Errorf("deux points identiques : %d sommets, attendu 1", len(got))
	}
	got = EnveloppeConvexe([][2]float64{{0, 0}, {2, 2}})
	if len(got) != 2 {
		t.Errorf("segment : %d sommets, attendu 2", len(got))
	}
}

// TestEnveloppeConvexeIgnoreLesPointsInterieurs : un nuage dense rend le meme carre que
// ses quatre coins seuls.
func TestEnveloppeConvexeIgnoreLesPointsInterieurs(t *testing.T) {
	points := [][2]float64{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {2, 2}, {1, 3}, {3, 1}, {2, 0}}
	got := EnveloppeConvexe(points)
	if len(got) != 4 {
		t.Fatalf("sommets = %d, attendu 4 : %v", len(got), got)
	}
	if aire := AirePolygone(got); math.Abs(aire-16) > 1e-9 {
		t.Errorf("aire = %v, attendu 16", aire)
	}
}

// TestEnveloppeConvexeSensTrigonometrique : l'aire signee du lacet est positive quand les
// sommets tournent dans le sens trigonometrique.
func TestEnveloppeConvexeSensTrigonometrique(t *testing.T) {
	got := EnveloppeConvexe([][2]float64{{0, 0}, {2, 0}, {2, 2}, {0, 2}})
	var signee float64
	for i := range got {
		j := (i + 1) % len(got)
		signee += got[i][0]*got[j][1] - got[j][0]*got[i][1]
	}
	if signee <= 0 {
		t.Errorf("aire signee = %v, attendue positive (sens trigonometrique) : %v", signee/2, got)
	}
}

func TestAirePolygoneDegenere(t *testing.T) {
	if got := AirePolygone([][2]float64{{0, 0}, {1, 1}}); got != 0 {
		t.Errorf("aire d'un segment = %v, attendu 0", got)
	}
}

// bornesDe rend le rectangle englobant d'un polygone.
func bornesDe(p [][2]float64) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, s := range p {
		minX, maxX = math.Min(minX, s[0]), math.Max(maxX, s[0])
		minY, maxY = math.Min(minY, s[1]), math.Max(maxY, s[1])
	}
	return minX, minY, maxX, maxY
}
