package shieldcycle

import (
	"reflect"
	"testing"
)

func ms(pairs ...float32) []Measure {
	out := make([]Measure, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, Measure{Frame: int(pairs[i]), Shield: pairs[i+1]})
	}
	return out
}

func ev(kinds ...any) []Event {
	out := []Event{}
	for i := 0; i+1 < len(kinds); i += 2 {
		out = append(out, Event{Frame: kinds[i].(int), Kind: kinds[i+1].(Kind)})
	}
	return out
}

func TestCycles(t *testing.T) {
	cases := []struct {
		name  string
		lives []Life
		want  map[string][]Event
	}{
		{"aucune vie", nil, map[string][]Event{}},
		{"vie sans mesure : non mesuré", []Life{{XUID: "a"}}, map[string][]Event{}},
		{"vie sans xuid ignorée", []Life{{Measures: ms(1, 0)}}, map[string][]Event{}},
		{"mesuré sans événement", []Life{{XUID: "a", Measures: ms(1, 1, 2, 0.5)}}, map[string][]Event{"a": nil}},
		{"bris puis recharge", []Life{{XUID: "a", Measures: ms(1, 1, 2, 0, 3, 0.4, 4, 1)}},
			map[string][]Event{"a": ev(2, Break, 4, Recharge)}},
		{"première mesure à 0 compte comme bris", []Life{{XUID: "a", Measures: ms(5, 0, 6, 0, 7, 1)}},
			map[string][]Event{"a": ev(5, Break, 7, Recharge)}},
		{"mesures à 0 consécutives : un seul bris", []Life{{XUID: "a", Measures: ms(1, 0.5, 2, 0, 3, 0, 4, 0)}},
			map[string][]Event{"a": ev(2, Break)}},
		{"recharge partielle puis nouveau bris : deux bris, aucune recharge", []Life{{XUID: "a", Measures: ms(1, 1, 2, 0, 3, 0.6, 4, 0)}},
			map[string][]Event{"a": ev(2, Break, 4, Break)}},
		{"deux recharges pour un bris : une seule compte", []Life{{XUID: "a", Measures: ms(1, 0, 2, 1, 3, 1.2, 4, 1)}},
			map[string][]Event{"a": ev(1, Break, 2, Recharge)}},
		{"surbouclier : plein atteint", []Life{{XUID: "a", Measures: ms(1, 0, 2, 1.5)}},
			map[string][]Event{"a": ev(1, Break, 2, Recharge)}},
		{"plein sans bris : aucune recharge", []Life{{XUID: "a", Measures: ms(1, 1, 2, 1.5)}},
			map[string][]Event{"a": nil}},
		{"bris, recharge, nouveau bris, nouvelle recharge", []Life{{XUID: "a", Measures: ms(1, 0.9, 2, 0, 3, 1, 4, 0, 5, 1)}},
			map[string][]Event{"a": ev(2, Break, 3, Recharge, 4, Break, 5, Recharge)}},
		{"un respawn n'est jamais une recharge", []Life{
			{XUID: "a", Measures: ms(1, 0.5, 2, 0)},
			{XUID: "a", Measures: ms(10, 1)},
		}, map[string][]Event{"a": ev(2, Break)}},
		{"un bris en attente ne passe pas à la vie suivante", []Life{
			{XUID: "a", Measures: ms(1, 0)},
			{XUID: "a", Measures: ms(10, 0.2, 11, 1)},
		}, map[string][]Event{"a": ev(1, Break)}},
		{"vies mises bout à bout : événements triés par image", []Life{
			{XUID: "a", Measures: ms(20, 0, 21, 1)},
			{XUID: "a", Measures: ms(3, 0, 4, 1)},
		}, map[string][]Event{"a": ev(3, Break, 4, Recharge, 20, Break, 21, Recharge)}},
		{"deux joueurs", []Life{
			{XUID: "a", Measures: ms(1, 0)},
			{XUID: "b", Measures: ms(2, 1, 3, 0)},
		}, map[string][]Event{"a": ev(1, Break), "b": ev(3, Break)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Cycles(c.lives)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Cycles = %v, attendu %v", got, c.want)
			}
		})
	}
}

func TestTotals(t *testing.T) {
	b, r := Totals(ev(1, Break, 2, Recharge, 3, Break, 4, Break))
	if b != 3 || r != 1 {
		t.Fatalf("Totals = (%d, %d), attendu (3, 1)", b, r)
	}
	if b, r := Totals(nil); b != 0 || r != 0 {
		t.Fatalf("Totals(nil) = (%d, %d)", b, r)
	}
}
