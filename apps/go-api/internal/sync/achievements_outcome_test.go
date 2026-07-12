// Package sync — achievements_outcome_test.go : garde-rail du fix « erreurs
// partielles » récurrentes du sync achievements H5.
//
// CONTRAT VERROUILLÉ : un skip BÉNIN du sync achievements (pas de provider /
// capability / token Xbox ce cycle — cas attendu et récurrent pour Halo 5) ne doit
// PAS remonter d'erreur au runner (sinon chaque cycle affiche « sync terminée avec
// erreurs partielles » + un WARN, alors que le sync des matchs a réussi). Seul un
// échec RÉEL (XSTS / metadata / HTTP / DB) → erreur. RunAchievementsHook encode cette
// règle ; RunAchievementsOnly (CLI) reste true UNIQUEMENT sur un succès.
package sync

import (
	"context"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
)

// TestRunAchievementsHook_BenignSkipIsNotAnError : provider nil = skip bénin le plus
// simple (même issue achievementsSkipped que le cas prod « aucun access_token
// disponible »). Le hook doit rendre nil (pas d'« erreurs partielles ») et le bool
// CLI doit rester false.
func TestRunAchievementsHook_BenignSkipIsNotAnError(t *testing.T) {
	e := &SyncEngine{gamertag: "TestPlayer", titleSlug: titlePkg.DefaultSlug} // provider nil

	if err := e.RunAchievementsHook(context.Background()); err != nil {
		t.Errorf("skip bénin (provider nil) → RunAchievementsHook err=%v, want nil "+
			"(le runner ne doit pas marquer le cycle en erreurs partielles)", err)
	}
	if e.RunAchievementsOnly(context.Background()) {
		t.Error("skip bénin → RunAchievementsOnly=true, want false (pas un succès)")
	}
}
