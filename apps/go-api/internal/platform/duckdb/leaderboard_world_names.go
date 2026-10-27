package duckdb

// leaderboard_world_names.go — noms de playlists du catalogue du classement mondial : cascade de
// résolution par langue (playlistName) et lecture du catalogue metadata mutualisé.

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	"levelup/go-api/internal/i18n"
)

// playlistName applique la cascade de résolution d'un nom de playlist dans la langue `loc` :
// nom officiel (asset_translations) puis nom de rankedplaylists dans la langue, puis
// name_canonical (EN brut), puis les mêmes sources dans la langue de repli et la langue par
// défaut, enfin l'id. La langue de repli (EN) lit donc : officiel EN > rankedplaylists EN >
// canonical > officiel FR > rankedplaylists FR > id.
func playlistName(id string, loc i18n.Locale, official i18n.Text, canonical string) string {
	pl, hasPL := rankedplaylists.Lookup(id)
	named := func(l i18n.Locale) string {
		if v := official[l]; v != "" {
			return v
		}
		if hasPL {
			return pl.Name(l)
		}
		return ""
	}
	order := []i18n.Locale{loc}
	for _, l := range []i18n.Locale{i18n.Fallback, i18n.Default} {
		if !slices.Contains(order, l) {
			order = append(order, l)
		}
	}
	if v := named(order[0]); v != "" {
		return v
	}
	if canonical != "" {
		return canonical
	}
	for _, l := range order[1:] {
		if v := named(l); v != "" {
			return v
		}
	}
	return id
}

// resolvePlaylistNamesFromCatalog lit le catalogue metadata MUTUALISÉ et retourne les noms
// officiels localisés (asset_translations, une map id -> nom par langue de l'application) et
// canonMap (playlists_catalog.name_canonical, EN brut). Vides si metadata indisponible (le
// caller retombe sur rankedplaylists). Ne touche jamais la shared DB.
func (r *LeaderboardRepo) resolvePlaylistNamesFromCatalog(ctx context.Context, ids []string) (official map[i18n.Locale]map[string]string, canonMap map[string]string) {
	official = map[i18n.Locale]map[string]string{}
	for _, l := range i18n.Locales() {
		official[l] = map[string]string{}
	}
	canonMap = map[string]string{}
	if len(ids) == 0 || r.pdb == nil || r.pdb.Metadata == nil {
		return official, canonMap
	}
	meta := r.pdb.Metadata
	scan := func(q string, args []any, dst map[string]string) {
		rows, err := meta.QueryRecovered(ctx, q, args...)
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) == nil && strings.TrimSpace(name) != "" {
				dst[id] = name
			}
		}
	}
	scan(fmt.Sprintf(
		`SELECT playlist_asset_id, COALESCE(name_canonical, '') FROM playlists_catalog
		 WHERE title_slug = ? AND playlist_asset_id IN (%s)`, Placeholders(len(ids))),
		append([]any{defaultLeaderboardTitleSlug}, ToAnySlice(ids)...), canonMap)
	// asset_translations de toutes les langues en une requête : on route par la colonne lang.
	langs := i18n.Locales()
	langArgs := make([]any, 0, len(langs)+len(ids))
	for _, l := range langs {
		langArgs = append(langArgs, string(l))
	}
	langArgs = append(langArgs, ToAnySlice(ids)...)
	rows, err := meta.QueryRecovered(ctx, fmt.Sprintf(
		`SELECT asset_id, name, lang FROM asset_translations
		 WHERE asset_type = 'playlist' AND lang IN (%s) AND asset_id IN (%s)
		   AND name IS NOT NULL AND TRIM(name) <> ''`, Placeholders(len(langs)), Placeholders(len(ids))), langArgs...)
	if err != nil {
		return official, canonMap
	}
	defer rows.Close()
	for rows.Next() {
		var id, name, lang string
		if rows.Scan(&id, &name, &lang) == nil && strings.TrimSpace(name) != "" {
			if l := i18n.Locale(lang); i18n.Known(l) {
				official[l][id] = name
			}
		}
	}
	return official, canonMap
}

// countNamesPerLocale compte les noms officiels lus, par langue, pour le journal de diagnostic.
func countNamesPerLocale(official map[i18n.Locale]map[string]string) map[string]int {
	out := make(map[string]int, len(official))
	for l, names := range official {
		out[string(l)] = len(names)
	}
	return out
}
