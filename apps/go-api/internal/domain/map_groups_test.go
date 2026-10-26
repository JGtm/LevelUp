package domain

import (
	"reflect"
	"testing"
)

func TestGroupesDeCartes_HorsGroupe(t *testing.T) {
	var g GroupesDeCartes
	if !g.Vide() {
		t.Error("la valeur zéro doit être vide")
	}
	if got := g.Groupe("m1"); got != "m1" {
		t.Errorf("Groupe(hors groupe) = %q, attendu m1", got)
	}
	if got := g.Membres("m1"); !reflect.DeepEqual(got, []string{"m1"}) {
		t.Errorf("Membres(hors groupe) = %v, attendu [m1]", got)
	}
	if g.Regroupe("m1") {
		t.Error("Regroupe(hors groupe) = true")
	}
}

func TestGroupesDeCartes_BaseEtVariantes(t *testing.T) {
	g := NewGroupesDeCartes(map[string]string{
		"var-ranked":  "base-a",
		"var-heavies": "base-a",
		"base-b":      "base-a", // second identifiant portant le nom de la base
	})
	for _, id := range []string{"base-a", "base-b", "var-ranked", "var-heavies"} {
		if got := g.Groupe(id); got != "base-a" {
			t.Errorf("Groupe(%s) = %q, attendu base-a", id, got)
		}
		if !g.Regroupe(id) {
			t.Errorf("Regroupe(%s) = false", id)
		}
	}
	want := []string{"base-a", "base-b", "var-heavies", "var-ranked"}
	if got := g.Membres("var-ranked"); !reflect.DeepEqual(got, want) {
		t.Errorf("Membres(variante) = %v, attendu %v", got, want)
	}
	if got := g.Membres("base-a"); !reflect.DeepEqual(got, want) {
		t.Errorf("Membres(base) = %v, attendu %v", got, want)
	}
	if g.Vide() {
		t.Error("table non vide déclarée vide")
	}
}

func TestGroupesDeCartes_MembresRendUneCopie(t *testing.T) {
	g := NewGroupesDeCartes(map[string]string{"v": "b"})
	m := g.Membres("b")
	m[0] = "modifie"
	if got := g.Membres("b"); got[0] != "b" {
		t.Errorf("Membres expose son état interne : %v", got)
	}
}

func TestGroupesDeCartes_EntreesSansGroupeIgnorees(t *testing.T) {
	g := NewGroupesDeCartes(map[string]string{
		"seul":  "seul", // un identifiant rattaché à lui-même ne forme pas un groupe
		"":      "base",
		"vide":  "",
		"v":     "b",
		"autre": "autre",
	})
	if g.Regroupe("seul") || g.Regroupe("vide") || g.Regroupe("autre") {
		t.Error("une entrée sans second membre ne doit pas former de groupe")
	}
	if !g.Regroupe("v") || g.Groupe("v") != "b" {
		t.Errorf("le groupe (v, b) doit exister : Groupe(v) = %q", g.Groupe("v"))
	}
}

func TestGroupesDeCartes_ChaineEtCycle(t *testing.T) {
	// Une base elle-même rattachée à une autre est remontée à la racine.
	g := NewGroupesDeCartes(map[string]string{"c": "b", "b": "a"})
	for _, id := range []string{"a", "b", "c"} {
		if got := g.Groupe(id); got != "a" {
			t.Errorf("chaîne : Groupe(%s) = %q, attendu a", id, got)
		}
	}
	// Un cycle ne boucle pas et rend un résultat stable.
	cycle := NewGroupesDeCartes(map[string]string{"x": "y", "y": "x"})
	if cycle.Groupe("x") != cycle.Groupe("y") {
		t.Errorf("cycle : x et y doivent partager un groupe (%q, %q)", cycle.Groupe("x"), cycle.Groupe("y"))
	}
}
