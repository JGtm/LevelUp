//go:build gamefiles

package himap

// Temoins du lecteur sddt sur les FICHIERS DU JEU (le jeu seul suffit). L'oracle et le banc de
// Cliffhanger, qui lisent le depot hote, sont des instruments : sddt_research_test.go.
//
// Promus des sondes d'investigation du 2026-08-10 (INVESTIGATION_SDDT_PFND_2026-08-10.md) :
// chaque invariant affirme ici a d'abord ete etabli par une mutation jouee dans la sonde.
// Les chiffres de reference (233 volumes, 29 221 positions, accord 66,7 %, positions gardees
// 93,82 %) sont ceux consignes dans l'investigation — toute regression est un echec.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestSddtLecture verifie le decodage complet sur les deux cartes etalonnees.
func TestSddtLecture(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	cas := []struct {
		carte      string
		volumes    int
		frontieres int
		plans      int
	}{
		{"ridgeline", 233, 12, 6},
		{"catalyst", 0, 20, 8},
	}
	for _, c := range cas {
		t.Run(c.carte, func(t *testing.T) {
			s, err := LitSddt(moduleAny(t, c.carte))
			if err != nil {
				t.Fatal(err)
			}
			if len(s.VolumesEau) != c.volumes {
				t.Errorf("volumes d'eau : %d, attendu %d", len(s.VolumesEau), c.volumes)
			}
			if len(s.Frontieres) != c.frontieres {
				t.Errorf("triangles-frontieres : %d, attendu %d", len(s.Frontieres), c.frontieres)
			}
			if coq := s.Coquille(); len(coq) != c.plans {
				t.Errorf("plans de coquille dedoublonnes : %d, attendu %d", len(coq), c.plans)
			}
			verifieInvariantsSddt(t, s)
		})
	}
}

// verifieInvariantsSddt rejoue sur les octets reels les temoins qui ont etabli la lecture :
// normales unitaires, sommets sur leurs plans, orientation des prismes (les deux sens),
// centroides des frontieres.
func verifieInvariantsSddt(t *testing.T, s Sddt) {
	t.Helper()
	// Tous les plans (volumes et coquille) portent une normale UNITAIRE — le temoin qui a
	// designe l'offset 0 (la lecture a +4 octets rendait 0 % d'unitaires).
	for vi, v := range s.VolumesEau {
		for _, p := range v.Plans {
			if n := math.Sqrt(p[0]*p[0] + p[1]*p[1] + p[2]*p[2]); math.Abs(n-1) > 1e-3 {
				t.Fatalf("volume %d : normale non unitaire (%f)", vi, n)
			}
		}
	}
	for fi, f := range s.Frontieres {
		n := math.Sqrt(f.Normale[0]*f.Normale[0] + f.Normale[1]*f.Normale[1] + f.Normale[2]*f.Normale[2])
		if math.Abs(n-1) > 1e-3 {
			t.Fatalf("frontiere %d : normale non unitaire (%f)", fi, n)
		}
		// Centroide lu == moyenne des sommets (temoin arithmetique du record de 68 o).
		for a := 0; a < 3; a++ {
			m := (f.Sommets[0][a] + f.Sommets[1][a] + f.Sommets[2][a]) / 3
			if math.Abs(m-f.Centroide[a]) > 1e-3 {
				t.Fatalf("frontiere %d : centroide %v != moyenne des sommets", fi, f.Centroide)
			}
		}
	}
	// Coherence plans <-> sommets et ORIENTATION : le centroide des sommets de chaque volume
	// est dans son propre prisme (`n.p <= d`), et le sens inverse echoue pour TOUS. C'est la
	// mutation qui a etabli le sens sur 233/233 contre 0/233.
	dedans, inverses := 0, 0
	residuMax := 0.0
	for _, v := range s.VolumesEau {
		c := centroideSommets(v)
		if v.Contient(c, 1e-4) {
			dedans++
		}
		horsUnPlan := false
		for _, pl := range v.Plans {
			if pl[0]*c[0]+pl[1]*c[1]+pl[2]*c[2] < pl[3]-1e-4 {
				horsUnPlan = true
				break
			}
		}
		if !horsUnPlan {
			inverses++
		}
		for _, tri := range v.Tris {
			for s := 0; s < 3; s++ {
				best := math.Inf(1)
				for _, pl := range v.Plans {
					r := math.Abs(pl[0]*tri[s][0] + pl[1]*tri[s][1] + pl[2]*tri[s][2] - pl[3])
					if r < best {
						best = r
					}
				}
				if best > residuMax && !math.IsInf(best, 1) {
					residuMax = best
				}
			}
		}
	}
	if n := len(s.VolumesEau); n > 0 {
		if dedans != n {
			t.Errorf("orientation : %d/%d centroides dans leur prisme", dedans, n)
		}
		if inverses != 0 {
			t.Errorf("orientation : le sens inverse accepterait %d/%d volumes — le temoin ne separe plus", inverses, n)
		}
		if residuMax > 1e-3 {
			t.Errorf("residu max sommet -> plan le plus proche : %f (attendu ~0)", residuMax)
		}
		t.Logf("%d volumes : orientation %d/%d, sens inverse %d, residu max %.4f", n, dedans, n, inverses, residuMax)
	}
}

