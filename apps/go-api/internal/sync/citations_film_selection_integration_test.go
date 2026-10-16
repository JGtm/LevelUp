//go:build integration

package sync

// citations_film_selection_integration_test.go — LE RECALCUL DES CITATIONS D'UN MATCH QUAND SON
// FILM EST DÉRIVÉ APRÈS ELLES : un match calculé sans résumé d'usage, puis dérivé, revient dans la
// sélection du post-sync ; le recalcul complet pose la citation du film et laisse les autres
// intactes ; la sélection se tait ensuite.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/sync/citationsfilm"
)

// ajouterResumeUsage crée les deux vues du résumé d'usage (forme réduite à ce que les citations
// lisent) et y écrit la dérivation de fixM1, datée après la génération de citations courante.
func ajouterResumeUsage(t *testing.T, f *pipelineFixture, derive time.Time) {
	t.Helper()
	mustExec(t, f.shared, `CREATE TABLE usage_films (match_id VARCHAR, written_at TIMESTAMP)`)
	mustExec(t, f.shared, `CREATE TABLE usage_players (match_id VARCHAR, xuid VARCHAR, pad_pickups INTEGER, shieldless_kills INTEGER)`)
	mustExec(t, f.shared, `CREATE VIEW match_usage_films_latest AS SELECT * FROM usage_films`)
	mustExec(t, f.shared, `CREATE VIEW match_usage_players_latest AS SELECT * FROM usage_players`)
	mustExec(t, f.shared, `INSERT INTO usage_films VALUES (?, ?)`, fixM1, derive)
	mustExec(t, f.shared, `INSERT INTO usage_players VALUES (?, ?, 1, 2)`, fixM1, fixXUID)
}

func TestCitationsFilm_RecalculApresDerivation(t *testing.T) {
	f := buildPipelineFixture(t)
	ctx := context.Background()
	mustExec(t, f.metadata, `
		INSERT INTO citation_mappings
			(citation_name_norm, citation_name_display, mapping_type, stat_name, tier_targets, enabled)
		VALUES ('almost_doesnt_count', 'Presque, ça ne compte pas', 'stat', 'shieldless_kills', '10,20,30,50,100', TRUE)`)

	runCitationsFor(t, f, fixM1)
	avant := realCitationValue(t, f.player, fixM1, "bulltrue")
	if avant == 0 || realCitationValue(t, f.player, fixM1, "almost_doesnt_count") != 0 {
		t.Fatalf("premier calcul : bulltrue=%d (attendu > 0), presque=%d (attendu 0, film absent)",
			avant, realCitationValue(t, f.player, fixM1, "almost_doesnt_count"))
	}

	var calcule time.Time
	if err := f.player.QueryRow(`SELECT MAX(written_at) FROM match_citations_latest WHERE match_id = ?`, fixM1).
		Scan(&calcule); err != nil {
		t.Fatal(err)
	}
	ajouterResumeUsage(t, f, calcule.Add(time.Millisecond))
	// Le recalcul se produit APRÈS la dérivation, comme en production.
	time.Sleep(20 * time.Millisecond)

	ids, err := citationsfilm.SelectMatchesFilmPlusRecent(ctx, f.player, f.shared)
	if err != nil || len(ids) != 1 || ids[0] != fixM1 {
		t.Fatalf("film dérivé après les citations : sélection %v (err %v), attendu [%s]", ids, err, fixM1)
	}
	if err := BackfillMatchCitations(ctx, f.metadata, f.shared, f.player, nil, fixXUID, ids); err != nil {
		t.Fatal(err)
	}
	if got := realCitationValue(t, f.player, fixM1, "almost_doesnt_count"); got != 2 {
		t.Fatalf("après recalcul : presque=%d, attendu 2", got)
	}
	if got := realCitationValue(t, f.player, fixM1, "bulltrue"); got != avant {
		t.Fatalf("après recalcul : bulltrue=%d, attendu %d (citations hors film inchangées)", got, avant)
	}
	if ids, err := citationsfilm.SelectMatchesFilmPlusRecent(ctx, f.player, f.shared); err != nil || len(ids) != 0 {
		t.Fatalf("citations à jour : sélection %v (err %v), attendu vide", ids, err)
	}
}

func TestCitationsFilm_TitreSansFilm(t *testing.T) {
	f := buildPipelineFixture(t)
	runCitationsFor(t, f, fixM1)
	if ids, err := citationsfilm.SelectMatchesFilmPlusRecent(context.Background(), f.player, f.shared); err != nil || len(ids) != 0 {
		t.Fatalf("sans résumé d'usage : sélection %v (err %v), attendu vide et sans erreur", ids, err)
	}
}
