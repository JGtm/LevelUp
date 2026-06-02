// Package ops — media_store.go : plomberie DB+FS de l'indexation (schéma
// media_files idempotent, scan disque, hash). Extrait de media.go (refactor god-file).
package ops

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ensureMediaTables(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE SEQUENCE IF NOT EXISTS media_files_id_seq START 1`); err != nil {
		return err
	}

	// Si la table existe avec l'ancien schéma (id VARCHAR, issu de create_base_player_schema)
	// et qu'elle est vide, on la supprime pour la recréer correctement.
	if err := dropLegacyMediaFilesIfNeeded(ctx, db); err != nil {
		return err
	}

	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS media_files (
			id INTEGER PRIMARY KEY DEFAULT nextval('media_files_id_seq'),
			player_slug VARCHAR,
			file_path VARCHAR UNIQUE,
			file_name VARCHAR,
			file_hash VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			status VARCHAR,
			mtime TIMESTAMPTZ,
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMPTZ,
			discord_notified BOOLEAN DEFAULT FALSE,
			indexed_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	// Migration idempotente : ajoute les colonnes absentes dans les DBs créées
	// par d'anciennes migrations qui n'avaient pas capture_start_utc.
	// ADD COLUMN IF NOT EXISTS évite d'avorter la connexion.
	for _, col := range []struct{ name, typ string }{
		{"capture_start_utc", colTypeTimestampTZ},
		{"capture_end_utc", colTypeTimestampTZ},
		{"file_hash", colTypeVarchar},
		{"kind", colTypeVarchar},
		{"thumbnail_path", colTypeVarchar},
		{"player_slug", colTypeVarchar},
		{"duration_seconds", "DOUBLE"},
		{"status", colTypeVarchar},
		{"mtime", colTypeTimestampTZ},
		{"indexed_at", "TIMESTAMPTZ DEFAULT NOW()"},
		{"liked", "BOOLEAN DEFAULT FALSE"},
		{"liked_at", colTypeTimestampTZ},
		{"discord_notified", "BOOLEAN DEFAULT FALSE"},
		{"file_stem", colTypeVarchar},
		{"file_ext", colTypeVarchar},
		// HLS (transcoding à l'ingestion) : hls_path = pointeur {slug}/hls/{stem}/master.m3u8
		// (NULL = média servi en direct, non transcodé) ; transcode_status =
		// processing|ready|failed (NULL = pas de transcodage). Colonnes dédiées :
		// ne PAS réutiliser `status` (sémantique 'active' déjà filtrée par le rail home).
		{"hls_path", colTypeVarchar},
		{"transcode_status", colTypeVarchar},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE media_files ADD COLUMN IF NOT EXISTS "+col.name+" "+col.typ); err != nil {
			return fmt.Errorf("ensureMediaTables: ajout colonne %s: %w", col.name, err)
		}
	}
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR,
			delta_seconds INTEGER,
			PRIMARY KEY (media_file_id, match_id)
		)
	`)
	return err
}

// dropLegacyMediaFilesIfNeeded supprime la table media_files si elle a l'ancien schéma
// (id VARCHAR, issue de create_base_player_schema) et qu'elle est vide.
// Ceci permet à ensureMediaTables de recréer la table avec le bon schéma.
func dropLegacyMediaFilesIfNeeded(ctx context.Context, db *sql.DB) error {
	// Vérifier si la colonne id est de type VARCHAR (ancien schéma).
	var dataType string
	err := db.QueryRowContext(ctx,
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'media_files' AND column_name = 'id'",
	).Scan(&dataType)
	if err != nil {
		// Table inexistante ou autre erreur → rien à faire.
		return nil //nolint:nilerr
	}
	if dataType != "VARCHAR" {
		return nil // Schéma déjà correct ou inconnu.
	}
	// Vérifier que la table est vide avant de la supprimer.
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_files").Scan(&count); err != nil || count > 0 {
		return nil // Table non-vide ou erreur : on ne touche pas aux données.
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE media_files"); err != nil {
		return fmt.Errorf("dropLegacyMediaFilesIfNeeded: DROP TABLE: %w", err)
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS media_match_associations"); err != nil {
		return fmt.Errorf("dropLegacyMediaFilesIfNeeded: DROP TABLE media_match_associations: %w", err)
	}
	return nil
}

func loadKnownHashes(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT file_hash FROM media_files WHERE file_hash IS NOT NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	known := make(map[string]bool)
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			continue
		}
		known[h] = true
	}
	return known, rows.Err()
}

func walkMediaDir(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignorer thumbs/ (miniatures) et hls/ (arbres HLS-fMP4 générés :
		// init.mp4, segments .m4s, playlists .m3u8). Sans le skip de hls/, le
		// fichier init.mp4 serait indexé comme un faux média (cf. PLAN_MEDIA_HLS
		// piège n°1).
		if info.IsDir() {
			if info.Name() == "thumbs" || info.Name() == "hls" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := supportedExtensions[ext]; ok {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

// insertMediaFile insère ou met à jour une ligne media_files.
//
// path est le chemin absolu sur disque (requis pour ffprobe + Stat des
// fichiers existants en cas de stem conflict). Le path ECRIT en DB est
// dérivé via store.ToRel : format stable {owner_slug}/{rel_in_owner_dir}.
// Si le store est en mode legacy (CapturesBase vide ou conversion échoue),
// le path absolu est stocké tel quel — comportement pré-refactor.
