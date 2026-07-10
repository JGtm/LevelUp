package duckdb

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

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
	enriched = applyCrossDBMediaFilters(enriched, f, whereCfg, r.modeTax)
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
// translateModeFilterOptions via la ModeTaxonomy injectée).
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
