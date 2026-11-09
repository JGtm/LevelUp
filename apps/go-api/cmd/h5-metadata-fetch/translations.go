package main

// translations.go — noms de playlists, cartes et modes de Halo 5 dans asset_translations : une
// ligne par langue de config/locales.toml pour laquelle le seed a un nom (ADR 0040 D-5). La
// langue de l'API (sans en-tête) vient du catalogue ou de l'API ; les autres, de l'override
// TOML et d'une passe Accept-Language par langue.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/i18n"
)

// haloTag est l'étiquette de langue de asset_translations pour `l` : la première étiquette Halo
// de la langue, celle que lit le résolveur de noms (ResolveAssetNamesBulk →
// PreferredLangsForLocale). IMPÉRATIF : l'écrire par i18n.HaloTags (config/locales.toml), la
// même source que le côté lecture, jamais par un littéral.
func haloTag(l i18n.Locale) string {
	tags := i18n.HaloTags(l)
	if len(tags) == 0 {
		return string(l)
	}
	return tags[0]
}

// apiGameBaseVariant — élément de l'API Metadata officielle /game-base-variants.
// `id` = le GameBaseVariantId porté par chaque match Halo 5 (cf.
// internal/games/halo_5/mapping.go : GameVariant = assetRef("game_variant",
// r.GameBaseVariantId)). Le NOM du mode vit ici, pas dans la donnée de match.
// Shape minimale alignée sur apiPlaylist/apiMap (tous les types officiels exposent
// `name` + `id`).
type apiGameBaseVariant struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// applyMapIDOverrides applique les overrides de nom de map keyés par asset_id :
// name_canonical (maps_catalog, langue de l'API) + une ligne asset_translations par langue
// renseignée dans l'override. Idempotent (UPDATE + INSERT OR REPLACE). Best-effort par entrée :
// une erreur est loguée et n'interrompt pas les autres.
func applyMapIDOverrides(db *sql.DB, overrides []mapIDEntry) {
	if len(overrides) == 0 {
		return
	}
	applied := 0
	for _, ov := range overrides {
		id := strings.TrimSpace(ov.ID)
		canonical := strings.TrimSpace(ov.Name[i18n.Fallback])
		if id == "" || canonical == "" {
			fmt.Printf("maps_by_id: entrée ignorée (id ou nom %s vide): %+v\n", i18n.Fallback, ov)
			continue
		}
		if _, err := db.Exec(
			`UPDATE maps_catalog SET name_canonical = ? WHERE title_slug = ? AND lower(map_asset_id) = lower(?)`,
			canonical, halo5.TitleSlug, id); err != nil {
			fmt.Printf("maps_by_id: update name_canonical %s: %v\n", id, err)
		}
		for _, l := range i18n.Locales() {
			upsertAssetTranslation(db, id, "map", haloTag(l), ov.Name[l])
		}
		applied++
	}
	fmt.Printf("maps_by_id: %d override(s) de map appliqué(s)\n", applied)
}

// seedAssetTranslations PEUPLE asset_translations (la table que lit le résolveur de
// noms) depuis les catalogues déjà seedés + l'endpoint officiel des modes. C'est le
// pont DATA-via-code qui débloque l'affichage mode/carte/playlist pour Halo 5.
//
// Mapping asset_type → source (formats alignés sur GetDistinctAssetIDs /
// ResolveAssetNamesBulk côté lecture) :
//   - playlist     ← table playlists (id, name)          [match.HopperId]
//   - map          ← table maps_catalog (map_asset_id, name_canonical) [match.MapId]
//   - game_variant ← API /game-base-variants (id, name)  [match.GameBaseVariantId]
//
// Langue de l'API (Fallback) d'abord, puis, pour chaque autre langue de l'application, deux
// sources complémentaires :
//  1. Overrides versionnés (section [maps] du TOML, labels.Maps[nom de l'API][langue]) —
//     prioritaires.
//  2. Pass par langue via l'API officielle : l'API Metadata Halo 5 HONORE l'en-tête
//     `Accept-Language` (info confirmée) et renvoie les noms localisés. On refetch
//     /playlists, /maps, /game-base-variants dans la langue et on seede ses lignes.
//     Certains assets (noms propres de maps) ne sont pas localisés → l'API renvoie
//     le même nom que dans la langue de l'API ; on seede quand même (la cascade prend la
//     langue demandée ; si égal, dégradation propre) et on logge un compteur « différent »
//     pour visibilité.
//
// Idempotent (INSERT OR REPLACE sur PK asset_id+asset_type+lang). Best-effort : une
// insertion ou un endpoint par langue qui échoue est logué et n'interrompt pas le pass de
// la langue de l'API.
func seedAssetTranslations(db *sql.DB, key string, labels assetLabels) {
	n := 0
	n += seedPlaylistTranslations(db)
	n += seedMapTranslations(db, labels.Maps)
	n += seedModeTranslations(db, key)
	fmt.Printf("asset_translations[%s]: %d seedées\n", haloTag(i18n.Fallback), n)
	// Pass par langue (best-effort, ne casse pas le pass de la langue de l'API ci-dessus).
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			continue
		}
		fmt.Printf("asset_translations[%s]: %d seedées\n", haloTag(l), seedLocalizedTranslations(db, key, l))
	}
}

