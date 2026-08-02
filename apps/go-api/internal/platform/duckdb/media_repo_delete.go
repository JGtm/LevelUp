// Package duckdb — media_repo_delete.go : suppression définitive d'un média
// (v7.3 lot 2, item 3.1).
//
// Ce fichier porte les DEUX faces de la suppression côté base :
//   - l'écriture : MarkMediaDeleted (soft-delete `status='deleted'`) ;
//   - la lecture : MediaVisiblePredicate, le fragment SQL que TOUTE lecture de
//     media_files doit porter pour ne jamais resservir un média supprimé.
//
// Le choix soft-delete plutôt que DELETE, et le « zéro écriture » sur les tables
// append-only de likes/associations, sont justifiés en tête de
// internal/domain/media_delete.go (risque ART, ADR 0022/0026).
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MediaVisiblePredicate retourne le fragment SQL excluant les médias supprimés,
// pour l'alias de table donné (vide = colonnes non qualifiées).
//
// UNIQUE source de ce prédicat (règle CLAUDE.md n°6 : à la 3e copie on
// centralise + garde-rail). Le garde-rail associé est
// TestAllMediaFilesReadsFilterDeleted (media_repo_delete_guard_test.go) : il
// refuse tout nouveau `FROM media_files` qui ne porterait pas ce filtre — sans
// lui, un média supprimé réapparaîtrait sur la surface oubliée (galerie, rail
// home, onglet match, auteurs, candidats d'association).
//
// COALESCE indispensable : la colonne `status` vaut NULL sur la majorité des
// lignes historiques et 'active' sur celles écrites par le rail home ; un
// `status <> 'deleted'` nu éliminerait silencieusement toutes les lignes NULL.
func MediaVisiblePredicate(alias string) string {
	col := "status"
	if alias != "" {
		col = alias + ".status"
	}
	return "COALESCE(" + col + ", '') <> '" + domain.MediaStatusDeleted + "'"
}

// LoadMediaForDeletion charge le propriétaire et les chemins physiques d'un
// média ENCORE VISIBLE. Retourne domain.ErrNotFound si le chemin est inconnu ou
// si le média est déjà supprimé — une suppression n'est pas rejouable, et c'est
// ce qui produit le 404 attendu côté HTTP.
func (r *MediaRepo) LoadMediaForDeletion(ctx context.Context, filePath string) (domain.MediaDeletionTarget, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if r.pdb.SharedSocial == nil {
		return domain.MediaDeletionTarget{}, domain.ErrNotFound("media", filePath)
	}

	var target domain.MediaDeletionTarget
	var ownerSlug, thumbnail sql.NullString
	err := r.socialDB().QueryRow(ctx, `
		SELECT id, player_slug, file_path, thumbnail_path
		FROM media_files
		WHERE file_path = ? AND `+MediaVisiblePredicate("")+`
		LIMIT 1
	`, filePath).Scan(&target.ID, &ownerSlug, &target.FilePath, &thumbnail)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.MediaDeletionTarget{}, domain.ErrNotFound("media", filePath)
	}
	if err != nil {
		return domain.MediaDeletionTarget{}, fmt.Errorf("LoadMediaForDeletion: %w", err)
	}
	if ownerSlug.Valid {
		target.OwnerSlug = ownerSlug.String
	}
	if thumbnail.Valid {
		target.ThumbnailPath = thumbnail.String
	}
	target.HLSPath = r.loadMediaHLSPath(ctx, target.ID)
	return target, nil
}

// loadMediaHLSPath lit hls_path en requête SÉPARÉE et TOLÉRANTE.
//
// La colonne n'est pas créée par la migration de schéma mais par l'ALTER runtime
// d'ensureMediaTables (ops/media_store.go) : elle est absente des bases qui
// n'ont jamais indexé de média (dont les DuckDB :memory: des tests). La
// référencer dans le SELECT principal ferait échouer la suppression entière sur
// ces bases. Absence de colonne ⇒ pas de HLS à retirer : chaîne vide, tracée.
func (r *MediaRepo) loadMediaHLSPath(ctx context.Context, mediaID int64) string {
	var hls sql.NullString
	if err := r.socialDB().QueryRow(ctx,
		`SELECT hls_path FROM media_files WHERE id = ? LIMIT 1`, mediaID,
	).Scan(&hls); err != nil {
		slog.DebugContext(ctx, "media_delete: hls_path indisponible",
			"err", err, "media_id", mediaID)
		return ""
	}
	if !hls.Valid {
		return ""
	}
	return hls.String
}

// MarkMediaDeleted masque le média : UPDATE de la seule colonne `status`, qui
// n'appartient à AUCUN index de media_files.
//
// NE JAMAIS remplacer par un `DELETE FROM media_files` : la table porte 4 index
// ART (PK id + idx_mf_player_slug + idx_mf_created + idx_mf_player_stem) et un
// DELETE simple sur une PK ART a déjà FATAL-invalidé une base de ce repo en
// production (incident catalog_fetch_queue du 2026-06-19, documenté dans
// internal/sync/no_art_patterns_test.go). Sur shared_social, une invalidation
// emporterait médias, likes, followers et activité pour tout le process.
//
// exec est la transaction du LeasedWriter appelant : le commit et son CHECKPOINT
// sont la responsabilité du service (cf. media_service_delete.go).
func (r *MediaRepo) MarkMediaDeleted(ctx context.Context, exec port.DBExecutor, filePath string) (bool, error) {
	res, err := exec.ExecContext(ctx, `
		UPDATE media_files
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE file_path = ? AND `+MediaVisiblePredicate(""), domain.MediaStatusDeleted, filePath)
	if err != nil {
		return false, fmt.Errorf("MarkMediaDeleted: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("MarkMediaDeleted rows affected: %w", err)
	}
	return n > 0, nil
}
