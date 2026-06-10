//go:build integration

// Package sync — invariants_gate_integration_test.go : GATE au commit des
// contrats de données du pipeline multi-joueurs.
//
// Scénario cadenassé (incidents 2026-05-27 et 2026-06-10) : une escouade de
// 3 joueurs joue le MÊME match. Le joueur dont le sync passe en premier
// insère le match en shared (registry + participants des 3 joueurs). Les deux
// autres syncs voient alors le match comme « déjà connu » (loadKnownMatchIDs
// source 2) et SKIPPENT le traitement per-player (delta-skip, 0 inséré).
//
// CONTRAT : malgré le skip, la convergence (runConditionalPostSync →
// ensurePlayerEnrichmentRows) DOIT créer la row player_match_enrichment de
// chaque joueur. Avant le fix countSharedMatchesMissingEnrichment, un cycle
// « pur skip » (0 inséré + scores complets + events chargés) ne déclenchait
// jamais le pipeline → enrichment manquant à durée indéterminée.
//
// Les assertions passent par internal/sync/invariants (mêmes définitions que
// la future sentinelle runtime) : toute violation FAIL = test rouge = commit
// bloqué en CI (job go-coverage, tags=integration, CGO).
package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/invariants"
)

// makeSquadMatchJSON construit un match stats JSON où les participants sont
// EXACTEMENT les users fournis (gamertag + xuid réels du harnais), tous dans
// la même équipe gagnante. Contrairement à makeMatchJSON (xuids génériques),
// indispensable ici : le delta-skip repose sur la présence du xuid du joueur
// courant dans shared.match_participants.
func makeSquadMatchJSON(matchID string, users []*userSetup) map[string]any {
	playersArr := make([]any, 0, len(users))
	for i, u := range users {
		coreStats := map[string]any{
			"Kills":               float64(10 + i),
			"Deaths":              float64(5 + i),
			"Assists":             float64(3 + i),
			"ShotsFired":          float64(100),
			"ShotsHit":            float64(50),
			"PersonalScore":       float64(1000 + i*100),
			"DamageDealt":         float64(2500.0),
			"DamageTaken":         float64(2000.0),
			"AverageLifeDuration": "PT30S",
			"Medals": []any{
				map[string]any{"NameId": float64(100 + i), "Count": float64(2)},
			},
		}
		playersArr = append(playersArr, map[string]any{
			"PlayerId":   fmt.Sprintf("xuid(%s)", u.xuid),
			"PlayerName": u.gamertag,
			"LastTeamId": float64(0),
			"Outcome":    float64(2), // WIN
			"Rank":       float64(i + 1),
			"PlayerTeamStats": []any{
				map[string]any{"Stats": map[string]any{"CoreStats": coreStats}},
			},
			"ParticipationInfo": map[string]any{"TimePlayed": "PT10M0S"},
		})
	}
	return map[string]any{
		"MatchId": matchID,
		"MatchInfo": map[string]any{
			"StartTime":           "2026-06-09T19:14:22Z",
			"EndTime":             "2026-06-09T19:24:22Z",
			"MapVariant":          map[string]any{"AssetId": "map-asset-001", "PublicName": "Aquarius"},
			"GameVariantCategory": float64(9),
			"PlaylistExperience":  "Arena:Slayer",
			"Playlist":            map[string]any{"AssetId": "playlist-001", "PublicName": "Quick Play"},
			"UgcGameVariant":      map[string]any{"AssetId": "gv-001", "PublicName": "Slayer"},
			"Duration":            "PT10M0S",
			"PlayableDuration":    "PT9M30S",
			"LifecycleMode":       float64(3),
		},
		"Players": playersArr,
	}
}

