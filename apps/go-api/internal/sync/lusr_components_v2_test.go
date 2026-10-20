package sync

// lusr_components_v2_test.go — les 8 composantes de performance (lusr_component_history)
// s'écrivent depuis le LUSR v2 (D-6, plan backlog 2026-10-16). Avant ce lot, seule la v1
// les écrivait : le mode canonique (v1 sautée) les laissait figées, alors que Prestige, le
// profil et l'escouade les lisent.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/skill"
)

func TestPostSyncLUSRV2_WritesTheEightComponents(t *testing.T) {

	const nMatches = 12
	repoRoot := t.TempDir()
	pr := titlePkg.NewPathResolver(repoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	playerPath := pr.PlayerDBPath(titlePkg.DefaultSlug, "Tester")
	seedLUSRShared(t, sharedPath, nMatches)
	seedLUSRPlayer(t, playerPath)

	shared, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	defer shared.Close()
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()

	engine := NewSyncEngineForTitle(repoRoot, titlePkg.DefaultSlug, "Tester", recomputerTestXUID, nil, nil)
	var res domain.PostSyncResult
	engine.runSkillRatingSteps(context.Background(), player.SQLDb(), NewPinnedSharedAccess(shared.SQLDb()), &res)
	if res.LUSRUpdated != nMatches {
		t.Fatalf("LUSRUpdated = %d, attendu %d", res.LUSRUpdated, nMatches)
	}

	// Chaque match noté porte ses composantes dans la vue lue par Prestige, le profil et l'escouade.
	rows, err := player.SQLDb().Query(
		`SELECT match_id, COUNT(*) FROM lusr_component_history_latest GROUP BY match_id`)
	if err != nil {
		t.Fatalf("lecture lusr_component_history_latest: %v", err)
	}
	defer rows.Close()
	perMatch := make(map[string]int)
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		perMatch[id] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(perMatch) != nMatches {
		t.Fatalf("composantes pour %d match(s), attendu %d : %v", len(perMatch), nMatches, perMatch)
	}
	for id, n := range perMatch {
		// 7 composantes sont calculables sur la fixture (pas de médailles) ; la moyenne
		// glissante de précision n'existe qu'après MinMatchesForAccuracyDelta matchs.
		if n < 6 {
			t.Errorf("match %s : %d composante(s), attendu au moins 6", id, n)
		}
	}

	// Valeur et poids d'une composante déterministe : victoire = 1, poids de CompositeWeights.
	var value, weight float64
	if err := player.SQLDb().QueryRow(`SELECT value, weight FROM lusr_component_history_latest
		WHERE match_id = ? AND component_name = ?`,
		"00000000-0000-0000-0000-000000000001", skill.MetricKeyWinFactor).Scan(&value, &weight); err != nil {
		t.Fatalf("lecture de win_factor: %v", err)
	}
	if value != 1.0 || weight != skill.CompositeWeights[skill.MetricKeyWinFactor] {
		t.Errorf("win_factor = %v (poids %v), attendu 1.0 (poids %v)",
			value, weight, skill.CompositeWeights[skill.MetricKeyWinFactor])
	}
}
