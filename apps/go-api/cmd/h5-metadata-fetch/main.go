// Outil ops : peuple la metadata.duckdb Halo 5 (médailles, cartes, désignations CSR)
// depuis l'API Metadata OFFICIELLE Halo 5 (www.haloapi.com). Ces référentiels
// canoniques ne sont PAS sur les endpoints internes SpartanToken (cf.
// .ai/PLAN_H5_ASSETS.md) — seule l'API officielle les expose. Auth = clé
// d'abonnement Azure APIM (Ocp-Apim-Subscription-Key), lue dans l'env
// LEVELUP_HALOAPI_KEY — JAMAIS committée. Données Halo 5 figées → seed one-shot
// idempotent (INSERT OR REPLACE).
//
// Usage : LEVELUP_HALOAPI_KEY=<clé> LEVELUP_REPO_ROOT=<repo> go run ./cmd/h5-metadata-fetch
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	halo5 "levelup/go-api/internal/games/halo_5"
	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

const officialMetaBase = "https://www.haloapi.com/metadata/h5/metadata/"

// defaultConfigRoot : racine portant config/titles/<slug>/mappings (worktree, où
// vivent les TOML versionnés — distinct de cfg.RepoRoot = clone runtime de données).
const defaultConfigRoot = "c:/Users/Guillaume/Downloads/Scripts/levelup-multititre"

// frLabels — overrides de noms FR (localisation officielle Halo 5), versionnés dans
// config/titles/halo_5/mappings/asset_labels_fr.toml. Clé = nom EN exact de l'API.
//
// PAS de champ Weapons : la section [weapons] a été RETIRÉE du TOML par V72-06 —
// les noms d'armes FR sont la source unique de weapon_names.toml, keyée par
// weapon_key (résolue via weapon_id → weapon_ids → weapon_key) et non par nom EN
// brut. Le champ et sa branche d'override étaient donc morts (map toujours vide) ;
// supprimés à la contre-revue V7.2. Ne pas les réintroduire : le keying par nom EN
// est précisément la source de mismatch que V72-06 supprime.
type frLabels struct {
	Medals            map[string]string `toml:"medals"`
	Maps              map[string]string `toml:"maps"`
	MedalDescriptions map[string]string `toml:"medal_descriptions"`
	Commendations     map[string]string `toml:"commendations"`
	// MapsByID — overrides de nom de map par asset_id (canvas Forge non nommés par
	// l'API officielle /maps : name_canonical vide). Distinct de Maps (keyé par nom
	// EN) car ces maps N'ONT PAS de nom EN sur lequel keyer. Appliqués en fin de run.
	MapsByID []mapIDOverride `toml:"maps_by_id"`
}

// mapIDOverride — un override de nom de map keyé par asset_id (EN canonique + FR).
type mapIDOverride struct {
	ID string `toml:"id"`
	EN string `toml:"en"`
	FR string `toml:"fr"`
}

// loadFRLabels lit les overrides FR. Fichier absent / illisible → maps vides
// (name_fr = name_en, dégradation propre). Best-effort.
func loadFRLabels(path string) frLabels {
	fr := frLabels{Medals: map[string]string{}, Maps: map[string]string{}, MedalDescriptions: map[string]string{}, Commendations: map[string]string{}}
	b, err := os.ReadFile(path)
	if err != nil {
		return fr
	}
	_ = toml.Unmarshal(b, &fr)
	if fr.Medals == nil {
		fr.Medals = map[string]string{}
	}
	if fr.Maps == nil {
		fr.Maps = map[string]string{}
	}
	if fr.MedalDescriptions == nil {
		fr.MedalDescriptions = map[string]string{}
	}
	if fr.Commendations == nil {
		fr.Commendations = map[string]string{}
	}
	return fr
}

// frOr retourne la traduction FR si présente, sinon l'EN (fallback = la clé).
func frOr(m map[string]string, en string) string {
	return chooseFR(m, en, "")
}

// chooseFR choisit le nom FR à persister pour un asset, par ordre de priorité :
//  1. override TOML versionné (corrections manuelles, clé = nom EN exact) ;
//  2. nom FR localisé par l'API Metadata (Accept-Language: fr-FR) ;
//  3. le nom EN (dégradation propre — on ne stocke JAMAIS de name_fr vide).
//
// Un asset non localisé fait renvoyer par l'API le MÊME nom qu'en EN → on le stocke
// tel quel (cascade fr-FR == en-US, cf. seedFrenchSimple). apiFR vide reproduit
// exactement l'ancien comportement de frOr (override sinon EN).
func chooseFR(override map[string]string, en, apiFR string) string {
	if v, ok := override[en]; ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if strings.TrimSpace(apiFR) != "" {
		return strings.TrimSpace(apiFR)
	}
	return en
}

// lookupOr retourne m[key] si non vide, sinon fallback (ex. description_fr : FR si
// dispo, sinon la description EN officielle — pas le nom).
func lookupOr(m map[string]string, key, fallback string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return fallback
}

