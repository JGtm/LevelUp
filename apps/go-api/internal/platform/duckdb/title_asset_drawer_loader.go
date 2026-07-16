// Package duckdb — title_asset_drawer_loader.go : lecture des référentiels d'assets
// d'un titre additionnel (maps/armes/médailles + insignes CSR) depuis sa
// metadata.duckdb isolée. Extrait de api/server.go (K1g, 2026-07-06) : le SQL vit ici
// (couche données), le boot ne garde qu'un wrapper open+delegate (modèle
// LoadCareerRankImageURLs). Best-effort : table absente/vide → slice/map vide.
package duckdb

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// LoadTitleAssetDrawerData charge maps + armes + médailles (avec URLs d'image) d'un titre
// depuis une metadata.duckdb DÉJÀ OUVERTE (peuplée par cmd/h5-metadata-fetch). Le caller
// ouvre/ferme la connexion. Best-effort : chaque table absente/vide → slice nil.
func LoadTitleAssetDrawerData(ctx context.Context, metaDB *DB, slug string) (maps, weapons, medals []canonical.AssetMeta) {
	if rows, qerr := metaDB.Query(ctx,
		`SELECT map_asset_id, COALESCE(name_canonical, ''), COALESCE(image_url, '')
		 FROM maps_catalog WHERE title_slug = ? ORDER BY name_canonical`, slug); qerr == nil {
		for rows.Next() {
			var m canonical.AssetMeta
			if rows.Scan(&m.ID, &m.NameEN, &m.ImageURL) == nil && m.NameEN != "" {
				maps = append(maps, m)
			}
		}
		_ = rows.Close()
	}
	if rows, qerr := metaDB.Query(ctx,
		`SELECT weapon_id::VARCHAR, name_en, COALESCE(icon_url, '')
		 FROM weapon_labels ORDER BY name_en`); qerr == nil {
		for rows.Next() {
			var w canonical.AssetMeta
			if rows.Scan(&w.ID, &w.NameEN, &w.ImageURL) == nil && w.NameEN != "" {
				weapons = append(weapons, w)
			}
		}
		_ = rows.Close()
	}
	// Médailles : icône SPRITE (feuille + offset) + noms/descriptions FR/EN. Le name_fr
	// est peuplé par cmd/h5-metadata-fetch (Accept-Language: fr-FR). La cascade locale
	// finale (fr → name_fr sinon name_en) est faite côté front (AssetCard).
	if rows, qerr := metaDB.Query(ctx,
		`SELECT medal_name_id::VARCHAR, name_en, COALESCE(name_fr, ''),
		        COALESCE(description_en, ''), COALESCE(description_fr, ''),
		        COALESCE(sprite_sheet_url, ''),
		        COALESCE(sprite_left, 0), COALESCE(sprite_top, 0),
		        COALESCE(sprite_width, 0), COALESCE(sprite_height, 0)
		 FROM medal_definitions ORDER BY name_en`); qerr == nil {
		for rows.Next() {
			var m canonical.AssetMeta
			if rows.Scan(&m.ID, &m.NameEN, &m.NameFR, &m.Description, &m.DescriptionFR, &m.SpriteSheet,
				&m.SpriteLeft, &m.SpriteTop, &m.SpriteWidth, &m.SpriteHeight) == nil && m.NameEN != "" {
				medals = append(medals, m)
			}
		}
		_ = rows.Close()
	}
	return maps, weapons, medals
}

// TeamColorName : nom localisé EN/FR d'une couleur d'équipe Halo 5 (table team_colors).
type TeamColorName struct {
	NameEN string
	NameFR string
}

// LoadTeamColorNames charge la map (team_id → noms EN/FR) des couleurs d'équipe d'un
// titre depuis une metadata.duckdb DÉJÀ OUVERTE (peuplée par cmd/h5-metadata-fetch).
// Best-effort : table absente/vide → map vide. Le caller construit le résolveur closure
// et gère le cas vide. Alimente le libellé d'équipe « Rouge/Bleu » de la Match View H5.
func LoadTeamColorNames(ctx context.Context, metaDB *DB) map[int]TeamColorName {
	m := map[int]TeamColorName{}
	if rows, qerr := metaDB.Query(ctx,
		`SELECT team_id, COALESCE(name_en, ''), COALESCE(name_fr, '') FROM team_colors`); qerr == nil {
		for rows.Next() {
			var id int
			var en, fr string
			if rows.Scan(&id, &en, &fr) == nil && (en != "" || fr != "") {
				m[id] = TeamColorName{NameEN: en, NameFR: fr}
			}
		}
		_ = rows.Close()
	}
	return m
}

// LoadCSRBadgeMap charge la map (designation|tier → icon_url) des insignes CSR d'un titre
// depuis une metadata.duckdb DÉJÀ OUVERTE. Best-effort : renvoie une map (éventuellement
// vide). Le caller construit le résolveur closure et gère le cas vide.
func LoadCSRBadgeMap(ctx context.Context, metaDB *DB) map[string]string {
	m := map[string]string{}
	if rows, qerr := metaDB.Query(ctx,
		`SELECT designation_name, tier_id, COALESCE(icon_url, '') FROM csr_designations`); qerr == nil {
		for rows.Next() {
			var name, url string
			var tier int
			if rows.Scan(&name, &tier, &url) == nil && url != "" {
				m[fmt.Sprintf("%s|%d", strings.ToLower(name), tier)] = url
			}
		}
		_ = rows.Close()
	}
	return m
}
