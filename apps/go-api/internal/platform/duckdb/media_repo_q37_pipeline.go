// Package duckdb — media_repo_q37_pipeline.go : pipeline Q37 média sans
// requête cross-DB SQL. Charge les médias depuis SharedSocial, enrichit avec
// match_registry via SharedReader, puis applique filtres + dédup + tri +
// pagination côté Go. Cf. ADR 0016 + audit P0.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
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

	from := mediaCandidatesLegacyFromClause
	playerSlugExpr := "NULL"
	indexedAtExpr := "CAST(NULL AS TIMESTAMPTZ)"
	if cfg.useSharedSocialSchema() {
		from = mediaCandidatesSharedSocialFromClause
		playerSlugExpr = "mf.player_slug"
		indexedAtExpr = "mf.indexed_at"
	}

	// match_start_time est résolu via match_registry (SharedReader) en phase B,
	// pas depuis mma.match_start_time (colonne dénormalisée présente uniquement
	// dans le schéma legacy, absente du schéma shared_social).
	// indexed_at n'existe que dans shared_social — fallback NULL en legacy
	// (cf. seedPlayerSchema vs seedSharedSocialSchema).
	q := `SELECT
    mf.file_path,
    mf.file_name,
    mf.kind,
    mf.thumbnail_path,
    mf.capture_start_utc,
    mf.capture_end_utc,
    mf.mtime,
    ` + indexedAtExpr + ` AS indexed_at,
    COALESCE(mf.liked, FALSE) AS liked,
    mma.match_id,
    ` + playerSlugExpr + ` AS player_slug
` + from + `
` + whereClause

	return q, args
}

const (
	// Phase A : aucune référence à match_registry. La jointure cross-DB est
	// faite côté Go via loadMediaMatchRegistry.
	mediaCandidatesLegacyFromClause = `FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.file_path = mma.media_path`

	mediaCandidatesSharedSocialFromClause = `FROM media_files mf
LEFT JOIN media_match_associations mma ON mf.id = mma.media_file_id`
)

