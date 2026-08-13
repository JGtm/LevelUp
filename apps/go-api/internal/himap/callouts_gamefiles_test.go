package himap

// Témoins GAMEFILES du lecteur de callouts (exigent l'installation du jeu ; skip sinon).
//
// Les oracles sont EXTERNES au lecteur : le dump versionné
// .ai/V7.5/dumps/callout_zones_ridgeline_clipped.json (produit par les scripts Python de
// recherche, confronté aux positions de joueurs) et les totaux mesurés du balayage
// callouts_all.py (22 cartes, 816 zones, liaison 816/816).

import (
	"os"
	"path/filepath"
	"testing"
)

// moduleDS compose le chemin du module ds/ d'une carte.
func moduleDS(t *testing.T, carte string) string {
	t.Helper()
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	p := filepath.Join(dir, carte, carte+"-rtx-new.module")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("module absent (%s)", p)
	}
	return p
}

// TestCalloutsRidgeline confronte la lecture au dump de référence de Ridgeline.
func TestCalloutsRidgeline(t *testing.T) {
	cs, err := ReadModuleCallouts(moduleDS(t, "ridgeline"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 28 {
		t.Fatalf("ridgeline : %d zones, attendu 28 (dump de référence)", len(cs))
	}
	// vi=10 « Horseshoe » : la zone la mieux documentée (dump + CSV + POC).
	var horse *Callout
	shaped := 0
	for i := range cs {
		if cs[i].VolumeIndex == 10 {
			horse = &cs[i]
		}
		if cs[i].HasShape {
			shaped++
		}
	}
	if shaped != 16 {
		t.Errorf("zones à forme propre : %d, attendu 16 (dump : a_forme_propre)", shaped)
	}
	if horse == nil {
		t.Fatal("volume 10 (Horseshoe) absent")
	}
	if horse.Name != "ridgeline horses" {
		t.Errorf("nom = %q, attendu %q (tronqué à 32 o, préfixe retiré)", horse.Name, "ridgeline horses")
	}
	if horse.StringID != 0x3CF86E98 {
		t.Errorf("string_id = %08x, attendu 3CF86E98 (CSV)", horse.StringID)
	}
	if !horse.HasShape {
		t.Error("Horseshoe doit porter sa forme propre")
	}
	profile := func(v, attendu, tol float64, quoi string) {
		if d := v - attendu; d < -tol || d > tol {
			t.Errorf("%s = %.3f, attendu %.3f (dump)", quoi, v, attendu)
		}
	}
	profile(horse.Pos[0], 19.0, 0.3, "pos.x")
	profile(horse.Pos[1], 10.0, 0.3, "pos.y")
	profile(horse.Pos[2], 1.0, 0.05, "pos.z")
	// Tranche verticale du dump : [-0.2, 11.0].
	profile(horse.ZBottom(), -0.2, 0.05, "zBottom")
	profile(horse.ZTop(), 11.0, 0.05, "zTop")
	if len(horse.Polygon) != 12 {
		t.Errorf("polygone : %d sommets, attendu 12 (dump brut)", len(horse.Polygon))
	} else {
		// Le polygone est publié en MONDE (sommets relatifs du tag + pos, invariant AABB
		// vérifié à l'extraction) : premier sommet du dump brut = (14.8, 7.5).
		profile(horse.Polygon[0][0], 14.8, 0.01, "polygon[0].x")
		profile(horse.Polygon[0][1], 7.5, 0.01, "polygon[0].y")
	}
	t.Logf("ridgeline : 28 zones, Horseshoe pos=%.2f/%.2f/%.2f tranche=[%.2f;%.2f] poly=%d",
		horse.Pos[0], horse.Pos[1], horse.Pos[2], horse.ZBottom(), horse.ZTop(), len(horse.Polygon))
}

// TestCalloutsInstallationComplete rejoue le balayage de callouts_all.py : 22 cartes
// portent 816 zones, les canevas Forge en portent ZÉRO — par construction, pas par échec.
func TestCalloutsInstallationComplete(t *testing.T) {
	dir, err := LevelsDir("ds")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	total, avec, sans := 0, 0, 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name(), e.Name()+"-rtx-new.module")
		if _, statErr := os.Stat(p); statErr != nil {
			continue
		}
		cs, err := ReadModuleCallouts(p)
		if err != nil {
			t.Errorf("%s : %v", e.Name(), err)
			continue
		}
		if len(cs) == 0 {
			sans++
			continue
		}
		avec++
		total += len(cs)
	}
	if avec != 22 || total != 816 {
		t.Errorf("balayage : %d cartes avec callouts / %d zones, attendu 22 / 816 (mesure callouts_all.py)", avec, total)
	}
	t.Logf("balayage : %d cartes avec callouts (%d zones), %d sans (canevas Forge et assimilés)", avec, total, sans)
}
