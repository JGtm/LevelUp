//go:build cgo

// media_repo_delete_source_test.go — chargement de source_path par
// LoadMediaForDeletion : colonne absente tolérée, toute autre erreur rendue, et
// visibilité d'une ligne introuvable.
package duckdb

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
)

const deletionTestPath = "JGtm/clip.mp4"

func seedDeletionRow(t *testing.T, db *DB, status string) {
	t.Helper()
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, status)
		VALUES ('JGtm', ?, 'clip.mp4', NULLIF(?, ''))
	`, deletionTestPath, status); err != nil {
		t.Fatalf("seed media_files: %v", err)
	}
}

// Base jamais indexée : pas de colonne source_path, la suppression continue.
func TestLoadMediaForDeletion_SourcePathColumnAbsent(t *testing.T) {
	db := createSharedSocialSchemaForMediaTests(t)
	seedDeletionRow(t, db, "")
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})

	target, err := repo.LoadMediaForDeletion(context.Background(), deletionTestPath)
	if err != nil {
		t.Fatalf("LoadMediaForDeletion: %v", err)
	}
	if target.SourcePath != "" {
		t.Errorf("SourcePath = %q, want vide sans colonne", target.SourcePath)
	}
}

func TestLoadMediaForDeletion_SourcePathLoaded(t *testing.T) {
	db := createSharedSocialSchemaForMediaTests(t)
	if _, err := db.SQLDb().Exec(`ALTER TABLE media_files ADD COLUMN source_path VARCHAR`); err != nil {
		t.Fatalf("alter: %v", err)
	}
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, source_path)
		VALUES ('JGtm', ?, 'clip.mp4', 'JGtm/orig.mkv')
	`, deletionTestPath); err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})

	target, err := repo.LoadMediaForDeletion(context.Background(), deletionTestPath)
	if err != nil {
		t.Fatalf("LoadMediaForDeletion: %v", err)
	}
	if target.SourcePath != "JGtm/orig.mkv" {
		t.Errorf("SourcePath = %q, want JGtm/orig.mkv", target.SourcePath)
	}
}

// Une erreur qui n'est pas « colonne inconnue » fait échouer la suppression :
// la colonne existe mais sa valeur n'est pas lisible comme texte.
func TestLoadMediaForDeletion_SourcePathOtherErrorFails(t *testing.T) {
	db := createSharedSocialSchemaForMediaTests(t)
	if _, err := db.SQLDb().Exec(`ALTER TABLE media_files ADD COLUMN source_path INTEGER[]`); err != nil {
		t.Fatalf("alter: %v", err)
	}
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, source_path)
		VALUES ('JGtm', ?, 'clip.mp4', [1, 2])
	`, deletionTestPath); err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})

	if _, err := repo.LoadMediaForDeletion(context.Background(), deletionTestPath); err == nil {
		t.Fatal("LoadMediaForDeletion doit échouer sur une erreur autre que colonne inconnue")
	}
}

// Une ligne introuvable se supprime ; une ligne déjà supprimée reste un 404.
func TestLoadMediaForDeletion_MissingVisibleDeletedNot(t *testing.T) {
	cases := []struct {
		status   string
		wantFind bool
	}{
		{domain.MediaStatusMissing, true},
		{domain.MediaStatusDeleted, false},
		{"", true},
	}
	for _, tc := range cases {
		t.Run("status="+tc.status, func(t *testing.T) {
			db := createSharedSocialSchemaForMediaTests(t)
			seedDeletionRow(t, db, tc.status)
			repo := NewMediaRepo(&PlayerDB{SharedSocial: db, Gamertag: "JGtm"})

			_, err := repo.LoadMediaForDeletion(context.Background(), deletionTestPath)
			var notFound bool
			if err != nil {
				var apiErr *domain.APIError
				notFound = errors.As(err, &apiErr) && apiErr.Code == "not_found"
				if !notFound {
					t.Fatalf("erreur inattendue: %v", err)
				}
			}
			if found := !notFound; found != tc.wantFind {
				t.Errorf("trouvé = %v, want %v", found, tc.wantFind)
			}
		})
	}
}
