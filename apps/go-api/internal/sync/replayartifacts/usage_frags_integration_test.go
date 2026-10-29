//go:build integration

package replayartifacts

// usage_frags_integration_test.go — LES FRAGS SANS BOUCLIER DU RESUME D'USAGE, de bout en bout :
// fil des morts en base -> lecture avant tout segment d'ecriture -> projection -> colonne
// `shieldless_kills` servie par la vue `_latest`. NULL (non mesure) quand le match n'a aucune
// ligne au fil des morts.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/killscope"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

func artefactBouclier(t *testing.T, dir, matchID string) string {
	t.Helper()
	origin := int64(0)
	zero := float32(0)
	pts := make([]replay.Point, 0, 100)
	for i := 0; i < 100; i++ {
		p := replay.Point{T: i}
		if i == 10 {
			p.Sh = &zero
		}
		pts = append(pts, p)
	}
	doc := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: matchID,
		FrameIntervalMS: 100, FrameCount: 100, DurationMS: 10000, OriginMs: &origin,
		Tracks: []replay.Track{{Slot: 1, XUID: "111", Points: pts}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal artefact: %v", err)
	}
	path := filepath.Join(dir, matchID+".json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write artefact: %v", err)
	}
	return path
}

func insererFrag(t *testing.T, db *sql.DB, matchID, killer, victime string, timeMS int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO match_kill_events (
			match_id, decode_pass, decoder_rev, publishable, time_ms,
			victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
			feed_present, assist_known, read_path, read_origin
		) VALUES (?, ?, 'test-rev', TRUE, ?, 'Victime', ?, 'Tueur', ?, TRUE, FALSE, ?, 'credit-concordant')`,
		matchID, "pass-"+matchID, timeMS, victime, killer, killscope.ReadPathFilmWalk); err != nil {
		t.Fatalf("insert frag %s@%d: %v", matchID, timeMS, err)
	}
}

func TestResumeUsage_FragsSansBouclier(t *testing.T) {
	db := baseRegistre(t)
	dir := t.TempDir()
	acquis, relaches := 0, 0
	d := depsUsage(t, db, "halo_infinite", &acquis, &relaches)
	insererFrag(t, db, "m-bouclier", "111", "222", 2000)

	lot := lus(d,
		ArtefactRange{MatchID: "m-bouclier", Path: artefactBouclier(t, dir, "m-bouclier")},
		ArtefactRange{MatchID: "m-sans-fil", Path: artefactBouclier(t, dir, "m-sans-fil")},
	)
	preparerFragsDuResume(context.Background(), d, lot)
	persisterResumesUsage(context.Background(), d, &bilanDerivations{}, lot)

	var n sql.NullInt64
	if err := db.QueryRow(`SELECT shieldless_kills FROM match_usage_players_latest
		WHERE match_id = 'm-bouclier' AND xuid = '111'`).Scan(&n); err != nil {
		t.Fatalf("lecture m-bouclier: %v", err)
	}
	if !n.Valid || n.Int64 != 1 {
		t.Fatalf("m-bouclier : shieldless_kills = %+v, attendu 1", n)
	}
	var lignes int
	if err := db.QueryRow(`SELECT count(*) FROM match_usage_players_latest
		WHERE match_id = 'm-sans-fil' AND shieldless_kills IS NOT NULL`).Scan(&lignes); err != nil {
		t.Fatalf("lecture m-sans-fil: %v", err)
	}
	if lignes != 0 {
		t.Fatalf("m-sans-fil : %d lignes mesurees, attendu aucune (fil des morts absent = non mesure)", lignes)
	}
}
