// Package duckdb — media_repo_q37_pipeline.go : pipeline Q37 média sans
// requête cross-DB SQL. Charge les médias depuis SharedSocial, enrichit avec
// match_registry via SharedReader, puis applique filtres + dédup + tri +
// pagination côté Go. Cf. ADR 0016 + audit P0.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// mediaCandidateRow est une ligne brute (media_files × media_match_associations)
// chargée depuis SharedSocial — avant enrichissement match_registry.
//
// Un même media (file_path) peut apparaître plusieurs fois si plusieurs
// associations match existent (capture pendant une session de plusieurs
// matchs proches). Le pipeline post-load dédoublonne par file_path (cf.
// dedupCandidatesByFilePath qui reproduit le QUALIFY ROW_NUMBER de Q37).
type mediaCandidateRow struct {
	FilePath        string
	FileName        string
	Kind            string
	ThumbnailPath   *string
	CaptureStartUTC *time.Time // utilisé pour le tri secondaire vs match_registry.start_time
	CaptureEndUTC   *time.Time
	MTime           *time.Time
	IndexedAt       *time.Time
	Liked           bool
	MatchID         *string // nil si aucune assoc
	PlayerSlug      *string // nil en schéma legacy (pas de player_slug sur media_files)
}

// effectiveCaptureTime retourne la meilleure approximation du timestamp de
// capture (cf. timeOrderExpr historique : capture_start_utc → capture_end_utc
// → mtime → indexed_at).
func (c mediaCandidateRow) effectiveCaptureTime() *time.Time {
	if c.CaptureStartUTC != nil {
		return c.CaptureStartUTC
	}
	if c.CaptureEndUTC != nil {
		return c.CaptureEndUTC
	}
	if c.MTime != nil {
		return c.MTime
	}
	return c.IndexedAt
}

// captureForMatchDelta retourne capture_end_utc si dispo sinon capture_start_utc.
// Utilisé par la priorisation QUALIFY (delta vers match.start_time).
func (c mediaCandidateRow) captureForMatchDelta() *time.Time {
	if c.CaptureEndUTC != nil {
		return c.CaptureEndUTC
	}
	return c.CaptureStartUTC
}

