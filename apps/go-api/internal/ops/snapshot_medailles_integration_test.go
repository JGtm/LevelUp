//go:build integration

package ops

// snapshot_medailles_integration_test.go — les médailles d'un match servi du snapshot sont celles du
// live : la vue `v_medals_earned` existe dans le schéma reconstruit, et elle fusionne le Vengeur
// dérivé du film (qui remplace la ligne héritée) et la Cible difficile.

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	platformduckdb "levelup/go-api/internal/platform/duckdb"
)

const (
	medailleVengeurSnapshot = int64(9000000001)
	medailleCibleSnapshot   = int64(9000000002)
	medailleAutreSnapshot   = int64(55)
)

// medailleLue est une ligne de Q14 : identifiant de médaille et compte.
type medailleLue struct{ ID, Count int64 }

// lireMedaillesQ14 exécute la lecture de la vue du match (Q14) sur `db`.
func lireMedaillesQ14(t *testing.T, db *sql.DB, xuid, matchID string) []medailleLue {
	t.Helper()
	rows, err := db.Query(platformduckdb.Q14MatchMedals, xuid, matchID)
	if err != nil {
		t.Fatalf("Q14 %s/%s: %v", xuid, matchID, err)
	}
	defer rows.Close() //nolint:errcheck
	var out []medailleLue
	for rows.Next() {
		var m medailleLue
		if err := rows.Scan(&m.ID, &m.Count); err != nil {
			t.Fatalf("scan Q14: %v", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Q14 rows: %v", err)
	}
	return out
}

// seedMedaillesDuFilm écrit sur m1 un Vengeur hérité (7) et une autre médaille, puis la passe du
// résumé d'usage : deux vengeances dérivées et trois recharges de bouclier pour xKiller.
func seedMedaillesDuFilm(t *testing.T, shared *sql.DB) {
	t.Helper()
	snapExecT(t, shared, `INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES
		('m1', 'xKiller', ?, 7), ('m1', 'xKiller', ?, 1)`, medailleVengeurSnapshot, medailleAutreSnapshot)
	snapExecT(t, shared, `INSERT INTO match_usage_films
		(match_id, summary_pass, summary_rev, artifact_schema, frame_interval_ms, frame_count, duration_ms,
		 pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)
		VALUES ('m1', 'p1', 'us7', 70, 100, 10, 1000, 0, 0, 0, '[]', '{}')`)
	snapExecT(t, shared, `INSERT INTO match_usage_players
		(match_id, summary_pass, summary_rev, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills,
		 overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown,
		 pad_pickups, pad_pickups_json, revenge_kills, shield_recharges)
		VALUES ('m1', 'p1', 'us7', 'xKiller', 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}', 2, 3)`)
}

// TestSnapshot_ServeLesMedaillesDuFilm : un snapshot produit depuis une base portant un Vengeur
// hérité (7) et la dérivation du film (2 vengeances, 3 recharges) sert, par la lecture de la vue du
// match, le Vengeur dérivé et la Cible difficile, comme le live.
func TestSnapshot_ServeLesMedaillesDuFilm(t *testing.T) {
	ctx := context.Background()
	paths := title.NewPathResolver(t.TempDir(), nil)
	shared := seedFidelityShared(t)
	seedMedaillesDuFilm(t, shared)
	if err := migration.ApplyMedalsEarnedView(shared); err != nil { // la vue du live, témoin de la lecture
		t.Fatalf("ApplyMedalsEarnedView (live): %v", err)
	}
	player := seedPlayerDB(t, []string{"m1"}, nil)

	res, err := ProduceSnapshot(ctx, SnapshotOptions{
		TitleSlug:    title.DefaultSlug,
		Paths:        paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": player}},
		Players:      []string{"Solo"},
		Now:          time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}
	q, err := OpenSnapshotShared(ctx, paths, title.DefaultSlug)
	if err != nil {
		t.Fatalf("OpenSnapshotShared: %v", err)
	}
	defer q.Close()

	attendu := []medailleLue{{medailleCibleSnapshot, 3}, {medailleVengeurSnapshot, 2}, {medailleAutreSnapshot, 1}}
	if got := lireMedaillesQ14(t, q.DB, "xKiller", "m1"); !reflect.DeepEqual(got, attendu) {
		t.Errorf("Q14 sur le snapshot = %v, attendu %v (Vengeur dérivé 2 à la place de l'hérité 7, Cible difficile 3)", got, attendu)
	}
	if live := lireMedaillesQ14(t, shared, "xKiller", "m1"); !reflect.DeepEqual(live, attendu) {
		t.Errorf("Q14 sur le live = %v, attendu %v : le snapshot ne serait pas comparé à la même vérité", live, attendu)
	}
}