// localizedResult — bilan d'un pass par langue pour un asset_type (visibilité du garde-fou
// « langue == langue de l'API » décrit ci-dessus).
type localizedResult struct {
	seeded int // lignes de la langue insérées
	diff   int // dont nom strictement différent du nom de la langue de l'API connu
	total  int // assets retournés par l'API dans la langue
}

// seedLocalizedTranslations refetch les endpoints localisables dans la langue `l` via
// Accept-Language et seede ses lignes de asset_translations. Pour chaque asset_type on compare
// le nom au nom de la langue de l'API déjà seedé afin de tracer combien sont RÉELLEMENT
// localisés. Best-effort : un endpoint qui échoue est logué et n'empêche pas les autres.
func seedLocalizedTranslations(db *sql.DB, key string, l i18n.Locale) int {
	refPlaylist := readFallbackNames(db, "playlist")
	refMap := readFallbackNames(db, "map")
	refVariant := readFallbackNames(db, "game_variant")

	pl := seedLocalizedSimple(db, key, "playlists", "playlist", l, refPlaylist)
	mp := seedLocalizedSimple(db, key, "maps", "map", l, refMap)
	gv := seedLocalizedSimple(db, key, "game-base-variants", "game_variant", l, refVariant)

	fmt.Printf("%s localisés (!= %s) : playlists %d/%d, maps %d/%d, modes %d/%d\n",
		haloTag(l), haloTag(i18n.Fallback), pl.diff, pl.total, mp.diff, mp.total, gv.diff, gv.total)
	return pl.seeded + mp.seeded + gv.seeded
}