// TestGate_DeltaSkip_EnrichmentConverges_integration : le gate principal.
//
// Étapes :
//  1. user0 RunDelta → insère le match squad en shared (participants ×3).
//  2. user1 et user2 RunDelta → delta-skip (« match connu ») → 0 inséré →
//     la convergence du post-sync conditionnel doit créer leur enrichment.
//  3. invariants.CheckPlayer pour les 3 joueurs : AUCUNE violation FAIL.
func TestGate_DeltaSkip_EnrichmentConverges_integration(t *testing.T) {
	const nUsers = 3
	env := newMultiUserEnv(t, nUsers)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Le MÊME match dans l'historique des 3 joueurs, avec leurs 3 xuids en
	// participants (≠ seedMatches qui génère des matchs uniques par user).
	const squadMatchID = "a0000000-0000-4000-8000-000000000001"
	matchJSON := makeSquadMatchJSON(squadMatchID, env.users)
	for _, u := range env.users {
		u.mock.history = makeHistory(squadMatchID)
		u.mock.statsBody = map[string]map[string]any{squadMatchID: matchJSON}
	}

	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        1,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}

	// 1. user0 traite le match (insertion shared + son propre enrichment).
	res0, err := env.users[0].engine.RunDelta(ctx, opts)
	if err != nil {
		t.Fatalf("RunDelta user0: %v", err)
	}
	if res0.MatchesInserted != 1 {
		t.Fatalf("user0 MatchesInserted = %d (attendu 1)", res0.MatchesInserted)
	}

	// 2. user1 et user2 : delta-skip attendu (0 inséré), convergence requise.
	for i := 1; i < nUsers; i++ {
		res, err := env.users[i].engine.RunDelta(ctx, opts)
		if err != nil {
			t.Fatalf("RunDelta user%d: %v", i, err)
		}
		if res.MatchesInserted != 0 {
			t.Errorf("user%d MatchesInserted = %d (attendu 0 : delta-skip cross-player)",
				i, res.MatchesInserted)
		}
	}

	// 3. Invariants par joueur — le contrat du gate.
	sharedDB, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	defer release()

	for i, u := range env.users {
		playerSQL := u.pool.Player.SQLDb()
		if playerSQL == nil {
			t.Fatalf("user%d: SQLDb nil", i)
		}
		report, err := invariants.CheckPlayer(ctx, playerSQL, sharedDB, u.xuid)
		if err != nil {
			t.Fatalf("CheckPlayer user%d (%s): %v", i, u.gamertag, err)
		}
		for _, v := range report.Violations {
			t.Logf("user%d (%s) %s", i, u.gamertag, v.String())
		}
		if fails := report.Failures(); len(fails) > 0 {
			t.Errorf("user%d (%s) : %d violation(s) FAIL — le delta-skip n'a pas convergé : %v",
				i, u.gamertag, len(fails), fails)
		}
	}
}

// TestGate_PureSkipCycle_TriggersConvergence_integration : cadenasse le
// déclencheur lui-même. Un joueur dont la player DB est vierge mais dont le
// match est déjà en shared (inséré par un coéquipier) doit voir
// hasConvergenceBacklog=true — sans dépendre de scores NULL préexistants ni
// d'events manquants (la convergence prod reposait accidentellement dessus).
func TestGate_PureSkipCycle_TriggersConvergence_integration(t *testing.T) {
	env := newMultiUserEnv(t, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const squadMatchID = "b0000000-0000-4000-8000-000000000002"
	matchJSON := makeSquadMatchJSON(squadMatchID, env.users)
	for _, u := range env.users {
		u.mock.history = makeHistory(squadMatchID)
		u.mock.statsBody = map[string]map[string]any{squadMatchID: matchJSON}
	}
	opts := domain.SyncOptions{
		MatchType:         "matchmaking",
		MaxMatches:        1,
		WithParticipants:  true,
		WithMedals:        true,
		RequestsPerSecond: 100,
	}

	// user0 insère ; user1 n'a encore JAMAIS sync.
	if _, err := env.users[0].engine.RunDelta(ctx, opts); err != nil {
		t.Fatalf("RunDelta user0: %v", err)
	}

	// Avant le RunDelta de user1 : sa player DB n'a aucune row enrichment pour
	// ce match alors que shared.match_participants contient son xuid →
	// countSharedMatchesMissingEnrichment doit être > 0.
	sharedDB, release, err := env.provider.Get(ctx)
	if err != nil {
		t.Fatalf("provider.Get: %v", err)
	}
	playerSQL := env.users[1].pool.Player.SQLDb()
	missing := countSharedMatchesMissingEnrichment(ctx, playerSQL, sharedDB, env.users[1].xuid)
	release()
	if missing < 1 {
		t.Fatalf("countSharedMatchesMissingEnrichment = %d (attendu ≥ 1 : le déclencheur du cycle pur-skip)",
			missing)
	}
}
