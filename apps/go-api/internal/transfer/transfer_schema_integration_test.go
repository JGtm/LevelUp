//go:build cgo && integration

package transfer

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/knownset"
)

// Le vrai schéma : les bases du test sont montées par les migrations du titre, les matchs hébergés sont écrits
// par persist.SharedPersister (le chemin de la synchronisation), pas par des tables écrites à la main.

const (
	mineMatch  = "match-mine"
	otherMatch = "match-other"
	friendXuid = "2533274800000009"
)

func openMigrated(t *testing.T, path string, target migration.TargetDB) *sql.DB {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForTitleDB(db, slug, target); err != nil {
		t.Fatalf("migrations %s : %v", path, err)
	}
	return db
}

func matchBatch(matchID string, xuids ...string) *persist.MatchBatch {
	i := func(v int) *int { return &v }
	s := func(v string) *string { return &v }
	b := persist.NewBatchBuilder(slug, "SpartanB", meXuid, "test")
	b.SetMatch(&domain.MatchRegistryRow{MatchID: matchID, StartTime: time.Now().UTC(), ModeCategory: "PVP",
		MapName: s("Aquarius"), DurationSeconds: i(600), FirstSyncBy: "SpartanB"})
	var parts []domain.MatchParticipantRow
	var medals []domain.MedalRow
	var acc []persist.WeaponAccuracyInsert
	var events []persist.HighlightEventInsert
	var csrs []persist.MatchCSRInsert
	for k, x := range xuids {
		parts = append(parts, domain.MatchParticipantRow{MatchID: matchID, XUID: x, Gamertag: s("P" + x[len(x)-2:]),
			TeamID: i(k % 2), Kills: i(10 + k), Deaths: i(5), Assists: i(2)})
		medals = append(medals, domain.MedalRow{MatchID: matchID, XUID: x, MedalNameID: int64(100 + k), Count: 1})
		acc = append(acc, persist.WeaponAccuracyInsert{MatchID: matchID, XUID: x, WeaponID: uint64(7000 + k), ShotsFired: 40, ShotsLanded: 20, Drops: 1})
		events = append(events, persist.HighlightEventInsert{MatchID: matchID, XUID: s(x), EventType: "Kill", TimeMS: 1000 * (k + 1)})
		v := 1500.0 + float64(k)
		csrs = append(csrs, persist.MatchCSRInsert{MatchID: matchID, XUID: x, RatingType: "CSR", RatingValue: &v})
	}
	b.AddParticipants(parts).AddMedals(medals).AddWeaponAccuracy(acc).AddHighlightEvents(events).AddMatchCSRs(csrs)
	b.AddKillerVictim([]persist.KillerVictimInsert{
		{MatchID: matchID, KillerXUID: xuids[0], KillerGamertag: "A", VictimXUID: xuids[1], VictimGamertag: "B", Count: 1, TimeMS: 1000},
		{MatchID: matchID, KillerXUID: xuids[1], KillerGamertag: "B", VictimXUID: xuids[0], VictimGamertag: "A", Count: 1, TimeMS: 2000},
	})
	return b.Build()
}

func scalar(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s : %v", query, err)
	}
	return n
}

