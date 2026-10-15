//go:build integration

package persist

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
)

// La fusion credit/film du flux primaire lit les preuves de l API (evenements death) DANS la
// transaction du sync : les evenements du batch doivent y etre ecrits AVANT les couples, sans quoi
// le match parait sans evenement, les preuves restent inconnues et une ligne de film a victime
// humaine que l API ne porte pas est servie.
//
// Mutation qui doit le faire rougir : ecrire `persistKillerVictim` avant `persistHighlightEvents`
// dans [SharedPersister.Persist].
func TestSharedPersister_LesPreuvesLisentLesEvenementsDuSync(t *testing.T) {
	ctx := context.Background()
	db := openSharedTestDB(t)
	const matchID = "m_preuves_001"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, written_at, publishable, time_ms,
			victim_gamertag, victim_xuid, feed_present, assist_known, read_path, read_origin
		) VALUES (?, 'film-pass', 'film-rev', TIMESTAMP '2026-01-01 00:00:00', TRUE, 900,
			'Humain', '7777', TRUE, TRUE, ?, 'tueur-bot')`, matchID, killscope.ReadPathFilmWalk); err != nil {
		t.Fatalf("seed passe de film: %v", err)
	}
	b := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	b.SetMatch(&domain.MatchRegistryRow{MatchID: matchID, StartTime: time.Now().UTC(), ModeCategory: "PVP",
		FirstSyncBy: "Alice"})
	b.AddKillerVictim([]KillerVictimInsert{{MatchID: matchID, KillerXUID: "1111", KillerGamertag: "Alice",
		VictimXUID: "2222", VictimGamertag: "Bob", Count: 1, TimeMS: 5000}})
	k, d := "1111", "2222"
	b.AddHighlightEvents([]HighlightEventInsert{
		{MatchID: matchID, XUID: &k, EventType: "kill", TimeMS: 5000},
		{MatchID: matchID, XUID: &d, EventType: "death", TimeMS: 5001},
	})
	if err := NewSharedPersister(db).Persist(ctx, b.Build()); err != nil {
		t.Fatalf("Persist: %v", err)
	}
	if n := countRows(t, db, "match_kill_events_latest", "match_id = ? AND victim_xuid = '2222'", matchID); n != 1 {
		t.Fatalf("la mort de credit de Bob est servie %d fois, attendu 1 : la fusion n a pas eu lieu", n)
	}
	if n := countRows(t, db, "match_kill_events_latest", "match_id = ? AND victim_xuid = '7777'", matchID); n != 0 {
		t.Errorf("la mort de 7777 (aucun evenement death de l API) est servie %d fois : les preuves "+
			"ont ete lues avant les evenements du sync", n)
	}
}
