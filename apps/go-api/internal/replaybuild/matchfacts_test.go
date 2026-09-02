package replaybuild

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
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

// TestIdentifiedEventsSansFamilleNeNommeRien — LA GARDE DE MODE, ET ELLE COURT-CIRCUITE L'I/O.
//
// Sans famille d'objectif (mode sans table nommee ou variante inconnue), aucun emplacement n'est
// nomme : `identifiedEvents` rend nil AVANT de toucher au disque (le fil des morts n'est meme pas
// relu). Le fil des morts volontairement EN ERREUR le prouve — s'il etait consomme, l'appel
// journaliserait le refus et rendrait nil pour une autre raison que la garde de mode.
func TestIdentifiedEventsSansFamilleNeNommeRien(t *testing.T) {
	recs := []objectiveevents.StatRecord{{
		TimeMS: 1_000, Slot: 10, Round: 0,
		Comps: map[int]objectiveevents.StatValue{2: {A: 3, B: 1}, 3: {A: 1}},
	}}
	for nom, variant := range map[string]string{
		"mode sans famille d'objectif": "Slayer:Arena",
		"variante inconnue":            "",
	} {
		t.Run(nom, func(t *testing.T) {
			got := identifiedEvents(context.Background(), "m", filmDeaths{err: errors.New("film absent")}, recs, variant)
			if got != nil {
				t.Errorf("%d action(s), attendu nil : un mode sans table nommee ne nomme rien", len(got))
			}
		})
	}
}

// twoRoundFlagFixture — un film SYNTHETIQUE a deux manches ou le slot 22 est REATTRIBUE ("A" en
// manche 0, "B" en manche 1), avec une CAPTURE DE DRAPEAU (comp 21 A) posee en manche 1. Le score
// de mode (comp 0 A) marque les manches ; le compteur de morts (comp 2 B) RESET par manche apparie
// le slot au fil des morts. Meme grammaire que `twoRoundReassignedFixture` d'objectiveevents.
func twoRoundFlagFixture() ([]objectiveevents.NamedEvent, []objectiveevents.StatRecord, []objectiveevents.DeathInstant) {
	sv := func(comp int, side string, v int64) map[int]objectiveevents.StatValue {
		if side == "B" {
			return map[int]objectiveevents.StatValue{comp: {B: v}}
		}
		return map[int]objectiveevents.StatValue{comp: {A: v}}
	}
	rec := func(t, slot, round, comp int, side string, v int64) objectiveevents.StatRecord {
		return objectiveevents.StatRecord{TimeMS: t, Slot: slot, Round: round, Comps: sv(comp, side, v)}
	}
	recs := []objectiveevents.StatRecord{
		// Score de mode : trois emissions croissantes par manche -> manches 0 et 1 reelles.
		rec(900, 22, 0, 0, "A", 10), rec(1900, 22, 0, 0, "A", 20), rec(2900, 22, 0, 0, "A", 30),
		rec(10900, 22, 1, 0, "A", 10), rec(11900, 22, 1, 0, "A", 20), rec(12900, 22, 1, 0, "A", 30),
		// Compteur de morts, RESET par manche : slot 22.
		rec(1000, 22, 0, 2, "B", 1), rec(2000, 22, 0, 2, "B", 2), rec(3000, 22, 0, 2, "B", 3),
		rec(11000, 22, 1, 2, "B", 1), rec(12000, 22, 1, 2, "B", 2), rec(13000, 22, 1, 2, "B", 3),
		// Slot 20, stable = "C".
		rec(1500, 20, 0, 2, "B", 1), rec(2500, 20, 0, 2, "B", 2), rec(3500, 20, 0, 2, "B", 3),
		rec(11500, 20, 1, 2, "B", 1), rec(12500, 20, 1, 2, "B", 2), rec(13500, 20, 1, 2, "B", 3),
		// Capture de drapeau (comp 21 A) sur le slot 22, EN MANCHE 1.
		rec(12500, 22, 1, 21, "A", 1),
	}
	deaths := []objectiveevents.DeathInstant{
		{XUID: "A", TimeMS: 1000}, {XUID: "A", TimeMS: 2000}, {XUID: "A", TimeMS: 3000},
		{XUID: "B", TimeMS: 11000}, {XUID: "B", TimeMS: 12000}, {XUID: "B", TimeMS: 13000},
		{XUID: "C", TimeMS: 1500}, {XUID: "C", TimeMS: 2500}, {XUID: "C", TimeMS: 3500},
		{XUID: "C", TimeMS: 11500}, {XUID: "C", TimeMS: 12500}, {XUID: "C", TimeMS: 13500},
	}
	named := objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeFlag)
	return named, recs, deaths
}

// TestIdentifyRoundEventsMultiManche — LE CAS QUI FONDE LE LOT : une capture posee en manche 1 sur
// un slot REATTRIBUE est attribuee au joueur de LA MANCHE 1 ("B"), la ou le pont plat par instants
// de mort la donnait a "A" (il ne voit que la manche 0, le compteur repartant de zero).
func TestIdentifyRoundEventsMultiManche(t *testing.T) {
	named, recs, deaths := twoRoundFlagFixture()

	// La capture de drapeau doit bien avoir ete nommee sur le slot 22.
	var cap *objectiveevents.NamedEvent
	for i := range named {
		if named[i].Stat == objectiveevents.StatFlagCaptures && named[i].Slot == 22 {
			cap = &named[i]
		}
	}
	if cap == nil {
		t.Fatalf("la fixture ne nomme aucune capture de drapeau sur le slot 22 : %+v", named)
	}

	got := identifyRoundEvents(named, recs, deaths)
	var capX string
	for _, e := range got {
		if e.Stat == objectiveevents.StatFlagCaptures {
			capX = e.XUID
		}
	}
	if capX != "B" {
		t.Errorf("capture de manche 1 attribuee a %q, attendu \"B\" (slot 22 reattribue)", capX)
	}

	// CONTRE-EPREUVE : le pont plat par instants de mort la donne a "A".
	flat := objectiveevents.IdentifyNamedEvents(named, objectiveevents.SlotIdentityByDeaths(recs, deaths))
	var flatCapX string
	for _, e := range flat {
		if e.Stat == objectiveevents.StatFlagCaptures {
			flatCapX = e.XUID
		}
	}
	if flatCapX != "A" {
		t.Fatalf("pont plat : capture attribuee a %q, attendu \"A\" (temoin de la difference)", flatCapX)
	}
	if flatCapX == capX {
		t.Error("le pont par manche ne DIFFERE PAS du pont plat sur la capture de manche 1 — " +
			"la correction est nulle")
	}
}

// TestDeathInstantsOfConversion — la traduction du fil des morts (xuid decimal + instant) est la
// SEULE chose que `deathInstantsOf` fait, et une inversion casserait tout appariement en silence.
func TestDeathInstantsOfConversion(t *testing.T) {
	got := deathInstantsOf([]replay.Death{{XUID: 2533274, TimeMS: 4200}})
	if len(got) != 1 {
		t.Fatalf("%d instant(s), attendu 1", len(got))
	}
	want := objectiveevents.DeathInstant{XUID: "2533274", TimeMS: 4200}
	if got[0] != want {
		t.Errorf("instant = %+v, attendu %+v (xuid en decimal, instant en ms)", got[0], want)
	}
}
