//go:build integration

// post_sync_stores_test.go : PostSyncProgressionStore et PlayerSnapshotStore contre de vraies bases migrées
// (stats du joueur, base partagée du titre) : mêmes lignes, mêmes agrégats qu'avant leur sortie de api/wire.
package duckdb

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/progression/records"
)

const postSyncTestXUID = "xuid-post-sync-store"

// newPostSyncFixture : un joueur de halo_infinite dont la base partagée porte trois matchs — deux d'arène
// (victoire puis défaite) et un troisième sans instant de début (écarté des lectures de matchs).
func newPostSyncFixture(t *testing.T) (*PlayerDB, time.Time, time.Time) {
	t.Helper()
	shared := setupPrestigeDB(t, migration.TargetShared)
	player := setupPrestigeDB(t, migration.TargetPlayer)
	pdb := &PlayerDB{
		Player: player, Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: postSyncTestXUID, Gamertag: "PostSync", TitleSlug: "halo_infinite",
	}
	ctx := context.Background()
	d1 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	for _, m := range []struct {
		id      string
		at      *time.Time
		outcome int
		kills   int
		deaths  int
		kda     float64
		acc     float64
		secs    int
		dmg     int
	}{
		{"m1", &d1, 2, 10, 5, 1.5, 60, 600, 2000},
		{"m2", &d2, 3, 4, 8, -0.5, 40, 300, 800},
		{"m3", nil, 2, 99, 1, 9.0, 90, 600, 9000},
	} {
		if m.at != nil {
			if _, err := shared.Exec(ctx, `INSERT INTO match_registry (match_id, start_time, start_time_utc) VALUES (?, ?, ?)`,
				m.id, *m.at, *m.at); err != nil {
				t.Fatalf("registry %s: %v", m.id, err)
			}
		} else if _, err := shared.Exec(ctx, `INSERT INTO match_registry (match_id) VALUES (?)`, m.id); err != nil {
			t.Fatalf("registry %s: %v", m.id, err)
		}
		if _, err := shared.Exec(ctx, `INSERT INTO match_participants (
				match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists, kda, accuracy,
				personal_score, time_played_seconds, headshot_kills, damage_dealt, damage_taken
			) VALUES (?, ?, 'PostSync', 1, ?, ?, ?, 2, ?, ?, 1000, ?, 1, ?, 500)`,
			m.id, postSyncTestXUID, m.outcome, m.kills, m.deaths, m.kda, m.acc, m.secs, m.dmg); err != nil {
			t.Fatalf("participant %s: %v", m.id, err)
		}
	}
	if _, err := player.Exec(ctx, `INSERT INTO player_match_enrichment (match_id, performance_score, stage) VALUES ('m1', 71.5, 'perf')`); err != nil {
		t.Fatalf("enrichment: %v", err)
	}
	return pdb, d1, d2
}

func TestPostSyncProgressionStore_MatchsScoresEtAgregats(t *testing.T) {
	pdb, d1, d2 := newPostSyncFixture(t)
	st := NewPostSyncProgressionStore(pdb)
	ctx := context.Background()

	rows, err := st.MatchRows(ctx, d1.AddDate(0, -1, 0))
	if err != nil {
		t.Fatalf("MatchRows: %v", err)
	}
	if len(rows) != 2 || rows[0].MatchID != "m1" || rows[1].MatchID != "m2" {
		t.Fatalf("matchs = %+v, attendu m1 puis m2 (m3 sans début écarté)", rows)
	}
	want := port.ProgressionMatchRow{MatchID: "m1", KDA: 1.5, Kills: 10, Deaths: 5, Assists: 2, Accuracy: 0.6,
		PersonalScore: 1000, TimeSeconds: 600, DamageDealt: 2000, DamageTaken: 500}
	got := rows[0]
	if !got.StartTime.Equal(d1) {
		t.Errorf("début = %v, attendu %v", got.StartTime, d1)
	}
	got.StartTime = time.Time{}
	if got != want {
		t.Errorf("m1 = %+v, attendu %+v", got, want)
	}
	if rows, err := st.MatchRows(ctx, d2); err != nil || len(rows) != 1 || rows[0].MatchID != "m2" {
		t.Errorf("depuis d2 : %+v, %v (attendu m2 seul)", rows, err)
	}

	perf, err := st.PerformanceScores(ctx, []string{"m1", "m2"})
	if err != nil || len(perf) != 1 || perf["m1"] != 71.5 {
		t.Errorf("scores = %v, %v (attendu m1 seul)", perf, err)
	}
	if none, err := st.PerformanceScores(ctx, nil); err != nil || len(none) != 0 {
		t.Errorf("sans match : %v, %v", none, err)
	}

	agg, err := st.Aggregates(ctx, port.ProgressionAggregateParams{
		AccuracyThreshold: 0.5, OCThreshold: 0.83, DRThreshold: 1.59, HPToKill: 225})
	if err != nil {
		t.Fatalf("Aggregates: %v", err)
	}
	// m1 + m2 + m3 : le jeu d'essai ne porte pas de variante de Campagne. m3 n'a pas de début : il compte dans
	// les agrégats par participant, pas dans les jours de précision (jointure sur le registre datée).
	if agg.MatchesPlayed != 3 || agg.Wins != 2 || agg.Kills != 113 || agg.Headshots != 3 || agg.Assists != 6 {
		t.Errorf("agrégats = %+v", agg)
	}
	// La colonne est en pourcentage et le seuil en ratio : tout match à plus de 0,5 % compte (m1 et m2).
	if agg.AccuracyDays != 2 {
		t.Errorf("jours de précision = %d, attendu 2 (m1 et m2 ont un début ; seuil ratio contre colonne pourcentage)", agg.AccuracyDays)
	}

	times, err := st.LastMatchTimes(ctx)
	if err != nil || len(times) != 2 || !times[0].Equal(d2) || !times[1].Equal(d1) {
		t.Errorf("deux derniers matchs = %v, %v (attendu d2 puis d1)", times, err)
	}
}

