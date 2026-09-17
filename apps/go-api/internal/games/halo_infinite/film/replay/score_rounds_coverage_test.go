package replay

// score_rounds_coverage_test.go — CE QUE LA COUVERTURE DU SCORE DIT DES MANCHES (lot 1.9.11).
//
// Trois formes, trois questions : le designateur ECRIT est-il publie ? la CONTRADICTION est-elle
// comptee au lieu d'etre tue ? le REPLI est-il declare ET compte au registre ?

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// manche2SansManche1Records reproduit le motif mesure sur `fb1a1a72` et 23 autres films du cache
// (lot 1.9.11) : manche 0 pleine, manche 1 ABSENTE, designateur 2 materiel.
func manche2SansManche1Records() []types.StatRecord {
	var recs []types.StatRecord
	recs = append(recs, manchesRecordsDeSlotJoueur(0, 1_000, 900, 200)...)
	recs = append(recs, manchesRecordsDeSlotJoueur(2, 60_000, 148, 0)...)
	return recs
}

// manchesRecordsDeSlotJoueur fabrique une manche MATERIELLE : n enregistrements repartis sur les
// huit slots de joueur, un point de score de mode a la fin pour le slot qui marque.
func manchesRecordsDeSlotJoueur(round, startMS, n int, scoreFinal int64) []types.StatRecord {
	out := make([]types.StatRecord, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, types.StatRecord{
			TimeMS: startMS + i*100, Slot: 10 + 2*(i%8), Round: round,
			Comps: map[int]types.StatValue{0: {A: 0}},
		})
	}
	if n > 0 && scoreFinal > 0 {
		out[n-1].Comps = map[int]types.StatValue{0: {A: scoreFinal}}
	}
	return out
}

// TestCouvertureScorePublieLeDesignateurEcritEtLaContradiction — LE CAS `fb1a1a72`.
func TestCouvertureScorePublieLeDesignateurEcritEtLaContradiction(t *testing.T) {
	_, cov := buildScoreTimeline(
		&ScoreInput{Records: manche2SansManche1Records()}, nil, testClock(), nil)
	if cov == nil {
		t.Fatal("aucune couverture publiee")
	}

	if cov.Rounds != 1 {
		t.Errorf("rounds = %d, attendu 1 : la regle d'ordre garde son verdict", cov.Rounds)
	}
	if want := []int{0, 2}; !reflect.DeepEqual(cov.RoundsWritten, want) {
		t.Errorf("roundsWritten = %v, attendu %v : le denominateur du fait doit etre lisible "+
			"dans l'artefact, sans relire le film", cov.RoundsWritten, want)
	}
	if want := []int{2}; !reflect.DeepEqual(cov.RoundsContradicted, want) {
		t.Errorf("roundsContradicted = %v, attendu %v", cov.RoundsContradicted, want)
	}
	if cov.RoundsContradictedRecords != 148 {
		t.Errorf("roundsContradictedRecords = %d, attendu 148", cov.RoundsContradictedRecords)
	}
	if cov.RoundsDecreed {
		t.Error("roundsDecreed vrai alors que la manche 0 est admise par la matiere")
	}
}

// TestCouvertureScoreTaitLesChampsDeMancheSurUnFilmMonoManche — LA FORME NE BOUGE PAS POUR RIEN.
//
// Un film mono-manche sans contradiction ni repli ne porte AUCUN des champs neufs : la fixture
// web et les 1 300 artefacts du parc gardent la forme qu'ils ont.
func TestCouvertureScoreTaitLesChampsDeMancheSurUnFilmMonoManche(t *testing.T) {
	_, cov := buildScoreTimeline(
		&ScoreInput{Records: manchesRecordsDeSlotJoueur(0, 1_000, 900, 200)}, nil, testClock(), nil)
	if cov == nil {
		t.Fatal("aucune couverture publiee")
	}

	if cov.Rounds != 1 {
		t.Errorf("rounds = %d, attendu 1", cov.Rounds)
	}
	if cov.RoundsWritten != nil || cov.RoundsContradicted != nil ||
		cov.RoundsContradictedRecords != 0 || cov.RoundsDecreed {
		t.Errorf("champs de manche publies sur un film mono-manche : written %v, contredits %v, "+
			"records %d, decret %v — ils ne diraient rien que `rounds` ne dise deja",
			cov.RoundsWritten, cov.RoundsContradicted, cov.RoundsContradictedRecords, cov.RoundsDecreed)
	}
}

// TestCouvertureScoreCompteLeRepliDeLaMancheZero — LE REPLI EST DECLARE ET COMPTE.
//
// Un film dont AUCUNE manche n'est admise (ici : quelques enregistrements, trop peu pour la
// matiere comme pour une suite coherente) fait decreter la manche 0. L'artefact doit le DIRE
// (`roundsDecreed`) et le compteur de la cuisson doit l'avoir compte — sans quoi
// `coverage.fallbacks[]` tairait la part de repli du document.
func TestCouvertureScoreCompteLeRepliDeLaMancheZero(t *testing.T) {
	fb := fallback.NouveauCompteur()
	_, cov := buildScoreTimeline(
		&ScoreInput{Records: manchesRecordsDeSlotJoueur(3, 1_000, 4, 0)}, nil, testClock(), fb)
	if cov == nil {
		t.Fatal("aucune couverture publiee")
	}

	if !cov.RoundsDecreed {
		t.Error("roundsDecreed faux alors qu'aucune manche n'est admise")
	}
	if n := fb.Compte(fallback.NomReplayMancheZeroDecretee); n != 1 {
		t.Errorf("repli compte %d fois, attendu 1 : un repli qui ne se compte pas est un repli "+
			"anonyme, ce que D14 (a) interdit", n)
	}
}
