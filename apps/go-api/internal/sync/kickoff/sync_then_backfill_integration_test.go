//go:build integration

package kickoff_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/kickoff"
)

// Un match inséré par le chemin de la synchronisation (extraction, résolution sur les lignes,
// insertion) porte déjà ce que le rattrapage décide : le rattrapage qui suit n'écrit rien.
func TestInsertionDeLaSynchronisationPuisRattrapage_ZeroEcriture(t *testing.T) {
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	start := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) string { return start.Add(d).Format(time.RFC3339Nano) }
	payload := map[string]any{
		"MatchId":   "m-sync",
		"MatchInfo": map[string]any{"StartTime": start.Format(time.RFC3339), "Duration": "PT10M"},
		"Players": []any{
			map[string]any{"PlayerId": "xuid(2603225103667623)", "ParticipationInfo": map[string]any{
				"PresentAtBeginning": true, "FirstJoinedTime": at(20 * time.Second), "TimePlayed": "PT10M"}},
			map[string]any{"PlayerId": "xuid(2547408409520575)", "ParticipationInfo": map[string]any{
				"PresentAtBeginning": true, "FirstJoinedTime": at(20500 * time.Millisecond), "TimePlayed": "PT10M"}},
			map[string]any{"PlayerId": "xuid(2664271391936720)", "ParticipationInfo": map[string]any{
				"PresentAtBeginning": false, "FirstJoinedTime": at(200 * time.Second), "TimePlayed": "PT6M40S"}},
		},
	}
	reg, err := syncpkg.ExtractRegistry(payload, "testeur")
	if err != nil {
		t.Fatalf("ExtractRegistry: %v", err)
	}
	parts := syncpkg.ExtractParticipants(payload)
	kickoff.ApplyToRows(reg, parts)
	batch := persist.NewBatchBuilder("halo_infinite", "testeur", "2603225103667623", "test").
		SetMatch(reg).AddParticipants(parts).Build()
	if err := persist.NewSharedPersister(db).Persist(context.Background(), batch); err != nil {
		t.Fatalf("insertion: %v", err)
	}

	tenu := func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
	rep, err := kickoff.Backfill(context.Background(), kickoff.Access{Read: tenu, Write: tenu},
		kickoff.Options{TitleSlug: "halo_infinite"})
	if err != nil || rep.Matches != 1 || rep.Written != 0 || rep.PlayersChanged != 0 {
		t.Fatalf("rattrapage apres insertion : err=%v bilan=%+v, attendu un match et zero ecriture", err, rep)
	}
}
