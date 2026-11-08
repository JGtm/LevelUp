//go:build cgo

// media_repo_missing_cgo_test.go — lectures et fusion des médias introuvables, sur une vraie
// shared_social DuckDB : ce que liste la liste, l'ordre des candidats, et surtout ce que la fusion
// n'écrit PAS (aucune ligne supprimée, tables append-only seulement allongées).
package duckdb

import (
	"context"
	"errors"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
)

// missingSchemaDB : le schéma minimal de media_files + colonnes d'exécution, sans la ligne témoin.
func missingSchemaDB(t *testing.T) *DB {
	t.Helper()
	db := createSharedSocialSchemaForMediaTests(t)
	for _, stmt := range []string{
		`DELETE FROM media_files`,
		`ALTER TABLE media_files ADD COLUMN capture_start_utc TIMESTAMPTZ`,
		`ALTER TABLE media_files ADD COLUMN capture_end_utc TIMESTAMPTZ`,
		`ALTER TABLE media_files ADD COLUMN updated_at TIMESTAMP`,
		`ALTER TABLE media_files ADD COLUMN source_path VARCHAR`,
		`ALTER TABLE media_files ADD COLUMN hls_path VARCHAR`,
	} {
		if _, err := db.SQLDb().Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	return db
}

// seedMedia insère une fiche et rend son id. capture est « YYYY-MM-DD HH:MM:SS » en UTC, vide = aucune.
func seedMedia(t *testing.T, db *DB, owner, path, kind, status, capture string) int64 {
	t.Helper()
	var id int64
	err := db.SQLDb().QueryRow(`
		INSERT INTO media_files (player_slug, file_path, file_name, kind, thumbnail_path, status,
		                         capture_start_utc, capture_end_utc)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''),
		        CAST(NULLIF(?, '') || '+00' AS TIMESTAMPTZ), CAST(NULLIF(?, '') || '+00' AS TIMESTAMPTZ))
		RETURNING id`,
		owner, path, path[len(owner)+1:], kind, owner+"/thumbs/"+path[len(owner)+1:]+".webp", status,
		capture, capture).Scan(&id)
	if err != nil {
		t.Fatalf("seed %s: %v", path, err)
	}
	return id
}

func seedLike(t *testing.T, db *DB, path, liker string) {
	t.Helper()
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, ?, ?, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`, path, liker, liker); err != nil {
		t.Fatalf("seed like: %v", err)
	}
}

func seedManualAssociation(t *testing.T, db *DB, mediaID int64, matchID string) {
	t.Helper()
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_match_associations_history (media_file_id, match_id, delta_seconds, is_manual, is_active)
		VALUES (?, ?, 0, TRUE, TRUE)`, mediaID, matchID); err != nil {
		t.Fatalf("seed association: %v", err)
	}
}

