// Sentinelles documentaires pour la Phase 5 du plan principal
// (PLAN_FIX_SYNC_RELIABILITY_2026-05-24).
//
// Ces tests documentent le comportement attendu APRES fix Phase 5 :
//
//	E.3 — submitMatchAsBatch_AlreadyInRegistry_DoesNotInflateInserted (Bug #3b)
//	E.5 — runConditionalPostSync_ZeroActuallyInserted_Skipped (Bug #3c)
//
// Aujourd'hui : le code increment MatchesInserted optimistement avant Persist.
// Apres Phase 5 : un pre-check sur match_registry (via SharedPersister status)
// evite l'inflation pour les matchs idempotents.
//
// Tests rouges sous build tag `bug_repro` (cf. csr_art_repro_test.go pattern).
// La CI default skip ; lancement explicite via :
//   go test -tags 'integration bug_repro' -run E_Phase5 ./internal/sync/...
//
//go:build integration && bug_repro

package sync

import "testing"

// TestE3_SubmitMatchAsBatch_AlreadyInRegistry_DoesNotInflateInserted
// documente le bug #3b. Aujourd'hui, le compteur MatchesInserted est
// incremente meme quand SharedPersister va detecter l'idempotence et no-op.
//
// Apres Phase 5 : le test passera VERT car le code fera un pre-check
// SELECT EXISTS FROM match_registry avant d'incrementer.
func TestE3_SubmitMatchAsBatch_AlreadyInRegistry_DoesNotInflateInserted(t *testing.T) {
	t.Skip("Phase 5 du plan principal a implementer — cf. PLAN_FIX_SYNC_RELIABILITY_2026-05-24 section Bug #3b.\n" +
		"Sentinel : retirer le t.Skip apres impl du pre-check match_registry dans submitMatchAsBatch.")
}

// TestE5_RunConditionalPostSync_ZeroActuallyInserted_Skipped documente
// le bug #3c : le post-sync trigge sur MatchesInserted > 0 alors que les
// matchs sont des dupes idempotents → cascade de tasks inutiles
// (events heal, weapon kills re-download, etc.).
//
// Apres Phase 5 : ce test passera VERT car runConditionalPostSync ne
// trigge plus sur le compteur optimiste mais sur la liste filtrée des
// matchs reellement nouveaux.
func TestE5_RunConditionalPostSync_ZeroActuallyInserted_Skipped(t *testing.T) {
	t.Skip("Phase 5 du plan principal a implementer — cf. PLAN_FIX_SYNC_RELIABILITY_2026-05-24 section Bug #3c.\n" +
		"Sentinel : retirer le t.Skip apres impl du gating post-sync sur actuallyInsertedIDs.")
}
