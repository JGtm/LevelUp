package replay

import "testing"

func projectileNeA(t0 int, x, y float32) Projectile {
	return Projectile{T0: t0, P: [][3]float32{{0, x, y}, {1, x + 1, y}}}
}

func tirDe(t int, slot uint32, weapon string, x, y float32) Shot {
	return Shot{T: t, Slot: slot, Weapon: weapon, X: x, Y: y}
}

func lienDe(p Projectile) int {
	if p.Shot == nil {
		return -1
	}
	return *p.Shot
}

func TestLinkShots_Nominal(t *testing.T) {
	projs := []Projectile{projectileNeA(10, 5, 5)}
	shots := []Shot{tirDe(3, 1, "A", 5, 5), tirDe(10, 1, "A", 6, 5)}
	linked, ambiguous := linkShotsToProjectiles(projs, shots, nil)
	if linked != 1 || ambiguous != 0 || lienDe(projs[0]) != 1 {
		t.Errorf("linked=%d ambiguous=%d lien=%d, attendu 1/0/1", linked, ambiguous, lienDe(projs[0]))
	}
}

func TestLinkShots_FenetreDUneFrame(t *testing.T) {
	for _, tc := range []struct {
		frameTir int
		lie      bool
	}{{9, true}, {10, true}, {11, true}, {8, false}, {12, false}} {
		projs := []Projectile{projectileNeA(10, 0, 0)}
		linked, _ := linkShotsToProjectiles(projs, []Shot{tirDe(tc.frameTir, 1, "A", 0, 0)}, nil)
		if (linked == 1) != tc.lie {
			t.Errorf("tir a la frame %d : lie=%v, attendu %v", tc.frameTir, linked == 1, tc.lie)
		}
	}
}

func TestLinkShots_TropLoin(t *testing.T) {
	projs := []Projectile{projectileNeA(10, 0, 0)}
	linked, ambiguous := linkShotsToProjectiles(projs, []Shot{tirDe(10, 1, "A", projShotMaxDistM+0.5, 0)}, nil)
	if linked != 0 || ambiguous != 0 || projs[0].Shot != nil {
		t.Errorf("tir hors portee lie : linked=%d ambiguous=%d", linked, ambiguous)
	}
	// Pile a la borne : lie.
	projs = []Projectile{projectileNeA(10, 0, 0)}
	if linked, _ = linkShotsToProjectiles(projs, []Shot{tirDe(10, 1, "A", projShotMaxDistM, 0)}, nil); linked != 1 {
		t.Error("tir a la borne de distance non lie")
	}
}

func TestLinkShots_SalveMemeArmeMemeTireur(t *testing.T) {
	// Trois tirs de la meme salve : le plus proche l'emporte, a egalite le plus petit index.
	projs := []Projectile{projectileNeA(10, 0, 0), projectileNeA(10, 2, 0)}
	shots := []Shot{tirDe(10, 4, "M", 1, 0), tirDe(10, 4, "M", 2, 0), tirDe(10, 4, "M", 0, 0)}
	linked, ambiguous := linkShotsToProjectiles(projs, shots, nil)
	if linked != 2 || ambiguous != 0 {
		t.Fatalf("linked=%d ambiguous=%d, attendu 2/0", linked, ambiguous)
	}
	if lienDe(projs[0]) != 2 { // distance 0
		t.Errorf("projectile 0 lie au tir %d, attendu 2 (le plus proche)", lienDe(projs[0]))
	}
	if lienDe(projs[1]) != 1 { // distance 0 (tir 1 en x=2)
		t.Errorf("projectile 1 lie au tir %d, attendu 1", lienDe(projs[1]))
	}
	// Egalite de distance : le plus petit index.
	egal := []Projectile{projectileNeA(10, 1, 0)}
	linkShotsToProjectiles(egal, []Shot{tirDe(10, 4, "M", 0, 0), tirDe(10, 4, "M", 2, 0)}, nil)
	if lienDe(egal[0]) != 0 {
		t.Errorf("egalite : lie au tir %d, attendu 0", lienDe(egal[0]))
	}
}

