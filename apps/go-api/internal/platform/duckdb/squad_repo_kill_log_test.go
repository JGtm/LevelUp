// Package duckdb — squad_repo_kill_log_test.go : Q32e contre une vraie base DuckDB en
// mémoire, au schéma de production (lecture par la vue `_latest`).
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/killscope"
)

func TestQ32eSquadKillLog_Filtre(t *testing.T) {
	db := newAssistPairsDB(t, nil)
	const ins = `INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms, victim_gamertag, victim_xuid,
		 feed_killer_xuid, feed_present, assist_known, assist_xuid, killer_damage_pct,
		 read_path, read_origin)
		VALUES (?, 'pass-1', 'rev-1', ?, ?, 'v', ?, ?, TRUE, ?, ?, ?, ?, ?)`
	insert := func(match string, publishable bool, tms int, victim, killer, assist *string, known bool, pct *int) {
		t.Helper()
		if _, err := db.Exec(ins, match, publishable, tms, victim, killer, known, assist, pct,
			killscope.ReadPathFilmWalk, filmCreditOrigin); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	insert("m1", true, 1000, strPtr("E1"), strPtr("S1"), strPtr("S2"), true, intPtr(5))  // vol candidat
	insert("m1", true, 2000, strPtr("S2"), strPtr("E1"), nil, true, intPtr(100))         // mort d'un membre
	insert("m1", true, 3000, strPtr("E2"), strPtr("S1"), strPtr("X9"), true, intPtr(5))  // assistant hors escouade
	insert("m1", true, 4000, strPtr("E3"), strPtr("S1"), nil, false, nil)                // assistance inconnue
	insert("m2", false, 1000, strPtr("E1"), strPtr("S1"), strPtr("S2"), true, intPtr(5)) // non publiable
	// Mort de BOT : victime NULL, tueur ET assistant membres de l'escouade. La branche
	// « candidat au vol » ne parle que du tueur et de l'assistant : sans le filtre
	// `victim_xuid IS NOT NULL`, cette ligne sort et le badge Voleur compte un vol sur
	// une victime que personne ne peut nommer.
	insert("m1", true, 5000, nil, strPtr("S1"), strPtr("S2"), true, intPtr(5))

	query, args := buildSquadKillLogQuery([]string{"m1", "m2"}, []string{"S1", "S2"})
	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		t.Fatalf("Q32e: %v", err)
	}
	defer rows.Close()
	got, err := scanSquadKillLog(rows)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("2 lignes attendues, got %d : %+v", len(got), got)
	}
	if got[0].TimeMS != 1000 || got[0].AssistXUID != "S2" || got[0].KillerDamagePct == nil || *got[0].KillerDamagePct != 5 {
		t.Errorf("ligne vol inattendue : %+v", got[0])
	}
	if got[1].VictimXUID != "S2" || got[1].AssistXUID != "" {
		t.Errorf("ligne mort de membre inattendue : %+v", got[1])
	}
	for _, r := range got {
		if r.VictimXUID == "" {
			t.Errorf("mort de bot (victime NULL) retournée par Q32e : %+v", r)
		}
	}
}
