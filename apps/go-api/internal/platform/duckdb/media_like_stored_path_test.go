//go:build cgo

// media_like_stored_path_test.go — reproduction en base réelle du bug « likes
// médias cassés » (plan v7.3 lot 2, item 1.5).
//
// CE QUE CE TEST PROUVE. Depuis la migration des chemins médias
// (cmd/migrate-media-paths), media_files.file_path est stocké au format
// CANONIQUE relatif forward-slash « {owner_slug}/{rel} ». La clé d'écriture des
// likes est ce file_path : toute autre forme (chemin absolu reconstruit depuis
// le disque, ou séparateurs Windows) ne touche AUCUNE ligne — le service traduit
// alors rowsAffected == 0 en 404 et le like est perdu, sans qu'aucune erreur ne
// remonte de la base.
//
// C'est exactement ce que faisait le handler de like avant le correctif : il
// résolvait l'URL servable vers un chemin absolu quand le fichier existait sous
// media_captures_base_dir — c'est-à-dire toujours, côté serveur.
package duckdb

import (
	"context"
	"path/filepath"
	"testing"
)

// seedCanonicalMediaFile insère une ligne media_files au format stocké réel
// (relatif, forward-slash) et retourne sa clé.
func seedCanonicalMediaFile(t *testing.T, db *DB) string {
	t.Helper()
	const storedPath = "JGtm/clip.mp4"
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name)
		VALUES ('JGtm', ?, 'clip.mp4')
	`, storedPath); err != nil {
		t.Fatalf("seed media_files canonique: %v", err)
	}
	return storedPath
}

// TestMediaLike_OnlyStoredRelativePathMatches reproduit la panne et vérifie le
// comportement attendu après correctif : seul le chemin STOCKÉ met à jour la
// ligne ; les formes dérivées du disque sont silencieusement inopérantes.
func TestMediaLike_OnlyStoredRelativePathMatches(t *testing.T) {
	socialDB := createSharedSocialSchemaForMediaTests(t)
	storedPath := seedCanonicalMediaFile(t, socialDB)
	repo := NewMediaRepo(&PlayerDB{SharedSocial: socialDB, Gamertag: "JGtm"})
	ctx := context.Background()

	// (1) Chemin ABSOLU reconstruit depuis capturesBase — ce que le handler
	// envoyait avant le correctif. Aucune ligne touchée → 404 côté service.
	absPath := filepath.Join(t.TempDir(), "JGtm", "clip.mp4")
	updated, err := repo.SetMediaLike(ctx, absPath, true)
	if err != nil {
		t.Fatalf("SetMediaLike(abs): %v", err)
	}
	if updated {
		t.Fatalf("chemin absolu %q ne devrait toucher aucune ligne", absPath)
	}

	// (2) Chemin stocké avec séparateurs OS (effet de filepath.FromSlash sous
	// Windows) — inopérant lui aussi : la colonne contient des forward-slashes.
	updated, err = repo.SetMediaLike(ctx, `JGtm\clip.mp4`, true)
	if err != nil {
		t.Fatalf("SetMediaLike(backslash): %v", err)
	}
	if updated {
		t.Error(`chemin "JGtm\clip.mp4" ne devrait toucher aucune ligne (format DB = forward-slash)`)
	}

	// (3) Chemin STOCKÉ tel quel — la seule clé valide (comportement corrigé).
	updated, err = repo.SetMediaLike(ctx, storedPath, true)
	if err != nil {
		t.Fatalf("SetMediaLike(stored): %v", err)
	}
	if !updated {
		t.Fatalf("chemin stocké %q doit mettre à jour la ligne", storedPath)
	}

	got := reopenAndCount(t, socialDB,
		`SELECT COUNT(*) FROM media_files WHERE file_path = ? AND liked = TRUE`, storedPath)
	if got != 1 {
		t.Errorf("attendu liked=TRUE persisté pour %q, got %d ligne(s)", storedPath, got)
	}
}

// TestMediaLike_SocialEventVisibleForStoredPath vérifie le volet social : l'event
// append-only écrit sous la clé stockée est lisible via la vue media_likes_latest
// (ADR 0026 — lecture des tables append-only par la vue _latest uniquement), et
// l'écriture survit au CHECKPOINT + reopen (ADR 0022).
//
// C'est la moitié « badges ♥ » du bug : un like écrit sous une clé qui ne
// correspond pas au file_path servi par la galerie reste invisible pour toujours
// (cas observé en base : un event orphelin sous chemin absolu, 0 correspondance
// dans media_files).
func TestMediaLike_SocialEventVisibleForStoredPath(t *testing.T) {
	socialDB := createSharedSocialSchemaForMediaTests(t)
	storedPath := seedCanonicalMediaFile(t, socialDB)
	repo := NewMediaRepo(&PlayerDB{SharedSocial: socialDB, Gamertag: "JGtm"})
	ctx := context.Background()

	if err := repo.ToggleSharedLike(ctx, storedPath, "Chocoboflor", "Chocoboflor", true); err != nil {
		t.Fatalf("ToggleSharedLike: %v", err)
	}

	likers, err := repo.GetMediaLikers(ctx, []string{storedPath})
	if err != nil {
		t.Fatalf("GetMediaLikers: %v", err)
	}
	info, ok := likers[storedPath]
	if !ok || info.Total != 1 {
		t.Fatalf("likers pour %q = %+v, want Total=1", storedPath, info)
	}

	got := reopenAndCount(t, socialDB,
		`SELECT COUNT(*) FROM media_likes_latest WHERE media_path = ? AND is_liked = TRUE`, storedPath)
	if got != 1 {
		t.Errorf("attendu 1 event visible dans media_likes_latest après reopen, got %d", got)
	}
}
