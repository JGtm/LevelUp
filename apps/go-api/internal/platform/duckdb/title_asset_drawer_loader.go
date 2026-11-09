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
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
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
		var nameEN string
		if rows.Scan(&m.ID, &nameEN, &m.ImageURL) == nil && nameEN != "" {
			m.Names = i18n.FromColumns(nameEN, "")
			maps = append(maps, m)
		}
	}
	return maps
}

// loadTitleWeapons : icône (weapon_labels) + noms de chaque langue (weapon_label_names, une ligne par
// arme et par langue). La langue servie est choisie par AssetService (langue demandée, sinon `en`).
// Une arme sans nom dans la langue de repli est omise ; tri par nom dans la langue de repli.
func loadTitleWeapons(ctx context.Context, metaDB *DB) (weapons []canonical.AssetMeta) {
	if !metaTableExists(ctx, metaDB, "weapon_labels") {
		return nil
	}
	rows, qerr := metaDB.Query(ctx,
		`SELECT CAST(weapon_id AS VARCHAR), COALESCE(icon_url, '') FROM weapon_labels ORDER BY weapon_id`)
	if qerr != nil {
		slog.ErrorContext(ctx, "title_assets: weapon_labels illisible, armes omises", "err", qerr)
		return nil
	}
	var ids []int64
	byID := map[int64]canonical.AssetMeta{}
	for rows.Next() {
		var idStr string
		var w canonical.AssetMeta
		if err := rows.Scan(&idStr, &w.ImageURL); err != nil {
			slog.ErrorContext(ctx, "title_assets: weapon_labels, ligne illisible", "err", err)
			continue
		}
		u, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			slog.WarnContext(ctx, "title_assets: identifiant d'arme illisible ignoré", "id", idStr, "err", err)
			continue
		}
		w.ID = idStr
		id := int64(u) //nolint:gosec // reinterpretation bit-a-bit, cf. UBigint
		ids = append(ids, id)
		byID[id] = w
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "title_assets: weapon_labels, lecture interrompue", "err", err)
	}
	_ = rows.Close()
	names := loadWeaponLabelNames(ctx, metaDB, ids)
	for _, id := range ids {
		w := byID[id]
		w.Names = names[id]
		if w.Names[i18n.Fallback] != "" {
			weapons = append(weapons, w)
		}
	}
	sort.SliceStable(weapons, func(i, j int) bool {
		return weapons[i].Names[i18n.Fallback] < weapons[j].Names[i18n.Fallback]
	})
	return weapons
}

// loadTitleMedals : icône SPRITE (feuille + offset) + noms/descriptions de chaque langue
// (medal_translations, peuplée par cmd/h5-metadata-fetch). La cascade locale finale (langue
// demandée, sinon `en`) est faite par AssetService au moment de servir. Une médaille sans nom
// anglais est omise ; tri par nom anglais.
func loadTitleMedals(ctx context.Context, metaDB *DB) (medals []canonical.AssetMeta) {
	if !metaTableExists(ctx, metaDB, "medal_definitions") {
		return nil
	}
	rows, qerr := metaDB.Query(ctx,
		`SELECT medal_name_id, COALESCE(sprite_sheet_url, ''),
		        COALESCE(sprite_left, 0), COALESCE(sprite_top, 0),
		        COALESCE(sprite_width, 0), COALESCE(sprite_height, 0)
		 FROM medal_definitions ORDER BY medal_name_id`)
	if qerr != nil {
		return nil
	}
	var ids []int64
	byID := map[int64]canonical.AssetMeta{}
	for rows.Next() {
		var id int64
		var m canonical.AssetMeta
		if rows.Scan(&id, &m.SpriteSheet, &m.SpriteLeft, &m.SpriteTop, &m.SpriteWidth, &m.SpriteHeight) == nil {
			m.ID = strconv.FormatInt(id, 10)
			ids = append(ids, id)
			byID[id] = m
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "title_assets: medal_definitions, lecture interrompue", "err", err)
	}
	_ = rows.Close()
	texts, err := LoadMedalTexts(ctx, metaDB, ids)
	if err != nil {
		slog.ErrorContext(ctx, "title_assets: textes des médailles illisibles, médailles omises", "err", err)
		return nil
	}
	for _, id := range ids {
		m := byID[id]
		m.Names, m.Descriptions = texts[id].Name, texts[id].Description
		if m.Names[i18n.Fallback] != "" {
			medals = append(medals, m)
		}
	}
	sortMedalsByEnglishName(medals)
	return medals
}

// TeamColorName : nom par langue + couleur d'identité d'une équipe Halo 5
// (table team_colors). Color est la couleur d'identité hex (#RRGGBB) exposée telle
// quelle jusqu'au scoreboard front (accent d'en-tête d'équipe) ; "" quand absente.
type TeamColorName struct {
	Name  i18n.Text
	Color string
}

// LoadTeamColorNames charge la map (team_id → noms par langue + couleur) des couleurs d'équipe
// d'un titre depuis une metadata.duckdb DÉJÀ OUVERTE (peuplée par cmd/h5-metadata-fetch) : la
// couleur vient de team_colors, les noms de asset_translations (type « team_color », une ligne par
// langue, ADR 0040 D-5). Une équipe sans aucun nom est omise. Best-effort : table absente/vide →
// map vide ; une table présente dont la lecture échoue est journalisée. Le caller construit le
// résolveur closure et gère le cas vide. Alimente le libellé « Rouge/Bleu » ET la couleur
// d'identité de l'en-tête d'équipe de la Match View H5.
func LoadTeamColorNames(ctx context.Context, metaDB *DB) map[int]TeamColorName {
	m := map[int]TeamColorName{}
	if !metaTableExists(ctx, metaDB, "team_colors") || !metaTableExists(ctx, metaDB, "asset_translations") {
		return m
	}
	names, err := LoadAllAssetTexts(ctx, metaDB, AssetTypeTeamColor)
	if err != nil {
		slog.ErrorContext(ctx, "title_assets: noms des couleurs d'équipe illisibles", "err", err)
		return m
	}
	rows, err := metaDB.Query(ctx, `SELECT team_id, COALESCE(color, '') FROM team_colors`)
	if err != nil {
		slog.ErrorContext(ctx, "title_assets: team_colors illisible", "err", err)
		return m
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int
		var color string
		if err := rows.Scan(&id, &color); err != nil {
			slog.ErrorContext(ctx, "title_assets: team_colors, ligne illisible", "err", err)
			continue
		}
		if name := names[strconv.Itoa(id)].Name; !name.Empty() {
			m[id] = TeamColorName{Name: name, Color: color}
		}
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
