package migrations

// steps_shared_social_media_files_drop_liked.go — retrait DÉFINITIF des colonnes
// media_files.liked / liked_at de shared_social.duckdb (2026-08-04).
//
// **Pourquoi** : `liked` était un booléen GLOBAL par média — le like de n'importe
// quel joueur allumait le cœur de TOUT LE MONDE, son unlike l'éteignait partout.
// Depuis le passage du like au par-viewer, l'état vit dans media_likes_history
// (append-only, une ligne par liker) et se lit via media_likes_latest. Les deux
// colonnes n'étaient plus ni lues ni écrites : du résidu de schéma.
//
// **Le piège DuckDB** : `ALTER TABLE … DROP COLUMN` échoue avec
// « Cannot drop this column: an index depends on a column after it! » dès qu'un
// index porte sur une colonne d'ordinal SUPÉRIEUR à celle qu'on retire. Sur
// media_files c'est systématiquement le cas — liked/liked_at précèdent
// created_at (idx_mf_created) et file_stem (idx_mf_player_stem). Un DROP nu
// aurait donc échoué EN PROD au premier boot. Séquence obligatoire :
//
//	relever les index → les DROP → DROP COLUMN → les recréer à l'identique
//
// Les index sont relevés depuis duckdb_indexes() (colonne `sql` = le CREATE
// INDEX exact), pas depuis une liste codée en dur : une DB live peut porter un
// index qu'aucun step ne crée plus, et le perdre en silence serait une
// régression de perf invisible. Zéro perte de ligne (DROP COLUMN est une
// opération de catalogue) — vérifié par le compteur de rows.
//
// **ART** : aucune surface. Un ALTER n'est pas une mutation de ligne ; les index
// recréés le sont sur leurs colonnes d'origine (player_slug, created_at,
// player_slug+file_stem — toutes NON mutées, cf. media_files_drop_filepath_unique_v1
// qui a déjà éradiqué idx_mf_kind pour cette raison).
//
// **L'autre moitié du retrait** vit dans internal/ops/media_store.go : sans elle,
// ensureMediaTables re-créerait la colonne au prochain IndexMedia (ADD COLUMN IF
// NOT EXISTS sur toute colonne absente, à chaque scan). Garde-rail :
// TestEnsureMediaTables_DoesNotResurrectLikedColumns.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/migration"
)

// mediaFilesDroppedLikeColumns : les colonnes retirées par ce step.
var mediaFilesDroppedLikeColumns = []string{"liked", "liked_at"}

// mediaFilesDropLikedStep retourne le step de retrait, consommé par
// sharedSocialSteps(). Nommé dans migration.canonicalOrder.
func mediaFilesDropLikedStep() migration.Migration {
	return migration.Migration{
		Name:     "drop_media_files_liked_columns_v1",
		TargetDB: migration.TargetSharedSocial,
		Description: "Retire media_files.liked / liked_at (booléen de like GLOBAL, remplacé le 2026-08-04 par " +
			"media_likes_history + vue media_likes_latest, par liker) — démonte et remonte les index de media_files.",
		ApplySchema: applyDropMediaFilesLikedColumns,
	}
}

// mediaFilesIndex porte un index relevé sur media_files, avec le DDL qui le recrée.
type mediaFilesIndex struct{ name, ddl string }