func TestLinkShots_AmbiguArmesOuTireursDifferents(t *testing.T) {
	for nom, shots := range map[string][]Shot{
		"deux armes":      {tirDe(10, 1, "A", 0, 0), tirDe(10, 1, "B", 0, 0)},
		"deux tireurs":    {tirDe(10, 1, "A", 0, 0), tirDe(10, 2, "A", 0, 0)},
		"frames voisines": {tirDe(9, 1, "A", 0, 0), tirDe(11, 1, "B", 0, 0)},
	} {
		projs := []Projectile{projectileNeA(10, 0, 0)}
		linked, ambiguous := linkShotsToProjectiles(projs, shots, nil)
		if linked != 0 || ambiguous != 1 || projs[0].Shot != nil {
			t.Errorf("%s : linked=%d ambiguous=%d, attendu 0/1 sans lien", nom, linked, ambiguous)
		}
	}
}

func TestLinkShots_ExclutLesProjectilesDeGrenade(t *testing.T) {
	proj := 0
	projs := []Projectile{projectileNeA(10, 0, 0), projectileNeA(10, 1, 0)}
	gren := []Grenade{{Proj: &proj}, {}}
	linked, _ := linkShotsToProjectiles(projs, []Shot{tirDe(10, 1, "A", 0, 0)}, gren)
	if projs[0].Shot != nil || projs[1].Shot == nil || linked != 1 {
		t.Errorf("projectile de grenade lie ou autre non lie : %+v / %+v", projs[0].Shot, projs[1].Shot)
	}
}

func TestLinkShots_OrdreDesEntreesSansEffet(t *testing.T) {
	// Les memes tirs, rendus dans l'ordre inverse, donnent le meme TIR designe (par contenu).
	a := []Shot{tirDe(10, 1, "A", 0, 0), tirDe(11, 1, "A", 2.5, 0)}
	pa := []Projectile{projectileNeA(10, 2, 0)}
	linkShotsToProjectiles(pa, a, nil)
	if lienDe(pa[0]) != 1 {
		t.Errorf("lie au tir %d, attendu 1 (le plus proche)", lienDe(pa[0]))
	}
	// Deux appels identiques : meme resultat.
	pb := []Projectile{projectileNeA(10, 2, 0)}
	linkShotsToProjectiles(pb, a, nil)
	if lienDe(pb[0]) != lienDe(pa[0]) {
		t.Error("resultat different entre deux appels identiques")
	}
}

func TestLierLesProjectilesAuxTirs_Couverture(t *testing.T) {
	a := &assemblage{projCov: &ProjectileCoverage{Tracks: 3, Published: 3}}
	a.doc.Projectiles = []Projectile{projectileNeA(10, 0, 0), projectileNeA(20, 0, 0), projectileNeA(30, 0, 0)}
	a.doc.Shots = []Shot{tirDe(10, 1, "A", 0, 0), tirDe(20, 1, "A", 0, 0), tirDe(20, 2, "B", 0, 0)}
	a.lierLesProjectilesAuxTirs()
	if a.projCov.ShotLinked != 1 || a.projCov.ShotAmbiguous != 1 {
		t.Errorf("couverture = %+v, attendu 1 lie et 1 ambigu (le troisieme n'a pas de tir)", *a.projCov)
	}
	// Sans couverture (aucune piste fournie), le lien ne plante pas.
	b := &assemblage{}
	b.lierLesProjectilesAuxTirs()
}

func TestLinkShots_SansTirOuSansProjectile(t *testing.T) {
	if l, a := linkShotsToProjectiles(nil, []Shot{tirDe(1, 1, "A", 0, 0)}, nil); l != 0 || a != 0 {
		t.Error("sans projectile")
	}
	projs := []Projectile{projectileNeA(1, 0, 0)}
	if l, a := linkShotsToProjectiles(projs, nil, nil); l != 0 || a != 0 || projs[0].Shot != nil {
		t.Error("sans tir")
	}
}
