//go:build integration

package sync

// citations_vengeur_vue_integration_test.go — LA CITATION « VENGEUR » SUIT LA DÉRIVATION DU FILM ET
// LIT LA VUE. Un match dont les citations sont calculées avant son film porte la valeur de la table
// brute (ici un Vengeur hérité gonflé) ; une fois le film dérivé, la sélection du post-sync le
// reprend, et le recalcul lit `v_medals_earned` : la valeur dérivée remplace l'héritée.
// Mutation : lire `medals_earned` dans loadMedalsForMatch rend ce test rouge (la valeur reste celle
// de la table brute).

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync/citationsfilm"
)

const medailleVengeurTest = 9000000001

// poserResumeAvecVengeur pose le résumé d'usage au schéma livré, écrit la dérivation de fixM1 datée
// `derive` (une passe, `vengeances` vengeances, aucune recharge) et pose la vraie vue de fusion des
// médailles.
func poserResumeAvecVengeur(t *testing.T, f *pipelineFixture, derive time.Time, vengeances int) {
	t.Helper()
	if err := migration.EnsureMatchUsageSummary(f.shared); err != nil {
		t.Fatalf("EnsureMatchUsageSummary: %v", err)
	}
	mustExec(t, f.shared, `INSERT INTO match_usage_films
		(match_id, summary_pass, summary_rev, artifact_schema, written_at, frame_interval_ms, frame_count, duration_ms,
		 pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)
		VALUES (?, 'p1', 'us7', 70, ?, 100, 10, 1000, 0, 0, 0, '[]', '{}')`, fixM1, derive)
	mustExec(t, f.shared, `INSERT INTO match_usage_players
		(match_id, summary_pass, summary_rev, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills,
		 overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown,
		 pad_pickups, pad_pickups_json, revenge_kills, shield_recharges)
		VALUES (?, 'p1', 'us7', ?, 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}', ?, 0)`, fixM1, fixXUID, vengeances)
	if err := migration.ApplyMedalsEarnedView(f.shared); err != nil {
		t.Fatalf("ApplyMedalsEarnedView: %v", err)
	}
}

func TestCitationVengeur_RecalculeeApresLaDerivationEtLitLaVue(t *testing.T) {
	f := buildPipelineFixture(t)
	ctx := context.Background()
	mustExec(t, f.metadata, `
		INSERT INTO citation_mappings
			(citation_name_norm, citation_name_display, mapping_type, medal_id, tier_targets, enabled)
		VALUES ('avenger', 'Vengeur', 'medal', ?, '5,15,30,55,105', TRUE)`, uint64(medailleVengeurTest))
	// Vengeur hérité (table brute) : 7 sur le match, avant toute dérivation.
	mustExec(t, f.shared, `INSERT INTO medals_earned VALUES (?, ?, ?, 7)`, fixM1, fixXUID, int64(medailleVengeurTest))

	runCitationsFor(t, f, fixM1)
	if got := realCitationValue(t, f.player, fixM1, "avenger"); got != 7 {
		t.Fatalf("avant la dérivation : avenger=%d, attendu 7 (valeur héritée de la table)", got)
	}

	var calcule time.Time
	if err := f.player.QueryRow(`SELECT MAX(written_at) FROM match_citations_latest WHERE match_id = ?`, fixM1).
		Scan(&calcule); err != nil {
		t.Fatal(err)
	}
	poserResumeAvecVengeur(t, f, calcule.Add(time.Millisecond), 2)
	time.Sleep(20 * time.Millisecond)

	ids, err := citationsfilm.SelectMatchesFilmPlusRecent(ctx, f.player, f.shared)
	if err != nil || len(ids) != 1 || ids[0] != fixM1 {
		t.Fatalf("film dérivé après les citations : sélection %v (err %v), attendu [%s]", ids, err, fixM1)
	}
	if err := BackfillMatchCitations(ctx, f.metadata, f.shared, f.player, nil, fixXUID, ids); err != nil {
		t.Fatal(err)
	}
	if got := realCitationValue(t, f.player, fixM1, "avenger"); got != 2 {
		t.Fatalf("après la dérivation : avenger=%d, attendu 2 (valeur dérivée lue par la vue, l'héritée 7 est remplacée)", got)
	}
	if ids, err := citationsfilm.SelectMatchesFilmPlusRecent(ctx, f.player, f.shared); err != nil || len(ids) != 0 {
		t.Fatalf("citations à jour : sélection %v (err %v), attendu vide", ids, err)
	}
}
