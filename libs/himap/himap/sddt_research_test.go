//go:build gamefiles && research

package himap

// sddt_research_test.go — INSTRUMENTS de la lecture sddt qui lisent le depot hote (D1 lot 2c,
// 2026-10-28, decision utilisateur : conserves comme instruments de recherche, pas supprimes).
//
// L'oracle de Cliffhanger et le banc comparent la coquille sddt aux ancres d'objectifs
// (`data/titles/.../map_objectives.json`) et aux positions jouees (`data/cache/replays/...`),
// deux donnees que la brique ne possede pas. Ils tournent a la demande :
// `go test -tags=gamefiles,research ./himap/ -run 'TestSddtOracle|TestBancCliffhanger'`.
// Le decodage lui-meme (`TestSddtLecture`, `TestSddtBalayageEau`) reste sous le tag `gamefiles`.

import (
	"context"
	"image/png"
	"os"
	"testing"
)

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
	ref := fixtureCarteValidee
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
