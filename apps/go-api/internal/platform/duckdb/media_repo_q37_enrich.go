package duckdb

import (
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

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
	modeTax analysis.ModeTaxonomy,
) []mediaEnrichedRow {
	out := rows[:0]
	for _, row := range rows {
		if whereCfg.includeMapFilter && f.MapFilter != "" {
			if !mediaRowMatchesMap(row, f.MapFilter) {
				continue
			}
		}
		if whereCfg.includeModeFilter && f.ModeFilter != "" {
			if !mediaRowMatchesMode(row, f.ModeFilter, modeTax) {
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
func mediaRowMatchesMode(row mediaEnrichedRow, filter string, tax analysis.ModeTaxonomy) bool {
	if row.Match == nil {
		return false
	}
	pairName := row.Match.PairName
	if pairName == "" {
		return false
	}
	category, submode, hasSubmode := strings.Cut(filter, "/")
	prefixes := tax.Prefixes(category)
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
	case tax.Other != "" && category == tax.Other:
		// Other = NOT IN les préfixes connus
		matchesCategory = true
		for _, p := range tax.KnownPrefixes() {
			pLower := strings.ToLower(p)
			if strings.HasPrefix(pairLower, pLower+":") || pairLower == pLower {
				matchesCategory = false
				break
			}
		}
	default:
		// Catégorie inconnue (ou titre sans taxonomie) → pas de match
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
