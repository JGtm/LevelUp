//go:build integration

// Package duckdb — match_view_repo_erreurs_remontees_test.go : E4.8a. Les lectures de la vue
// match ne transforment plus une défaillance en « pas de donnée » : une requête en échec
// (relation absente d'une base non migrée comprise) REMONTE en erreur, le service la dit.
// L'absence d'une relation que le lecteur ne porte légitimement pas est CONSTATÉE par une
// sonde, jamais déduite d'une erreur quelconque.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

func TestMatchViewRepo_GetMatchEvents_RequeteEnEchec_Remonte(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `DROP VIEW highlight_events`); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	events, err := NewMatchViewRepo(pdb, pTestXUID).GetMatchEvents(ctx, "m1")
	if err == nil {
		t.Fatalf("une requête Q21 en échec doit remonter en erreur, obtenu %d events et aucune erreur", len(events))
	}
}

// Q21b, Q21c et Q20 lisent la même relation canonique : son absence (base non migrée) est une
// panne, pas une liste vide.
func TestMatchViewRepo_LecturesDuKillFeed_RequeteEnEchec_Remonte(t *testing.T) {
	lectures := map[string]func(r *MatchViewRepo, ctx context.Context) error{
		"GetMatchKillSources": func(r *MatchViewRepo, ctx context.Context) error {
			_, err := r.GetMatchKillSources(ctx, "m1")
			return err
		},
		"GetMatchKillAssists": func(r *MatchViewRepo, ctx context.Context) error {
			_, err := r.GetMatchKillAssists(ctx, "m1")
			return err
		},
		"GetMatchKVPairs": func(r *MatchViewRepo, ctx context.Context) error {
			_, err := r.GetMatchKVPairs(ctx, "m1")
			return err
		},
	}
	for nom, lire := range lectures {
		t.Run(nom, func(t *testing.T) {
			pdb := newTestPlayerDB(t)
			ctx := context.Background()
			if _, err := pdb.Player.Exec(ctx, `DROP VIEW match_kill_events_latest`); err != nil {
				t.Fatalf("fixture : %v", err)
			}
			if err := lire(NewMatchViewRepo(pdb, pTestXUID), ctx); err == nil {
				t.Fatalf("%s : relation absente = panne, une erreur est attendue", nom)
			}
		})
	}
}

// Les mêmes lectures sur la relation PRÉSENTE (schéma de production) et vide rendent « pas de
// donnée » sans erreur : c'est l'état normal d'un match qui n'est pas passé par le décodeur de film.
func TestMatchViewRepo_LecturesDuKillFeed_RelationVide_PasUneErreur(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `DROP VIEW match_kill_events_latest`); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	if err := migration.EnsureMatchKillEvents(pdb.Player.SQLDb()); err != nil {
		t.Fatalf("EnsureMatchKillEvents : %v", err)
	}
	repo := NewMatchViewRepo(pdb, pTestXUID)
	if _, err := repo.GetMatchKillSources(ctx, "m1"); err != nil {
		t.Errorf("GetMatchKillSources : %v", err)
	}
	if _, err := repo.GetMatchKillAssists(ctx, "m1"); err != nil {
		t.Errorf("GetMatchKillAssists : %v", err)
	}
	if _, err := repo.GetMatchKVPairs(ctx, "m1"); err != nil {
		t.Errorf("GetMatchKVPairs : %v", err)
	}
}

func TestMatchViewRepo_GetMatchObjectiveScore_AbsenceConstatee_Vs_Panne(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	repo := NewMatchViewRepo(pdb, pTestXUID)

	// Table absente (titre ou joueur sans awards) : absence constatée, 0, pas d'erreur.
	n, err := repo.GetMatchObjectiveScore(ctx, pTestXUID, "m1")
	if err != nil || n != 0 {
		t.Fatalf("table absente : attendu (0, nil), obtenu (%d, %v)", n, err)
	}

	// Table présente mais lecture en échec (vue `_latest` manquante) : PANNE, elle remonte.
	if _, err := pdb.Player.Exec(ctx, `CREATE TABLE personal_score_awards (x INTEGER)`); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	if _, err := repo.GetMatchObjectiveScore(ctx, pTestXUID, "m1"); err == nil {
		t.Fatalf("table présente et lecture en échec : une erreur est attendue")
	}
}

// Stats d'Assaut : la vue ABSENTE du lecteur est une absence constatée (le snapshot ne
// l'exporte pas) ; une requête qui échoue sur une vue PRÉSENTE est une panne, DITE par une
// SectionDegradedError, et le tableau des scores reste servi.
func TestGetMatchScoreboard_BombStats_RequeteEnEchec_SectionDegradee(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	// Vue présente mais de schéma incompatible avec Q12c : la requête échoue.
	if _, err := pdb.Player.Exec(ctx, `CREATE OR REPLACE VIEW match_bomb_stats_latest AS SELECT 'm1' AS match_id`); err != nil {
		t.Fatalf("fixture : %v", err)
	}
	buf := captureSlog(t)
	rows, err := NewMatchViewRepo(pdb, pTestXUID).WithBombStats(true).GetMatchScoreboard(ctx, "m1")
	if !domain.IsSectionDegraded(err) {
		t.Fatalf("panne de la section d'Assaut : *domain.SectionDegradedError attendue, obtenu %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("le tableau des scores doit rester servi, obtenu %d lignes", len(rows))
	}
	if !hasLogLevelContaining(t, buf, "ERROR", "Assaut") {
		t.Errorf("aucune ERREUR journalisée avant la dégradation ; logs=%s", buf.String())
	}
}
