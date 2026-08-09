package himap

import (
	"math"
	"testing"
)

// grille rend un carre de `cote` metres decoupe en n x n cases, chacune en deux triangles.
// Le nombre de subdivisions fixe donc la FINESSE, a emprise au sol constante — c'est
// exactement le couple que le tri doit distinguer.
func grille(cote float64, n int) *Mesh {
	m := &Mesh{}
	pas := cote / float64(n)
	idx := func(i, j int) int { return j*(n+1) + i }
	for j := 0; j <= n; j++ {
		for i := 0; i <= n; i++ {
			m.Vertices = append(m.Vertices, [3]float64{float64(i) * pas, float64(j) * pas, 0})
		}
	}
	for j := 0; j < n; j++ {
		for i := 0; i < n; i++ {
			m.Triangles = append(m.Triangles,
				[3]int{idx(i, j), idx(i+1, j), idx(i+1, j+1)},
				[3]int{idx(i, j), idx(i+1, j+1), idx(i, j+1)})
		}
	}
	return m
}

// TestAireMedianeLitLaFinessePasLaTaille — a EMPRISE EGALE, l'aire mediane doit suivre la
// subdivision. C'est ce qui rend le critere transposable : il ne mesure pas si l'objet est
// grand, il mesure a quel grain il est modelise.
func TestAireMedianeLitLaFinessePasLaTaille(t *testing.T) {
	in := instanceIdentite()
	// 20 m de cote en 200 subdivisions -> cases de 10 cm -> triangles de 50 cm2.
	fin := AireMedianeProjetee(grille(20, 200), in)
	// Meme emprise, 4 subdivisions -> cases de 5 m -> triangles de 12,5 m2.
	grossier := AireMedianeProjetee(grille(20, 4), in)
	if math.Abs(fin-0.005) > 1e-6 {
		t.Fatalf("maillage fin : aire mediane %.6f m2, attendu 0,005", fin)
	}
	if math.Abs(grossier-12.5) > 1e-6 {
		t.Fatalf("maillage grossier : aire mediane %.4f m2, attendu 12,5", grossier)
	}
	if fin >= grossier {
		t.Fatal("le critere doit croitre avec la GROSSIERETE, pas avec la taille de l'objet")
	}
}

// TestMedianeResistgeAuSocle — LE temoin de la mediane contre la moyenne.
//
// Il existe parce que le premier jeu de temoins ne separait PAS : sur des grilles uniformes,
// moyenne et mediane coincident, et la mutation « mediane -> moyenne » passait au vert. Le
// cas qui separe est celui que la doc de `AireMedianeProjetee` decrit — un sol finement
// maille qui porte quelques grandes faces de socle.
//
// MUTATION QUI LE FAIT ROUGIR : remplacer la mediane par la moyenne dans
// `AireMedianeProjetee` (verifie le 2026-08-09).
func TestMedianeResisteAuSocle(t *testing.T) {
	in := instanceIdentite()
	m := grille(20, 200) // 80 000 triangles de 50 cm2
	// Deux faces de socle de 5 000 m2 chacune : negligeables en nombre, ecrasantes en aire.
	base := len(m.Vertices)
	m.Vertices = append(m.Vertices,
		[3]float64{-50, -50, -1}, [3]float64{50, -50, -1},
		[3]float64{50, 50, -1}, [3]float64{-50, 50, -1})
	m.Triangles = append(m.Triangles,
		[3]int{base, base + 1, base + 2}, [3]int{base, base + 2, base + 3})

	if got := AireMedianeProjetee(m, in); math.Abs(got-0.005) > 1e-6 {
		t.Fatalf("la mediane doit ignorer le socle : %.6f m2, attendu 0,005", got)
	}
	if EstDecorGrossier(m, in, AireMaxTriangleJouable) {
		t.Error("un sol fin pose sur un socle n'est PAS du decor — c'est le cas que la moyenne rate")
	}
}

// TestEstDecorGrossierTrieAuSeuilRetenu — le tri au seuil valide (0,005 m2) garde le maillage
// au grain decimetrique et jette celui a grandes facettes.
//
// MUTATION QUI LE FAIT ROUGIR : inverser la comparaison de `EstDecorGrossier`.
func TestEstDecorGrossierTrieAuSeuilRetenu(t *testing.T) {
	in := instanceIdentite()
	jouable := grille(20, 400) // cases de 5 cm -> 12,5 cm2, sous le seuil
	decor := grille(20, 4)     // 12,5 m2, trois ordres de grandeur au-dessus
	if EstDecorGrossier(jouable, in, AireMaxTriangleJouable) {
		t.Error("un sol modelise au grain decimetrique n'est PAS du decor")
	}
	if !EstDecorGrossier(decor, in, AireMaxTriangleJouable) {
		t.Error("une dalle a facettes de 12,5 m2 EST du decor")
	}
	// Seuil nul = tri desactive : les deux passent. C'est la lecture concurrente des temoins.
	if EstDecorGrossier(decor, in, 0) {
		t.Error("aireMax <= 0 doit desactiver le tri, pas l'inverser")
	}
}

// TestAireMedianeSuitLEchelleDeLInstance — l'aire est mesuree en MONDE : une instance mise a
// l'echelle 10 rend des triangles 100 fois plus grands. Sans cela, une dalle de decor
// agrandie par son instance passerait pour un maillage fin.
func TestAireMedianeSuitLEchelleDeLInstance(t *testing.T) {
	m := grille(20, 200)
	unite := AireMedianeProjetee(m, instanceIdentite())
	agrandie := instanceIdentite()
	agrandie.Scale = [3]float64{10, 10, 10}
	if got, veut := AireMedianeProjetee(m, agrandie), unite*100; math.Abs(got-veut) > 1e-6 {
		t.Fatalf("instance a l'echelle 10 : aire mediane %.6f m2, attendu %.6f", got, veut)
	}
}