func applyDropMediaFilesLikedColumns(db *sql.DB) error {
	ctx := migration.BootCtx()

	// Garde table : le IF EXISTS de DuckDB porte sur la COLONNE, pas sur la table —
	// un ALTER sur une table absente est une Catalog Error.
	hasTable, err := migration.TableExists(db, "media_files")
	if err != nil {
		return fmt.Errorf("drop_media_files_liked: check table: %w", err)
	}
	if !hasTable {
		return nil
	}

	// Idempotence : plus aucune des deux colonnes → no-op complet. Évite surtout de
	// démonter/remonter les index pour rien à chaque boot.
	remaining, err := presentLikeColumns(db)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return nil
	}

	indexes, err := loadMediaFilesIndexes(ctx, db)
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS `+idx.name); err != nil {
			return fmt.Errorf("drop_media_files_liked: drop index %s: %w", idx.name, err)
		}
	}

	for _, col := range mediaFilesDroppedLikeColumns {
		if _, err := db.ExecContext(ctx,
			`ALTER TABLE media_files DROP COLUMN IF EXISTS `+col); err != nil {
			return fmt.Errorf("drop_media_files_liked: drop column %s: %w", col, err)
		}
	}

	// Remontage : un index dont le DDL référence une colonne retirée ne peut pas
	// être recréé — il n'en existe aucun (aucun step n'indexe liked), mais si une DB
	// live en portait un, le perdre est le comportement voulu ET doit être TRACÉ.
	for _, idx := range indexes {
		if ddlReferencesDroppedColumn(idx.ddl) {
			slog.WarnContext(ctx, "drop_media_files_liked: index non recréé (portait une colonne retirée)",
				"index", idx.name, "ddl", idx.ddl)
			continue
		}
		if _, err := db.ExecContext(ctx, idx.ddl); err != nil {
			return fmt.Errorf("drop_media_files_liked: recreate index %s: %w", idx.name, err)
		}
	}

	slog.InfoContext(ctx, "drop_media_files_liked: colonnes de like globales retirées de media_files",
		"columns", strings.Join(remaining, ","), "indexes_rebuilt", len(indexes))
	return nil
}

// presentLikeColumns retourne celles des colonnes cibles encore présentes.
func presentLikeColumns(db *sql.DB) ([]string, error) {
	var present []string
	for _, col := range mediaFilesDroppedLikeColumns {
		has, err := migration.ColumnExists(db, "media_files", col)
		if err != nil {
			return nil, fmt.Errorf("drop_media_files_liked: check colonne %s: %w", col, err)
		}
		if has {
			present = append(present, col)
		}
	}
	return present, nil
}

// loadMediaFilesIndexes relève les index de media_files et leur DDL de recréation.
func loadMediaFilesIndexes(ctx context.Context, db *sql.DB) ([]mediaFilesIndex, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT index_name, sql FROM duckdb_indexes() WHERE table_name = 'media_files'`)
	if err != nil {
		return nil, fmt.Errorf("drop_media_files_liked: liste des index: %w", err)
	}
	defer rows.Close()

	var out []mediaFilesIndex
	for rows.Next() {
		var name string
		var ddl sql.NullString
		if err := rows.Scan(&name, &ddl); err != nil {
			return nil, fmt.Errorf("drop_media_files_liked: scan index: %w", err)
		}
		if !ddl.Valid || strings.TrimSpace(ddl.String) == "" {
			// Sans DDL on ne saurait pas le recréer : refuser plutôt que droper à l'aveugle.
			return nil, fmt.Errorf("drop_media_files_liked: index %s sans DDL dans duckdb_indexes()", name)
		}
		out = append(out, mediaFilesIndex{name: name, ddl: ddl.String})
	}
	return out, rows.Err()
}

// ddlReferencesDroppedColumn indique si un CREATE INDEX porte sur une colonne retirée.
// Comparaison bornée par des non-caractères de mot pour ne pas confondre `liked`
// avec `liked_at` ni avec un préfixe d'un autre identifiant.
func ddlReferencesDroppedColumn(ddl string) bool {
	lowered := strings.ToLower(ddl)
	open := strings.Index(lowered, "(")
	if open < 0 {
		return false
	}
	isWordRune := func(r rune) bool {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
	}
	for _, field := range strings.FieldsFunc(lowered[open:], func(r rune) bool {
		return !isWordRune(r)
	}) {
		for _, col := range mediaFilesDroppedLikeColumns {
			if field == col {
				return true
			}
		}
	}
	return false
}