func likersOf(t *testing.T, db *DB, path string) []string {
	t.Helper()
	rows, err := db.SQLDb().Query(
		`SELECT liker_slug FROM media_likes_latest WHERE media_path = ? AND is_liked = TRUE ORDER BY liker_slug`, path)
	if err != nil {
		t.Fatalf("likers: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func countRows(t *testing.T, db *DB, table string) int {
	t.Helper()
	var n int
	if err := db.SQLDb().QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestListMissingMedia_OnlyMissingOfThePlayer_NewestFirst(t *testing.T) {
	db := missingSchemaDB(t)
	old := seedMedia(t, db, "JGtm", "JGtm/old.mp4", "video", "missing", "2026-01-01 10:00:00")
	seedMedia(t, db, "JGtm", "JGtm/recent.mp4", "video", "missing", "2026-02-01 10:00:00")
	seedMedia(t, db, "JGtm", "JGtm/present.mp4", "video", "", "2026-03-01 10:00:00")
	seedMedia(t, db, "JGtm", "JGtm/gone.mp4", "video", "deleted", "2026-03-02 10:00:00")
	seedMedia(t, db, "Autre", "Autre/theirs.mp4", "video", "missing", "2026-03-03 10:00:00")
	seedManualAssociation(t, db, old, "match-1")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})

	items, err := repo.ListMissingMedia(context.Background())
	if err != nil {
		t.Fatalf("ListMissingMedia: %v", err)
	}
	var paths []string
	for _, it := range items {
		paths = append(paths, it.FilePath)
	}
	if !slices.Equal(paths, []string{"JGtm/recent.mp4", "JGtm/old.mp4"}) {
		t.Fatalf("liste = %v, attendu [recent old] et rien d'autre", paths)
	}
	oldItem := items[1]
	if oldItem.Basename != "old.mp4" || oldItem.Kind != "video" || oldItem.CaptureEndUTC == nil {
		t.Errorf("fiche old = %+v", oldItem)
	}
	if oldItem.ThumbnailPath == nil || *oldItem.ThumbnailPath != "JGtm/thumbs/old.mp4.webp" {
		t.Errorf("miniature = %v", oldItem.ThumbnailPath)
	}
	if oldItem.MatchID == nil || *oldItem.MatchID != "match-1" {
		t.Errorf("match = %v, attendu match-1", oldItem.MatchID)
	}
	if items[0].MatchID != nil {
		t.Errorf("fiche sans rattachement : match = %v", items[0].MatchID)
	}
}

func TestListMergeCandidates_OrderedByDateProximity(t *testing.T) {
	db := missingSchemaDB(t)
	seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "2026-01-01 10:00:00")
	seedMedia(t, db, "JGtm", "JGtm/far.mp4", "video", "", "2026-01-01 09:00:00")
	seedMedia(t, db, "JGtm", "JGtm/near.mp4", "video", "", "2026-01-01 10:05:00")
	seedMedia(t, db, "JGtm", "JGtm/undated.mp4", "video", "", "")
	seedMedia(t, db, "JGtm", "JGtm/photo.png", "image", "", "2026-01-01 10:00:10")
	seedMedia(t, db, "JGtm", "JGtm/other_missing.mp4", "video", "missing", "2026-01-01 10:00:20")
	seedMedia(t, db, "JGtm", "JGtm/deleted.mp4", "video", "deleted", "2026-01-01 10:00:30")
	seedMedia(t, db, "Autre", "Autre/theirs.mp4", "video", "", "2026-01-01 10:00:40")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
	ctx := context.Background()

	ref, err := repo.LoadMissingTarget(ctx, "JGtm/missing.mp4")
	if err != nil {
		t.Fatalf("LoadMissingTarget: %v", err)
	}
	if ref.CaptureUTC == nil || ref.Kind != "video" || ref.OwnerSlug != "JGtm" {
		t.Fatalf("ref = %+v", ref)
	}
	cands, err := repo.ListMergeCandidates(ctx, ref, 50)
	if err != nil {
		t.Fatalf("ListMergeCandidates: %v", err)
	}
	var paths []string
	for _, c := range cands {
		paths = append(paths, c.FilePath)
	}
	if want := []string{"JGtm/near.mp4", "JGtm/far.mp4", "JGtm/undated.mp4"}; !slices.Equal(paths, want) {
		t.Fatalf("candidats = %v, attendu %v (même nature, même joueur, présentes, par proximité)", paths, want)
	}
	if d := cands[0].DeltaSeconds; d == nil || *d != 300 {
		t.Errorf("écart du plus proche = %v, attendu 300", d)
	}
	if d := cands[1].DeltaSeconds; d == nil || *d != 3600 {
		t.Errorf("écart du plus lointain = %v, attendu 3600", d)
	}
	if cands[2].DeltaSeconds != nil {
		t.Errorf("fiche sans date : écart = %v, attendu nil", cands[2].DeltaSeconds)
	}

	limited, err := repo.ListMergeCandidates(ctx, ref, 2)
	if err != nil || len(limited) != 2 {
		t.Fatalf("limite 2 : %d candidats, err=%v", len(limited), err)
	}
}

func TestLoadTargets_StatusDecidesWhichLoaderSeesTheRow(t *testing.T) {
	db := missingSchemaDB(t)
	seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "")
	seedMedia(t, db, "JGtm", "JGtm/present.mp4", "video", "", "")
	seedMedia(t, db, "JGtm", "JGtm/gone.mp4", "video", "deleted", "")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		load func(context.Context, string) (domain.MediaFileRef, error)
		path string
		want bool
	}{
		{"introuvable vue par LoadMissingTarget", repo.LoadMissingTarget, "JGtm/missing.mp4", true},
		{"présente refusée par LoadMissingTarget", repo.LoadMissingTarget, "JGtm/present.mp4", false},
		{"supprimée refusée par LoadMissingTarget", repo.LoadMissingTarget, "JGtm/gone.mp4", false},
		{"présente vue par LoadMergeTarget", repo.LoadMergeTarget, "JGtm/present.mp4", true},
		{"introuvable refusée par LoadMergeTarget", repo.LoadMergeTarget, "JGtm/missing.mp4", false},
		{"supprimée refusée par LoadMergeTarget", repo.LoadMergeTarget, "JGtm/gone.mp4", false},
		{"inconnue refusée", repo.LoadMergeTarget, "JGtm/unknown.mp4", false},
	} {
		_, err := tc.load(ctx, tc.path)
		var apiErr *domain.APIError
		switch {
		case tc.want && err != nil:
			t.Errorf("%s : %v", tc.name, err)
		case !tc.want && !(errors.As(err, &apiErr) && apiErr.Code == "not_found"):
			t.Errorf("%s : err = %v, attendu not_found", tc.name, err)
		}
	}
}

