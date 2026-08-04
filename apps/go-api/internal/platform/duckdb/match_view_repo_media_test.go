//go:build integration

// Package duckdb — match_view_repo_media_test.go : tests d'intégration pour
// MatchViewRepo.GetMatchMedia (Q24).
//
// Lance via : go test -tags=integration ./internal/platform/duckdb/ -run TestMatchViewRepo_GetMatchMedia -v
package duckdb

import (
	"context"
	"sort"
	"testing"
)

// TestMatchViewRepo_GetMatchMedia_CrossPlayer vérifie que la query Q24
// retourne les médias de tous les auteurs associés à un match — y compris
// ceux uploadés par un coéquipier.
//
// Verrou anti-régression du fix 2026-05-07 : avant, la query filtrait sur
// mf.player_slug, ce qui (a) excluait les coéquipiers et (b) renvoyait 0
// résultat lorsque le caller passait l'XUID au lieu du gamertag stocké.
func TestMatchViewRepo_GetMatchMedia_CrossPlayer(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMatchViewRepo(pdb, mediaTestPlayerXUID)

	// Match m1 a 2 médias associés : med-A1 (Alice, /A1.mp4) et med-C1 (HeroPlayer, /C1.png).
	// Le filtre player_slug-disparu doit faire remonter les deux.
	rows, err := repo.GetMatchMedia(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchMedia(m1): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("GetMatchMedia(m1) = %d rows, want 2 (Alice + HeroPlayer)", len(rows))
	}
	paths := []string{rows[0].FilePath, rows[1].FilePath}
	sort.Strings(paths)
	want := []string{"/A1.mp4", "/C1.png"}
	for i, p := range want {
		if paths[i] != p {
			t.Errorf("paths[%d] = %q, want %q (full=%v)", i, paths[i], p, paths)
		}
	}
}

// TestMatchViewRepo_GetMatchMedia_TeammateOnly vérifie qu'un match associé
// uniquement à un média de coéquipier remonte bien (alors que le caller
// est l'XUID de HeroPlayer).
func TestMatchViewRepo_GetMatchMedia_TeammateOnly(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMatchViewRepo(pdb, mediaTestPlayerXUID)

	// Match m3 est uniquement associé à med-B1 (Bob, coéquipier).
	rows, err := repo.GetMatchMedia(context.Background(), "m3")
	if err != nil {
		t.Fatalf("GetMatchMedia(m3): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("GetMatchMedia(m3) = %d rows, want 1 (Bob teammate)", len(rows))
	}
	if rows[0].FilePath != "/B1.mp4" {
		t.Errorf("FilePath = %q, want /B1.mp4", rows[0].FilePath)
	}
}

// TestMatchViewRepo_GetMatchMedia_NoAssociations vérifie qu'un match sans
// association renvoie un slice vide sans erreur (cas dégradé propre).
func TestMatchViewRepo_GetMatchMedia_NoAssociations(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMatchViewRepo(pdb, mediaTestPlayerXUID)

	rows, err := repo.GetMatchMedia(context.Background(), "m_unknown")
	if err != nil {
		t.Fatalf("GetMatchMedia(m_unknown): %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("GetMatchMedia(m_unknown) = %d rows, want 0", len(rows))
	}
}

// TestMatchViewRepo_GetMatchMedia_OrderByCaptureTimeASC vérifie l'ordre :
// capture_end_utc ASC NULLS LAST. m1 a 2 médias capturés à 14:30 et 16:00.
func TestMatchViewRepo_GetMatchMedia_OrderByCaptureTimeASC(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	repo := NewMatchViewRepo(pdb, mediaTestPlayerXUID)

	rows, err := repo.GetMatchMedia(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchMedia(m1): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("GetMatchMedia(m1) = %d rows, want 2", len(rows))
	}
	// med-A1 capture_end_utc = 2025-01-10 14:30 → doit venir avant
	// med-C1 capture_end_utc = 2025-01-10 16:00.
	if rows[0].FilePath != "/A1.mp4" {
		t.Errorf("rows[0].FilePath = %q, want /A1.mp4 (earlier capture)", rows[0].FilePath)
	}
	if rows[1].FilePath != "/C1.png" {
		t.Errorf("rows[1].FilePath = %q, want /C1.png (later capture)", rows[1].FilePath)
	}
}

// TestMatchViewRepo_GetMatchMedia_LikedIsPerViewer : l'onglet Médias d'un match
// sert le cœur DU VIEWER, comme la galerie.
//
// Sans ce verrou, la page match resterait la dernière surface à lire l'ancienne
// colonne globale : le clip d'un coéquipier y apparaîtrait liké par tout le
// monde alors que la galerie, elle, afficherait le bon état — une incohérence
// entre deux pages qui montrent LE MÊME média, plus déroutante encore que le
// bug d'origine.
//
// Dataset : /A1.mp4 est liké par HeroPlayer (cf. fixture), /C1.png par personne.
func TestMatchViewRepo_GetMatchMedia_LikedIsPerViewer(t *testing.T) {
	pdb := newTestPlayerDBForMediaScenario(t)
	ctx := context.Background()

	likedByViewer := func(t *testing.T, viewer, path string) bool {
		t.Helper()
		repo := NewMatchViewRepo(pdb, mediaTestPlayerXUID).WithViewer(viewer)
		rows, err := repo.GetMatchMedia(ctx, "m1")
		if err != nil {
			t.Fatalf("GetMatchMedia(m1) viewer=%s: %v", viewer, err)
		}
		for _, r := range rows {
			if r.FilePath == path {
				return r.Liked
			}
		}
		t.Fatalf("média %q absent de l'onglet Médias pour le viewer %s", path, viewer)
		return false
	}

	if !likedByViewer(t, mediaTestPlayerSlug, "/A1.mp4") {
		t.Error("le viewer qui a liké /A1.mp4 doit voir son cœur allumé sur la page match")
	}
	if likedByViewer(t, "Bob", "/A1.mp4") {
		t.Error("RÉGRESSION : Bob voit allumé le cœur d'un like qui appartient à HeroPlayer")
	}
	if likedByViewer(t, mediaTestPlayerSlug, "/C1.png") {
		t.Error("/C1.png n'est liké par personne : cœur attendu vide")
	}
}
