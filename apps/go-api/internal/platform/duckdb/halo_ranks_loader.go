package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/games/mappings"
)

// LoadCareerRankImageURLs charge les URLs d'images pour tous les rangs depuis
// career_ranks (metadata.duckdb). Retourne une map rank_id → imageURL.
// Silencieux si metaDB est nil.
func LoadCareerRankImageURLs(ctx context.Context, metaDB *DB, titleSlug string) (map[int]*string, error) {
	if metaDB == nil {
		return nil, nil
	}
	rows, err := metaDB.Query(ctx, `
		SELECT rank_id,
		       COALESCE(NULLIF(TRIM(large_icon_path), ''), NULLIF(TRIM(icon_path), '')) AS image_path
		FROM career_ranks
		WHERE COALESCE(NULLIF(TRIM(large_icon_path), ''), NULLIF(TRIM(icon_path), '')) IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("LoadCareerRankImageURLs: query: %w", err)
	}
	defer rows.Close()

	urls := make(map[int]*string)
	for rows.Next() {
		var rankID int
		var imagePath sql.NullString
		if err := rows.Scan(&rankID, &imagePath); err != nil {
			return nil, fmt.Errorf("LoadCareerRankImageURLs: scan: %w", err)
		}
		if imagePath.Valid {
			urls[rankID] = buildHomeIdentityAssetURL("career-rank", titleSlug, imagePath.String)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("LoadCareerRankImageURLs: iter: %w", err)
	}
	return urls, nil
}

// LoadRankCatalog construit un mappings.RankCatalog en interrogeant la table
// career_rank_translations de metadata.duckdb.
//
// Les libellÃ©s sont peuplÃ©s via la CLI cmd/refresh-career-ranks (one-shot,
// fetch GameCMS). Si la table est vide, retourne un catalog vide â€” le service
// home utilisera alors un libellÃ© minimal (rank_id seul).
//
// Les codes lang Waypoint ("fr-FR", "de-DE") sont normalisÃ©s en codes courts
// ("fr", "de") au moment de la lecture, en cohÃ©rence avec mappings.LocaleEN/FR.
func LoadRankCatalog(ctx context.Context, metaDB *DB, titleSlug string) (*mappings.RankCatalog, error) {
	if metaDB == nil {
		return mappings.NewRankCatalog(titleSlug, nil), nil
	}

	rows, err := metaDB.Query(ctx, `
		SELECT rank_id, lang, title, subtitle, tier
		FROM career_rank_translations
		ORDER BY rank_id, lang
	`)
	if err != nil {
		return nil, fmt.Errorf("query career_rank_translations: %w", err)
	}
	defer rows.Close()

	type acc struct {
		entry mappings.RankEntry
	}
	byID := make(map[int]*acc)

	for rows.Next() {
		var (
			id               int
			lang             string
			title, sub, tier sql.NullString
		)
		if err := rows.Scan(&id, &lang, &title, &sub, &tier); err != nil {
			return nil, fmt.Errorf("scan career_rank_translations: %w", err)
		}
		norm := mappings.NormalizeLang(lang)
		a, ok := byID[id]
		if !ok {
			a = &acc{entry: mappings.RankEntry{
				ID:       id,
				Title:    make(map[string]string),
				Subtitle: make(map[string]string),
				Tier:     make(map[string]string),
			}}
			byID[id] = a
		}
		if title.Valid {
			a.entry.Title[norm] = title.String
		}
		if sub.Valid {
			a.entry.Subtitle[norm] = sub.String
		}
		if tier.Valid {
			a.entry.Tier[norm] = tier.String
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iter career_rank_translations: %w", err)
	}

	// Enrichit chaque rang avec son seuil XP (career_ranks.xp_required, table
	// distincte des traductions). Best-effort : table/colonne absente (tests,
	// schéma partiel) → libellés conservés sans XP (le fallback de progression
	// dans buildHomeCareerRank reste simplement inactif, pas d'échec).
	if xpRows, xerr := metaDB.Query(ctx, `SELECT rank_id, xp_required FROM career_ranks`); xerr == nil {
		for xpRows.Next() {
			var id int
			var xp sql.NullInt64
			if scanErr := xpRows.Scan(&id, &xp); scanErr != nil {
				break
			}
			if a, ok := byID[id]; ok && xp.Valid {
				a.entry.XPRequired = int(xp.Int64)
			}
		}
		xpRows.Close()
	}

	entries := make([]mappings.RankEntry, 0, len(byID))
	for _, a := range byID {
		entries = append(entries, a.entry)
	}
	return mappings.NewRankCatalog(titleSlug, entries), nil
}