func TestApplyMediaMerge_TransfersLikesAndManualAssociation_DeletesNothing(t *testing.T) {
	db := missingSchemaDB(t)
	missingID := seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "")
	targetID := seedMedia(t, db, "JGtm", "JGtm/target.mp4", "video", "", "")
	seedLike(t, db, "JGtm/missing.mp4", "alice")
	seedLike(t, db, "JGtm/missing.mp4", "bob")
	seedLike(t, db, "JGtm/target.mp4", "bob")
	seedLike(t, db, "JGtm/target.mp4", "carol")
	seedManualAssociation(t, db, missingID, "match-7")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
	ctx := context.Background()
	missing, err := repo.LoadMissingTarget(ctx, "JGtm/missing.mp4")
	if err != nil {
		t.Fatal(err)
	}
	target, err := repo.LoadMergeTarget(ctx, "JGtm/target.mp4")
	if err != nil {
		t.Fatal(err)
	}
	filesBefore, likesBefore, assocBefore := countRows(t, db, "media_files"), countRows(t, db, "media_likes_history"),
		countRows(t, db, "media_match_associations_history")

	tx, err := db.SQLDb().BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ApplyMediaMerge(ctx, tx, missing, target); err != nil {
		_ = tx.Rollback()
		t.Fatalf("ApplyMediaMerge: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	if got := likersOf(t, db, "JGtm/target.mp4"); !slices.Equal(got, []string{"alice", "bob", "carol"}) {
		t.Errorf("likers de la cible = %v, attendu alice bob carol (additionnés, sans doublon)", got)
	}
	if got := likersOf(t, db, "JGtm/missing.mp4"); len(got) != 0 {
		t.Errorf("likers restant sur la fiche fondue = %v, attendu aucun", got)
	}
	var match string
	if err := db.SQLDb().QueryRow(
		`SELECT match_id FROM media_match_associations_latest WHERE media_file_id = ? AND is_manual`, targetID,
	).Scan(&match); err != nil || match != "match-7" {
		t.Errorf("rattachement manuel de la cible = %q (err=%v), attendu match-7", match, err)
	}
	var status string
	if err := db.SQLDb().QueryRow(`SELECT status FROM media_files WHERE id = ?`, missingID).Scan(&status); err != nil ||
		status != domain.MediaStatusDeleted {
		t.Errorf("statut de la fiche fondue = %q (err=%v), attendu deleted", status, err)
	}
	if err := db.SQLDb().QueryRow(`SELECT COALESCE(status, '') FROM media_files WHERE id = ?`, targetID).Scan(&status); err != nil ||
		status != "" {
		t.Errorf("statut de la cible = %q (err=%v), attendu inchangé", status, err)
	}
	if n := countRows(t, db, "media_files"); n != filesBefore {
		t.Errorf("media_files : %d lignes, attendu %d (aucune ligne supprimée)", n, filesBefore)
	}
	if n := countRows(t, db, "media_likes_history"); n < likesBefore {
		t.Errorf("media_likes_history : %d lignes, attendu au moins %d (INSERT seulement)", n, likesBefore)
	}
	if n := countRows(t, db, "media_match_associations_history"); n != assocBefore+1 {
		t.Errorf("media_match_associations_history : %d lignes, attendu %d", n, assocBefore+1)
	}
}

