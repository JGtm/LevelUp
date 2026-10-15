//go:build integration

// Package sync — csr_ranked_rule_integration_test.go : le chemin de sync complet
// (fetchMatchData -> persistFetchedMatch, DuckDB en mémoire, client Halo simulé) applique la règle
// unique de classement (D-10) : is_ranked du registre et CSR par match viennent du RankRecap du
// payload skill, jamais du nom de la playlist (absent du payload stats).
package sync

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func TestSyncPath_RankedByRankRecap_NotByPlaylistName(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	patchSharedSchemaForBatch(t, sharedDB)

	const rankedID = "aabbccdd-0000-4000-8000-0000000000a1"
	const socialID = "aabbccdd-0000-4000-8000-0000000000a2"
	const me = "0000000000000000"
	const other = "0000000000000001"

	rankedSkill := loadSkillFixture(t, "skill_match_ranked.json")[ruleRankedXUID]
	socialSkill := loadSkillFixture(t, "skill_match_social.json")[rulePlayerXUID]
	mock := &mockHaloClient{
		history: makeHistory(rankedID, socialID),
		statsBody: map[string]map[string]any{
			rankedID: makeMatchJSON(rankedID, 2),
			socialID: makeMatchJSON(socialID, 2),
		},
		skillBody: map[string]map[string]*MatchSkillData{
			rankedID: {me: rankedSkill, other: rankedSkill},
			socialID: {me: socialSkill, other: socialSkill},
		},
	}
	e := &SyncEngine{gamertag: "Player0", xuid: me, titleSlug: "halo_infinite"}
	opts := domain.SyncOptions{MatchType: "matchmaking", MaxMatches: 10, WithParticipants: true, WithMedals: true}
	result := &domain.SyncResult{}
	for _, id := range []string{rankedID, socialID} {
		fm, err := e.fetchMatchData(context.Background(), mock, id, opts)
		if err != nil {
			t.Fatalf("fetchMatchData(%s): %v", id, err)
		}
		if err := e.persistFetchedMatch(context.Background(), sharedDB, playerDB, result, fm); err != nil {
			t.Fatalf("persistFetchedMatch(%s): %v", id, err)
		}
	}

	var rankedFlag, socialFlag bool
	if err := sharedDB.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id = ?`, rankedID).Scan(&rankedFlag); err != nil {
		t.Fatal(err)
	}
	if err := sharedDB.QueryRow(`SELECT is_ranked FROM match_registry WHERE match_id = ?`, socialID).Scan(&socialFlag); err != nil {
		t.Fatal(err)
	}
	if !rankedFlag || socialFlag {
		t.Errorf("is_ranked : classé=%v (want true), social=%v (want false)", rankedFlag, socialFlag)
	}
	var sharedRanked, sharedSocial int
	_ = sharedDB.QueryRow(`SELECT COUNT(*) FROM match_csrs_latest WHERE match_id = ?`, rankedID).Scan(&sharedRanked)
	_ = sharedDB.QueryRow(`SELECT COUNT(*) FROM match_csrs_latest WHERE match_id = ?`, socialID).Scan(&sharedSocial)
	if sharedRanked != 2 || sharedSocial != 0 {
		t.Errorf("match_csrs : classé=%d (want 2), social=%d (want 0)", sharedRanked, sharedSocial)
	}
	var playerRanked, playerSocial int
	_ = playerDB.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest WHERE match_id = ? AND rating_type = 'CSR'`, rankedID).Scan(&playerRanked)
	_ = playerDB.QueryRow(`SELECT COUNT(*) FROM match_skill_rank_latest WHERE match_id = ? AND rating_type = 'CSR'`, socialID).Scan(&playerSocial)
	if playerRanked != 1 || playerSocial != 0 {
		t.Errorf("match_skill_rank CSR : classé=%d (want 1), social=%d (want 0)", playerRanked, playerSocial)
	}
}