func main() {
	// --overrides-only : applique UNIQUEMENT les overrides TOML locaux (maps_by_id)
	// à la metadata.duckdb existante, SANS fetch réseau ni clé API. Sert à rejouer le
	// seed d'un override (ex. nom de map Tidal) sans marteler l'API. Les autres args
	// (config root) restent positionnels.
	overridesOnly := false
	var positional []string
	for _, a := range os.Args[1:] {
		if a == "--overrides-only" {
			overridesOnly = true
			continue
		}
		positional = append(positional, a)
	}

	key := os.Getenv("LEVELUP_HALOAPI_KEY")
	if key == "" && !overridesOnly {
		fatal("LEVELUP_HALOAPI_KEY absent de l'env (clé d'abonnement www.haloapi.com)")
	}
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	metaPath := pr.MetadataDBPath(halo5.TitleSlug)

	// Provisionne le schéma metadata h5 (set isolé : medal_definitions, maps_catalog,
	// csr_designations…) avant le seed. Idempotent.
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	halo5migrations.Register()
	db, err := sql.Open("duckdb", metaPath)
	if err != nil {
		fatal("open metadata %s: %v", metaPath, err)
	}
	defer db.Close()
	if err := migration.RunForTitleDB(db, halo5.TitleSlug, migration.TargetMetadata); err != nil {
		fatal("provision metadata h5: %v", err)
	}
	fmt.Printf("metadata h5: %s\n", metaPath)

	// Overrides FR (versionnés worktree). Arg positionnel 1 = config root (défaut worktree).
	configRoot := defaultConfigRoot
	if len(positional) > 0 {
		configRoot = positional[0]
	}
	fr := loadFRLabels(filepath.Join(configRoot, "config", "titles", halo5.TitleSlug, "mappings", "asset_labels_fr.toml"))
	fmt.Printf("FR overrides: %d médailles, %d maps (nom EN), %d maps (par id), %d commendations\n",
		len(fr.Medals), len(fr.Maps), len(fr.MapsByID), len(fr.Commendations))

	if !overridesOnly {
		seedMedals(db, key, fr.Medals, fr.MedalDescriptions)
		seedMaps(db, key)
		seedWeapons(db, key)
		seedCSRDesignations(db, key)
		seedTeamColors(db, key)
		seedCommendations(db, key, fr.Commendations)
		seedPlaylists(db, key)

		// PONT catalogue → résolveur de noms. Sans cette étape, `asset_translations`
		// reste VIDE pour Halo 5 → ResolveAssetNamesBulk (home tile, match-view,
		// playlist favorite) renvoie 0 ligne → mode/carte/playlist s'affichent vides
		// PARTOUT. On peuple asset_translations depuis les catalogues déjà seedés
		// (playlists, maps_catalog) + l'endpoint officiel game-base-variants (modes).
		// À lancer EN DERNIER (dépend de playlists + maps_catalog seedés ci-dessus).
		seedAssetTranslations(db, key, fr)
	} else {
		fmt.Println("--overrides-only : fetch réseau sauté, application des overrides locaux uniquement")
	}

	// Overrides de map par asset_id — EN DERNIER (idempotent), pour survivre à un
	// re-fetch qui aurait réécrit name_canonical vide sur un canvas Forge.
	applyMapIDOverrides(db, fr.MapsByID)

	// Garde-fou : signale les maps référencées par le registre H5 sans nom résolu
	// dans asset_translations (retour silencieux du problème « carte vide »).
	logUnresolvedMaps(db, pr.SharedDBPath(halo5.TitleSlug))
}

// applyMapIDOverrides applique les overrides de nom de map keyés par asset_id :
// name_canonical (maps_catalog) + asset_translations en-US/fr-FR. Idempotent
// (UPDATE + INSERT OR REPLACE). Best-effort par entrée : une erreur est loguée et
// n'interrompt pas les autres.
func applyMapIDOverrides(db *sql.DB, overrides []mapIDOverride) {
	if len(overrides) == 0 {
		return
	}
	applied := 0
	for _, ov := range overrides {
		id := strings.TrimSpace(ov.ID)
		en := strings.TrimSpace(ov.EN)
		fr := strings.TrimSpace(ov.FR)
		if id == "" || en == "" {
			fmt.Printf("maps_by_id: entrée ignorée (id ou en vide): %+v\n", ov)
			continue
		}
		if fr == "" {
			fr = en
		}
		if _, err := db.Exec(
			`UPDATE maps_catalog SET name_canonical = ? WHERE title_slug = ? AND lower(map_asset_id) = lower(?)`,
			en, halo5.TitleSlug, id); err != nil {
			fmt.Printf("maps_by_id: update name_canonical %s: %v\n", id, err)
		}
		upsertAssetTranslation(db, id, "map", langEN, en)
		upsertAssetTranslation(db, id, "map", langFR, fr)
		applied++
	}
	fmt.Printf("maps_by_id: %d override(s) de map appliqué(s)\n", applied)
}