// Le rattachement automatique de la fiche introuvable est repris, en manuel, par la cible qui n'en
// a pas.
func TestApplyMediaMerge_AutomaticAssociationIsCarriedAsManual(t *testing.T) {
	db := missingSchemaDB(t)
	missingID := seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "")
	targetID := seedMedia(t, db, "JGtm", "JGtm/target.mp4", "video", "", "")
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_match_associations_history (media_file_id, match_id, delta_seconds, is_manual, is_active)
		VALUES (?, 'auto-1', 12, FALSE, TRUE)`, missingID); err != nil {
		t.Fatal(err)
	}
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
	ctx := context.Background()
	missing, _ := repo.LoadMissingTarget(ctx, "JGtm/missing.mp4")
	target, _ := repo.LoadMergeTarget(ctx, "JGtm/target.mp4")

	tx, _ := db.SQLDb().BeginTx(ctx, nil)
	if err := repo.ApplyMediaMerge(ctx, tx, missing, target); err != nil {
		_ = tx.Rollback()
		t.Fatalf("ApplyMediaMerge: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var match string
	if err := db.SQLDb().QueryRow(
		`SELECT match_id FROM media_match_associations_latest WHERE media_file_id = ? AND is_manual`, targetID,
	).Scan(&match); err != nil || match != "auto-1" {
		t.Errorf("rattachement manuel de la cible = %q (err=%v), attendu auto-1", match, err)
	}
}

// Une fiche introuvable sans aucun rattachement n'en invente pas à sa cible.
func TestApplyMediaMerge_NoAssociationCarriesNothing(t *testing.T) {
	db := missingSchemaDB(t)
	seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "")
	seedMedia(t, db, "JGtm", "JGtm/target.mp4", "video", "", "")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
	ctx := context.Background()
	missing, _ := repo.LoadMissingTarget(ctx, "JGtm/missing.mp4")
	target, _ := repo.LoadMergeTarget(ctx, "JGtm/target.mp4")

	tx, _ := db.SQLDb().BeginTx(ctx, nil)
	if err := repo.ApplyMediaMerge(ctx, tx, missing, target); err != nil {
		_ = tx.Rollback()
		t.Fatalf("ApplyMediaMerge: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, db, "media_match_associations_history"); n != 0 {
		t.Errorf("associations : %d lignes, attendu 0", n)
	}
}

// La cible a déjà un rattachement manuel : celui de la fiche introuvable, manuel ou automatique, ne
// le remplace pas.
func TestApplyMediaMerge_TargetManualAssociationIsKept(t *testing.T) {
	for _, manualOnMissing := range []bool{false, true} {
		db := missingSchemaDB(t)
		missingID := seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "")
		targetID := seedMedia(t, db, "JGtm", "JGtm/target.mp4", "video", "", "")
		if _, err := db.SQLDb().Exec(`
			INSERT INTO media_match_associations_history (media_file_id, match_id, delta_seconds, is_manual, is_active)
			VALUES (?, 'missing-match', 12, ?, TRUE)`, missingID, manualOnMissing); err != nil {
			t.Fatal(err)
		}
		seedManualAssociation(t, db, targetID, "target-match")
		repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
		ctx := context.Background()
		missing, _ := repo.LoadMissingTarget(ctx, "JGtm/missing.mp4")
		target, _ := repo.LoadMergeTarget(ctx, "JGtm/target.mp4")
		before := countRows(t, db, "media_match_associations_history")

		tx, _ := db.SQLDb().BeginTx(ctx, nil)
		if err := repo.ApplyMediaMerge(ctx, tx, missing, target); err != nil {
			_ = tx.Rollback()
			t.Fatalf("ApplyMediaMerge: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatal(err)
		}
		var match string
		if err := db.SQLDb().QueryRow(
			`SELECT match_id FROM media_match_associations_latest WHERE media_file_id = ? AND is_manual`, targetID,
		).Scan(&match); err != nil || match != "target-match" {
			t.Errorf("manuel sur la fiche introuvable = %v : rattachement de la cible = %q (err=%v), attendu target-match",
				manualOnMissing, match, err)
		}
		if n := countRows(t, db, "media_match_associations_history"); n != before {
			t.Errorf("manuel sur la fiche introuvable = %v : %d lignes, attendu %d (rien d'écrit)", manualOnMissing, n, before)
		}
	}
}

// La fiche a cessé d'être introuvable entre le chargement et l'écriture : erreur, et la transaction
// annulée ne laisse aucun like déplacé.
func TestApplyMediaMerge_NoLongerMissingFailsAndRollbackLeavesLikes(t *testing.T) {
	db := missingSchemaDB(t)
	missingID := seedMedia(t, db, "JGtm", "JGtm/missing.mp4", "video", "missing", "")
	seedMedia(t, db, "JGtm", "JGtm/target.mp4", "video", "", "")
	seedLike(t, db, "JGtm/missing.mp4", "alice")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})
	ctx := context.Background()
	missing, _ := repo.LoadMissingTarget(ctx, "JGtm/missing.mp4")
	target, _ := repo.LoadMergeTarget(ctx, "JGtm/target.mp4")
	if _, err := db.SQLDb().Exec(`UPDATE media_files SET status = NULL WHERE id = ?`, missingID); err != nil {
		t.Fatal(err)
	}
	likesBefore := countRows(t, db, "media_likes_history")

	tx, _ := db.SQLDb().BeginTx(ctx, nil)
	err := repo.ApplyMediaMerge(ctx, tx, missing, target)
	_ = tx.Rollback()
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "not_found" {
		t.Fatalf("err = %v, attendu not_found", err)
	}
	if n := countRows(t, db, "media_likes_history"); n != likesBefore {
		t.Errorf("media_likes_history : %d lignes après annulation, attendu %d", n, likesBefore)
	}
	if got := likersOf(t, db, "JGtm/target.mp4"); len(got) != 0 {
		t.Errorf("likers de la cible après annulation = %v, attendu aucun", got)
	}
}
