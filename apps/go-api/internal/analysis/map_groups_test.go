package analysis

import (
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
