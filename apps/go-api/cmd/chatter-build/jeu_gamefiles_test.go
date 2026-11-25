//go:build gamefiles

package main

import (
	"path/filepath"
	"testing"
)

// TestTextesLusDansLeJeuInstalle : la chaîne `textes` relit, dans le module des tags du jeu installé, la
// variante d'un son de réplique et son texte français et anglais. Le son 00950095 (« Ennemi repéré ») a trois
// variantes, relevées à la mesure du 2026-11-24 sur le build HI_1_13_0. Sans jeu installé, le test passe son tour.
func TestTextesLusDansLeJeuInstalle(t *testing.T) {
	racine, err := racineDuJeu("")
	if err != nil {
		t.Skipf("jeu non installé : %v", err)
	}
	mod, err := ouvrirModule(filepath.Join(racine, filepath.FromSlash(moduleTags)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mod.Close() }()
	got, err := extraireTextes(mod, map[uint32]bool{0x00950095: true}, languesDeTest(t), journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Variantes) != 3 {
		t.Fatalf("son lu = %+v, une variante par évènement attendue (trois)", got)
	}
	v := got[0].Variantes[0]
	if v.Textes["fr"] != "Je l'ai dans mon viseur !" || v.Textes["en"] != "In my sights." {
		t.Errorf("première variante = %+v", v)
	}
}
