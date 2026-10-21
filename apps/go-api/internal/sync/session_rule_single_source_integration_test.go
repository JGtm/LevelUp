//go:build integration

package sync

// session_rule_single_source_integration_test.go — la synchro (étape sessions du post-sync,
// incrémentale) et le recalcul complet admin écrivent session_id avec la MÊME règle : celle
// du réglage du titre (settings.SessionOptions). Même réglage, mêmes matchs ⇒ mêmes sessions.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	settings_platform "levelup/go-api/internal/platform/settings"
)

const (
	ruleMatchCount = 9
	ruleFriendA    = "xuid_rule_friend_a"
	ruleFriendB    = "xuid_rule_friend_b"
	ruleFriendAGT  = "RuleFriendA"
	ruleFriendBGT  = "RuleFriendB"
)

// buildSessionRuleFixture : 9 matchs espacés de 10 min ; dans l'équipe du joueur, deux amis
// constants et un 4e coéquipier différent à chaque match (inconnu, hors liste d'amis).
func buildSessionRuleFixture(t *testing.T) (player, shared *sql.DB) {
	t.Helper()
	shared = openFixtureDB(t, buildSharedDDL())
	player = openFixtureDB(t, buildPlayerDDL())
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(player); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	base := time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC)
	for i := 0; i < ruleMatchCount; i++ {
		matchID := fmt.Sprintf("rule-match-%02d", i)
		start := base.Add(time.Duration(i) * 10 * time.Minute)
		mustExec(t, shared, `
			INSERT INTO match_registry (match_id, start_time, end_time, start_time_utc, end_time_utc, is_ranked)
			VALUES (?, ?, ?, ?, ?, FALSE)`,
			matchID, start, start.Add(8*time.Minute), start.Format(time.RFC3339), start.Add(8*time.Minute).Format(time.RFC3339))
		team := []string{fixXUID, ruleFriendA, ruleFriendB, fmt.Sprintf("xuid_rule_random_%02d", i)}
		for _, x := range team {
			mustExec(t, shared, `
				INSERT INTO match_participants (match_id, xuid, team_id, time_played_seconds)
				VALUES (?, ?, 0, 480)`, matchID, x)
		}
		mustExec(t, player, `INSERT INTO player_match_enrichment (match_id) VALUES (?)`, matchID)
	}
	mustExec(t, shared, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?), (?, ?)`,
		ruleFriendA, ruleFriendAGT, ruleFriendB, ruleFriendBGT)
	return player, shared
}

// readSessionAssignments : match_id → "session_id|session_label" (vue _latest).
func readSessionAssignments(t *testing.T, player *sql.DB) map[string]string {
	t.Helper()
	rows, err := player.Query(`
		SELECT match_id, COALESCE(CAST(session_id AS VARCHAR), ''), COALESCE(session_label, '')
		FROM player_match_enrichment_latest`)
	if err != nil {
		t.Fatalf("lecture des sessions: %v", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, sid, label string
		if err := rows.Scan(&id, &sid, &label); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out[id] = sid + "|" + label
	}
	return out
}

func distinctSessions(assignments map[string]string) int {
	seen := map[string]bool{}
	for _, v := range assignments {
		seen[v] = true
	}
	return len(seen)
}

func TestSessionRule_SyncAndAdminRecalcApplyTheSameSetting(t *testing.T) {
	cases := []struct {
		mode         string
		wantSessions int
	}{
		// Rupture dès qu'un coéquipier change : le 4e change à chaque match.
		{mode: "group", wantSessions: ruleMatchCount},
		// Rupture seulement si un AMI change : les deux amis restent, une seule session.
		{mode: "friends", wantSessions: 1},
	}
	friends := []string{ruleFriendAGT, ruleFriendBGT}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			ctx := context.Background()
			settingsPath := filepath.Join(t.TempDir(), "app_settings.json")
			body := fmt.Sprintf(`{"session_team_change_mode": %q, "session_gap_minutes": 120}`, tc.mode)
			if err := os.WriteFile(settingsPath, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			store := settings_platform.NewStore(settingsPath)
			overlay := filepath.Join(t.TempDir(), "absent_overlay.json")

			// Chemin 1 : synchro (post-sync incrémental).
			syncPlayer, syncShared := buildSessionRuleFixture(t)
			engine := NewSyncEngine(t.TempDir(), fixGamertag, fixXUID, nil, nil).
				WithFriendsLoader(func() ([]string, error) { return friends, nil }).
				WithSessionOptionsLoader(settings_platform.SessionOptionsLoader(store, overlay))
			engine.runSessionAssignment(ctx, syncPlayer, syncShared, &domain.PostSyncResult{})
			bySync := readSessionAssignments(t, syncPlayer)

			// Chemin 2 : recalcul complet admin.
			adminPlayer, adminShared := buildSessionRuleFixture(t)
			if _, err := recalculateSessionsFull(ctx, adminPlayer, adminShared, fixXUID,
				settings_platform.SessionOptions(ctx, store, overlay), friends); err != nil {
				t.Fatalf("recalcul complet: %v", err)
			}
			byAdmin := readSessionAssignments(t, adminPlayer)

			if got := distinctSessions(byAdmin); got != tc.wantSessions {
				t.Errorf("recalcul admin : %d sessions, attendu %d", got, tc.wantSessions)
			}
			if got := distinctSessions(bySync); got != tc.wantSessions {
				t.Errorf("synchro : %d sessions, attendu %d (réglage %q ignoré ?)", got, tc.wantSessions, tc.mode)
			}
			for id, admin := range byAdmin {
				if bySync[id] != admin {
					t.Errorf("match %s : synchro %q, recalcul admin %q — deux règles", id, bySync[id], admin)
				}
			}
		})
	}
}
