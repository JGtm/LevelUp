package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// sieges_tireur_test.go — LA VIE QUI TIRE REFUSE LA DEDUCTION QU'ELLE CONTREDIT (lot R2 du plan de
// suite de l'audit du decodeur, 2026-09-28, constat C6 du G-corpus J11.1).
//
// La figure de `4f77afc1` : la place 5 de la table prend l'equipe 0 de son titulaire, present
// TARD ; un arrivant d'equipe 1 (`320`) tire sous l'index 5 avant, et l'equipe l'ecarte ; l'unique
// arrivant d'equipe 0 present (`310`), qui n'a pas tire, recevait ces votes. Ses propres tirs
// (place 6) etaient alors « contestes » et sa place perdue.
//
// ROUGE AVANT : `310` conteste (votes {5, 6}), source `apparie`.
// MUTATION : retirer le refus des votes contredits -> rouge.
func TestPlaceDesTirs_LaVieQuiTireRefuseLeVoteQuElleContredit(t *testing.T) {
	roster := []RosterEntry{entree(0, "100"), {FilmIndex: 5, XUID: "500"}, {FilmIndex: 6, XUID: "600"},
		entree(10, "310"), entree(11, "320")}
	occ := occupantsFabriques([]int{0, 0, 0, 0, 1}, iv(0, 99), iv(80, 99), nil, iv(20, 99), iv(10, 40))
	tirs := []FireEventRef{
		{FilmIndex: 5, TimestampUS: 3_000_000}, {FilmIndex: 5, TimestampUS: 3_500_000}, // tires par 320
		{FilmIndex: 6, TimestampUS: 5_000_000}, // tire par 310
	}
	tireurs := []string{"320", "320", "310"}
	cov := poserLesSieges(roster, occ, entreesDesPlaces{table: tableDeDebut(0, 5, 6), fire: tirs,
		tireurs: tireurs, horloge: horlogeDeSieges(nil)})

	if roster[3].Seat != 6 || roster[3].SeatSource != SeatSourceTirs {
		t.Fatalf("310 : place %d (%s), attendu 6 lue par SES tirs — les tirs de 320 ne votent pas pour lui",
			roster[3].Seat, roster[3].SeatSource)
	}
	if cov.TirsContestes != 0 || cov.PlacesTirs != 1 {
		t.Errorf("couverture %+v : attendu 1 place lue, 0 contestee", cov)
	}
	// Sans la lecture de l'unite, la deduction d'avant reste (et conteste).
	roster2 := []RosterEntry{entree(0, "100"), {FilmIndex: 5, XUID: "500"}, {FilmIndex: 6, XUID: "600"},
		entree(10, "310"), entree(11, "320")}
	cov2 := poserLesSieges(roster2, occ, entreesDesPlaces{table: tableDeDebut(0, 5, 6), fire: tirs,
		horloge: horlogeDeSieges(nil)})
	if cov2.TirsContestes != 1 {
		t.Errorf("sans unite lue : %d contestation(s), attendu 1 (la deduction seule)", cov2.TirsContestes)
	}
}

// TestTireursDesTirs : l'unite du tir designe la piste publiee de son slot qui couvre la frame ; un
// tir sans unite, ou dont l'unite ne porte aucune piste a cet instant, ne designe personne.
func TestTireursDesTirs(t *testing.T) {
	tracks := []Track{
		{Slot: 544, StartFrame: 8, EndFrame: 14, XUID: "28"},
		{Slot: 544, StartFrame: 30, EndFrame: 40, XUID: "1"},
		{Slot: 600, StartFrame: 0, EndFrame: 99, Bot: "343 Bot"},
	}
	unite := func(slot uint32) grammar.UnitRef { return grammar.UnitRef{Present: true, Slot: slot} }
	fire := []grammar.FireEvent{
		{TimestampUS: 1_000_000, Unit: unite(544)}, // frame 10 : premier corps
		{TimestampUS: 3_500_000, Unit: unite(544)}, // frame 35 : second corps du meme slot
		{TimestampUS: 2_000_000, Unit: unite(544)}, // frame 20 : aucun corps
		{TimestampUS: 1_000_000},                   // sans unite
		{TimestampUS: 1_000_000, Unit: unite(600)}, // un bot
	}
	got := tireursDesTirs(fire, tracks, horlogeDeSieges(nil))
	want := []string{"28", "1", "", "", "bot:343 Bot"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tir %d : tireur %q, attendu %q", i, got[i], want[i])
		}
	}
}
