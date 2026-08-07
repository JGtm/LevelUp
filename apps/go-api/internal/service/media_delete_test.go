//go:build cgo

// media_delete_test.go — suppression définitive d'un média (v7.3 lot 2, item 3.1).
//
// Ce fichier verrouille les quatre invariants de la décision de design (cf.
// domain/media_delete.go), sur une vraie shared_social DuckDB et de vrais
// fichiers sur disque :
//
//  1. INVISIBILITÉ — après suppression, la galerie ne sert plus le média.
//  2. DURABILITÉ — le WAL est flushé à l'instant du 200 : un redémarrage ne
//     peut pas faire réapparaître un média dont les fichiers sont détruits
//     (l'état fantôme le plus dommageable de cette opération).
//  3. APPEND-ONLY INTACT — pas une ligne de media_likes_history n'est
//     supprimée ni modifiée : elles deviennent des orphelins invisibles.
//  4. AUTORISATION — un non-propriétaire ne détruit rien, ni en base ni sur
//     le disque.
package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

// deleteFixture monte une shared_social minimale + les fichiers disque
// correspondants, et retourne le service prêt à supprimer.
type deleteFixture struct {
	db           *duckdb.DB
	dbPath       string
	capturesBase string
	svc          *MediaService
	clipAbs      string
	thumbAbs     string
}

func newDeleteFixture(t *testing.T) *deleteFixture {
	t.Helper()
	root := t.TempDir()
	dbPath := filepath.Join(root, "shared_social.duckdb")
	db, err := duckdb.OpenReadWriteShared(dbPath, "")
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}

	// Schéma aligné sur la prod pour ce qui touche la suppression : `status`
	// (support du soft-delete) et media_likes_history + sa vue _latest.
	// Pas de colonne hls_path : c'est le cas des bases n'ayant jamais indexé de
	// média, et LoadMediaForDeletion doit le tolérer.
	ddl := `
		CREATE SEQUENCE IF NOT EXISTS media_id_seq START 1;
		CREATE TABLE media_files (
			id INTEGER DEFAULT nextval('media_id_seq') PRIMARY KEY,
			player_slug VARCHAR, file_path VARCHAR NOT NULL, file_name VARCHAR,
			kind VARCHAR DEFAULT 'video', thumbnail_path VARCHAR, status VARCHAR,
			mtime TIMESTAMP WITH TIME ZONE, updated_at TIMESTAMP,
			capture_start_utc TIMESTAMP WITH TIME ZONE,
			capture_end_utc TIMESTAMP WITH TIME ZONE,
			indexed_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);
		CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1;
		CREATE TABLE media_likes_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
			media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
			is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW media_likes_latest AS
			SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
			FROM media_likes_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
				ORDER BY written_at DESC, id DESC) = 1;
		CREATE SEQUENCE IF NOT EXISTS media_mmah_id_seq START 1;
		CREATE TABLE media_match_associations_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_mmah_id_seq'),
			media_file_id BIGINT NOT NULL, match_id VARCHAR, delta_seconds INTEGER,
			is_manual BOOLEAN DEFAULT FALSE, is_active BOOLEAN DEFAULT TRUE,
			associated_at TIMESTAMP, written_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW media_match_associations_latest AS
			SELECT id, media_file_id, match_id, delta_seconds, is_manual, is_active,
			       associated_at, written_at
			FROM media_match_associations_history
			WHERE is_active
			QUALIFY ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id
				ORDER BY written_at DESC, id DESC) = 1;
		INSERT INTO media_files (player_slug, file_path, file_name, thumbnail_path)
		VALUES ('JGtm', 'JGtm/clip.mp4', 'clip.mp4', 'JGtm/thumbs/clip.webp');
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES ('JGtm/clip.mp4', 'Chocoboflor', 'Chocoboflor', TRUE, CURRENT_TIMESTAMP);
	`
	if _, err := db.SQLDb().Exec(ddl); err != nil {
		t.Fatalf("seed shared_social: %v", err)
	}
	// Le seed appartient au fichier, pas au WAL : sans ce CHECKPOINT l'assertion
	// de durabilité mesurerait le seed au lieu de la suppression.
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint seed: %v", err)
	}

	capturesBase := filepath.Join(root, "captures")
	clipAbs := filepath.Join(capturesBase, "JGtm", "clip.mp4")
	thumbAbs := filepath.Join(capturesBase, "JGtm", "thumbs", "clip.webp")
	for _, p := range []string{clipAbs, thumbAbs} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
		if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	pdb := &duckdb.PlayerDB{SharedSocial: db, Gamertag: "JGtm"}
	svc := NewMediaService(duckdb.NewMediaRepo(pdb), "",
		WithMediaWriterAcquirer(func() (*dblease.LeasedWriter, error) {
			return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
		}),
		WithMediaFileRemover(ops.NewOSMediaFileRemover(func() string { return capturesBase })))

	t.Cleanup(func() { _ = db.Close() })
	return &deleteFixture{
		db: db, dbPath: dbPath, capturesBase: capturesBase,
		svc: svc, clipAbs: clipAbs, thumbAbs: thumbAbs,
	}
}

