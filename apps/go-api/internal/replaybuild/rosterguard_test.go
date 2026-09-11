package replaybuild

// rosterguard_test.go — LA GARDE D'EFFECTIF DU CALQUE DES ACTIONS (lot 6.7-B1, item 5).
//
// PREUVE D'ENTREE (audit du 2026-09-10 §6, L1). Le calque du PORTAGE du drapeau avait sa garde
// de mode et se taisait sur les films BTB ; le calque des ACTIONS n'en avait AUCUNE et publiait
// 129 actions sur les trois films BTB du parc, dont 65 « prises de drapeau » sur `4f77afc1` la
// ou l'oracle API en compte QUATRE pour les 36 participants reunis.

import (
	"context"
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/port"
)

// rosterDe fabrique des faits de match a `n` joueurs presents au coup d'envoi, sur une
// variante CTF.
func rosterDe(n int) port.MatchFacts {
	f := port.MatchFacts{GameVariantName: "CTF:Arena"}
	for i := 0; i < n; i++ {
		f.Players = append(f.Players, port.MatchPlayerFact{
			XUID: fmt.Sprintf("25334%03d", i), Kills: i, Deaths: i, Assists: i,
		})
	}
	return f
}

// TestGardeDEffectifRefuseUnFilmAuDelaDeHuitJoueurs — LE POINT DU LOT.
//
// MUTATION : retirer l'appel a `objectiveevents.RosterFitsStatborg` dans [identifiedEvents]
// rougit ce test — les actions ressortent alors publiees, exactement comme avant ce lot.
func TestGardeDEffectifRefuseUnFilmAuDelaDeHuitJoueurs(t *testing.T) {
	recs := recordsCTFDeTest()
	pont := func() *pontParManche { return &pontParManche{recs: recs} }

	nommees := len(objectiveevents.NamedEventsFrom(recs, objectiveevents.ObjectiveTypeFlag))
	if nommees == 0 {
		t.Fatal("la fixture ne nomme aucune action : le test ne prouverait rien")
	}

	// TEMOIN — a huit joueurs, le calque n'est PAS refuse : ce que le film nomme lui parvient
	// (ici sans pont d'identite, donc tout part en `nonNommes` — le sujet est le REFUS).
	_, nonNommes, refuses := identifiedEvents(context.Background(), "m", filmDeaths{}, recs,
		rosterDe(objectiveevents.StatPlayerSlots), pont())
	if refuses != 0 || nonNommes != nommees {
		t.Fatalf("temoin : %d refusee(s), %d non nommee(s) sur %d — attendu 0 refus a "+
			"%d joueurs", refuses, nonNommes, nommees, objectiveevents.StatPlayerSlots)
	}

	// LA REGLE — un joueur de plus, et le calque se tait ENTIEREMENT.
	got, nonNommes, refuses := identifiedEvents(context.Background(), "m", filmDeaths{}, recs,
		rosterDe(objectiveevents.StatPlayerSlots+1), pont())
	if len(got) != 0 {
		t.Errorf("%d action(s) publiee(s), attendu aucune : l'effectif depasse le format",
			len(got))
	}
	if nonNommes != 0 {
		t.Errorf("nonNommes = %d, attendu 0 : un refus d'effectif n'est pas un trou de pont",
			nonNommes)
	}
	if refuses != nommees {
		t.Errorf("refuses = %d, attendu %d : le compte du refus est le DENOMINATEUR, il porte "+
			"tout ce que le film nommait", refuses, nommees)
	}
}

// TestGardeDEffectifNeRefuseRienSansFaitsDeMatch — LA CONTRE-EPREUVE. Une cuisson sans faits de
// match ne connait pas l'effectif : la garde refuse ce qu'elle MESURE, jamais ce qu'elle ignore.
func TestGardeDEffectifNeRefuseRienSansFaitsDeMatch(t *testing.T) {
	recs := recordsCTFDeTest()
	got, _, refuses := identifiedEvents(context.Background(), "m", filmDeaths{}, recs,
		port.MatchFacts{GameVariantName: "CTF:Arena"}, &pontParManche{recs: recs})
	if refuses != 0 {
		t.Errorf("refuses = %d, attendu 0 : sans ligne de match, l'effectif est INCONNU", refuses)
	}
	if got == nil {
		t.Error("sortie nil : le calque hors ligne doit rester publiable")
	}
}

// TestSiegesAuCoupDEnvoiNeCompteNiBotNiRemplacant — LE SIEGE, PAS LA PERSONNE.
//
// C'est la forme EXACTE de cinq films d'arene du parc (`8bc6074f`, `396cfc92`, `572e236b`,
// `4ecdf3e7`, `bf5ced1b`) : neuf ou dix lignes de feuille pour HUIT sieges — un joueur part,
// un bot tient la place, un remplacant arrive. Compter les lignes faisait refuser ces films et
// coutait 633 actions EXACTES.
//
// MUTATION : compter `len(facts.Players)` au lieu des sieges rougit ce test.
func TestSiegesAuCoupDEnvoiNeCompteNiBotNiRemplacant(t *testing.T) {
	f := rosterDe(objectiveevents.StatPlayerSlots) // huit joueurs au coup d'envoi
	f.Players = append(f.Players,
		port.MatchPlayerFact{XUID: "bid(7.0)"},                                 // le bot qui tient la place
		port.MatchPlayerFact{XUID: "2535461109438273", JoinedInProgress: true}) // le remplacant

	if got := siegesAuCoupDEnvoi(f); got != objectiveevents.StatPlayerSlots {
		t.Errorf("%d sieges pour %d lignes, attendu %d", got, len(f.Players),
			objectiveevents.StatPlayerSlots)
	}
	// Et le calque n'est donc PAS refuse.
	_, _, refuses := identifiedEvents(context.Background(), "m", filmDeaths{}, recordsCTFDeTest(),
		f, &pontParManche{recs: recordsCTFDeTest()})
	if refuses != 0 {
		t.Errorf("%d action(s) refusee(s) : un 4v4 a remplacants tient dans le format", refuses)
	}
}

// TestRosterFitsStatborgSuitLesConstantesDeFormat — le seuil est DERIVE de la bande de slots du
// statborg, pas ecrit en dur.
func TestRosterFitsStatborgSuitLesConstantesDeFormat(t *testing.T) {
	if objectiveevents.StatPlayerSlots != 8 {
		t.Fatalf("StatPlayerSlots = %d, attendu 8 (slots 10, 12, ... 24)",
			objectiveevents.StatPlayerSlots)
	}
	for _, n := range []int{0, 1, 8} {
		if !objectiveevents.RosterFitsStatborg(n) {
			t.Errorf("un effectif de %d refuse alors qu'il tient", n)
		}
	}
	for _, n := range []int{9, 26, 36} {
		if objectiveevents.RosterFitsStatborg(n) {
			t.Errorf("un effectif de %d accepte alors qu'il ne tient pas", n)
		}
	}
}

// recordsCTFDeTest fabrique des enregistrements de statborg qui portent des prises de drapeau
// (`comp 22 A`) sur deux slots de joueur, plus les progressions du compteur de morts qui les
// nomment.
func recordsCTFDeTest() []objectiveevents.StatRecord {
	var out []objectiveevents.StatRecord
	for i := 1; i <= 4; i++ {
		for _, slot := range []int{10, 12} {
			out = append(out, objectiveevents.StatRecord{
				TimeMS: i * 1_000, Slot: slot,
				Comps: map[int]objectiveevents.StatValue{22: {A: int64(i)}},
			})
		}
	}
	return out
}