func TestPlayerSnapshotStore_LitLeJoueurEtLaBasePartagee(t *testing.T) {
	pdb, _, d2 := newPostSyncFixture(t)
	ctx := context.Background()
	if _, err := pdb.Player.Exec(ctx, `INSERT INTO career_progression (rank, rank_name, recorded_at) VALUES (5, 'Hero', now())`); err != nil {
		t.Fatalf("career_progression : %v", err)
	}
	if _, err := pdb.Shared.Exec(ctx, `INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES ('m1', ?, 100, 2), ('m2', ?, 200, 0)`,
		postSyncTestXUID, postSyncTestXUID); err != nil {
		t.Fatalf("medals: %v", err)
	}

	s := &port.PlayerSnapshot{SkillTierByPlaylist: map[string]string{}}
	var rd port.PlayerSnapshotReader = NewPlayerSnapshotStore(pdb)
	rd.FillPlayerCounters(ctx, s)
	if s.CurrentRank != 5 || s.CurrentRankName != "Hero" {
		t.Errorf("rang = %d %q, attendu 5 Hero", s.CurrentRank, s.CurrentRankName)
	}
	if s.PersonalAwardCount != 0 || s.ChallengePathsCount != 0 || len(s.SkillTierByPlaylist) != 0 {
		t.Errorf("tables vides : compteurs à zéro attendus, obtenu %+v", s)
	}

	rd.FillSharedAggregates(ctx, s)
	if want := float64(113) / 14; s.KDRatio != want {
		t.Errorf("KD = %v, attendu %v", s.KDRatio, want)
	}
	if want := 2.0 / 3.0; s.Winrate != want {
		t.Errorf("taux de victoire = %v, attendu %v", s.Winrate, want)
	}
	if s.BestKDAMatchID != "m3" || s.BestKDA != float64(99+2)/1 {
		t.Errorf("meilleur KDA = %v (%s), attendu m3 à 101", s.BestKDA, s.BestKDAMatchID)
	}
	if !s.LastMatchStartTime.Equal(d2) {
		t.Errorf("dernier match = %v, attendu %v", s.LastMatchStartTime, d2)
	}
	if len(s.EarnedMedalIDs) != 1 {
		t.Errorf("médailles obtenues = %v, attendu {100}", s.EarnedMedalIDs)
	} else if _, ok := s.EarnedMedalIDs[100]; !ok {
		t.Errorf("médailles obtenues = %v, attendu {100}", s.EarnedMedalIDs)
	}

	// Sans xuid, la base partagée n'est pas lue.
	empty := &port.PlayerSnapshot{}
	NewPlayerSnapshotStore(&PlayerDB{Player: pdb.Player, SharedReader: pdb.SharedReader}).FillSharedAggregates(ctx, empty)
	if empty.KDRatio != 0 || empty.EarnedMedalIDs != nil {
		t.Errorf("sans xuid : rien ne doit être lu, obtenu %+v", empty)
	}
}

// La précision stockée en pourcentage (61,73 : la valeur du journal de démarrage) sort en ratio dans
// les bornes des records ; sans la conversion, aucun record de précision ne se pose.
func TestPostSyncProgressionStore_PrecisionEnRatio(t *testing.T) {
	pdb, d1, _ := newPostSyncFixture(t)
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx, `UPDATE match_participants SET accuracy = 61.73 WHERE match_id = 'm1'`); err != nil {
		t.Fatalf("maj précision: %v", err)
	}
	rows, err := NewPostSyncProgressionStore(pdb).MatchRows(ctx, d1.AddDate(0, -1, 0))
	if err != nil || len(rows) != 2 {
		t.Fatalf("MatchRows: %v (%d lignes)", err, len(rows))
	}
	if got := rows[0].Accuracy; got < 0.6172 || got > 0.6174 {
		t.Errorf("précision de m1 = %v, attendu 0,6173 (ratio)", got)
	}
	for _, r := range rows {
		if !records.IsPlausibleValue(records.MetricAccuracy, r.Accuracy) {
			t.Errorf("précision %v de %s hors des bornes des records", r.Accuracy, r.MatchID)
		}
	}
}
