package hinavmesh

// navmesh_test.go — TEMOINS DE STRUCTURE du maillage decode.
//
// L'oracle des ancres (oracle_ancres_test.go) prouve que le maillage est au bon endroit.
// Ces temoins-ci figent ce qu'il CONTIENT, pour qu'une regression de decodage se voie
// meme si elle laisse la geometrie plausible.

import (
	"math"
	"testing"
)

func TestMaillageIsolationConforme(t *testing.T) {
	m := decodeTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	verifieStructure(t, m, temoinStructure{
		sommets: 3350, faces: 2348,
		aireMin: 2150, aireMax: 2250,
	})
}

func TestMaillageKikennaConforme(t *testing.T) {
	m := decodeTemoin(t, "df7dbf08-b8de-4ade-9d7f-1947128c9ae4")
	verifieStructure(t, m, temoinStructure{
		sommets: 1367, faces: 689,
		aireMin: 600, aireMax: 650,
	})
}

type temoinStructure struct {
	sommets, faces   int
	aireMin, aireMax float64
}

func verifieStructure(t *testing.T, m *Maillage, veut temoinStructure) {
	t.Helper()
	if len(m.Sommets) != veut.sommets {
		t.Errorf("%d sommets, %d attendus", len(m.Sommets), veut.sommets)
	}
	if len(m.Faces) != veut.faces {
		t.Errorf("%d faces, %d attendues", len(m.Faces), veut.faces)
	}
	if aire := m.AireAuSol(); aire < veut.aireMin || aire > veut.aireMax {
		t.Errorf("aire au sol %.0f m2, attendue entre %.0f et %.0f", aire, veut.aireMin, veut.aireMax)
	}
	// Le repere du jeu est Z vers le haut : c'est ce qui autorise a projeter le fond dans
	// le plan XY. On le LIT dans le fichier plutot que de le supposer.
	if m.Haut.X != 0 || m.Haut.Y != 0 || m.Haut.Z != 1 {
		t.Errorf("vecteur haut (%.3f, %.3f, %.3f), (0, 0, 1) attendu", m.Haut.X, m.Haut.Y, m.Haut.Z)
	}
	// Les polygones d'un navmesh sont convexes, d'au moins 3 cotes, et tous leurs sommets
	// sont references : un sommet hors bornes aurait deja fait echouer le decodage.
	cotes := map[int]int{}
	for i, f := range m.Faces {
		if len(f.Sommets) < cotesFaceMinimum {
			t.Fatalf("face %d n'a que %d cotes", i, len(f.Sommets))
		}
		cotes[len(f.Sommets)]++
	}
	if cotes[3] == 0 {
		t.Errorf("aucune face triangulaire : la repartition des cotes est %v", cotes)
	}
	// L'eventail de triangulation doit rendre exactement somme(cotes - 2) triangles.
	var attendus int
	for n, k := range cotes {
		attendus += (n - 2) * k
	}
	if tris := m.Triangles(); len(tris) != attendus {
		t.Errorf("%d triangles, %d attendus pour la repartition %v", len(tris), attendus, cotes)
	}
	t.Logf("%d sommets, %d faces, %.0f m2, repartition des cotes %v, rayon d'erosion %.3f",
		len(m.Sommets), len(m.Faces), m.AireAuSol(), cotes, m.RayonErosion)
}

// TestEmpriseMesureeSurLesFacesPasSurLAabb : le fichier porte un hkAabb qui decrit le
// CANEVAS Forge (plus de 400 m de cote), pas l'aire jouable. L'emprise du maillage doit
// venir des sommets, sinon le fond de carte serait cadre sur le canevas et l'arene y
// occuperait quelques pixels.
func TestEmpriseMesureeSurLesFacesPasSurLAabb(t *testing.T) {
	m := decodeTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	largeur, profondeur := m.Max.X-m.Min.X, m.Max.Y-m.Min.Y
	if largeur > 100 || profondeur > 100 {
		t.Errorf("emprise de %.0f x %.0f m : c'est le canevas, pas l'arene", largeur, profondeur)
	}
	for _, f := range m.Faces {
		for _, p := range m.Contour(f) {
			if p.X < m.Min.X || p.X > m.Max.X || p.Y < m.Min.Y || p.Y > m.Max.Y {
				t.Fatalf("sommet (%.2f, %.2f) hors de l'emprise annoncee", p.X, p.Y)
			}
		}
	}
	t.Logf("Isolation : arene de %.1f x %.1f m sur %.1f m de denivele", largeur, profondeur, m.Max.Z-m.Min.Z)
}

// TestMaillageMultiNiveaux mesure le risque qui SUBSISTE apres ce chantier : le navmesh
// n'est pas un plan, c'est une surface a plusieurs planchers. Le rendu vu de dessus devra
// toujours arbitrer par z-buffer — mais entre des SOLS reels, sur une dizaine de metres,
// et non plus sous quarante metres de coques.
func TestMaillageMultiNiveaux(t *testing.T) {
	m := decodeTemoin(t, "01af558d-53ab-4f05-ba68-92d805fc6260")
	// Deux faces se superposent si leurs emprises XY se recouvrent et que leurs altitudes
	// medianes different de plus d'une hauteur de joueur.
	const hauteurJoueur = 2.0
	superposees := 0
	for i := range m.Faces {
		for j := i + 1; j < len(m.Faces); j++ {
			if !empriseSeRecouvre(m, i, j) {
				continue
			}
			if math.Abs(altitudeMediane(m, i)-altitudeMediane(m, j)) > hauteurJoueur {
				superposees++
			}
		}
	}
	if superposees == 0 {
		t.Errorf("aucun plancher superpose detecte : le denivele mesure est pourtant de %.1f m",
			m.Max.Z-m.Min.Z)
	}
	t.Logf("Isolation : %d paires de faces superposees de plus de %.0f m — le fond restera "+
		"un arbitrage z-buffer entre planchers", superposees, hauteurJoueur)
}

func empriseSeRecouvre(m *Maillage, i, j int) bool {
	aMin, aMax := empriseFace(m, i)
	bMin, bMax := empriseFace(m, j)
	return aMin.X < bMax.X && bMin.X < aMax.X && aMin.Y < bMax.Y && bMin.Y < aMax.Y
}

func empriseFace(m *Maillage, i int) (Point, Point) {
	inf := math.Inf(1)
	mn, mx := Point{inf, inf, inf}, Point{-inf, -inf, -inf}
	for _, p := range m.Contour(m.Faces[i]) {
		mn = Point{math.Min(mn.X, p.X), math.Min(mn.Y, p.Y), math.Min(mn.Z, p.Z)}
		mx = Point{math.Max(mx.X, p.X), math.Max(mx.Y, p.Y), math.Max(mx.Z, p.Z)}
	}
	return mn, mx
}

func altitudeMediane(m *Maillage, i int) float64 {
	var somme float64
	contour := m.Contour(m.Faces[i])
	for _, p := range contour {
		somme += p.Z
	}
	return somme / float64(len(contour))
}