// logUnresolvedMaps ouvre le registre H5 en lecture seule et WARN (slog) la liste
// des map_id référencés par match_registry qui n'ont PAS d'entrée asset_translations
// (map, en-US) dans la metadata — le cas d'origine du bug « carte vide » (Tidal).
// Best-effort : registre indisponible (tenu RW par un writer) → skip loggé.
func logUnresolvedMaps(metaDB *sql.DB, registryPath string) {
	regDB, err := sql.Open("duckdb", registryPath+"?access_mode=read_only")
	if err != nil {
		slog.Warn("h5_metadata: garde-fou maps non résolues sauté (open registre)", "err", err)
		return
	}
	defer regDB.Close()

	resolved := readENNames(metaDB, "map") // asset_id → nom EN seedé
	rows, err := regDB.Query(
		`SELECT DISTINCT map_id FROM match_registry WHERE TRIM(COALESCE(map_id,'')) != ''`)
	if err != nil {
		slog.Warn("h5_metadata: garde-fou maps non résolues sauté (query registre)", "err", err)
		return
	}
	defer rows.Close()

	var unresolved []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if name, ok := resolved[id]; !ok || strings.TrimSpace(name) == "" {
			unresolved = append(unresolved, id)
		}
	}
	if len(unresolved) == 0 {
		slog.Info("h5_metadata: toutes les maps du registre sont résolues", "count_registry_maps", len(resolved))
		return
	}
	slog.Warn("h5_metadata: maps référencées par le registre SANS nom résolu (ajouter un override maps_by_id)",
		"count", len(unresolved), "map_ids", unresolved)
}

// apiPlaylist — élément de l'API Metadata officielle /playlists. `isRanked` fait foi
// pour classer match_registry.is_ranked (par playlist_id = UUID). Source autoritative
// du ranked H5 (cf. .ai, ne PAS dériver des parties).
type apiPlaylist struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	IsRanked bool   `json:"isRanked"`
}

// seedPlaylists peuple la table `playlists` (id, name, is_ranked) depuis l'API
// Metadata officielle. Table de référence (catalogue) créée à la volée si absente.
// Sert à classer is_ranked des matchs H5 (jointure offline sur playlist_id).
func seedPlaylists(db *sql.DB, key string) {
	body, err := fetchMeta(key, "playlists")
	if err != nil {
		fmt.Printf("playlists: SKIP (%v)\n", err)
		return
	}
	var pls []apiPlaylist
	if err := json.Unmarshal(body, &pls); err != nil {
		fmt.Printf("playlists: parse %v\n", err)
		return
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS playlists (
		id VARCHAR PRIMARY KEY, name VARCHAR, is_ranked BOOLEAN)`); err != nil {
		fmt.Printf("playlists: create table %v\n", err)
		return
	}
	n, ranked := 0, 0
	for _, p := range pls {
		if p.ID == "" {
			continue
		}
		if _, err := db.Exec(`INSERT OR REPLACE INTO playlists (id, name, is_ranked) VALUES (?,?,?)`,
			p.ID, strings.TrimSpace(p.Name), p.IsRanked); err != nil {
			fmt.Printf("playlists: insert %s: %v\n", p.ID, err)
			continue
		}
		n++
		if p.IsRanked {
			ranked++
		}
	}
	fmt.Printf("playlists: %d seedées (%d ranked) sur %d retournées\n", n, ranked, len(pls))
}

// fetchMeta récupère un type de métadonnée officiel (corps JSON brut) en EN
// (langue par défaut de l'API). Conserve la signature historique des seeders EN.
func fetchMeta(key, typ string) ([]byte, error) {
	return fetchMetaLang(key, typ, "")
}

// fetchMetaLang récupère un type de métadonnée officiel avec un override de langue.
// L'API Metadata Halo 5 honore l'en-tête HTTP `Accept-Language` (BCP-47, p.ex.
// "fr-FR") et renvoie les noms localisés des assets (maps, playlists, modes,
// armes, médailles, commendations). `lang` vide → en-tête NON posé (l'API sert
// l'EN par défaut), comportement identique à l'ancien fetchMeta.
func fetchMetaLang(key, typ, lang string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, officialMetaBase+typ, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", key)
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(lang) != "" {
		req.Header.Set("Accept-Language", lang)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d sur %s (lang=%q): %.200s", resp.StatusCode, typ, lang, body)
	}
	return body, nil
}

type apiMedal struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Classification string `json:"classification"`
	Difficulty     int    `json:"difficulty"`
	ID             string `json:"id"`
	SpriteLocation struct {
		SpriteSheetURI string `json:"spriteSheetUri"`
		Left           int    `json:"left"`
		Top            int    `json:"top"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
	} `json:"spriteLocation"`
}

