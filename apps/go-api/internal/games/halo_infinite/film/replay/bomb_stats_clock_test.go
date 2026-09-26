package replay

// bomb_stats_clock_test.go — SANS HORLOGE DU FILM, `bomb_arms` EST ABSENT, JAMAIS UN ZERO MESURE
// (constat RB1-3 de l audit du 2026-09-24, lot J9.6).
//
// LE DEFAUT : la jointure de `bomb_arms` recale l instant arme (horloge du FILM) sur l horloge du
// MATCH par `premierPaquetDuFilmUS/1000 - deathOffsetMS`. Quand l origine d horloge du film est
// illisible (`FilmClockOriginUS == 0` : chunk 1 absent ou tronque), ce terme valait
// `-deathOffsetMS` — un recalage FAUX de plusieurs secondes, et la jointure tournait quand meme :
// les armements tombaient hors de toute fenetre, et chaque joueur sortait avec `bomb_arms = 0`,
// persiste comme un zero MESURE. Ou pire, comme ici, un armement tombait par hasard dans la
// fenetre d un autre lacher.

import "testing"

// bombClockDoc monte un document arme : un armement date a 17 000 ms sur l horloge du FILM, et
// une periode de portage de « 7 » fermee PAR LACHER a 12 000 ms sur celle du MATCH. Avec une
// origine lue (1 000 000 us) et un calage de 6 000 ms, le recalage vaut -5 000 ms : l armement
// tombe a 12 000 ms, sur le lacher.
func bombClockDoc(originUS uint64) (ReplayDocument, Options, IdentityRegistry, HeldObjectCarry) {
	doc := ReplayDocument{
		MatchID:     "m",
		BombArmings: []BombArming{{TimeMS: 17_000}},
		Coverage:    &Coverage{BombArmings: &BombArmingsCoverage{Scanned: true}},
	}
	opt, own := bombDocOptions(MatchKillsInput{})
	opt.FilmClockOriginUS = originUS
	return doc, opt, own, bombCarryDe(bombPeriode(7, 10_000, 12_000))
}

// TestBombArmsAbsentSansHorlogeDuFilm — LE POINT DU LOT. Origine illisible : aucun armement n est
// attribue, `bomb_arms` est ABSENT chez tous, et l armement reste publie comme fait date, sans
// acteur.
//
// MUTATION : retirer la garde `ClockRead` de `bombArmsByXUID` rougit ce test.
func TestBombArmsAbsentSansHorlogeDuFilm(t *testing.T) {
	doc, opt, own, carry := bombClockDoc(0)
	attachBombStats(&doc, opt, own, carry)
	if doc.BombStats == nil {
		t.Fatal("aucune statistique posee sur le document")
	}
	assertBombAbsent(t, *doc.BombStats, func(p BombPlayerStats) bool { return p.Arms != nil }, "bomb_arms")
	if len(doc.BombEvents) != 1 || doc.BombEvents[0].Type != BombEventArmed || doc.BombEvents[0].XUID != "" {
		t.Errorf("faits dates %+v : attendu l armement publie SANS acteur", doc.BombEvents)
	}
	cov := doc.BombStats.Coverage
	if !cov.ArmingsRead || cov.Armings != 1 || cov.ArmingsNoClock != 1 || cov.ArmingsAttributed != 0 ||
		cov.ArmingsNoCarrier != 0 {
		t.Errorf("couverture %+v : attendu 1 armement lu, publie sans acteur faute d horloge", cov)
	}
}

// TestBombArmsZeroMesureSansArmement — sans AUCUN armement, il n y a rien a joindre : le zero de
// `bomb_arms` est mesure, horloge du film lue ou non.
func TestBombArmsZeroMesureSansArmement(t *testing.T) {
	doc, opt, own, carry := bombClockDoc(0)
	doc.BombArmings = nil
	attachBombStats(&doc, opt, own, carry)
	if doc.BombStats == nil {
		t.Fatal("aucune statistique posee sur le document")
	}
	assertBombInt(t, *doc.BombStats, "7", func(p BombPlayerStats) *int { return p.Arms }, 0, "bomb_arms")
	if doc.BombStats.Coverage.ArmingsNoClock != 0 {
		t.Errorf("%d armements sans horloge, attendu 0", doc.BombStats.Coverage.ArmingsNoClock)
	}
}

// TestBombArmsMesureAvecHorlogeDuFilm — LE TEMOIN : origine lue, l armement est attribue a « 7 »
// par son lacher, et `bomb_arms` est mesure.
func TestBombArmsMesureAvecHorlogeDuFilm(t *testing.T) {
	doc, opt, own, carry := bombClockDoc(1_000_000)
	attachBombStats(&doc, opt, own, carry)
	if doc.BombStats == nil {
		t.Fatal("aucune statistique posee sur le document")
	}
	assertBombInt(t, *doc.BombStats, "7", func(p BombPlayerStats) *int { return p.Arms }, 1, "bomb_arms")
}