// ownerDeleteRequest : requête du propriétaire légitime, auth appliquée.
func ownerDeleteRequest() domain.MediaDeleteRequest {
	return domain.MediaDeleteRequest{
		FilePath:      "JGtm/clip.mp4",
		RequesterSlug: "JGtm",
		AuthEnforced:  true,
	}
}

// TestMediaService_DeleteMedia_OwnerRemovesFilesAndHidesMedia couvre le happy
// path complet : fichiers effacés, média invisible, likes append-only intacts.
func TestMediaService_DeleteMedia_OwnerRemovesFilesAndHidesMedia(t *testing.T) {
	f := newDeleteFixture(t)
	ctx := context.Background()

	resp, err := f.svc.DeleteMedia(ctx, ownerDeleteRequest())
	if err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}
	if !resp.Deleted {
		t.Error("réponse Deleted=false alors que la suppression a réussi")
	}
	if resp.FilesRemoved != 2 {
		t.Errorf("FilesRemoved = %d, want 2 (source + miniature)", resp.FilesRemoved)
	}

	// 1. Les fichiers ne sont plus servables : ServeMediaFile ne consulte pas la
	//    base, c'est leur absence qui garantit « plus jamais servi ».
	for _, p := range []string{f.clipAbs, f.thumbAbs} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("fichier toujours présent après suppression: %s (err=%v)", p, err)
		}
	}

	// 2. Le média n'est plus listé par la galerie (chemin de lecture réel).
	rows, err := duckdb.NewMediaRepo(&duckdb.PlayerDB{
		SharedSocial: f.db, Gamertag: "JGtm",
	}).LoadMediaFiles(ctx, domain.MediaFilters{}, 50, 0)
	if err != nil {
		t.Fatalf("LoadMediaFiles: %v", err)
	}
	for _, r := range rows {
		if r.FilePath == "JGtm/clip.mp4" {
			t.Error("média supprimé toujours servi par la galerie")
		}
	}

	// 3. La ligne existe toujours (soft-delete) : on ne DELETE jamais une table
	//    portant des index ART.
	var status string
	if err := f.db.SQLDb().QueryRow(
		`SELECT COALESCE(status,'') FROM media_files WHERE file_path = ?`, "JGtm/clip.mp4",
	).Scan(&status); err != nil {
		t.Fatalf("relecture status: %v", err)
	}
	if status != domain.MediaStatusDeleted {
		t.Errorf("status = %q, want %q", status, domain.MediaStatusDeleted)
	}

	// 4. APPEND-ONLY INTACT : l'event de like est toujours là, non modifié.
	//    Il devient un orphelin invisible — jamais lu, jamais réécrit.
	var events int
	if err := f.db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes_history WHERE media_path = ?`, "JGtm/clip.mp4",
	).Scan(&events); err != nil {
		t.Fatalf("relecture media_likes_history: %v", err)
	}
	if events != 1 {
		t.Errorf("media_likes_history = %d event(s), want 1 — la suppression ne doit "+
			"RIEN écrire sur une table append-only (ADR 0026)", events)
	}
}

// TestMediaService_DeleteMedia_CommitsWithCheckpoint verrouille la durabilité.
//
// Sans CommitWithCheckpoint, le masquage reste dans le WAL jusqu'à 5 min : tout
// redémarrage dans cette fenêtre RESSUSCITE un média dont les fichiers ont déjà
// été détruits — une entrée fantôme illisible et non réparable. Mesure à chaud,
// base ouverte : un Close déclencherait un CHECKPOINT qui masquerait la panne.
func TestMediaService_DeleteMedia_CommitsWithCheckpoint(t *testing.T) {
	f := newDeleteFixture(t)

	if _, err := f.svc.DeleteMedia(context.Background(), ownerDeleteRequest()); err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}

	if st, err := os.Stat(f.dbPath + ".wal"); err == nil && st.Size() > 0 {
		t.Errorf("WAL non flushé (%d octets) juste après la réponse : la suppression "+
			"n'est pas durable — utiliser LeasedWriter.CommitWithCheckpoint, pas tx.Commit", st.Size())
	}
}

// TestMediaService_DeleteMedia_ForeignRequesterRefused : un utilisateur
// authentifié qui n'est ni propriétaire ni admin ne détruit RIEN.
//
// C'est le cas que le middleware d'ownership ne couvre pas : un co-membre de
// groupe passe RequirePlayerOwnership (authz.CanAccessPlayer) et arriverait
// jusqu'ici sans cette règle.
func TestMediaService_DeleteMedia_ForeignRequesterRefused(t *testing.T) {
	f := newDeleteFixture(t)

	_, err := f.svc.DeleteMedia(context.Background(), domain.MediaDeleteRequest{
		FilePath:      "JGtm/clip.mp4",
		RequesterSlug: "Chocoboflor",
		AuthEnforced:  true,
	})
	if err == nil {
		t.Fatal("suppression acceptée pour un non-propriétaire")
	}
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "forbidden" {
		t.Fatalf("erreur = %v, want APIError code=forbidden", err)
	}

	// Aucun effet de bord : ni sur le disque, ni en base.
	if _, statErr := os.Stat(f.clipAbs); statErr != nil {
		t.Errorf("fichier supprimé malgré le refus: %v", statErr)
	}
	var status string
	if err := f.db.SQLDb().QueryRow(
		`SELECT COALESCE(status,'') FROM media_files WHERE file_path = ?`, "JGtm/clip.mp4",
	).Scan(&status); err != nil {
		t.Fatalf("relecture status: %v", err)
	}
	if status == domain.MediaStatusDeleted {
		t.Error("média masqué malgré le refus d'autorisation")
	}
}

// TestMediaService_DeleteMedia_AdminAllowed : l'admin supprime le média d'un
// autre joueur (décision utilisateur : propriétaire + admin).
func TestMediaService_DeleteMedia_AdminAllowed(t *testing.T) {
	f := newDeleteFixture(t)

	resp, err := f.svc.DeleteMedia(context.Background(), domain.MediaDeleteRequest{
		FilePath:         "JGtm/clip.mp4",
		RequesterSlug:    "Chocoboflor",
		RequesterIsAdmin: true,
		AuthEnforced:     true,
	})
	if err != nil {
		t.Fatalf("DeleteMedia (admin): %v", err)
	}
	if !resp.Deleted {
		t.Error("admin: Deleted=false")
	}
}

// TestMediaService_DeleteMedia_UnknownMedia : chemin inconnu → not_found, et
// aucune écriture n'est tentée.
func TestMediaService_DeleteMedia_UnknownMedia(t *testing.T) {
	f := newDeleteFixture(t)

	_, err := f.svc.DeleteMedia(context.Background(), domain.MediaDeleteRequest{
		FilePath:      "JGtm/inconnu.mp4",
		RequesterSlug: "JGtm",
		AuthEnforced:  true,
	})
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "not_found" {
		t.Fatalf("erreur = %v, want APIError code=not_found", err)
	}
}

// TestMediaService_DeleteMedia_NotRepeatable : une suppression déjà effectuée
// répond not_found — le média n'est plus visible, donc plus supprimable.
func TestMediaService_DeleteMedia_NotRepeatable(t *testing.T) {
	f := newDeleteFixture(t)
	ctx := context.Background()

	if _, err := f.svc.DeleteMedia(ctx, ownerDeleteRequest()); err != nil {
		t.Fatalf("1re suppression: %v", err)
	}
	_, err := f.svc.DeleteMedia(ctx, ownerDeleteRequest())
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "not_found" {
		t.Fatalf("2e suppression: erreur = %v, want APIError code=not_found", err)
	}
}

// TestMediaService_DeleteMedia_LikeOnDeletedMediaFails : après suppression, un
// like sur le média ne doit produire AUCUN event dans media_likes_history —
// cette table étant append-only, un event écrit par erreur serait définitif.
func TestMediaService_DeleteMedia_LikeOnDeletedMediaFails(t *testing.T) {
	f := newDeleteFixture(t)
	ctx := context.Background()

	if _, err := f.svc.DeleteMedia(ctx, ownerDeleteRequest()); err != nil {
		t.Fatalf("DeleteMedia: %v", err)
	}

	_, err := f.svc.SetMediaLike(ctx, domain.MediaLikeRequest{
		FilePath:  "JGtm/clip.mp4",
		LikerSlug: "Madina97294",
		Liked:     true,
	})
	if err == nil {
		t.Fatal("like accepté sur un média supprimé")
	}

	var events int
	if err := f.db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes_history WHERE media_path = ?`, "JGtm/clip.mp4",
	).Scan(&events); err != nil {
		t.Fatalf("relecture media_likes_history: %v", err)
	}
	if events != 1 {
		t.Errorf("media_likes_history = %d event(s), want 1 : un like sur média supprimé "+
			"a écrit un event IRRÉVERSIBLE dans une table append-only", events)
	}
}
