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

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
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
			-- file_path NON UNIQUE : file_path est MUTÉE par 3 UPDATE (conversion/HLS/
			-- reconcile) → une contrainte UNIQUE (index ART) sur une colonne mutée
			-- déclenche le bug DuckDB #23046 (FATAL invalidated, blast MAX shared_social).
			-- La dédup file_path passe en applicatif (insertMediaFile SELECT-then-INSERT).
			-- Migration media_files_drop_filepath_unique_v1 retire l'UNIQUE des DB existantes.
			file_path VARCHAR,
			file_name VARCHAR,
			file_hash VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			status VARCHAR,
			mtime TIMESTAMPTZ,
			-- RETIRÉES le 2026-08-04 : liked / liked_at. C'était un booléen GLOBAL par
			-- média (le like de n'importe quel joueur allumait le cœur de TOUT LE MONDE).
			-- L'état d'un like vit dans media_likes_history (append-only, une ligne par
			-- liker) et se lit via media_likes_latest. Les colonnes sont droppées des DB
			-- existantes par la migration drop_media_files_liked_columns_v1
			-- (games/halo_infinite/migrations/steps_shared_social_media_files_drop_liked.go).
			-- NE PAS les
			-- réintroduire ici NI dans la liste ADD COLUMN ci-dessous : ce CREATE et cet
			-- ALTER tournent à CHAQUE IndexMedia et re-créeraient la colonne droppée
			-- (garde-rail : TestEnsureMediaTables_DoesNotResurrectLikedColumns).
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
		// PAS de liked / liked_at ici : droppées le 2026-08-04 (cf. CREATE ci-dessus).
		{"discord_notified", "BOOLEAN DEFAULT FALSE"},
		{"file_stem", colTypeVarchar},
		{"file_ext", colTypeVarchar},
		// HLS (transcoding à l'ingestion) : hls_path = pointeur {slug}/hls/{stem}/master.m3u8
		// (NULL = média servi en direct, non transcodé) ; transcode_status =
		// processing|ready|failed|direct (NULL = pas encore évalué). Colonnes dédiées :
		// ne PAS réutiliser `status` (sémantique 'active' déjà filtrée par le rail home).
		{"hls_path", colTypeVarchar},
		{"transcode_status", colTypeVarchar},
		// transcode_started_at : horodatage (UTC) du passage à 'processing', posé par
		// MarkTranscodeProcessing (upload ET balayage). Sert la récupération d'orphelins
		// de crash — un 'processing' plus vieux que transcodeStaleAfter redevient
		// éligible au balayage (media_hls_sweep.go).
		{"transcode_started_at", colTypeTimestampTZ},
	} {
		if _, err := db.ExecContext(ctx, "ALTER TABLE media_files ADD COLUMN IF NOT EXISTS "+col.name+" "+col.typ); err != nil {
			return fmt.Errorf("ensureMediaTables: ajout colonne %s: %w", col.name, err)
		}
	}
	if _, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR,
			delta_seconds INTEGER,
			PRIMARY KEY (media_file_id, match_id)
		)
	`); err != nil {
		return err
	}
	// Substrat APPEND-ONLY (media_match_associations_history + vue _latest) — réconcilié
	// ici car ensureMediaTables peut gagner la course aux migrations au 1er IndexMedia sur
	// une DB fraîche : on garantit que _history + la vue existent TOUJOURS. La table legacy
	// ci-dessus est conservée vide (vestige) ; tous les writers/readers passent par
	// _history/_latest. Cf. migration shared_social_media_assoc_append_only_v1.
	_, err = db.ExecContext(ctx, `
		CREATE SEQUENCE IF NOT EXISTS media_match_associations_history_id_seq START 1;
		CREATE TABLE IF NOT EXISTS media_match_associations_history (
			id            BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
			media_file_id BIGINT NOT NULL,
			match_id      VARCHAR NOT NULL,
			delta_seconds INTEGER,
			is_manual     BOOLEAN NOT NULL DEFAULT FALSE,
			is_active     BOOLEAN NOT NULL DEFAULT TRUE,
			associated_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			written_at    TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE INDEX IF NOT EXISTS idx_mmah_lookup ON media_match_associations_history(media_file_id, written_at DESC);
		CREATE INDEX IF NOT EXISTS idx_mmah_match  ON media_match_associations_history(match_id);
		CREATE OR REPLACE VIEW media_match_associations_latest AS
			WITH lpp AS (
				SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
					ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
				FROM media_match_associations_history
			),
			act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
			hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
			SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
			FROM act a JOIN hm ON hm.media_file_id = a.media_file_id
			WHERE a.is_manual = hm.has_manual;
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

// loadKnownHashes liste les empreintes déjà indexées, pour ignorer un fichier
// dont le contenu est connu.
//
// Les médias SUPPRIMÉS (item 3.1) en sont exclus : re-déposer le fichier dans le
// dossier captures est le geste par lequel l'utilisateur annule sa suppression.
// Garder leur hash ici le rendrait définitivement non ré-indexable, donc
// invisible pour toujours malgré un ré-upload explicite (insertMediaFile
// ressuscite alors la ligne au lieu d'en créer une seconde).
func loadKnownHashes(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT file_hash FROM media_files WHERE file_hash IS NOT NULL AND "+
			platform_duckdb.MediaVisiblePredicate(""))
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

// HashFile calcule le hash de contenu d'un fichier sur disque (sha256 tronqué à
// 16 hex, lu en streaming). Doit produire la même valeur que HashBytes pour un
// contenu identique — c'est la clé de dédup utilisée par l'indexation.
func HashFile(path string) (string, error) {
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

// HashBytes calcule le même hash de contenu que HashFile, mais depuis un buffer
// en mémoire (cas upload : les octets sont déjà chargés). Permet de détecter un
// ré-upload avant l'écriture disque.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])[:16]
}

// insertMediaFile insère ou met à jour une ligne media_files.
//
// path est le chemin absolu sur disque (requis pour ffprobe + Stat des
// fichiers existants en cas de stem conflict). Le path ECRIT en DB est
// dérivé via store.ToRel : format stable {owner_slug}/{rel_in_owner_dir}.
// Si le store est en mode legacy (CapturesBase vide ou conversion échoue),
// le path absolu est stocké tel quel — comportement pré-refactor.
