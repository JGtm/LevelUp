package main

import (
	"testing"

	"levelup/go-api/internal/analysis/tactical"
)

func TestFamilleDeMode(t *testing.T) {
	cas := []struct{ variante, categorie, attendu string }{
		{"Team Slayer:Arena", "Assassin", familleAssassin},
		{"Ranked:Slayer", "Ranked", familleAssassin},
		{"Slayer:Arena Super Fiesta", "Super Fiesta", familleFiesta},
		{"Slayer:Arena Fiesta", "Assassin", familleFiesta},
		{"CTF:Arena", "Assassin", familleDrapeau},
		{"Strongholds:Arena", "Assassin", familleBases},
		{"KOTH:Arena", "Assassin", familleColline},
		{"Ranked:King of the Hill", "Ranked", familleColline},
		{"Oddball:Arena", "Assassin", familleOddball},
		{"BTB:CTF", "BTB", ""},
		{"CTF:BTB", "BTB", ""},
		{"Slayer:BTB Heavies", "BTB", ""},
		{"BTB:Fiesta Slayer", "BTB", ""},
		{"Husky Raid:CTF", "Husky Raid", ""},
		{"Arena:VIP", "Assassin", ""},
		{"", "", ""},
	}
	for _, c := range cas {
		if got := familleDeMode(c.variante, c.categorie); got != c.attendu {
			t.Errorf("familleDeMode(%q, %q) = %q, attendu %q", c.variante, c.categorie, got, c.attendu)
		}
	}
}

func TestEvenementsDuMatch_DeuxInstantanes(t *testing.T) {
	pistes := []tactical.Piste{
		{XUID: "a", Equipe: 0, Points: []tactical.PointPiste{{T: 0, X: 1}, {T: 100, X: 2}}},
		{XUID: "a", Equipe: 0, Points: []tactical.PointPiste{{T: 150, X: 30}, {T: 300, X: 31}}},
		{XUID: "e", Equipe: 1, Points: []tactical.PointPiste{{T: 0, X: 50}, {T: 400, X: 60}}},
		{XUID: "m", Equipe: 0, Points: []tactical.PointPiste{{T: 0, X: 10}, {T: 400, X: 10}}},
		{XUID: "sans", Equipe: tactical.EquipeInconnue, Points: []tactical.PointPiste{{T: 0, X: 5}, {T: 100, X: 5}}},
		{XUID: "sans", Equipe: tactical.EquipeInconnue, Points: []tactical.PointPiste{{T: 150, X: 6}, {T: 300, X: 6}}},
	}
	e := tactical.EntreeOccupation{MatchID: "m1", IntervalleFrameMs: 100, Pistes: pistes}
	jeu := &jeuReapparitions{}
	jeu.bilan.parFamille = map[string]int{}
	jeu.evenementsDuMatch(e, infoMatch{matchID: "m1", carteID: "c", famille: familleAssassin})

	if len(jeu.evts) != 1 {
		t.Fatalf("une reapparition mesurable attendue (celle de \"sans\" n'a pas d'equipe), obtenu %d", len(jeu.evts))
	}
	if jeu.bilan.reapRepli != 2 || jeu.bilan.sansEquipe != 1 {
		t.Errorf("bilan = %+v", jeu.bilan)
	}
	ev := jeu.evts[0]
	// A la reapparition (frame 150) : l'adversaire "e" tient sa derniere position connue (x = 50),
	// le coequipier "m" est a 10 ; "sans" (equipe inconnue) n'est dans aucun des deux camps.
	if len(ev.reap.Ennemis) != 1 || ev.reap.Ennemis[0].X != 50 || len(ev.reap.Equipiers) != 1 {
		t.Errorf("instant de la reapparition = %+v", ev.reap)
	}
	// A la mort (frame 100) : mort de "a" ; les autres sont les memes.
	if len(ev.mort.Ennemis) != 1 || ev.mort.Mort.X != 2 || ev.mort.Reel.X != 30 {
		t.Errorf("instant de la mort = %+v", ev.mort)
	}
}

func TestPoidsArrondis(t *testing.T) {
	w := poidsArrondis(map[string]float64{"a": 2.001339496453945, "b": -0.65318830})
	if w["a"] != 2.0013 || w["b"] != -0.6532 {
		t.Errorf("poids arrondis au dix-millieme : %v", w)
	}
	if arrondi1(62.76) != 62.8 {
		t.Errorf("arrondi1(62.76) = %v", arrondi1(62.76))
	}
}