// TestExportImportRealSchema_MatchComplete : un match exporté arrive ENTIER (tous ses participants, ses armes, ses
// événements et son journal des morts), la synchronisation le tient pour connu et n'a rien à redemander, rejouer
// l'import n'écrit rien, et ni le match d'un autre joueur ni la base joueur d'un autre profil ne voyagent.
func TestExportImportRealSchema_MatchComplete(t *testing.T) {
	ctx := context.Background()
	hosted := roots(t)
	hostedShared := openMigrated(t, hosted.SharedDBPath(slug), migration.TargetShared)
	sp := persist.NewSharedPersister(hostedShared)
	for _, b := range []*persist.MatchBatch{
		matchBatch(mineMatch, meXuid, herXuid, friendXuid),
		matchBatch(otherMatch, herXuid, friendXuid),
	} {
		if err := sp.Persist(ctx, b); err != nil {
			t.Fatalf("Persist hébergé : %v", err)
		}
	}
	mePlayer := openMigrated(t, hosted.PlayerDBPath(slug, meTag), migration.TargetPlayer)
	if _, err := mePlayer.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES (?, 71.5)`, mineMatch); err != nil {
		t.Fatal(err)
	}
	herPlayer := openMigrated(t, hosted.PlayerDBPath(slug, "Her"), migration.TargetPlayer)
	if _, err := herPlayer.Exec(`INSERT INTO player_match_enrichment (match_id, performance_score) VALUES (?, 55.0)`, otherMatch); err != nil {
		t.Fatal(err)
	}
	_ = hostedShared.Close()
	_ = mePlayer.Close()
	_ = herPlayer.Close()

	zipPath, m := exportFor(t, hosted, meXuid, meTag)
	tables := map[string]int64{}
	for _, tb := range m.Tables {
		tables[tb.Name] = tb.Rows
	}
	for name, want := range map[string]int64{"match_registry": 1, "match_participants": 3, "medals_earned": 3,
		"weapon_accuracy": 3, "highlight_events": 3, "match_csrs": 3, "player_match_enrichment": 1} {
		if tables[name] != want {
			t.Errorf("%s : %d ligne(s) exportée(s), attendu %d", name, tables[name], want)
		}
	}
	if tables["match_kill_events"] == 0 || tables["match_kill_heads"] == 0 || tables["killer_victim_pairs"] != 2 {
		t.Errorf("journal des morts incomplet dans l'archive : %v", tables)
	}

	local := roots(t)
	localShared := openMigrated(t, local.SharedDBPath(slug), migration.TargetShared)
	localPlayer := openMigrated(t, local.PlayerDBPath(slug, meTag), migration.TargetPlayer)
	in := ImportInput{ZipPath: zipPath, ExpectXuid: meXuid,
		AcquireShared: func(context.Context) (*sql.DB, func(), error) { return localShared, func() {}, nil },
		AcquirePlayer: func(context.Context) (*sql.DB, func(), error) { return localPlayer, func() {}, nil }}
	if _, err := Import(ctx, in); err != nil {
		t.Fatalf("Import : %v", err)
	}

	for _, c := range []struct {
		query string
		want  int
	}{
		{`SELECT count(*) FROM match_participants WHERE match_id = 'match-mine'`, 3},
		{`SELECT count(*) FROM weapon_accuracy WHERE match_id = 'match-mine'`, 3},
		{`SELECT count(*) FROM highlight_events WHERE match_id = 'match-mine'`, 3},
		{`SELECT count(*) FROM medals_earned WHERE match_id = 'match-mine'`, 3},
		{`SELECT count(*) FROM killer_victim_pairs WHERE match_id = 'match-mine'`, 2},
		{`SELECT count(*) FROM match_kill_events_latest WHERE match_id = 'match-mine'`, 2},
		{`SELECT count(*) FROM match_kill_heads_latest WHERE match_id = 'match-mine'`, 1},
		{`SELECT count(*) FROM match_csrs_latest WHERE match_id = 'match-mine'`, 3},
		{`SELECT count(*) FROM match_registry WHERE events_loaded AND match_id = 'match-mine'`, 1},
		{`SELECT count(*) FROM match_registry WHERE match_id = 'match-other'`, 0},
		{`SELECT count(*) FROM match_participants WHERE match_id = 'match-other'`, 0},
	} {
		if got := scalar(t, localShared, c.query); got != c.want {
			t.Errorf("%s : %d, attendu %d", c.query, got, c.want)
		}
	}
	if got := scalar(t, localPlayer, `SELECT count(*) FROM player_match_enrichment WHERE match_id = 'match-other'`); got != 0 {
		t.Errorf("l'enrichissement d'un autre profil suivi ne doit pas voyager : %d ligne(s)", got)
	}

	// La synchronisation ne redemande rien : le match est connu, aucun orphelin à récupérer.
	set, err := knownset.Load(ctx, localPlayer, localShared, meXuid)
	if err != nil {
		t.Fatalf("knownset.Load : %v", err)
	}
	if !set.Known[mineMatch] || len(set.Recover) != 0 {
		t.Errorf("ensemble connu après import : %v, à récupérer : %v", set.Known, set.Recover)
	}

	again, err := Import(ctx, in)
	if err != nil {
		t.Fatalf("second Import : %v", err)
	}
	for k, v := range again.Inserted {
		if v != 0 {
			t.Errorf("%s : %d ligne(s) au second import, attendu 0", k, v)
		}
	}
}
