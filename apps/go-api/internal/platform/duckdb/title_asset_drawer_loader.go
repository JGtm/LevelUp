// Package duckdb — title_asset_drawer_loader.go : lecture des référentiels d'assets
// d'un titre additionnel (maps/armes/médailles + insignes CSR) depuis sa
// metadata.duckdb isolée. Extrait de api/server.go (K1g, 2026-07-06) : le SQL vit ici
// (couche données), le boot ne garde qu'un wrapper open+delegate (modèle
// LoadCareerRankImageURLs). Best-effort : table absente/vide → slice/map vide, SANS erreur
// journalisée pour l'absence (attendue : titre jamais synchronisé, base copiée avant une
// migration) ; une table présente dont la lecture échoue reste journalisée en ERROR.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/games/canonical"
)

// metaTableExists dit si la table existe dans la metadata. Si la vérification elle-même échoue,
// rend true : la requête qui suit tranche et journalise sa propre erreur.
func metaTableExists(ctx context.Context, metaDB *DB, table string) bool {
	var n int
	if err := metaDB.QueryRow(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, table).Scan(&n); err != nil {
		return true
	}
	if n == 0 {
		slog.DebugContext(ctx, "title_assets: table absente, référentiel ignoré", "table", table, "path", metaDB.path)
		return false
	}
	return true
}

// LoadTitleAssetDrawerData charge maps + armes + médailles (avec URLs d'image) d'un titre
// depuis une metadata.duckdb DÉJÀ OUVERTE (peuplée par cmd/h5-metadata-fetch). Le caller
// ouvre/ferme la connexion. Best-effort : chaque table absente/vide → slice nil.
func LoadTitleAssetDrawerData(ctx context.Context, metaDB *DB, slug string) (maps, weapons, medals []canonical.AssetMeta) {
	return loadTitleMaps(ctx, metaDB, slug), loadTitleWeapons(ctx, metaDB), loadTitleMedals(ctx, metaDB)
}

func loadTitleMaps(ctx context.Context, metaDB *DB, slug string) (maps []canonical.AssetMeta) {
	if !metaTableExists(ctx, metaDB, "maps_catalog") {
		return nil
	}
	rows, qerr := metaDB.Query(ctx,
		`SELECT map_asset_id, COALESCE(name_canonical, ''), COALESCE(image_url, '')
		 FROM maps_catalog WHERE title_slug = ? ORDER BY name_canonical`, slug)
	if qerr != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m canonical.AssetMeta
		if rows.Scan(&m.ID, &m.NameEN, &m.ImageURL) == nil && m.NameEN != "" {
			maps = append(maps, m)
		}
	}
	return maps
}

func loadTitleWeapons(ctx context.Context, metaDB *DB) (weapons []canonical.AssetMeta) {
	if !metaTableExists(ctx, metaDB, "weapon_labels") {
		return nil
	}
	rows, qerr := metaDB.Query(ctx,
		`SELECT weapon_id::VARCHAR, name_en, COALESCE(icon_url, '')
		 FROM weapon_labels ORDER BY name_en`)
	if qerr != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var w canonical.AssetMeta
		if rows.Scan(&w.ID, &w.NameEN, &w.ImageURL) == nil && w.NameEN != "" {
			weapons = append(weapons, w)
		}
	}
	return weapons
}

// loadTitleMedals : icône SPRITE (feuille + offset) + noms/descriptions FR/EN. Le name_fr
// est peuplé par cmd/h5-metadata-fetch (Accept-Language: fr-FR). La cascade locale
// finale (fr → name_fr sinon name_en) est faite côté front (AssetCard).
func loadTitleMedals(ctx context.Context, metaDB *DB) (medals []canonical.AssetMeta) {
	if !metaTableExists(ctx, metaDB, "medal_definitions") {
		return nil
	}
	rows, qerr := metaDB.Query(ctx,
		`SELECT medal_name_id::VARCHAR, name_en, COALESCE(name_fr, ''),
		        COALESCE(description_en, ''), COALESCE(description_fr, ''),
		        COALESCE(sprite_sheet_url, ''),
		        COALESCE(sprite_left, 0), COALESCE(sprite_top, 0),
		        COALESCE(sprite_width, 0), COALESCE(sprite_height, 0)
		 FROM medal_definitions ORDER BY name_en`)
	if qerr != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m canonical.AssetMeta
		if rows.Scan(&m.ID, &m.NameEN, &m.NameFR, &m.Description, &m.DescriptionFR, &m.SpriteSheet,
			&m.SpriteLeft, &m.SpriteTop, &m.SpriteWidth, &m.SpriteHeight) == nil && m.NameEN != "" {
			medals = append(medals, m)
		}
	}
	return medals
}

// TeamColorName : nom localisé EN/FR + couleur d'identité d'une équipe Halo 5
// (table team_colors). Color est la couleur d'identité hex (#RRGGBB) exposée telle
// quelle jusqu'au scoreboard front (accent d'en-tête d'équipe) ; "" quand absente.
type TeamColorName struct {
	NameEN string
	NameFR string
	Color  string
}

// LoadTeamColorNames charge la map (team_id → noms EN/FR + couleur) des couleurs d'équipe
// d'un titre depuis une metadata.duckdb DÉJÀ OUVERTE (peuplée par cmd/h5-metadata-fetch).
// Best-effort : table absente/vide → map vide. Le caller construit le résolveur closure
// et gère le cas vide. Alimente le libellé « Rouge/Bleu » ET la couleur d'identité de
// l'en-tête d'équipe de la Match View H5.
func LoadTeamColorNames(ctx context.Context, metaDB *DB) map[int]TeamColorName {
	m := map[int]TeamColorName{}
	if !metaTableExists(ctx, metaDB, "team_colors") {
		return m
	}
	if rows, qerr := metaDB.Query(ctx,
		`SELECT team_id, COALESCE(name_en, ''), COALESCE(name_fr, ''), COALESCE(color, '') FROM team_colors`); qerr == nil {
		for rows.Next() {
			var id int
			var en, fr, color string
			if rows.Scan(&id, &en, &fr, &color) == nil && (en != "" || fr != "") {
				m[id] = TeamColorName{NameEN: en, NameFR: fr, Color: color}
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
	if !metaTableExists(ctx, metaDB, "csr_designations") {
		return m
	}
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
