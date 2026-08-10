package himap

// Temoins du lecteur sddt sur les FICHIERS DU JEU, et le banc de Cliffhanger.
//
// Promus des sondes d'investigation du 2026-08-10 (INVESTIGATION_SDDT_PFND_2026-08-10.md) :
// chaque invariant affirme ici a d'abord ete etabli par une mutation jouee dans la sonde.
// Les chiffres de reference (233 volumes, 29 221 positions, accord 66,7 %, positions gardees
// 93,82 %) sont ceux consignes dans l'investigation — toute regression est un echec.

import (
	"context"
	"image/png"
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

// TestSddtOracleCliffhanger — les deux natures contre les 29 221 positions jouees :
// la coquille de survie les contient TOUTES ; les volumes d'eau presque aucune (on traverse
// la riviere, on n'y sejourne pas).
func TestSddtOracleCliffhanger(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	s, err := LitSddt(moduleAny(t, "ridgeline"))
	if err != nil {
		t.Fatal(err)
	}
	coq := s.Coquille()
	pos := chargePositions(t)
	dansCoq, dansEau := 0, 0
	for _, p := range pos {
		q := [3]float64{p.X, p.Y, p.Z}
		if coq.Contient(q, 0) {
			dansCoq++
		}
		for _, v := range s.VolumesEau {
			if v.Contient(q, 0) {
				dansEau++
				break
			}
		}
	}
	t.Logf("ORACLE : coquille %d/%d · volumes d'eau %d/%d (%.3f %%)",
		dansCoq, len(pos), dansEau, len(pos), 100*float64(dansEau)/float64(len(pos)))
	if dansCoq != len(pos) {
		t.Errorf("la coquille de survie doit contenir TOUTES les positions jouees (%d/%d)", dansCoq, len(pos))
	}
	if partEau := 100 * float64(dansEau) / float64(len(pos)); partEau == 0 || partEau > 1 {
		t.Errorf("les volumes d'eau doivent contenir une part infime mais non nulle des positions (%.3f %%)", partEau)
	}
}

// TestBancCliffhanger — LE BANC : la carte par defaut (grain, tranche, tous modules) contre
// la reference validee et l'oracle des positions jouees. Toute regression est un echec de lot,
// quelle que soit la beaute du resultat.
//
// REFERENCES RE-BASEES LE 2026-08-10, PAR DECISION UTILISATEUR, ET IL FAUT QUE CE SOIT ECRIT :
//
//	                  avant (tranche ABSOLUE)   apres (tranche RELATIVE au sol joue)
//	accord                    66,7 %                    64,7 %
//	exces                     33,8 %                    39,1 %
//	positions jouees          93,82 %                   93,95 %
//
// Ce n'est PAS une regression toleree, c'est un ARBITRAGE : la tranche absolue rendait `chasm`
// (5/17 ancres) et `btb_highpower` (14/51) inexploitables — Chasm se reduisait litteralement a
// deux traits. La translater au sol joue les repare (17/17 et 38/51) et coute a Cliffhanger
// 5,3 points d'exces, c'est-a-dire un peu de vallee en plus autour de l'arene. L'utilisateur a
// tranche sur pieces, images a l'appui. Detail : `PLAN_PORT_TRIANGLES_GO.md` §7 D3.
//
// Le banc applique desormais `TrancheDeJeu`, LA MEME fonction que la production : une copie de
// la translation ici cesserait de garder ce qui est reellement cuit.
//
// L'eau est ensuite posee et le test PROUVE qu'elle ne touche pas le terrain : z-buffer et
// eclairement identiques a l'octet.
func TestBancCliffhanger(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ref, err := cheminDepuisDepot(".ai/V7.5/dumps/carte_validee_v1.png")
	if err != nil {
		t.Skip(err)
	}
	f, err := os.Open(ref)
	if err != nil {
		t.Skip(err)
	}
	defer func() { _ = f.Close() }()
	validee, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bo := validee.Bounds()
	larg, haut := bo.Dx(), bo.Dy()
	lo := [2]float64{gateX0, gateY1 - float64(haut)*gateEchelle}
	hi := [2]float64{gateX0 + float64(larg)*gateEchelle, gateY1}

	rendu := NewRendu(lo, hi, gateEchelle)
	rendu.Tranche(TrancheDeJeu(MedianeZ(ancresCliffhanger(t)) - AncrageDecalageSol))

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	bsp := ChoisitBSP(bsps, ancresCliffhanger(t))
	// LE BANC PASSE PAR LA CHAINE DE PRODUCTION (`PeupleRendu`), et c'est le point : une boucle
	// jumelle ici aurait diverge de la production, et le banc aurait alors cesse de la garder.
	// Seul le CADRE reste propre au banc — il est celui de la carte validee, pas celui des
	// ancres, pour que la comparaison pixel a pixel ait un sens.
	dessinees, decor := PeupleRendu(context.Background(), rendu, idx, bsp)
	t.Logf("BANC : %d instances dessinees · %d ecartees comme decor", dessinees, decor)

	// ACCORD contre la silhouette de la reference (memes formules que comparePixels).
	refPlein, _ := matiereReference(validee, bo, larg, haut)
	nRef, manquants, exces := 0, 0, 0
	for py := 0; py < haut; py++ {
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			r := refPlein[py*larg+px]
			_, n := rendu.Eclairement(px, j)
			if r {
				nRef++
			}
			switch {
			case r && !n:
				manquants++
			case n && !r:
				exces++
			}
		}
	}
	accord := 100 * float64(nRef-manquants) / float64(nRef+exces)
	t.Logf("BANC : manquants %.1f %% · exces %.1f %% · ACCORD %.1f %%",
		100*float64(manquants)/float64(nRef), 100*float64(exces)/float64(nRef), accord)
	if accord < 64.6 {
		t.Errorf("ACCORD %.1f %% sous la reference RE-BASEE du 2026-08-10 (64,7 %%, tranche relative)", accord)
	}

	// L'ORACLE : part des positions jouees qui tombent sur une cellule de matiere.
	pos := chargePositions(t)
	dans, horsCadre := 0, 0
	for _, p := range pos {
		i := int((p.X - rendu.Min[0]) / rendu.Cell)
		j := int((p.Y - rendu.Min[1]) / rendu.Cell)
		if i < 0 || i >= rendu.NX || j < 0 || j >= rendu.NY {
			horsCadre++
			continue
		}
		if _, ok := rendu.Eclairement(i, j); ok {
			dans++
		}
	}
	nb := len(pos) - horsCadre
	gardees := 100 * float64(dans) / float64(nb)
	t.Logf("BANC : %d positions dans le cadre · gardees %.2f %%", nb, gardees)
	if gardees < 93.9 {
		t.Errorf("positions gardees %.2f %% sous la reference RE-BASEE du 2026-08-10 (93,95 %%, tranche relative)", gardees)
	}

	// L'EAU NE TOUCHE PAS LE TERRAIN : z-buffer et normales identiques avant/apres.
	zAvant := make([]float64, len(rendu.z))
	copy(zAvant, rendu.z)
	nAvant := make([][3]float64, len(rendu.n))
	copy(nAvant, rendu.n)
	s, err := LitSddt(moduleAny(t, "ridgeline"))
	if err != nil {
		t.Fatal(err)
	}
	rendu.PoseEau(s.VolumesEau)
	for k := range zAvant {
		if zAvant[k] != rendu.z[k] || nAvant[k] != rendu.n[k] {
			t.Fatalf("PoseEau a modifie le terrain (cellule %d)", k)
		}
	}
	nEau := 0
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if rendu.Eau(i, j) {
				nEau++
			}
		}
	}
	t.Logf("BANC : %d volumes d'eau -> %d cellules d'eau, terrain intact", len(s.VolumesEau), nEau)
	if nEau == 0 {
		t.Error("la riviere de Cliffhanger doit poser des cellules d'eau dans le cadre")
	}
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
