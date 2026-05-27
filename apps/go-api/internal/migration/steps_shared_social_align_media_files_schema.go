package migration

// steps_shared_social_align_media_files_schema.go — ADR 0021 Bonus 12.
//
// Aligne le schéma `media_files` de la DB live (créée historiquement par
// ops.IndexMedia AVANT que les migrations Go n'existent) avec le schéma
// cible définit dans `create_base_shared_social_schema`.
//
// **Divergences détectées 27/05/2026** (cf. audit_shared_social_writes_2026-05-27.md) :
//
//	| Colonne legacy live          | Schéma migré actuel       | Action          |
//	|------------------------------|---------------------------|-----------------|
//	| `id INTEGER auto`            | `id VARCHAR PRIMARY KEY`  | TODO séparé (risque PK) |
//	| `capture_start_utc TS+TZ`    | absent                    | conserver (lecture seule) |
//	| `duration_seconds DOUBLE`    | absent                    | conserver |
//	| `mtime TS+TZ`                | `mtime TS+TZ`             | aligné |
//	| `discord_notified BOOLEAN`   | `discord_notified_at TS`  | ADD nouvelle col + backfill |
//	| `indexed_at TS+TZ`           | `created_at TS DEFAULT NOW` | ADD si manquant |
//	| (absent)                     | `updated_at TS DEFAULT NOW` | ADD si manquant |
//	| (absent)                     | `file_size INTEGER DEFAULT 0` | ADD si manquant |
//
// Stratégie : ADD COLUMN IF NOT EXISTS uniquement (jamais DROP). Backfill
// best-effort. Migration idempotente.
//
// **Hors scope** : conversion `id INTEGER → VARCHAR` (PK rename + FK refactor —
// trop risqué pour une migration auto, requires plan séparé).

import (
	"database/sql"
	"fmt"
)

func init() {
	Register(Migration{
		Name:        "align_media_files_legacy_schema",
		TargetDB:    TargetSharedSocial,
		Description: "ADR 0021 Bonus 12 : aligne media_files legacy vers schéma actuel (ADD COLUMN IF NOT EXISTS).",
		ApplySchema: func(db *sql.DB) error {
			// Détecter le schéma actuel : si `created_at` existe, le schéma est
			// déjà migré (création via create_base_shared_social_schema) → no-op.
			// Sinon, on est sur la version legacy → appliquer les ADD COLUMN.
			//
			// DuckDB v1.4 ne supporte pas `IF NOT EXISTS` sur ADD COLUMN dans tous
			// les cas — on guarde explicitement via information_schema.
			cols, err := loadMediaFilesColumns(db)
			if err != nil {
				return fmt.Errorf("load media_files columns: %w", err)
			}
			if cols == nil {
				// Table media_files n'existe pas (fresh DB) → laisser
				// create_base_shared_social_schema faire son boulot.
				return nil
			}

			addIfMissing := func(name, ddl string) error {
				if _, ok := cols[name]; ok {
					return nil
				}
				if _, err := db.Exec(`ALTER TABLE media_files ADD COLUMN ` + name + ` ` + ddl); err != nil {
					return fmt.Errorf("ADD COLUMN %s: %w", name, err)
				}
				cols[name] = struct{}{}
				return nil
			}

			if err := addIfMissing("file_size", "INTEGER DEFAULT 0"); err != nil {
				return err
			}
			if err := addIfMissing("created_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
				return err
			}
			if err := addIfMissing("updated_at", "TIMESTAMP DEFAULT CURRENT_TIMESTAMP"); err != nil {
				return err
			}
			if err := addIfMissing("discord_notified_at", "TIMESTAMP"); err != nil {
				return err
			}

			// Backfill discord_notified_at depuis discord_notified BOOLEAN si présent.
			// Pattern : mettre CURRENT_TIMESTAMP pour les rows où le bool était TRUE.
			if _, ok := cols["discord_notified"]; ok {
				if _, err := db.Exec(`
					UPDATE media_files
					SET discord_notified_at = CURRENT_TIMESTAMP
					WHERE discord_notified = TRUE AND discord_notified_at IS NULL
				`); err != nil {
					// Best-effort : ne bloque pas la migration si UPDATE échoue.
					return nil
				}
			}

			// Backfill created_at depuis indexed_at si présent (lecture seule).
			if _, ok := cols["indexed_at"]; ok {
				if _, err := db.Exec(`
					UPDATE media_files
					SET created_at = indexed_at
					WHERE created_at IS NULL AND indexed_at IS NOT NULL
				`); err != nil {
					return nil
				}
			}

			return nil
		},
	})
}

// loadMediaFilesColumns retourne l'ensemble des colonnes de media_files si
// la table existe, nil si absente.
func loadMediaFilesColumns(db *sql.DB) (map[string]struct{}, error) {
	// Vérifier que la table existe avant de lister ses colonnes.
	var tableCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'media_files'`,
	).Scan(&tableCount); err != nil {
		return nil, err
	}
	if tableCount == 0 {
		return nil, nil
	}
	rows, err := db.Query(
		`SELECT column_name FROM information_schema.columns WHERE table_name = 'media_files'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = struct{}{}
	}
	return out, rows.Err()
}
