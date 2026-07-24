// Package wire — post_sync_deltas_bespoke_test.go : tests de la résolution
// FR du libellé de rang carrière (V72-27a). Fichier séparé de
// post_sync_deltas_test.go (verrouillé côté chantier parallèle) — réutilise
// ses helpers `recordingEmitter`/`hasCategory` (même package `wire`).
package wire

import (
	"context"
	"testing"

	"levelup/go-api/internal/notifications"
)

// withRankLabelResolver câble un résolveur pour la durée du test et le
// désarme via t.Cleanup — évite qu'un résolveur de test fuite vers un autre
// test du package (var globale boot-only).
func withRankLabelResolver(t *testing.T, fn RankLabelResolver) {
	t.Helper()
	SetRankLabelResolver(fn)
	t.Cleanup(func() { SetRankLabelResolver(nil) })
}

// TestEmitCareerRankDelta_RankNameFR_UsesResolver : quand le résolveur est
// câblé et connaît (titleSlug, rankID), rank_name porte le libellé FR — pas
// CurrentRankName (baké EN depuis career_ranks.title_en, cf. career.go).
func TestEmitCareerRankDelta_RankNameFR_UsesResolver(t *testing.T) {
	withRankLabelResolver(t, func(titleSlug string, rankID int) (string, bool) {
		if titleSlug == "halo_infinite" && rankID == 192 {
			return "Colonel Platine 2", true
		}
		return "", false
	})

	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 190, CurrentRankName: "Colonel Platinum 2"}
	after := &PlayerSnapshot{CurrentRank: 192, CurrentRankName: "Colonel Platinum 2"}
	emitCareerRankDelta(context.Background(), em, "halo_infinite", "p1", before, after)

	if !hasCategory(em.emitted, notifications.CategoryCareerRank) {
		t.Fatal("career_rank aurait dû être émis")
	}
	got := em.emitted[0].Params["rank_name"]
	// rankSubRoman convertit le "2" final en chiffre romain.
	if got != "Colonel Platine II" {
		t.Errorf("rank_name = %q, want %q (résolveur FR + roman)", got, "Colonel Platine II")
	}
}

// TestEmitCareerRankDelta_RankNameFR_FallsBackWithoutResolver : résolveur nil
// (dégradation gracieuse) → rank_name retombe sur CurrentRankName (EN existant).
func TestEmitCareerRankDelta_RankNameFR_FallsBackWithoutResolver(t *testing.T) {
	withRankLabelResolver(t, nil)

	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 190, CurrentRankName: "Colonel Platinum 2"}
	after := &PlayerSnapshot{CurrentRank: 192, CurrentRankName: "Colonel Platinum 2"}
	emitCareerRankDelta(context.Background(), em, "halo_infinite", "p1", before, after)

	if !hasCategory(em.emitted, notifications.CategoryCareerRank) {
		t.Fatal("career_rank aurait dû être émis")
	}
	got := em.emitted[0].Params["rank_name"]
	if got != "Colonel Platinum II" {
		t.Errorf("rank_name = %q, want %q (fallback EN + roman)", got, "Colonel Platinum II")
	}
}

// TestEmitCareerRankDelta_RankNameFR_FallsBackOnUnknownRank : résolveur câblé
// mais qui ne connaît PAS ce (titleSlug, rankID) (ex. autre titre, rank_id
// absent du catalog) → fallback EN, jamais de chaîne vide.
func TestEmitCareerRankDelta_RankNameFR_FallsBackOnUnknownRank(t *testing.T) {
	withRankLabelResolver(t, func(string, int) (string, bool) { return "", false })

	em := &recordingEmitter{}
	before := &PlayerSnapshot{CurrentRank: 1, CurrentRankName: "Recruit"}
	after := &PlayerSnapshot{CurrentRank: 2, CurrentRankName: "Private"}
	emitCareerRankDelta(context.Background(), em, "halo_5", "p1", before, after)

	got := em.emitted[0].Params["rank_name"]
	if got != "Private" {
		t.Errorf("rank_name = %q, want %q (fallback EN — résolveur sans match)", got, "Private")
	}
}