// loadMediaCandidates exécute la phase A sur SharedSocial : SELECT media_files
// LEFT JOIN media_match_associations avec uniquement les filtres LOCAUX
// (kind, liked, date, player_slug, AuthorSlugs, UnassignedOnly).
//
// Les filtres cross-DB (map/mode/playlist) sont appliqués post-load en Go
// après chargement de match_registry via SharedReader.
func (r *MediaRepo) loadMediaCandidates(
	ctx context.Context,
	f domain.MediaFilters,
) ([]mediaCandidateRow, error) {
	// media_files réside dans shared_social.duckdb depuis drop_media_from_player_db.
	// Si SharedSocial est nil (échec d'ouverture transitoire au démarrage du pool),
	// on retourne vide plutôt que de propager un crash via le fallback Player DB.
	if r.pdb.SharedSocial == nil {
		return nil, nil
	}
	// Garde Gamertag vide. mediaQueryConfig.playerSlug porte TOUT le scoping
	// d'ownership de la galerie (baseWhereClause : « mine » → mf.player_slug = ?,
	// « teammate » → mf.player_slug <> ?). Un Gamertag vide ne dégrade pas le
	// résultat, il l'INVERSE : « mine » ne remonte rien et « teammate » remonte
	// TOUT, y compris les médias du joueur courant. Le repli « config vide » qui
	// neutralisait ce cas a disparu avec la branche SQL legacy (2026-08-03) →
	// refus explicite et tracé, jamais un scoping silencieusement faux.
	if r.pdb.Gamertag == "" {
		slog.ErrorContext(ctx, "loadMediaCandidates: Gamertag vide — scoping ownership des médias impossible",
			"titleSlug", r.pdb.TitleSlug)
		return nil, errors.New("loadMediaCandidates: gamertag du PlayerDB vide — scoping ownership impossible")
	}
	cfg := r.queryConfig()
	q, args := buildMediaCandidatesQuery(f, cfg)

	rows, err := r.socialDB().QueryRecovered(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("loadMediaCandidates: %w", err)
	}
	defer rows.Close()

	var out []mediaCandidateRow
	for rows.Next() {
		var row mediaCandidateRow
		var thumbnail, matchID, playerSlug sql.NullString
		var captureStart, captureEnd, mtime, indexedAt sql.NullTime
		if err := rows.Scan(
			&row.FilePath, &row.FileName, &row.Kind, &thumbnail,
			&captureStart, &captureEnd, &mtime, &indexedAt,
			&row.Liked, &matchID, &playerSlug,
		); err != nil {
			return nil, fmt.Errorf("loadMediaCandidates: scan: %w", err)
		}
		if thumbnail.Valid {
			s := thumbnail.String
			row.ThumbnailPath = &s
		}
		if captureStart.Valid {
			t := captureStart.Time
			row.CaptureStartUTC = &t
		}
		if captureEnd.Valid {
			t := captureEnd.Time
			row.CaptureEndUTC = &t
		}
		if mtime.Valid {
			t := mtime.Time
			row.MTime = &t
		}
		if indexedAt.Valid {
			t := indexedAt.Time
			row.IndexedAt = &t
		}
		if matchID.Valid && matchID.String != "" {
			s := matchID.String
			row.MatchID = &s
		}
		if playerSlug.Valid && playerSlug.String != "" {
			s := playerSlug.String
			row.PlayerSlug = &s
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// buildMediaCandidatesQuery construit la query SQL phase A. Pas de jointure
// shared.match_registry : tout ce qui dépend de match_registry sera appliqué
// post-load en Go.
func buildMediaCandidatesQuery(f domain.MediaFilters, cfg mediaQueryConfig) (string, []any) {
	whereParts, args := buildMediaLocalWhere(f, cfg)
	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	// Schéma shared_social INCONDITIONNEL : le seul appelant (loadMediaCandidates)
	// retourne vide si pdb.SharedSocial == nil, et la query s'exécute sur socialDB().
	// La variante « legacy » qui joignait sur mma.media_path a été supprimée le
	// 2026-08-03 (règle CLAUDE.md n°7) : elle était inatteignable ET cassée — cette
	// colonne n'existe pas dans shared_social (media_match_associations y porte
	// media_file_id), donc l'atteindre aurait produit un Binder Error, pas un repli.
	const from = mediaCandidatesSharedSocialFromClause

	// match_start_time est résolu via match_registry (SharedReader) en phase B,
	// pas depuis mma.match_start_time (colonne dénormalisée du défunt schéma legacy).
	q := `SELECT
    mf.file_path,
    mf.file_name,
    mf.kind,
    mf.thumbnail_path,
    mf.capture_start_utc,
    mf.capture_end_utc,
    mf.mtime,
    mf.indexed_at AS indexed_at,
    COALESCE(mf.liked, FALSE) AS liked,
    mma.match_id,
    mf.player_slug AS player_slug
` + from + `
` + whereClause

	return q, args
}

// Phase A : aucune référence à match_registry. La jointure cross-DB est
// faite côté Go via loadMediaMatchRegistry. Lecture via la vue _latest
// (media_match_associations est append-only — règle ART n°2).
const mediaCandidatesSharedSocialFromClause = `FROM media_files mf
LEFT JOIN media_match_associations_latest mma ON mf.id = mma.media_file_id`

// buildMediaLocalWhere construit la clause WHERE avec UNIQUEMENT les filtres
// qui ne nécessitent pas match_registry (kind, liked, date, player_slug,
// AuthorSlugs, UnassignedOnly).
//
// Les filtres MapFilter / ModeFilter / PlaylistFilter sont appliqués post-load
// en Go par applyCrossDBMediaFilters.
func buildMediaLocalWhere(f domain.MediaFilters, cfg mediaQueryConfig) ([]string, []any) {
	var where []string
	var args []any

	// Médias supprimés (item 3.1) : exclus de TOUTE liste, quels que soient les
	// autres filtres. Posé en premier pour qu'aucune branche ci-dessous ne puisse
	// l'oublier — la galerie est la surface principale d'un média supprimé.
	where = append(where, MediaVisiblePredicate("mf"))

	if len(f.AuthorSlugs) > 0 {
		placeholders := make([]string, len(f.AuthorSlugs))
		for i, slug := range f.AuthorSlugs {
			placeholders[i] = "?"
			args = append(args, slug)
		}
		where = append(where, "mf.player_slug IN ("+strings.Join(placeholders, ",")+")")
	} else {
		baseWhere, baseArgs := cfg.baseWhereClause(f.SectionFilter)
		if len(baseWhere) > 0 {
			where = append(where, baseWhere...)
			args = append(args, baseArgs...)
		}
	}

	if f.KindFilter != "" {
		equivalents := mediaKindEquivalents(f.KindFilter)
		placeholders := make([]string, len(equivalents))
		for i, eq := range equivalents {
			placeholders[i] = "?"
			args = append(args, eq)
		}
		where = append(where, "mf.kind IN ("+strings.Join(placeholders, ",")+")")
	}
	if f.LikedOnly {
		where = append(where, "COALESCE(mf.liked, FALSE) = TRUE")
	}
	if f.UnassignedOnly {
		where = append(where, "mma.match_id IS NULL")
	}
	if len(where) == 0 {
		where = append(where, "TRUE")
	}
	return where, args
}

// mediaEnrichedRow est une candidate row enrichie de son match_registry.
// Match peut être nil si la candidate row n'a pas d'assoc ou si le match_id
// n'existe pas dans match_registry (orphelin).
