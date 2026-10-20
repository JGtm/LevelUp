package service

// match_history_rounds_test.go — LA LIGNE D'HISTORIQUE LIT-ELLE VRAIMENT LA TABLE DES
// MODES À MANCHES ?
//
// Sans ce test, débrancher `WithRoundsDecide`, se tromper de clé, ou retirer le `TrimSpace`
// ne ferait bouger AUCUN test : la colonne « Score » de l'historique et de l'Explorateur
// redeviendrait silencieusement le cumul de points, et contredirait l'en-tête de la vue
// match sur le même match (constat de revue adversariale du 2026-08-29).

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// oddballRow : le témoin 293a763e — victoire 2 manches à 1, alors que les points disent
// 181-186. Toutes les valeurs sont distinctes, un 2-2 ne prouverait rien.
func oddballRow() domain.MatchHistoryRawRow {
	variant := "Arena:Oddball"
	p, e := 181, 186
	pr, er, total := 2, 1, 3
	return domain.MatchHistoryRawRow{
		MatchID:         "m-oddball",
		GameVariantName: &variant,
		MyTeamScore:     &p, EnemyTeamScore: &e,
		MyRoundsWon: &pr, EnemyRoundsWon: &er, RoundsTotal: &total,
	}
}

func TestEnrichRow_VarianteDeclareeAfficheLesManches(t *testing.T) {
	fmts := rowFormatters{roundsDecide: map[string]bool{"Arena:Oddball": true}}
	got := enrichRow(oddballRow(), nil, fmts)
	if got.ScoreLabel != "2 - 1" {
		t.Errorf("ScoreLabel = %q, want \"2 - 1\"", got.ScoreLabel)
	}
	if got.ScoreKind != "rounds" {
		t.Errorf("ScoreKind = %q, want \"rounds\"", got.ScoreKind)
	}
}

// La table ABSENTE (service non câblé) doit rendre les points, pas planter : c'est la
// dégradation qui garantit qu'un titre sans déclaration ne régresse pas.
func TestEnrichRow_SansTableAfficheLesPoints(t *testing.T) {
	got := enrichRow(oddballRow(), nil, rowFormatters{})
	if got.ScoreLabel != "181 - 186" || got.ScoreKind != "points" {
		t.Errorf("got %q / %q, want \"181 - 186\" / \"points\"", got.ScoreLabel, got.ScoreKind)
	}
}

// Variante NON déclarée alors que la table existe : c'est le cas du CTF d'arène (deux
// mi-temps, score = captures) — il doit garder ses points.
func TestEnrichRow_VarianteNonDeclareeGardeLesPoints(t *testing.T) {
	row := oddballRow()
	variant := "CTF:Arena"
	row.GameVariantName = &variant
	fmts := rowFormatters{roundsDecide: map[string]bool{"Arena:Oddball": true}}
	got := enrichRow(row, nil, fmts)
	if got.ScoreLabel != "181 - 186" || got.ScoreKind != "points" {
		t.Errorf("got %q / %q, want \"181 - 186\" / \"points\"", got.ScoreLabel, got.ScoreKind)
	}
}

// La clé est comparée TRIMÉE des deux côtés : un `game_variant_name` qui traîne une espace
// en base ne doit pas faire basculer la ligne en points.
func TestEnrichRow_CleTrimee(t *testing.T) {
	row := oddballRow()
	variant := "  Arena:Oddball  "
	row.GameVariantName = &variant
	fmts := rowFormatters{roundsDecide: map[string]bool{"Arena:Oddball": true}}
	if got := enrichRow(row, nil, fmts); got.ScoreKind != "rounds" {
		t.Errorf("ScoreKind = %q, want \"rounds\" (clé trimée)", got.ScoreKind)
	}
}
