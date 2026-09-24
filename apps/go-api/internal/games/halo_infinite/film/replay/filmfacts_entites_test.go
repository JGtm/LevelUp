package replay

// filmfacts_entites_test.go — LES OCCUPANTS DU MATCH TRAVERSENT LE FICHIER DE FAITS A L IDENTIQUE
// (lot M2.2, 2026-09-23).
//
// `TestCodecCouvreFilmInputs` ne demande qu une chose d un champ : qu il ne se relise pas VIDE.
// Ce test-ci demande l EGALITE : chaque entite (slot, index, designateur, rangs, trous comptes,
// instabilite), les instants des images-cles porteuses et le temoin `Scanned` — y compris le cas
// « balaye, personne », qui ne doit pas se relire « non balaye ».

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/killsource"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

func TestFaitsTransportentLesEntitesDesJoueurs(t *testing.T) {
	entry := goldenEntryPourTest(t)
	cas := map[string]grammar.PlayerEntityScan{
		"temoin b1ad85eb": {
			Scanned:     true,
			KeyframesUS: []uint64{1_576_675_905, 1_576_695_905, 1_576_715_905, 1_576_736_005},
			Entities: []grammar.PlayerEntity{
				{Slot: 1297, Index: 0, Team: 0, FirstKF: 0, LastKF: 3, Seen: 3},
				{Slot: 1530, Index: 8, Team: 0, FirstKF: 0, LastKF: 1, Seen: 2},
				{Slot: 2145, Index: 8, Team: 1, FirstKF: 3, LastKF: 3, Seen: 1},
				{Slot: 9, Index: 2, Team: grammar.TeamNone, FirstKF: 2, LastKF: 2, Seen: 1, Unstable: true},
			},
			// Les absences non prouvees (lot D-fix) voyagent avec les entites.
			Doutes: []grammar.DouteDAbsence{{Rang: 0, Slot: 2145}, {Rang: 2, Slot: 1530}},
		},
		"balaye, personne": {Scanned: true},
		"non balaye":       {},
	}
	for nom, scan := range cas {
		t.Run(nom, func(t *testing.T) {
			f := &FilmFactsFile{Facts: FilmFacts{Film: goldenFilm, MapModule: entry.Module,
				AxisW: entry.AxisWidths}}
			f.Facts.PlayerEntities = scan
			blob, err := EncodeFilmFactsFile(f)
			if err != nil {
				t.Fatalf("encodage : %v", err)
			}
			relu, err := DecodeFilmFactsFile(blob, entry)
			if err != nil {
				t.Fatalf("relecture : %v", err)
			}
			if !reflect.DeepEqual(relu.Facts.PlayerEntities, scan) {
				t.Fatalf("entites relues :\n %+v\n attendu :\n %+v", relu.Facts.PlayerEntities, scan)
			}
		})
	}
}

// TestFaitsTransportentLesDeclarationsDesBots : la section 5 (le resultat du kill-feed, en JSON)
// porte les INSTANTS de BOT_METADATA de chaque bot (lot M2.1) — une declaration fermee et une
// declaration ouverte (`ToUS` nul = declare jusqu'au bout) se relisent a l'identique, et le
// compteur des paquets incomplets aussi. Sans elles, un rejeu depuis les faits perdrait le lien
// bot -> entite.
func TestFaitsTransportentLesDeclarationsDesBots(t *testing.T) {
	entry := goldenEntryPourTest(t)
	bots := []killsource.BotEntry{
		{Slot: 8, BotID: 16, Name: "343 Hundy", Declarations: []killsource.BotDeclaration{
			{FromUS: 1_576_675_905, ToUS: 1_576_702_905}}},
		{Slot: 8, BotID: 19, Name: "343 Brew Dog", Declarations: []killsource.BotDeclaration{
			{FromUS: 1_576_990_905}}},
	}
	f := &FilmFactsFile{Facts: FilmFacts{Film: goldenFilm, MapModule: entry.Module, AxisW: entry.AxisWidths},
		Kills: &killsource.Result{Roster: killsource.Roster{Bots: bots, BotPaquetsIncomplets: 2}}}
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	relu, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("relecture : %v", err)
	}
	if relu.Kills == nil || !reflect.DeepEqual(relu.Kills.Roster.Bots, bots) ||
		relu.Kills.Roster.BotPaquetsIncomplets != 2 {
		t.Fatalf("bots relus %+v : attendu %+v et 2 paquets incomplets", relu.Kills, bots)
	}
}