func centroideSommets(v VolumeEauSddt) [3]float64 {
	var c [3]float64
	n := 0
	for _, tri := range v.Tris {
		for s := 0; s < 3; s++ {
			for a := 0; a < 3; a++ {
				c[a] += tri[s][a]
			}
			n++
		}
	}
	if n > 0 {
		for a := 0; a < 3; a++ {
			c[a] /= float64(n)
		}
	}
	return c
}

// TestSddtBalayageEau — l'eau sddt est-elle une feature GENERALE ou propre a Cliffhanger ?
// Balayage de toutes les cartes installees (variante any/) : compte des volumes d'eau et des
// frontieres par carte. Le verdict (mono-carte ou general) se lit dans le journal du test.
func TestSddtBalayageEau(t *testing.T) {
	dir, err := LevelsDir("any")
	if err != nil {
		t.Skip(err)
	}
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Skip(err)
	}
	cartes, avecEau, avecCoquille := 0, 0, 0
	for _, e := range entrees {
		if !e.IsDir() {
			continue
		}
		chemin := filepath.Join(dir, e.Name(), e.Name()+"-rtx-new.module")
		if _, err := os.Stat(chemin); err != nil {
			continue // dossier sans module rtx (dossiers annexes)
		}
		s, err := LitSddt(chemin)
		if err != nil {
			t.Errorf("%s : lecture sddt : %v", e.Name(), err)
			continue
		}
		cartes++
		if len(s.VolumesEau) > 0 {
			avecEau++
			emprise := empriseVolumes(s.VolumesEau)
			t.Logf("%-24s : %4d volumes d'eau · %2d frontieres · emprise x [%.1f ; %.1f] y [%.1f ; %.1f] z [%.1f ; %.1f]",
				e.Name(), len(s.VolumesEau), len(s.Frontieres),
				emprise[0][0], emprise[1][0], emprise[0][1], emprise[1][1], emprise[0][2], emprise[1][2])
		} else {
			t.Logf("%-24s : aucun volume d'eau · %2d frontieres", e.Name(), len(s.Frontieres))
		}
		if len(s.Frontieres) > 0 {
			avecCoquille++
		}
	}
	if cartes == 0 {
		t.Skip("aucun module any/ lisible — balayage sans objet")
	}
	t.Logf("BILAN : %d cartes lues · %d avec volumes d'eau · %d avec coquille-frontiere",
		cartes, avecEau, avecCoquille)
}

func empriseVolumes(vols []VolumeEauSddt) [2][3]float64 {
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, v := range vols {
		for a := 0; a < 3; a++ {
			lo[a] = math.Min(lo[a], v.AABBMin[a])
			hi[a] = math.Max(hi[a], v.AABBMax[a])
		}
	}
	return [2][3]float64{lo, hi}
}
