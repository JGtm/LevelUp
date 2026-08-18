package replaybuild

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/port"
)

// matchfacts_test.go — LE PONT ENTRE LES FAITS DE BASE ET LE DECODEUR (revue R1, 2026-08-18).
//
// POURQUOI CES TESTS EXISTENT. Ce fichier traduit `port.MatchFacts` en entrees du decodeur, et
// AUCUN test ne le couvrait : une inversion de deux champs du triplet (morts et assistances, par
// exemple) aurait apparie les slots aux mauvais joueurs, sans que rien ne rougisse — l'erreur
// aurait ete invisible a l'ecran et credible.

// TestPlayerLinesGardentLOrdreDuTriplet — LE TRIPLET EST UNE CLE, PAS UN AFFICHAGE.
//
// Les trois compteurs sont volontairement DISTINCTS deux a deux : intervertir n'importe quelle
// paire fait echouer ce test. Avec des valeurs egales, il ne prouverait rien.
func TestPlayerLinesGardentLOrdreDuTriplet(t *testing.T) {
	facts := port.MatchFacts{Players: []port.MatchPlayerFact{
		{XUID: "2533274", Kills: 11, Deaths: 7, Assists: 3, TeamID: 0},
	}}
	got := playerLines(facts)
	if len(got) != 1 {
		t.Fatalf("%d ligne(s), attendu 1", len(got))
	}
	want := objectiveevents.PlayerLine{XUID: "2533274", Kills: 11, Deaths: 7, Assists: 3}
	if got[0] != want {
		t.Errorf("ligne = %+v, attendu %+v — l'ordre du triplet (frags, morts, assistances) "+
			"est la CLE d'appariement du slot d'entite au xuid", got[0], want)
	}
}

// TestPlayerLinesSansJoueurRendNil — sans ligne de match, aucune ligne : le pont d'identite ne
// peut pas s'appuyer sur du vide.
func TestPlayerLinesSansJoueurRendNil(t *testing.T) {
	if got := playerLines(port.MatchFacts{}); got != nil {
		t.Errorf("obtenu %+v, attendu nil", got)
	}
}

// TestTeamByXUIDEcarteUnCampInconnu — un camp inconnu vaut -1 et N'ENTRE PAS dans la table.
//
// L'y faire entrer ferait entrer un faux camp dans la somme des frags qui identifie les slots
// d'equipe — et le camp 0 est un camp REEL, on ne peut pas s'en servir comme valeur d'absence.
func TestTeamByXUIDEcarteUnCampInconnu(t *testing.T) {
	facts := port.MatchFacts{Players: []port.MatchPlayerFact{
		{XUID: "a", TeamID: 0}, {XUID: "b", TeamID: 1}, {XUID: "c", TeamID: -1},
	}}
	got := teamByXUID(facts)
	if len(got) != 2 {
		t.Fatalf("%d entree(s), attendu 2 : %+v", len(got), got)
	}
	if _, present := got["c"]; present {
		t.Error("le joueur au camp inconnu est entre dans la table")
	}
	if got["a"] != 0 {
		t.Errorf("le camp 0 est un camp REEL, il doit etre conserve : %+v", got)
	}
}

// TestTeamByXUIDSansCampConnuRendNil — quand AUCUN camp n'est connu, la table est nil et non
// vide : la resolution d'identite (b) doit pouvoir distinguer les deux.
func TestTeamByXUIDSansCampConnuRendNil(t *testing.T) {
	facts := port.MatchFacts{Players: []port.MatchPlayerFact{{XUID: "a", TeamID: -1}}}
	if got := teamByXUID(facts); got != nil {
		t.Errorf("obtenu %+v, attendu nil", got)
	}
}

// TestIdentifiedEventsRefuseDeuxCas — LES DEUX REFUS SONT EXPLICITES.
//
// Sans famille d'objectif, aucun nom n'est possible ; sans ligne de match, aucun slot ne peut
// etre apparie. Dans les deux cas on ne publie RIEN — poser une action sur un slot arbitraire est
// precisement l'erreur que le pont d'identite existe pour eviter.
func TestIdentifiedEventsRefuseDeuxCas(t *testing.T) {
	recs := []objectiveevents.StatRecord{{
		TimeMS: 1_000, Slot: 10, Round: 0,
		Comps: map[int]objectiveevents.StatValue{2: {A: 3, B: 1}, 3: {A: 1}},
	}}
	lignes := []objectiveevents.PlayerLine{{XUID: "x10", Kills: 3, Deaths: 1, Assists: 1}}

	for nom, cas := range map[string]struct {
		lignes  []objectiveevents.PlayerLine
		variant string
	}{
		"mode sans famille d'objectif": {lignes, "Slayer:Arena"},
		"variante inconnue":            {lignes, ""},
		"aucune ligne de match":        {nil, "CTF:Arena"},
	} {
		t.Run(nom, func(t *testing.T) {
			got := identifiedEvents(context.Background(), "m", recs, cas.lignes, cas.variant)
			if got != nil {
				t.Errorf("%d action(s) publiee(s), attendu aucune : %+v", len(got), got)
			}
		})
	}
}

// TestIdentifiedEventsNommeQuandLesDeuxSontLa — le temoin positif : avec une famille d'objectif
// ET des lignes de match, le pont fonctionne. Sans lui, les refus ci-dessus ne prouveraient rien.
func TestIdentifiedEventsNommeQuandLesDeuxSontLa(t *testing.T) {
	var recs []objectiveevents.StatRecord
	// Composant 2 (frags et morts) et 3 (assistances) : le triplet qui apparie le slot.
	// Composant 4 valeur A : les captures de drapeau nommees par la table `flag`.
	for i, v := range []int64{1, 2} {
		recs = append(recs, objectiveevents.StatRecord{
			TimeMS: 1_000 + i*1_000, Slot: 10, Round: 0,
			Comps: map[int]objectiveevents.StatValue{
				2: {A: 3, B: 1}, 3: {A: 1}, 4: {A: v},
			},
		})
	}
	lignes := []objectiveevents.PlayerLine{{XUID: "x10", Kills: 3, Deaths: 1, Assists: 1}}
	got := identifiedEvents(context.Background(), "m", recs, lignes, "CTF:Arena")
	if len(got) == 0 {
		t.Fatal("aucune action identifiee alors que la famille et les lignes sont la")
	}
	for _, e := range got {
		if e.XUID != "x10" {
			t.Errorf("action attribuee a %q, attendu x10 : %+v", e.XUID, e)
		}
	}
}