// readFallbackNames relit les noms de la langue de l'API déjà seedés pour un asset_type (clé =
// asset_id). Sert de référence pour le compteur « localisé ». Erreur → map vide (le compteur
// diff comptera alors 0, sans casser le seed de la langue).
func readFallbackNames(db *sql.DB, assetType string) map[string]string {
	out := map[string]string{}
	rows, err := db.Query(
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = ? AND lang = ?`,
		assetType, haloTag(i18n.Fallback))
	if err != nil {
		fmt.Printf("asset_translations[%s]: read %s ref %v\n", assetType, haloTag(i18n.Fallback), err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			fmt.Printf("asset_translations[%s]: scan ref %v\n", assetType, err)
			continue
		}
		out[id] = name
	}
	return out
}

// seedLocalizedSimple fetche un endpoint dans la langue `l` (shape {id,name} commune à
// playlists, maps, game-base-variants) et seede asset_translations(asset_type, lang de `l`).
// On compare au nom de référence connu pour le compteur de localisation. Best-effort.
func seedLocalizedSimple(db *sql.DB, key, typ, assetType string, l i18n.Locale, refNames map[string]string) localizedResult {
	res := localizedResult{}
	body, err := fetchMetaLang(key, typ, acceptLanguage(l))
	if err != nil {
		fmt.Printf("asset_translations[%s][%s]: SKIP (%v)\n", assetType, haloTag(l), err)
		return res
	}
	// /maps expose l'id sous map_asset_id côté catalogue mais l'API le renvoie
	// toujours dans le champ `id` (cf. apiMap/apiPlaylist/apiGameBaseVariant) → shape
	// commune {id,name}.
	var items []apiGameBaseVariant
	if err := json.Unmarshal(body, &items); err != nil {
		fmt.Printf("asset_translations[%s][%s]: parse %v\n", assetType, haloTag(l), err)
		return res
	}
	res.total = len(items)
	for _, it := range items {
		name := strings.TrimSpace(it.Name)
		if !upsertAssetTranslation(db, it.ID, assetType, haloTag(l), name) {
			continue
		}
		res.seeded++
		if ref, ok := refNames[it.ID]; ok && name != strings.TrimSpace(ref) {
			res.diff++
		}
	}
	return res
}

// upsertAssetTranslation insère/réécrit une ligne asset_translations. Best-effort.
func upsertAssetTranslation(db *sql.DB, assetID, assetType, lang, name string) bool {
	if strings.TrimSpace(assetID) == "" || strings.TrimSpace(name) == "" {
		return false
	}
	if _, err := db.Exec(
		`INSERT OR REPLACE INTO asset_translations (asset_id, asset_type, lang, name)
		 VALUES (?,?,?,?)`,
		assetID, assetType, lang, strings.TrimSpace(name)); err != nil {
		fmt.Printf("asset_translations: insert %s/%s/%s: %v\n", assetType, assetID, lang, err)
		return false
	}
	return true
}

// seedPlaylistTranslations relit la table `playlists` (déjà seedée par seedPlaylists) et écrit
// asset_type='playlist' dans la langue de l'API. Les autres langues viennent du pass par langue.
func seedPlaylistTranslations(db *sql.DB) int {
	rows, err := db.Query(`SELECT id, name FROM playlists WHERE TRIM(COALESCE(name,'')) != ''`)
	if err != nil {
		fmt.Printf("asset_translations[playlist]: read playlists %v\n", err)
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			fmt.Printf("asset_translations[playlist]: scan %v\n", err)
			continue
		}
		if upsertAssetTranslation(db, id, "playlist", haloTag(i18n.Fallback), name) {
			n++
		}
	}
	return n
}

// seedMapTranslations relit `maps_catalog` (déjà seedée par seedMaps) et écrit
// asset_type='map' dans la langue de l'API (name_canonical) puis, pour chaque autre langue, la
// traduction de la section [maps] du TOML (maps[nom de l'API][langue]) quand elle existe.
func seedMapTranslations(db *sql.DB, maps map[string]i18n.Text) int {
	rows, err := db.Query(`SELECT map_asset_id, name_canonical FROM maps_catalog
		WHERE title_slug = ? AND TRIM(COALESCE(name_canonical,'')) != ''`, halo5.TitleSlug)
	if err != nil {
		fmt.Printf("asset_translations[map]: read maps_catalog %v\n", err)
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, nameAPI string
		if err := rows.Scan(&id, &nameAPI); err != nil {
			fmt.Printf("asset_translations[map]: scan %v\n", err)
			continue
		}
		if upsertAssetTranslation(db, id, "map", haloTag(i18n.Fallback), nameAPI) {
			n++
		}
		for _, l := range i18n.Locales() {
			if l == i18n.Fallback {
				continue
			}
			if upsertAssetTranslation(db, id, "map", haloTag(l), maps[nameAPI][l]) {
				n++
			}
		}
	}
	return n
}

// seedModeTranslations fetche l'endpoint officiel /game-base-variants et écrit
// asset_type='game_variant' dans la langue de l'API. C'est la SOURCE de nom de mode pour Halo 5
// (le match ne porte que le GameBaseVariantId, jamais le nom). Aucune section modes dans le TOML.
func seedModeTranslations(db *sql.DB, key string) int {
	body, err := fetchMeta(key, "game-base-variants")
	if err != nil {
		fmt.Printf("asset_translations[game_variant]: SKIP (%v)\n", err)
		return 0
	}
	var variants []apiGameBaseVariant
	if err := json.Unmarshal(body, &variants); err != nil {
		fmt.Printf("asset_translations[game_variant]: parse %v\n", err)
		return 0
	}
	n := 0
	for _, v := range variants {
		if upsertAssetTranslation(db, v.ID, "game_variant", haloTag(i18n.Fallback), v.Name) {
			n++
		}
	}
	return n
}
