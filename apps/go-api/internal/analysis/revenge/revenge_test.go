package revenge

import (
	"reflect"
	"testing"
)

func TestCount(t *testing.T) {
	teams := map[string]int{"k": 0, "v": 1, "k2": 0, "v2": 1, "m": 0}
	cases := []struct {
		name  string
		frags []Frag
		want  map[string]int
	}{
		{"aucun frag", nil, map[string]int{}},
		{"vengeance simple", []Frag{{"v", "k", 1}, {"k", "v", 2}}, map[string]int{"k": 1}},
		{"mort sans vengeance", []Frag{{"v", "k", 1}}, map[string]int{}},
		{"double frag sur le même tueur : un seul compte", []Frag{{"v", "k", 1}, {"k", "v", 2}, {"k", "v", 3}}, map[string]int{"k": 1}},
		{"échange de frags : chaque mort vengée une fois", []Frag{
			{"v", "k", 1}, {"k", "v", 2}, {"v", "k", 3}, {"k", "v", 4},
		}, map[string]int{"k": 2, "v": 1}},
		{"seule la dernière mort compte", []Frag{
			{"v", "k", 1}, {"v2", "k", 2}, {"k", "v", 3},
		}, map[string]int{}},
		{"la dernière mort vengée sur son tueur", []Frag{
			{"v", "k", 1}, {"v2", "k", 2}, {"k", "v2", 3},
		}, map[string]int{"k": 1}},
		{"frag sur un autre que le tueur de la dernière mort", []Frag{
			{"v", "k", 1}, {"k", "v2", 2},
		}, map[string]int{}},
		{"trahison exclue", []Frag{{"m", "k", 1}, {"k", "m", 2}}, map[string]int{}},
		{"suicide : la mort n'est pas vengeable", []Frag{{"k", "k", 1}, {"k", "k", 2}}, map[string]int{}},
		{"un suicide efface la mort précédente", []Frag{{"v", "k", 1}, {"k", "k", 2}, {"k", "v", 3}}, map[string]int{}},
		{"bot ou joueur non résolu : tueur vide", []Frag{{"", "k", 1}, {"k", "", 2}}, map[string]int{}},
		{"mort par un bot : pas de vengeance sur lui", []Frag{{"", "k", 1}, {"k", "", 2}, {"k", "v", 3}}, map[string]int{}},
		{"mort par un bot efface la mort précédente", []Frag{{"v", "k", 1}, {"", "k", 2}, {"k", "v", 3}}, map[string]int{}},
		{"équipe inconnue : frag gardé", []Frag{{"x", "y", 1}, {"y", "x", 2}}, map[string]int{"y": 1}},
		{"ordre du temps, pas de l'entrée", []Frag{{"k", "v", 20}, {"v", "k", 10}}, map[string]int{"k": 1}},
		{"à temps égal, ordre de l'entrée", []Frag{{"v", "k", 5}, {"k", "v", 5}}, map[string]int{"k": 1}},
		{"deux vengeurs indépendants", []Frag{
			{"v", "k", 1}, {"v2", "k2", 2}, {"k", "v", 3}, {"k2", "v2", 4},
		}, map[string]int{"k": 1, "k2": 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Count(c.frags, teams)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Count = %v, attendu %v", got, c.want)
			}
		})
	}
}

func TestCountNeModifiePasLEntree(t *testing.T) {
	in := []Frag{{"k", "v", 20}, {"v", "k", 10}}
	Count(in, nil)
	if in[0].TimeMS != 20 || in[1].TimeMS != 10 {
		t.Fatalf("l'entrée a été réordonnée : %v", in)
	}
}
