package teammates

// squad_rounds_test.go — LE TABLEAU DE L'ESCOUADE LIT-IL VRAIMENT LA TABLE DES MODES À
// MANCHES ?
//
// Sans ce test, débrancher `WithRoundsDecide` ou se tromper de clé ne ferait bouger AUCUN
// test : la colonne « Score » de l'escouade redeviendrait le cumul de points, en
// contradiction avec la vue match et l'Explorateur sur le même match — alors que le
// commentaire du câblage affirme le contraire (constat de revue adversariale du 2026-08-29).

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// oddballSquadRow : le témoin 293a763e — 2 manches à 1, points 181-186. Valeurs toutes
// distinctes : un 2-2 ne prouverait rien.
func oddballSquadRow() []domain.SquadMatchRow {
	p, e := 181, 186
	pr, er, total := 2, 1, 3
	return []domain.SquadMatchRow{{
		MatchID:         "m-oddball",
		GameVariantName: "Arena:Oddball",
		MyTeamScore:     &p, EnemyTeamScore: &e,
		MyRoundsWon: &pr, EnemyRoundsWon: &er, RoundsTotal: &total,
	}}
}

func TestBuildSquadMatchHistory_VarianteDeclareeAfficheLesManches(t *testing.T) {
	hist := buildSquadMatchHistory(oddballSquadRow(), nil, "halo_infinite", nil,
		map[string]bool{"Arena:Oddball": true})
	if len(hist) != 1 {
		t.Fatalf("want 1 row, got %d", len(hist))
	}
	if hist[0].ScoreLabel != "2 - 1" || hist[0].ScoreKind != "rounds" {
		t.Errorf("got %q / %q, want \"2 - 1\" / \"rounds\"", hist[0].ScoreLabel, hist[0].ScoreKind)
	}
}

// Table absente (service non câblé) : on garde les points, jamais un plantage.
func TestBuildSquadMatchHistory_SansTableAfficheLesPoints(t *testing.T) {
	hist := buildSquadMatchHistory(oddballSquadRow(), nil, "halo_infinite", nil, nil)
	if len(hist) != 1 {
		t.Fatalf("want 1 row, got %d", len(hist))
	}
	if hist[0].ScoreLabel != "181 - 186" || hist[0].ScoreKind != "points" {
		t.Errorf("got %q / %q, want \"181 - 186\" / \"points\"", hist[0].ScoreLabel, hist[0].ScoreKind)
	}
}

// Variante non déclarée (le CTF d'arène et ses deux mi-temps) : les points restent.
func TestBuildSquadMatchHistory_VarianteNonDeclareeGardeLesPoints(t *testing.T) {
	rows := oddballSquadRow()
	rows[0].GameVariantName = "CTF:Arena"
	hist := buildSquadMatchHistory(rows, nil, "halo_infinite", nil,
		map[string]bool{"Arena:Oddball": true})
	if len(hist) != 1 {
		t.Fatalf("want 1 row, got %d", len(hist))
	}
	if hist[0].ScoreLabel != "181 - 186" || hist[0].ScoreKind != "points" {
		t.Errorf("got %q / %q, want \"181 - 186\" / \"points\"", hist[0].ScoreLabel, hist[0].ScoreKind)
	}
}