func seedMedals(db *sql.DB, key string, fr, descFR map[string]string) {
	body, err := fetchMeta(key, "medals")
	if err != nil {
		fmt.Printf("medals: SKIP (%v)\n", err)
		return
	}
	var medals []apiMedal
	if err := json.Unmarshal(body, &medals); err != nil {
		fmt.Printf("medals: parse %v\n", err)
		return
	}
	n := 0
	for _, m := range medals {
		id, perr := strconv.ParseInt(m.ID, 10, 64)
		if perr != nil {
			continue // id non numérique → ignoré (medal_name_id = BIGINT)
		}
		// h5 : `difficulty` (0..245) n'est PAS l'enum 0-3 d'Infinite → stocké brut en
		// VARCHAR (difficulty), difficulty_index laissé à 0 (non applicable). medal_type
		// = classification normalisée vers les clés canoniques inter-titres
		// (multikill/spree/skill/style/mode/proficiency/other) via
		// canonical.NormalizeMedalCategory — sinon l'enum brut H5 (MultiKill, Style,
		// CaptureTheFlag…) court-circuite la traduction frontend (categoryLabels).
		// Icône = sprite (feuille + offset).
		_, err := db.Exec(`INSERT OR REPLACE INTO medal_definitions
			(medal_name_id, name_en, name_fr, description_en, description_fr,
			 difficulty_index, difficulty, medal_type,
			 sprite_sheet_url, sprite_left, sprite_top, sprite_width, sprite_height)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, m.Name, frOr(fr, m.Name), m.Description, lookupOr(descFR, m.Name, m.Description),
			0, strconv.Itoa(m.Difficulty), canonical.NormalizeMedalCategory(m.Classification),
			m.SpriteLocation.SpriteSheetURI, m.SpriteLocation.Left, m.SpriteLocation.Top,
			m.SpriteLocation.Width, m.SpriteLocation.Height)
		if err != nil {
			fmt.Printf("medals: insert %s: %v\n", m.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("medals: %d seedées (sur %d)\n", n, len(medals))
}

type apiMap struct {
	Name               string   `json:"name"`
	ImageURL           string   `json:"imageUrl"`
	SupportedGameModes []string `json:"supportedGameModes"`
	ID                 string   `json:"id"`
}

func seedMaps(db *sql.DB, key string) {
	body, err := fetchMeta(key, "maps")
	if err != nil {
		fmt.Printf("maps: SKIP (%v)\n", err)
		return
	}
	var maps []apiMap
	if err := json.Unmarshal(body, &maps); err != nil {
		fmt.Printf("maps: parse %v\n", err)
		return
	}
	n := 0
	now := time.Now()
	for _, m := range maps {
		if m.ID == "" {
			continue
		}
		_, err := db.Exec(`INSERT OR REPLACE INTO maps_catalog
			(title_slug, map_asset_id, name_canonical, image_url, last_fetched_at)
			VALUES (?,?,?,?,?)`,
			halo5.TitleSlug, m.ID, m.Name, m.ImageURL, now)
		if err != nil {
			fmt.Printf("maps: insert %s: %v\n", m.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("maps: %d seedées (sur %d)\n", n, len(maps))
}

type apiCSRDesignation struct {
	Name           string `json:"name"`
	BannerImageURL string `json:"bannerImageUrl"`
	Tiers          []struct {
		IconImageURL string `json:"iconImageUrl"`
		ID           string `json:"id"`
	} `json:"tiers"`
}

type apiWeapon struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	LargeIconImageURL string `json:"largeIconImageUrl"`
	ID                string `json:"id"`
}

// fetchWeaponsFR refetch /weapons en fr-FR (l'API Metadata HONORE Accept-Language, cf.
// fetchMetaLang) et indexe le nom FR par id d'arme. Best-effort : endpoint FR en échec
// ou parse KO -> map vide ; seedWeapons retombera alors sur le nom EN.
func fetchWeaponsFR(key string) map[string]string {
	out := map[string]string{}
	body, err := fetchMetaLang(key, "weapons", langFR)
	if err != nil {
		fmt.Printf("weapons[fr-FR]: SKIP (%v)\n", err)
		return out
	}
	var weapons []apiWeapon
	if err := json.Unmarshal(body, &weapons); err != nil {
		fmt.Printf("weapons[fr-FR]: parse %v\n", err)
		return out
	}
	for _, w := range weapons {
		if name := strings.TrimSpace(w.Name); name != "" && w.ID != "" {
			out[w.ID] = name
		}
	}
	return out
}

// seedWeapons peuple weapon_labels (id -> nom EN/FR, icône, type) depuis l'API
// Metadata officielle. name_fr : la localisation FR vient de l'API elle-même (pass
// fr-FR via fetchWeaponsFR) ; EN en dernier repli (jamais name_fr vide). Sans ce
// pass FR, les armes restaient en anglais (« lightrifle », « FRAG GRENADE ») côté
// match view.
//
// AUCUN override TOML ici (contre-revue V7.2) : la section [weapons] de
// asset_labels_fr.toml a été retirée par V72-06 au profit de weapon_names.toml
// keyé par weapon_key — la branche d'override par nom EN était inatteignable.
func seedWeapons(db *sql.DB, key string) {
	body, err := fetchMeta(key, "weapons")
	if err != nil {
		fmt.Printf("weapons: SKIP (%v)\n", err)
		return
	}
	var weapons []apiWeapon
	if err := json.Unmarshal(body, &weapons); err != nil {
		fmt.Printf("weapons: parse %v\n", err)
		return
	}
	// Pass FR : noms localisés par l'API (Accept-Language: fr-FR), indexés par id.
	frByID := fetchWeaponsFR(key)
	n, localized := 0, 0
	for _, w := range weapons {
		// id officiel = numérique (tient dans weapon_labels.weapon_id UBIGINT).
		id, perr := strconv.ParseInt(w.ID, 10, 64)
		if perr != nil {
			continue
		}
		// Pas d'override TOML pour les armes (cf. en-tête) : nom FR de l'API,
		// sinon EN. chooseFR reste la SEULE implémentation de cette cascade
		// (override nil = branche neutre) — pas de 2e variante à maintenir.
		nameFR := chooseFR(nil, w.Name, frByID[w.ID])
		_, err := db.Exec(`INSERT OR REPLACE INTO weapon_labels
			(weapon_id, name_en, name_fr, icon_url, weapon_type) VALUES (?,?,?,?,?)`,
			id, w.Name, nameFR, w.LargeIconImageURL, w.Type)
		if err != nil {
			fmt.Printf("weapons: insert %s: %v\n", w.ID, err)
			continue
		}
		n++
		if nameFR != w.Name {
			localized++
		}
	}
	fmt.Printf("weapons: %d seedées (sur %d, %d localisées FR)\n", n, len(weapons), localized)
}

func seedCSRDesignations(db *sql.DB, key string) {
	body, err := fetchMeta(key, "csr-designations")
	if err != nil {
		fmt.Printf("csr-designations: SKIP (%v)\n", err)
		return
	}
	var desigs []apiCSRDesignation
	if err := json.Unmarshal(body, &desigs); err != nil {
		fmt.Printf("csr-designations: parse %v\n", err)
		return
	}
	n := 0
	for _, d := range desigs {
		for _, t := range d.Tiers {
			tierID, perr := strconv.Atoi(t.ID)
			if perr != nil {
				continue
			}
			_, err := db.Exec(`INSERT OR REPLACE INTO csr_designations
				(designation_name, tier_id, icon_url, banner_url) VALUES (?,?,?,?)`,
				d.Name, tierID, t.IconImageURL, d.BannerImageURL)
			if err != nil {
				fmt.Printf("csr: insert %s/%s: %v\n", d.Name, t.ID, err)
				continue
			}
			n++
		}
	}
	fmt.Printf("csr-designations: %d tiers seedés (sur %d désignations)\n", n, len(desigs))
}

// apiTeamColor — élément de l'API Metadata officielle /team-colors. `id` = le TeamId
// porté par chaque équipe des résultats Stats API Halo 5 (H5Team.Id → match_participants.team_id).
// `name` (+ `description`) sont localisés via Accept-Language ; `color` est un hex
// "#RRGGBB" ; `iconUrl` est nullable. On persiste name/color/icône (la description
// n'est pas consommée par la Match View). NB : l'API sérialise `id` en STRING (ex.
// "0","1"), comme tous les autres id de cette API (medals/weapons/csr/commendations) →
// typé string et converti en int au persist (team_id INTEGER) ; un id non numérique est
// ignoré (dégradation gracieuse → l'exposition retombe sur le libellé d'équipe existant).
type apiTeamColor struct {
	Name    string `json:"name"`
	Color   string `json:"color"`
	IconURL string `json:"iconUrl"`
	ID      string `json:"id"`
}

// fetchTeamColorsFR refetch /team-colors en fr-FR (l'API HONORE Accept-Language, cf.
// fetchMetaLang) et indexe le nom FR par TeamId. Best-effort : échec/parse KO → map
// vide (le seed retombe alors sur le nom EN).
func fetchTeamColorsFR(key string) map[string]string {
	out := map[string]string{}
	body, err := fetchMetaLang(key, "team-colors", langFR)
	if err != nil {
		fmt.Printf("team-colors[fr-FR]: SKIP (%v)\n", err)
		return out
	}
	var colors []apiTeamColor
	if err := json.Unmarshal(body, &colors); err != nil {
		fmt.Printf("team-colors[fr-FR]: parse %v\n", err)
		return out
	}
	for _, c := range colors {
		if n := strings.TrimSpace(c.Name); n != "" {
			out[c.ID] = n
		}
	}
	return out
}

// seedTeamColors peuple team_colors depuis l'API Metadata officielle (/team-colors) :
// TeamId → couleur (#RRGGBB) + nom localisé EN/FR + icône. Source des libellés d'équipe
// « Rouge/Bleu » de la Match View H5 (jointure read-time sur team_id). Best-effort.
func seedTeamColors(db *sql.DB, key string) {
	body, err := fetchMeta(key, "team-colors")
	if err != nil {
		fmt.Printf("team-colors: SKIP (%v)\n", err)
		return
	}
	var colors []apiTeamColor
	if err := json.Unmarshal(body, &colors); err != nil {
		fmt.Printf("team-colors: parse %v\n", err)
		return
	}
	n := persistTeamColors(db, colors, fetchTeamColorsFR(key))
	fmt.Printf("team-colors: %d seedées (sur %d)\n", n, len(colors))
}

// persistTeamColors écrit les team_colors en INSERT OR REPLACE (idempotent). frByID
// porte les noms FR localisés par l'API (Accept-Language), indexés par l'id string de
// l'API ; fallback sur le nom EN (jamais name_fr vide). L'id string est converti en int
// pour team_id (INTEGER) ; un id non numérique est ignoré (best-effort par entrée).
// Retourne le nombre de lignes écrites.
func persistTeamColors(db *sql.DB, colors []apiTeamColor, frByID map[string]string) int {
	n := 0
	for _, c := range colors {
		// id officiel = entier sérialisé en string (spec API /team-colors) → team_id INTEGER.
		teamID, perr := strconv.Atoi(strings.TrimSpace(c.ID))
		if perr != nil {
			fmt.Printf("team-colors: id non numérique %q: %v\n", c.ID, perr)
			continue
		}
		nameEN := strings.TrimSpace(c.Name)
		// Pas d'override TOML pour les couleurs d'équipe (contrairement aux
		// armes/médailles) → sélection directe : nom FR de l'API sinon EN.
		nameFR := strings.TrimSpace(frByID[c.ID])
		if nameFR == "" {
			nameFR = nameEN
		}
		if _, err := db.Exec(`INSERT OR REPLACE INTO team_colors
			(team_id, name_en, name_fr, color, icon_url) VALUES (?,?,?,?,?)`,
			teamID, nameEN, nameFR, strings.TrimSpace(c.Color), strings.TrimSpace(c.IconURL)); err != nil {
			fmt.Printf("team-colors: insert %d: %v\n", teamID, err)
			continue
		}
		n++
	}
	return n
}

// apiCommendation — élément de l'API Metadata officielle /commendations. `id` (UUID)
// = la clé naturelle référencée par carnage ProgressiveCommendationDeltas[].Id.
//
// `levels[]` (Progressive uniquement) porte les PALIERS : chaque level a un
// `threshold` = score cumulé cible du palier (≈ 5 levels/commendation, p.ex.
// "Spartan Slayer" → [1, 41, …]). On extrait la suite croissante des thresholds
// (CSV) pour réutiliser la mécanique de progression des citations Infinite
// (parseTierTargets/ComputeTierProgression).
type apiCommendation struct {
	Type         string `json:"type"` // Progressive | Meta | Daily
	Name         string `json:"name"`
	Description  string `json:"description"`
	IconImageURL string `json:"iconImageUrl"`
	ID           string `json:"id"`
	Category     struct {
		Name string `json:"name"`
	} `json:"category"`
	Levels []struct {
		Threshold int `json:"threshold"`
	} `json:"levels"`
}

// tierTargetsCSV projette les paliers d'une commendation en CSV croissant
// IDENTIQUE au format citation_mappings.tier_targets d'Infinite (réutilise
// analysis.ParseTierTargets / ComputeTierProgression côté lecture). Seuils <= 0
// ignorés ; tri croissant ; chaîne vide si aucun palier exploitable (commendations
// Meta/Daily sans levels → dégradation propre : anneau vide côté front).
func tierTargetsCSV(levels []struct {
	Threshold int `json:"threshold"`
}) string {
	thresholds := make([]int, 0, len(levels))
	for _, l := range levels {
		if l.Threshold > 0 {
			thresholds = append(thresholds, l.Threshold)
		}
	}
	if len(thresholds) == 0 {
		return ""
	}
	sort.Ints(thresholds)
	parts := make([]string, len(thresholds))
	for i, t := range thresholds {
		parts[i] = strconv.Itoa(t)
	}
	return strings.Join(parts, ",")
}

// commFR — nom + description FR d'une commendation (Accept-Language: fr-FR), indexés
// par UUID pour alimenter name_fr / description_fr au seed.
type commFR struct {
	name string
	desc string
}

// fetchCommendationsFR refetch /commendations en fr-FR (l'API Metadata HONORE
// Accept-Language, cf. fetchMetaLang) et indexe nom+description par UUID. Best-effort :
// endpoint FR en échec ou parse KO → map vide ; chooseFR retombera alors sur l'EN.
func fetchCommendationsFR(key string) map[string]commFR {
	out := map[string]commFR{}
	body, err := fetchMetaLang(key, "commendations", langFR)
	if err != nil {
		fmt.Printf("commendations[fr-FR]: SKIP (%v)\n", err)
		return out
	}
	var comms []apiCommendation
	if err := json.Unmarshal(body, &comms); err != nil {
		fmt.Printf("commendations[fr-FR]: parse %v\n", err)
		return out
	}
	for _, c := range comms {
		if c.ID == "" {
			continue
		}
		out[c.ID] = commFR{name: strings.TrimSpace(c.Name), desc: strings.TrimSpace(c.Description)}
	}
	return out
}

// seedCommendations peuple commendation_definitions depuis l'API Metadata officielle
// (/commendations) → nom/description/icône CDN + type + catégorie, par UUID. La clé
// `id` est exactement le ProgressiveCommendationDeltas[].Id du carnage (jointure
// read-time pour peupler Name/IconURL des commendations, cf.
// halo_5.enrichCommendationTotals). Tous les types (Progressive/Meta/Daily) sont seedés.
//
// name_fr / description_fr : la localisation FR vient de l'API elle-même (pass fr-FR
// via fetchCommendationsFR), les overrides TOML restant prioritaires (cf. chooseFR).
// Les noms de l'API portent des espaces parasites en fin → TrimSpace.
func seedCommendations(db *sql.DB, key string, fr map[string]string) {
	body, err := fetchMeta(key, "commendations")
	if err != nil {
		fmt.Printf("commendations: SKIP (%v)\n", err)
		return
	}
	var comms []apiCommendation
	if err := json.Unmarshal(body, &comms); err != nil {
		fmt.Printf("commendations: parse %v\n", err)
		return
	}
	// Pass FR : noms/descriptions localisés par l'API, indexés par UUID.
	frByID := fetchCommendationsFR(key)
	// Idempotent : garantit la colonne tier_targets même si la DB a été provisionnée
	// avant l'ajout de la migration (parité ALTER idempotent côté schéma).
	if _, err := db.Exec(`ALTER TABLE commendation_definitions ADD COLUMN IF NOT EXISTS tier_targets VARCHAR`); err != nil {
		fmt.Printf("commendations: ensure tier_targets column: %v\n", err)
	}
	n, withTiers, localized := 0, 0, 0
	for _, c := range comms {
		if c.ID == "" {
			continue // pas de clé naturelle → ignoré
		}
		nameEN := strings.TrimSpace(c.Name)
		apiFR := frByID[c.ID]
		nameFR := chooseFR(fr, nameEN, apiFR.name)
		tierTargets := tierTargetsCSV(c.Levels)
		_, err := db.Exec(`INSERT OR REPLACE INTO commendation_definitions
			(commendation_id, name_en, name_fr, description_en, description_fr,
			 commendation_type, category, icon_url, tier_targets)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			c.ID, nameEN, nameFR, strings.TrimSpace(c.Description), apiFR.desc,
			c.Type, c.Category.Name, c.IconImageURL, tierTargets)
		if err != nil {
			fmt.Printf("commendations: insert %s: %v\n", c.ID, err)
			continue
		}
		n++
		if tierTargets != "" {
			withTiers++
		}
		if nameFR != nameEN {
			localized++
		}
	}
	fmt.Printf("commendations: %d seedées (sur %d, %d avec paliers, %d localisées FR)\n",
		n, len(comms), withTiers, localized)
}

