package analysis

import (
	"context"
	"reflect"
	"testing"
)

var suffixesGroupes = []string{"- Ranked", "Heavies"}

func TestGroupMapsByBase_VarianteEtBaseADeuxIdentifiants(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "live-b", Name: "Live Fire", Matches: 17},
		{MapID: "live-a", Name: "Live Fire", Matches: 432},
		{MapID: "live-rk", Name: "Live Fire - Ranked", Matches: 512},
		{MapID: "pit", Name: "The Pit", Matches: 26},
		{MapID: "pit2", Name: "The Pit", Matches: 10}, // deux identifiants sans variante : pas de groupe
	}, suffixesGroupes)

	for _, id := range []string{"live-a", "live-b", "live-rk"} {
		if got := g.Groups.Groupe(id); got != "live-a" {
			t.Errorf("Groupe(%s) = %q, attendu live-a (la base la plus jouée)", id, got)
		}
	}
	if got, want := g.Groups.Membres("live-rk"), []string{"live-a", "live-b", "live-rk"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Membres = %v, attendu %v", got, want)
	}
	base, ok := g.Bases["live-a"]
	if !ok || base.Name != "Live Fire" || !reflect.DeepEqual(base.IDs, []string{"live-a", "live-b"}) {
		t.Errorf("base = %+v, attendu Live Fire sur [live-a live-b]", base)
	}
	if g.Groups.Regroupe("pit") || g.Groups.Regroupe("pit2") {
		t.Error("deux identifiants de même nom SANS variante ne forment pas un groupe")
	}
}

func TestGroupMapsByBase_VarianteSansBase(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "lat", Name: "Lattice - Ranked", Matches: 187},
		{MapID: "th", Name: "Thunderhead Heavies", Matches: 8},
		{MapID: "thb", Name: "Thunderhead", Matches: 9},
	}, suffixesGroupes)
	if g.Groups.Regroupe("lat") {
		t.Error("une variante sans carte de base au registre reste seule")
	}
	if g.Groups.Groupe("th") != "thb" || g.Groups.Groupe("thb") != "thb" {
		t.Errorf("Thunderhead Heavies doit rejoindre Thunderhead : %q / %q", g.Groups.Groupe("th"), g.Groups.Groupe("thb"))
	}
}

func TestGroupMapsByBase_DeuxSuffixesSurUneBase(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "r", Name: "Refuge", Matches: 38},
		{MapID: "rk", Name: "Refuge - Ranked", Matches: 7},
		{MapID: "rh", Name: "Refuge Heavies", Matches: 3},
	}, suffixesGroupes)
	if got, want := g.Groups.Membres("rh"), []string{"r", "rh", "rk"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Membres = %v, attendu %v", got, want)
	}
}

// Une carte sans nom retenu (nom NULL partout, ou identifiant recopié) reste seule : le nom n'est
// jamais déduit d'une autre ligne.
func TestGroupMapsByBase_SansNomRetenu(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "sans-nom", Name: "", Matches: 40},
		{MapID: "base", Name: "Streets", Matches: 5},
		{MapID: "ranked", Name: "Streets - Ranked", Matches: 9},
	}, suffixesGroupes)
	if g.Groups.Regroupe("sans-nom") {
		t.Error("une carte sans nom retenu ne doit pas être regroupée")
	}
	if g.Groups.Groupe("ranked") != "base" {
		t.Errorf("Groupe(ranked) = %q, attendu base", g.Groups.Groupe("ranked"))
	}
}

func TestGroupMapsByBase_SansSuffixeOuSansCarte(t *testing.T) {
	maps := []MapNameCount{{MapID: "a", Name: "X"}, {MapID: "b", Name: "X Heavies"}}
	if g := GroupMapsByBase(maps, nil); !g.Groups.Vide() || len(g.Bases) != 0 {
		t.Errorf("titre sans suffixe : aucun groupe attendu, reçu %+v", g)
	}
	if g := GroupMapsByBase(nil, suffixesGroupes); !g.Groups.Vide() {
		t.Errorf("registre vide : aucun groupe attendu, reçu %+v", g)
	}
}

// Égalité de parties : le plus petit identifiant est le représentant, quel que soit l'ordre de lecture.
func TestGroupMapsByBase_RepresentantStable(t *testing.T) {
	a := []MapNameCount{{MapID: "z", Name: "X", Matches: 5}, {MapID: "m", Name: "X", Matches: 5}, {MapID: "v", Name: "X Heavies", Matches: 1}}
	b := []MapNameCount{{MapID: "v", Name: "X Heavies", Matches: 1}, {MapID: "m", Name: "X", Matches: 5}, {MapID: "z", Name: "X", Matches: 5}}
	if ga, gb := GroupMapsByBase(a, suffixesGroupes).Groups.Groupe("v"), GroupMapsByBase(b, suffixesGroupes).Groups.Groupe("v"); ga != "m" || gb != "m" {
		t.Errorf("représentant = %q / %q, attendu m dans les deux ordres", ga, gb)
	}
}

// « X Heavies - Ranked » remonte jusqu'à « X » quand « X Heavies » existe lui aussi.
func TestGroupMapsByBase_VarianteDeVariante(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "x", Name: "X", Matches: 3},
		{MapID: "xh", Name: "X Heavies", Matches: 2},
		{MapID: "xhr", Name: "X Heavies - Ranked", Matches: 1},
	}, suffixesGroupes)
	if g.Groups.Groupe("xhr") != "x" || g.Groups.Groupe("xh") != "x" {
		t.Errorf("chaîne : Groupe(xhr) = %q, Groupe(xh) = %q, attendu x", g.Groups.Groupe("xhr"), g.Groups.Groupe("xh"))
	}
}