// buildMediaLocalWhere construit la clause WHERE avec UNIQUEMENT les filtres
// qui ne nécessitent pas match_registry (kind, liked, date, player_slug,
// AuthorSlugs, UnassignedOnly).
//
// Les filtres MapFilter / ModeFilter / PlaylistFilter sont appliqués post-load
// en Go par applyCrossDBMediaFilters.
func buildMediaLocalWhere(f domain.MediaFilters, cfg mediaQueryConfig) ([]string, []any) {
	var where []string
	var args []any

	if len(f.AuthorSlugs) > 0 && cfg.useSharedSocialSchema() {
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
type mediaEnrichedRow struct {
	Cand  mediaCandidateRow
	Match *mediaMatchRegistryInfo
}

// computedMapLabel reproduit q37MediaMapLabelExpr côté Go :
//
//	NULLIF(TRIM(regexp_replace(regexp_replace(regexp_replace(
//	  COALESCE(mr.map_name_fr, mr.map_name, ''),
//	  '\s+v\d+$', '', 'i'),
//	  '\s*-\s*Forge.*$', '', 'i'),
//	  '\s*-\s*Ranked.*$', '', 'i')), '')
func (e mediaEnrichedRow) computedMapLabel() string {
	if e.Match == nil {
		return ""
	}
	raw := e.Match.MapNameFR
	if raw == "" {
		raw = e.Match.MapName
	}
	if raw == "" {
		return ""
	}
	return normalizeMediaMapName(raw)
}

// computedModeLabel reproduit q37MediaModeLabelExpr côté Go :
//   - Si pair_name contient ":" → prefix avant ":"
//   - Sinon → strip suffixes " on .+", " - Forge*", " - Ranked*"
//
// Aligné sur normalizeModeLabel (media_repo.go).
func (e mediaEnrichedRow) computedModeLabel() string {
	if e.Match == nil {
		return ""
	}
	raw := e.Match.PairNameFR
	if raw == "" {
		raw = e.Match.PairName
	}
	if raw == "" {
		return ""
	}
	return normalizeModeLabel(raw)
}

// computedPlaylistLabel reproduit q37MediaPlaylistLabelExpr :
//
//	NULLIF(TRIM(COALESCE(mr.playlist_name_fr, mr.playlist_name, '')), '')
func (e mediaEnrichedRow) computedPlaylistLabel() string {
	if e.Match == nil {
		return ""
	}
	raw := e.Match.PlaylistNameFR
	if raw == "" {
		raw = e.Match.PlaylistName
	}
	return strings.TrimSpace(raw)
}

// enrichCandidates joint chaque candidate row avec son match_registry (si dispo).
func enrichCandidates(
	cands []mediaCandidateRow,
	registry map[string]mediaMatchRegistryInfo,
) []mediaEnrichedRow {
	out := make([]mediaEnrichedRow, len(cands))
	for i, c := range cands {
		row := mediaEnrichedRow{Cand: c}
		if c.MatchID != nil && *c.MatchID != "" {
			if info, ok := registry[*c.MatchID]; ok {
				m := info
				row.Match = &m
			}
		}
		out[i] = row
	}
	return out
}

// applyCrossDBMediaFilters reproduit les filtres MapFilter / ModeFilter /
// PlaylistFilter qui dépendent de match_registry. Une row sans Match passe
// les filtres uniquement si f.UnassignedOnly est explicite (les médias non
// associés à un match ne peuvent pas matcher un filtre map/mode/playlist).
//
// Le caller passe whereCfg pour permettre la même fonction de servir aux 3
// options queries (chaque option exclut son propre filtre pour montrer les
// alternatives disponibles).
func applyCrossDBMediaFilters(
	rows []mediaEnrichedRow,
	f domain.MediaFilters,
	whereCfg mediaWhereConfig,
) []mediaEnrichedRow {
	out := rows[:0]
	for _, row := range rows {
		if whereCfg.includeMapFilter && f.MapFilter != "" {
			if !mediaRowMatchesMap(row, f.MapFilter) {
				continue
			}
		}
		if whereCfg.includeModeFilter && f.ModeFilter != "" {
			if !mediaRowMatchesMode(row, f.ModeFilter) {
				continue
			}
		}
		if whereCfg.includePlaylistFilter && f.PlaylistFilter != "" {
			if !mediaRowMatchesPlaylist(row, f.PlaylistFilter) {
				continue
			}
		}
		out = append(out, row)
	}
	return out
}

// mediaRowMatchesMap reproduit :
//
//	mr.map_id = ? OR LOWER(map_label) = LOWER(?)
//
// MapFilter peut être un map_id (UUID stable) ou un label brut (fallback).
func mediaRowMatchesMap(row mediaEnrichedRow, filter string) bool {
	if row.Match == nil {
		return false
	}
	if row.Match.MapID != "" && row.Match.MapID == filter {
		return true
	}
	label := row.computedMapLabel()
	if label == "" {
		return false
	}
	return strings.EqualFold(label, normalizeMediaMapName(filter))
}

// mediaRowMatchesMode reproduit le filtre Mode :
//
//	2 formats acceptés :
//	 "Assassin"        → préfixes pair_name de la catégorie
//	 "Assassin/Slayer" → catégorie + sous-mode normalisé
//	 "Other"           → NOT IN les préfixes connus
//
// Cf. buildQ37MediaWhereClause historique.
func mediaRowMatchesMode(row mediaEnrichedRow, filter string) bool {
	if row.Match == nil {
		return false
	}
	pairName := row.Match.PairName
	if pairName == "" {
		return false
	}
	category, submode, hasSubmode := strings.Cut(filter, "/")
	prefixes := halo_infinite.PairNamePrefixesForCategory(category)
	pairLower := strings.ToLower(pairName)

	matchesCategory := false
	switch {
	case len(prefixes) > 0:
		for _, p := range prefixes {
			pLower := strings.ToLower(p)
			if strings.HasPrefix(pairLower, pLower+":") || pairLower == pLower {
				matchesCategory = true
				break
			}
		}
	case category == halo_infinite.ModeCategoryOther:
		// Other = NOT IN les préfixes connus
		matchesCategory = true
		for _, p := range halo_infinite.AllKnownPairNamePrefixes() {
			pLower := strings.ToLower(p)
			if strings.HasPrefix(pairLower, pLower+":") || pairLower == pLower {
				matchesCategory = false
				break
			}
		}
	default:
		// Catégorie inconnue → pas de match
		return false
	}
	if !matchesCategory {
		return false
	}
	if hasSubmode {
		submode = strings.TrimSpace(submode)
		if submode == "" {
			return true
		}
		// Sous-mode : compare le label normalisé (NormalizeModeLabel) au submode.
		modeLabel := analysis.NormalizeModeLabel(pairName)
		return strings.EqualFold(modeLabel, submode)
	}
	return true
}

// mediaRowMatchesPlaylist reproduit :
//
//	mr.playlist_id = ? OR LOWER(playlist_label) = LOWER(?)
func mediaRowMatchesPlaylist(row mediaEnrichedRow, filter string) bool {
	if row.Match == nil {
		return false
	}
	if row.Match.PlaylistID != "" && row.Match.PlaylistID == filter {
		return true
	}
	label := row.computedPlaylistLabel()
	if label == "" {
		return false
	}
	return strings.EqualFold(label, filter)
}

// dedupCandidatesByFilePath reproduit le QUALIFY ROW_NUMBER OVER (PARTITION
// BY mf.file_path ORDER BY ...) historique :
//   - prioriser les rows avec match (Match != nil ET start_time non null)
//   - parmi celles-là, la plus proche temporellement de capture_*
//   - tiebreak : match_id ASC (lexicographique)
//
// Cf. q37 historique buildQ37MediaQuery (QUALIFY ROW_NUMBER lignes 804-810).
func dedupCandidatesByFilePath(rows []mediaEnrichedRow) []mediaEnrichedRow {
	// Pour chaque file_path, calculer la "meilleure" row.
	best := make(map[string]mediaEnrichedRow, len(rows))
	for _, row := range rows {
		existing, ok := best[row.Cand.FilePath]
		if !ok {
			best[row.Cand.FilePath] = row
			continue
		}
		if betterMediaRow(row, existing) {
			best[row.Cand.FilePath] = row
		}
	}
	// Reconstruire la liste en préservant un ordre stable (tri secondaire
	// appliqué plus tard par sortEnrichedRows). On utilise file_path ASC
	// comme ordre canonique post-dédup.
	out := make([]mediaEnrichedRow, 0, len(best))
	for _, row := range best {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Cand.FilePath < out[j].Cand.FilePath
	})
	return out
}

// betterMediaRow retourne true si `candidate` est meilleur que `existing`
// selon le scoring QUALIFY historique.
func betterMediaRow(candidate, existing mediaEnrichedRow) bool {
	// Critère 1 : présence d'un match (Match non nil + start_time non null).
	candHasMatch := candidate.Match != nil && candidate.Match.effectiveStart() != nil
	exHasMatch := existing.Match != nil && existing.Match.effectiveStart() != nil
	if candHasMatch != exHasMatch {
		return candHasMatch
	}

	// Critère 2 : delta avec capture le plus petit (parmi ceux avec match).
	if candHasMatch && exHasMatch {
		candDelta := absDeltaSeconds(candidate.Match.effectiveStart(), candidate.Cand.captureForMatchDelta())
		exDelta := absDeltaSeconds(existing.Match.effectiveStart(), existing.Cand.captureForMatchDelta())
		if candDelta != exDelta {
			return candDelta < exDelta
		}
	}

	// Critère 3 : tiebreak match_id ASC.
	candID := ""
	if candidate.Cand.MatchID != nil {
		candID = *candidate.Cand.MatchID
	}
	exID := ""
	if existing.Cand.MatchID != nil {
		exID = *existing.Cand.MatchID
	}
	return candID < exID
}

// absDeltaSeconds retourne |a - b| en secondes, ou math.MaxFloat64 si l'un
// des deux est nil (placé en queue par NULLS LAST dans Q37 historique).
func absDeltaSeconds(a, b *time.Time) float64 {
	if a == nil || b == nil {
		return math.MaxFloat64
	}
	d := a.Sub(*b).Seconds()
	if d < 0 {
		d = -d
	}
	return d
}

// sortEnrichedRows applique le tri selon f.Sort + f.GroupBy. Tiebreak
// déterministe sur file_path ASC.
func sortEnrichedRows(rows []mediaEnrichedRow, f domain.MediaFilters) {
	sort.SliceStable(rows, func(i, j int) bool {
		// Groupement (préfixe de tri si non vide).
		if cmp := compareGroupOrder(rows[i], rows[j], f.GroupBy); cmp != 0 {
			return cmp < 0
		}
		// Tri principal selon Sort.
		if cmp := compareSortOrder(rows[i], rows[j], f.Sort); cmp != 0 {
			return cmp < 0
		}
		// Tiebreaker stable : file_path ASC.
		return rows[i].Cand.FilePath < rows[j].Cand.FilePath
	})
}

// compareGroupOrder reproduit groupOrderExpr historique. Retourne <0, 0, >0.
func compareGroupOrder(a, b mediaEnrichedRow, groupBy string) int {
	switch groupBy {
	case "owner":
		ao := derefStr(a.Cand.PlayerSlug)
		bo := derefStr(b.Cand.PlayerSlug)
		return strings.Compare(ao, bo)
	case "map":
		ao := coalesceLabel(a.computedMapLabel())
		bo := coalesceLabel(b.computedMapLabel())
		return strings.Compare(ao, bo)
	case "mode":
		ao := coalesceLabel(a.computedModeLabel())
		bo := coalesceLabel(b.computedModeLabel())
		return strings.Compare(ao, bo)
	case "session":
		// Tri par date DESC (alignement sessions contiguës côté frontend).
		return compareTimePtrDesc(a.Cand.effectiveCaptureTime(), b.Cand.effectiveCaptureTime())
	case "liked":
		// liked=true en premier (DESC)
		if a.Cand.Liked != b.Cand.Liked {
			if a.Cand.Liked {
				return -1
			}
			return 1
		}
		return 0
	}
	return 0
}

// compareSortOrder reproduit le orderBy principal (date / map / mode).
func compareSortOrder(a, b mediaEnrichedRow, sortKey string) int {
	switch sortKey {
	case "date_asc":
		return compareTimePtrAsc(a.Cand.effectiveCaptureTime(), b.Cand.effectiveCaptureTime())
	case "map_asc":
		ao := a.computedMapLabel()
		bo := b.computedMapLabel()
		if cmp := strings.Compare(ao, bo); cmp != 0 {
			return cmp
		}
		return compareTimePtrDesc(a.Cand.effectiveCaptureTime(), b.Cand.effectiveCaptureTime())
	case "mode_asc":
		ao := a.computedModeLabel()
		bo := b.computedModeLabel()
		if cmp := strings.Compare(ao, bo); cmp != 0 {
			return cmp
		}
		return compareTimePtrDesc(a.Cand.effectiveCaptureTime(), b.Cand.effectiveCaptureTime())
	default: // date_desc
		return compareTimePtrDesc(a.Cand.effectiveCaptureTime(), b.Cand.effectiveCaptureTime())
	}
}

// coalesceLabel reproduit COALESCE(label, '~zzz') pour pousser les NULL en fin
// de liste en ordre ASC.
func coalesceLabel(label string) string {
	if label == "" {
		return "~zzz"
	}
	return label
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func compareTimePtrAsc(a, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1 // NULL LAST
	}
	if b == nil {
		return -1
	}
	if a.Equal(*b) {
		return 0
	}
	if a.Before(*b) {
		return -1
	}
	return 1
}

func compareTimePtrDesc(a, b *time.Time) int { return -compareTimePtrAsc(a, b) }

// paginateEnrichedRows applique offset + limit.
func paginateEnrichedRows(rows []mediaEnrichedRow, limit, offset int) []mediaEnrichedRow {
	if offset >= len(rows) {
		return nil
	}
	end := offset + limit
	if end > len(rows) {
		end = len(rows)
	}
	return rows[offset:end]
}

// toMediaFileRows convertit les enriched rows en domain.MediaFileRow pour la
// signature publique. Fait l'enrich final (MapName/ModeName/PairNameRaw).
func toMediaFileRows(rows []mediaEnrichedRow) []domain.MediaFileRow {
	out := make([]domain.MediaFileRow, 0, len(rows))
	for _, row := range rows {
		dest := domain.MediaFileRow{
			FilePath:      row.Cand.FilePath,
			FileName:      row.Cand.FileName,
			Kind:          row.Cand.Kind,
			ThumbnailPath: row.Cand.ThumbnailPath,
			CaptureEndUTC: row.Cand.CaptureEndUTC,
			Liked:         row.Cand.Liked,
			MatchID:       row.Cand.MatchID,
			PlayerSlug:    row.Cand.PlayerSlug,
		}
		if row.Match != nil {
			// MapName : label calculé (TRIM + regex strip)
			if label := row.computedMapLabel(); label != "" {
				s := label
				dest.MapName = &s
			}
			// ModeName : label calculé (préfixe avant ":" ou strip suffixes)
			if label := row.computedModeLabel(); label != "" {
				s := label
				dest.ModeName = &s
			}
			// PairNameRaw : valeur brute (EN ou FR) pour InferModeCategoryFromPairName.
			pair := row.Match.PairName
			if pair == "" {
				pair = row.Match.PairNameFR
			}
			if pair != "" {
				s := pair
				dest.PairNameRaw = &s
			}
			if row.Match.MapID != "" {
				s := row.Match.MapID
				dest.MapID = &s
			}
			// MatchStartTime : utiliser start_time_utc en priorité, fallback
			// sur start_time (réputé UTC en v6+) depuis match_registry chargé
			// via SharedReader.
			if t := row.Match.effectiveStart(); t != nil {
				utc := t.UTC()
				dest.MatchStartTime = &utc
			}
		}
		out = append(out, dest)
	}
	return out
}

// runMediaPipeline orchestre les 3 phases : load candidates → load match
// registry → enrich + filter + dedup + sort + paginate.
//
// Si limit <= 0 et offset <= 0, retourne TOUS les rows (cas options).
func (r *MediaRepo) runMediaPipeline(
	ctx context.Context,
	f domain.MediaFilters,
	whereCfg mediaWhereConfig,
	limit, offset int,
) ([]mediaEnrichedRow, error) {
	cands, err := r.loadMediaCandidates(ctx, f)
	if err != nil {
		return nil, err
	}
	matchIDs := extractDistinctMatchIDs(cands)
	registry, err := r.loadMediaMatchRegistry(ctx, matchIDs)
	if err != nil {
		return nil, err
	}
	enriched := enrichCandidates(cands, registry)
	enriched = applyCrossDBMediaFilters(enriched, f, whereCfg)
	enriched = dedupCandidatesByFilePath(enriched)
	sortEnrichedRows(enriched, f)
	if limit > 0 {
		enriched = paginateEnrichedRows(enriched, limit, offset)
	}
	return enriched, nil
}

// extractDistinctMatchIDs collecte les match_id non-NULL distincts des candidates.
func extractDistinctMatchIDs(cands []mediaCandidateRow) []string {
	seen := make(map[string]struct{}, len(cands))
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		if c.MatchID == nil || *c.MatchID == "" {
			continue
		}
		if _, ok := seen[*c.MatchID]; ok {
			continue
		}
		seen[*c.MatchID] = struct{}{}
		out = append(out, *c.MatchID)
	}
	return out
}