// langEN / langFR — codes de langue attendus par le résolveur de noms
// (ResolveAssetNamesBulk → PreferredLangsForLocale). IMPÉRATIF : matcher
// EXACTEMENT le format BCP-47 long utilisé côté lecture (cf.
// internal/platform/duckdb/medal_definitions_repo.go : LangCodeEN="en-US",
// LangCodeFR="fr-FR" ; et PreferredLangsForLocale qui cascade {fr-FR, fr, en-US,
// en}). Un "en"/"fr" court ne matcherait que par le fallback alphabétique, jamais
// la préférence → résolution fragile. On duplique les littéraux ici plutôt que
// d'importer le package duckdb (ce CLI n'a aucune dépendance duckdb runtime, et le
// package migrations isole déjà la metadata h5).
const (
	langEN = "en-US"
	langFR = "fr-FR"
)

// apiGameBaseVariant — élément de l'API Metadata officielle /game-base-variants.
// `id` = le GameBaseVariantId porté par chaque match Halo 5 (cf.
// internal/games/halo_5/mapping.go : GameVariant = assetRef("game_variant",
// r.GameBaseVariantId)). Le NOM du mode (EN) vit ici, pas dans la donnée de match.
// Shape minimale alignée sur apiPlaylist/apiMap (tous les types officiels exposent
// `name` + `id`).
type apiGameBaseVariant struct {
	Name string `json:"name"`
	ID   string `json:"id"`
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
// FR (deux sources complémentaires) :
//  1. Overrides versionnés (section [maps] du TOML, fr.Maps[name_en]) — prioritaires.
//  2. Pass FR via l'API officielle : l'API Metadata Halo 5 HONORE l'en-tête
//     `Accept-Language` (info confirmée) et renvoie les noms localisés. On refetch
//     /playlists, /maps, /game-base-variants en fr-FR et on seede lang='fr-FR'.
//     Certains assets (noms propres de maps) ne sont pas localisés → l'API renvoie
//     le même nom qu'en EN ; on seede quand même (la cascade prend fr-FR ; si == EN,
//     dégradation propre) et on logge un compteur "!= EN" pour visibilité.
//
// Idempotent (INSERT OR REPLACE sur PK asset_id+asset_type+lang). Best-effort : une
// insertion ou un endpoint FR qui échoue est logué et n'interrompt pas le pass EN.
func seedAssetTranslations(db *sql.DB, key string, fr frLabels) {
	n := 0
	n += seedPlaylistTranslations(db)
	n += seedMapTranslations(db, fr.Maps)
	n += seedModeTranslations(db, key)
	fmt.Printf("asset_translations[en-US]: %d seedées\n", n)
	// Pass FR (best-effort, ne casse pas le pass EN ci-dessus).
	nfr := seedFrenchTranslations(db, key)
	fmt.Printf("asset_translations[fr-FR]: %d seedées\n", nfr)
}

// frTransResult — bilan d'un pass FR par asset_type (visibilité du garde-fou
// "fr-FR == en-US" décrit ci-dessus).
type frTransResult struct {
	seeded int // lignes fr-FR insérées
	diff   int // dont nom FR strictement != nom EN connu
	total  int // assets retournés par l'API en fr-FR
}

// seedFrenchTranslations refetch les endpoints localisables en fr-FR via
// Accept-Language et seede asset_translations lang='fr-FR'. Pour chaque asset_type
// on compare le nom FR au nom EN déjà seedé (lecture asset_translations en-US) afin
// de tracer combien sont RÉELLEMENT localisés. Best-effort : un endpoint qui échoue
// est logué et n'empêche pas les autres.
func seedFrenchTranslations(db *sql.DB, key string) int {
	enPlaylist := readENNames(db, "playlist")
	enMap := readENNames(db, "map")
	enVariant := readENNames(db, "game_variant")

	pl := seedFrenchSimple(db, key, "playlists", "playlist", enPlaylist)
	mp := seedFrenchSimple(db, key, "maps", "map", enMap)
	gv := seedFrenchSimple(db, key, "game-base-variants", "game_variant", enVariant)

	fmt.Printf("fr-FR localisés (!= en-US) : playlists %d/%d, maps %d/%d, modes %d/%d\n",
		pl.diff, pl.total, mp.diff, mp.total, gv.diff, gv.total)
	return pl.seeded + mp.seeded + gv.seeded
}

// readENNames relit les noms en-US déjà seedés pour un asset_type (clé = asset_id).
// Sert de référence pour le compteur "fr != en". Erreur → map vide (le compteur
// diff comptera alors 0, sans casser le seed FR).
func readENNames(db *sql.DB, assetType string) map[string]string {
	out := map[string]string{}
	rows, err := db.Query(
		`SELECT asset_id, name FROM asset_translations WHERE asset_type = ? AND lang = ?`,
		assetType, langEN)
	if err != nil {
		fmt.Printf("asset_translations[%s]: read en-US ref %v\n", assetType, err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		out[id] = name
	}
	return out
}

// seedFrenchSimple fetche un endpoint en fr-FR (shape {id,name} commune à playlists,
// maps, game-base-variants) et seede asset_translations(asset_type, lang='fr-FR').
// On compare au nom EN connu pour le compteur de localisation. Best-effort.
func seedFrenchSimple(db *sql.DB, key, typ, assetType string, enNames map[string]string) frTransResult {
	res := frTransResult{}
	body, err := fetchMetaLang(key, typ, langFR)
	if err != nil {
		fmt.Printf("asset_translations[%s][fr-FR]: SKIP (%v)\n", assetType, err)
		return res
	}
	// /maps expose l'id sous map_asset_id côté catalogue mais l'API le renvoie
	// toujours dans le champ `id` (cf. apiMap/apiPlaylist/apiGameBaseVariant) → shape
	// commune {id,name}.
	var items []apiGameBaseVariant
	if err := json.Unmarshal(body, &items); err != nil {
		fmt.Printf("asset_translations[%s][fr-FR]: parse %v\n", assetType, err)
		return res
	}
	res.total = len(items)
	for _, it := range items {
		nameFR := strings.TrimSpace(it.Name)
		if !upsertAssetTranslation(db, it.ID, assetType, langFR, nameFR) {
			continue
		}
		res.seeded++
		if en, ok := enNames[it.ID]; ok && nameFR != strings.TrimSpace(en) {
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

// seedPlaylistTranslations relit la table `playlists` (déjà seedée par
// seedPlaylists) et écrit asset_type='playlist' lang='en-US'. EN seul (aucune trad
// FR de playlist disponible).
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
			continue
		}
		if upsertAssetTranslation(db, id, "playlist", langEN, name) {
			n++
		}
	}
	return n
}

// seedMapTranslations relit `maps_catalog` (déjà seedée par seedMaps) et écrit
// asset_type='map' lang='en-US' (name_canonical) + lang='fr-FR' si une trad existe
// dans la section [maps] du TOML (fr.Maps[name_en]).
func seedMapTranslations(db *sql.DB, frMaps map[string]string) int {
	rows, err := db.Query(`SELECT map_asset_id, name_canonical FROM maps_catalog
		WHERE title_slug = ? AND TRIM(COALESCE(name_canonical,'')) != ''`, halo5.TitleSlug)
	if err != nil {
		fmt.Printf("asset_translations[map]: read maps_catalog %v\n", err)
		return 0
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var id, nameEN string
		if err := rows.Scan(&id, &nameEN); err != nil {
			continue
		}
		if upsertAssetTranslation(db, id, "map", langEN, nameEN) {
			n++
		}
		if frName, ok := frMaps[nameEN]; ok && strings.TrimSpace(frName) != "" {
			if upsertAssetTranslation(db, id, "map", langFR, frName) {
				n++
			}
		}
	}
	return n
}

// seedModeTranslations fetche l'endpoint officiel /game-base-variants et écrit
// asset_type='game_variant' lang='en-US'. C'est la SOURCE de nom de mode pour Halo 5
// (le match ne porte que le GameBaseVariantId, jamais le nom). EN seul (l'API ne
// localise pas ; aucune section modes dans le TOML).
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
		if upsertAssetTranslation(db, v.ID, "game_variant", langEN, v.Name) {
			n++
		}
	}
	return n
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