// Une carte intégrée republiée sous un nouvel asset : même nom, même clé de module, aucune variante.
func TestGroupMapsByBase_IntegreeRepubliee(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "bz-old", Name: "Bazaar", Matches: 173, Module: "ctf_bazaar"},
		{MapID: "bz-new", Name: "Bazaar", Matches: 59, Module: "ctf_bazaar"},
		{MapID: "bz-bis", Name: "Bazaar", Matches: 3, Module: "ctf_bazaar"},
	}, suffixesGroupes)
	for _, id := range []string{"bz-old", "bz-new", "bz-bis"} {
		if got := g.Groups.Groupe(id); got != "bz-old" {
			t.Errorf("Groupe(%s) = %q, attendu bz-old (le plus joué)", id, got)
		}
	}
	base, ok := g.Bases["bz-old"]
	if !ok || base.Name != "Bazaar" || !reflect.DeepEqual(base.IDs, []string{"bz-bis", "bz-new", "bz-old"}) {
		t.Errorf("base = %+v, attendu Bazaar sur les trois identifiants", base)
	}
}

// Sans suffixe déclaré, la règle de republication fonctionne seule.
func TestGroupMapsByBase_RepublieeSansSuffixe(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "a", Name: "Chasm", Matches: 5, Module: "chasm"},
		{MapID: "b", Name: "Chasm", Matches: 9, Module: "chasm"},
	}, nil)
	if g.Groups.Groupe("a") != "b" || g.Groups.Groupe("b") != "b" {
		t.Errorf("groupes = %q / %q, attendu b / b", g.Groups.Groupe("a"), g.Groups.Groupe("b"))
	}
}

// Deux versions d'une carte Forge portent le même nom et aucune clé de module : elles restent deux
// cartes, quel que soit leur nom.
func TestGroupMapsByBase_DeuxVersionsForgeRestentSeparees(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "sh-1", Name: "Shogun", Matches: 15},
		{MapID: "sh-2", Name: "Shogun", Matches: 2},
	}, suffixesGroupes)
	if g.Groups.Regroupe("sh-1") || g.Groups.Regroupe("sh-2") || !g.Groups.Vide() {
		t.Error("deux identifiants Forge de même nom sans clé de module ne forment pas un groupe")
	}
}

// Une clé absente d'un côté : l'identifiant sans clé reste seul, les autres se réunissent. Deux clés
// différentes sous un même nom restent deux groupes.
func TestGroupMapsByBase_CleAbsenteOuDifferente(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "x1", Name: "Forest", Matches: 10, Module: "forest"},
		{MapID: "x2", Name: "Forest", Matches: 8, Module: "forest"},
		{MapID: "x3", Name: "Forest", Matches: 7}, // pas de clé : reste seule
		{MapID: "y1", Name: "Pit", Matches: 4, Module: "m1"},
		{MapID: "y2", Name: "Pit", Matches: 3, Module: "m2"}, // clé différente : reste seule
	}, suffixesGroupes)
	if !g.Groups.Regroupe("x1") || g.Groups.Groupe("x2") != "x1" {
		t.Errorf("x1 et x2 (même clé) doivent former un groupe : %q / %q", g.Groups.Groupe("x1"), g.Groups.Groupe("x2"))
	}
	if g.Groups.Regroupe("x3") {
		t.Error("un identifiant sans clé de module reste seul")
	}
	if g.Groups.Regroupe("y1") || g.Groups.Regroupe("y2") {
		t.Error("deux clés de module différentes sous un même nom restent deux cartes")
	}
}

// Variante et republication : le groupe de la base absorbe tous les identifiants de son nom et ceux
// de sa variante, sans second groupe ni doublon.
func TestGroupMapsByBase_VarianteEtRepublieeUnSeulGroupe(t *testing.T) {
	g := GroupMapsByBase([]MapNameCount{
		{MapID: "lf-1", Name: "Live Fire", Matches: 432, Module: "sgh_interlock"},
		{MapID: "lf-2", Name: "Live Fire", Matches: 58, Module: "sgh_interlock"},
		{MapID: "lf-rk", Name: "Live Fire - Ranked", Matches: 512, Module: "sgh_interlock"},
		{MapID: "lf-rk2", Name: "Live Fire - Ranked", Matches: 6, Module: "sgh_interlock"},
	}, suffixesGroupes)
	want := []string{"lf-1", "lf-2", "lf-rk", "lf-rk2"}
	if got := g.Groups.Membres("lf-rk2"); !reflect.DeepEqual(got, want) {
		t.Errorf("Membres = %v, attendu %v", got, want)
	}
	if len(g.Bases) != 1 {
		t.Errorf("un seul groupe attendu, bases = %+v", g.Bases)
	}
}

// Les règles : un résolveur pose la clé de module sans modifier les cartes d'origine ; sans résolveur
// ni suffixe, elles sont inactives.
func TestMapGroupRules(t *testing.T) {
	if (MapGroupRules{}).Active() {
		t.Error("règles vides : inactives")
	}
	cartes := []MapNameCount{{MapID: "a", Name: "Chasm"}, {MapID: "b", Name: "Chasm"}}
	r := MapGroupRules{Modules: func(_ context.Context, maps []MapNameCount) map[string]string {
		return map[string]string{"a": "chasm"}
	}}
	if !r.Active() {
		t.Error("un résolveur suffit à activer les règles")
	}
	got := r.WithModules(context.Background(), cartes)
	if got[0].Module != "chasm" || got[1].Module != "" || cartes[0].Module != "" {
		t.Errorf("modules = %+v (origine %+v)", got, cartes)
	}
}