// extractMapPairs reproduit BuildQ37MediaMapOptionsQuery historique :
// DISTINCT (map_id, label_normalisé) avec label non vide.
//
// Pour chaque enriched row, retourne (mr.map_id, computedMapLabel). Skip rows
// dont le label est vide (équivalent du WHERE label IS NOT NULL).
//
// Trié par label ASC (alignement avec ORDER BY label ASC historique).
func extractMapPairs(rows []mediaEnrichedRow) []mediaFilterOptionPair {
	type key struct{ id, label string }
	seen := make(map[key]struct{}, len(rows))
	out := make([]mediaFilterOptionPair, 0, len(rows))
	for _, row := range rows {
		if row.Match == nil {
			continue
		}
		label := row.computedMapLabel()
		if label == "" {
			continue
		}
		k := key{id: row.Match.MapID, label: label}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, mediaFilterOptionPair(k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].label < out[j].label })
	return out
}

// extractModePairs reproduit BuildQ37MediaModeOptionsQuery historique :
// DISTINCT (pair_name_raw, label_normalisé) avec label non vide.
//
// id = pair_name brut (pour normalisation FR ultérieure côté
// translateModeFilterOptions via halo_infinite.InferModeCategoryFromPairName).
// label = label normalisé (préfixe avant ":" ou strip suffixes).
func extractModePairs(rows []mediaEnrichedRow) []mediaFilterOptionPair {
	type key struct{ id, label string }
	seen := make(map[key]struct{}, len(rows))
	out := make([]mediaFilterOptionPair, 0, len(rows))
	for _, row := range rows {
		if row.Match == nil {
			continue
		}
		label := row.computedModeLabel()
		if label == "" {
			continue
		}
		// id : pair_name brut (EN ou FR si EN vide).
		raw := row.Match.PairName
		if raw == "" {
			raw = row.Match.PairNameFR
		}
		k := key{id: raw, label: label}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, mediaFilterOptionPair(k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].label < out[j].label })
	return out
}

// extractPlaylistPairs reproduit BuildQ37MediaPlaylistOptionsQuery :
// DISTINCT (playlist_id, label_raw) avec label non vide.
func extractPlaylistPairs(rows []mediaEnrichedRow) []mediaFilterOptionPair {
	type key struct{ id, label string }
	seen := make(map[key]struct{}, len(rows))
	out := make([]mediaFilterOptionPair, 0, len(rows))
	for _, row := range rows {
		if row.Match == nil {
			continue
		}
		label := row.computedPlaylistLabel()
		if label == "" {
			continue
		}
		k := key{id: row.Match.PlaylistID, label: label}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, mediaFilterOptionPair(k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].label < out[j].label })
	return out
}
