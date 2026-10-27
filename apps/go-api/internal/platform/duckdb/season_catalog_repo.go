// Package duckdb — season_catalog_repo.go : lecture de shared.season_catalog
// (noms + traduction FR des saisons CSR Waypoint, persisté par C2a) et composition
// du libellé "Saison N · Nom" localisé pour les sélecteurs de saison.
//
// Foyer canonique (règle ≤2 copies) : réutilisé par le sélecteur de la page
// classement (GetWorldLeaderboardCatalog) ET de la page player (AvailableCSRSeasons).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
)

// SeasonName : nom d'une saison CSR persisté dans season_catalog (C2).
type SeasonName struct {
	Name  i18n.Text // en : nom d'Operation ; fr : traduction fr-FR (peut être absente → repli EN)
	Major int
	Minor int
}

// LoadSeasonCatalogNames lit season_catalog (shared) et retourne un map keyé par
// season_id EN MINUSCULES — les appelants varient la casse (Waypoint "csrseason13-2"
// vs API carrière "CsrSeason13-2"). Best-effort : table absente (saison jamais
// scrapée, DB legacy) → map vide + nil error (dégradation gracieuse, l'appelant
// retombe sur son libellé dérivé).
func LoadSeasonCatalogNames(ctx context.Context, sharedDB *sql.DB, titleSlug string) (map[string]SeasonName, error) {
	rows, err := sharedDB.QueryContext(ctx,
		`SELECT season_id, COALESCE(display_name, ''), COALESCE(name_fr, ''),
			COALESCE(season_major, 0), COALESCE(season_minor, 0)
		 FROM season_catalog WHERE title_slug = ?`, titleSlug)
	if err != nil {
		if isTableNotFoundErr(err) {
			return map[string]SeasonName{}, nil
		}
		return nil, fmt.Errorf("LoadSeasonCatalogNames: %w", err)
	}
	defer rows.Close()
	out := make(map[string]SeasonName)
	for rows.Next() {
		var id string
		var sn SeasonName
		var nameEN, nameFR string
		if err := rows.Scan(&id, &nameEN, &nameFR, &sn.Major, &sn.Minor); err != nil {
			return nil, fmt.Errorf("LoadSeasonCatalogNames scan: %w", err)
		}
		sn.Name = i18n.FromColumns(nameEN, nameFR)
		out[strings.ToLower(strings.TrimSpace(id))] = sn
	}
	return out, rows.Err()
}

// SeasonSelectorLabel compose "Saison N · Nom" / "Season N · Name" (catalogue des messages,
// common.toml) pour un season_id, à partir des noms persistés (match insensible à la casse).
// Retourne `fallback` si la saison n'est pas au catalogue (jamais scrapée) — préserve
// le libellé dérivé existant.
func SeasonSelectorLabel(locale, seasonID string, names map[string]SeasonName, fallback string) string {
	sn, ok := names[strings.ToLower(strings.TrimSpace(seasonID))]
	if !ok {
		return fallback
	}
	loc := i18n.Parse(locale)
	name := strings.TrimSpace(sn.Name.Pick(loc))
	switch {
	case sn.Major > 0 && name != "":
		return i18n.Msg(loc, "common.season.label_named", i18n.Params{"n": sn.Major, "name": name})
	case sn.Major > 0:
		return i18n.Msg(loc, "common.season.label", i18n.Params{"n": sn.Major})
	case name != "":
		return name
	default:
		return fallback
	}
}
