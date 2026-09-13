package squadformes

// formes_prises_nettes_test.go — LA GRANDEUR HORS COLONNE dans la grille d'objectif.
//
// Deux propriétés, et la seconde est celle qui coûte cher si elle casse : la colonne apparaît
// bien dans le rôle « prendre » de la famille CTF, et UN MATCH SANS FILM N'Y REÇOIT PAS DE
// ZÉRO — son absence de clé porte le « non mesuré » jusqu'à l'écran.

import (
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
)

// scopeCTF : deux matchs de CTF, le premier mesuré par le film, le second non.
func scopeCTF() Input {
	return Input{
		PlayerXUID: "moi",
		Metas: []MatchMeta{
			{MatchID: "m1", ModeLabel: "CTF"},
			{MatchID: "m2", ModeLabel: "CTF"},
		},
		Objectives: []ObjectiveColumnRow{
			{MatchID: "m1", XUID: "moi", Family: narrative.FamilyCTF,
				Values: map[string]float64{
					"flag_captures":                1,
					narrative.GrandeurFlagGrabsNet: 4,
				},
				FlagJuggleWindowSeconds: 1.5},
			// m2 : le film n'a pas été lu — AUCUNE clé de prises nettes.
			{MatchID: "m2", XUID: "moi", Family: narrative.FamilyCTF,
				Values: map[string]float64{"flag_captures": 2}},
		},
	}
}

// colonne retrouve une colonne publiée par sa clé.
func colonne(t *testing.T, obj *domain.SquadFormesObjective, key string) domain.SquadFormesObjectiveColumn {
	t.Helper()
	for _, c := range obj.Columns {
		if c.Key == key {
			return c
		}
	}
	t.Fatalf("colonne %q absente des %d colonnes publiées", key, len(obj.Columns))
	return domain.SquadFormesObjectiveColumn{}
}

func TestPrisesNettes_ColonneDuRolePrendre(t *testing.T) {
	block := Build(scopeCTF())
	obj := block.Matches[0].Objective
	if obj == nil {
		t.Fatal("premier match sans bloc objectif")
	}
	c := colonne(t, obj, narrative.GrandeurFlagGrabsNet)
	if c.Role != string(narrative.ObjectiveRoleTake) {
		t.Errorf("rôle = %q, want %q", c.Role, narrative.ObjectiveRoleTake)
	}
	if !c.Optional {
		t.Error("colonne non marquée optionnelle — le web afficherait un zéro là où rien n'est mesuré")
	}
	if c.Duration {
		t.Error("colonne marquée durée — ce sont des prises, pas des secondes")
	}
	if obj.FlagJuggleWindowSeconds != 1.5 {
		t.Errorf("fenêtre publiée = %v, want 1.5 (la mesure ne se lit pas sans sa règle)",
			obj.FlagJuggleWindowSeconds)
	}
	if v := obj.Players[0].Values[narrative.GrandeurFlagGrabsNet]; v != 4 {
		t.Errorf("valeur = %v, want 4", v)
	}
}

// TestPrisesNettes_MatchSansFilmNaPasDeCle — LA propriété qui compte. La colonne existe pour la
// famille (un match du scope l'alimente), mais le match non mesuré n'a pas de valeur.
func TestPrisesNettes_MatchSansFilmNaPasDeCle(t *testing.T) {
	block := Build(scopeCTF())
	obj := block.Matches[1].Objective
	if obj == nil {
		t.Fatal("second match sans bloc objectif")
	}
	// La colonne est bien publiée — c'est la même grille pour tous les matchs de la famille.
	colonne(t, obj, narrative.GrandeurFlagGrabsNet)
	if _, ok := obj.Players[0].Values[narrative.GrandeurFlagGrabsNet]; ok {
		t.Errorf("le match sans film porte une valeur de prises nettes (%v) — ce serait un faux zéro",
			obj.Players[0].Values[narrative.GrandeurFlagGrabsNet])
	}
	// Les colonnes ORDINAIRES, elles, gardent leur zéro : il y est une mesure.
	if v, ok := obj.Players[0].Values["flag_captures"]; !ok || v != 2 {
		t.Errorf("flag_captures = (%v, %v), want (2, true)", v, ok)
	}
}

// TestPrisesNettes_HorsDuVocabulaireDesAutresFamilles — une grandeur de CTF ne doit pas
// apparaître sur la grille d'un mode à zones.
func TestPrisesNettes_HorsDuVocabulaireDesAutresFamilles(t *testing.T) {
	in := Input{
		PlayerXUID: "moi",
		Metas:      []MatchMeta{{MatchID: "z1"}},
		Objectives: []ObjectiveColumnRow{{MatchID: "z1", XUID: "moi",
			Family: narrative.FamilyZonesKOTH,
			Values: map[string]float64{"zone_captures": 3, narrative.GrandeurFlagGrabsNet: 9}}},
	}
	obj := Build(in).Matches[0].Objective
	if obj == nil {
		t.Fatal("match de zones sans bloc objectif")
	}
	for _, c := range obj.Columns {
		if c.Key == narrative.GrandeurFlagGrabsNet {
			t.Fatal("les prises nettes de drapeau figurent sur une grille de zones")
		}
	}
}
