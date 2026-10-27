// Outil ops : peuple la metadata.duckdb Halo 5 (médailles, cartes, désignations CSR)
// depuis l'API Metadata OFFICIELLE Halo 5 (www.haloapi.com). Ces référentiels
// canoniques ne sont PAS sur les endpoints internes SpartanToken (cf.
// .ai/archive/V7/PLAN_H5_ASSETS.md) — seule l'API officielle les expose. Auth = clé
// d'abonnement Azure APIM (Ocp-Apim-Subscription-Key), lue dans l'env
// LEVELUP_HALOAPI_KEY — JAMAIS committée. Données Halo 5 figées → seed one-shot
// idempotent (INSERT OR REPLACE).
//
// Usage : LEVELUP_HALOAPI_KEY=<clé> LEVELUP_REPO_ROOT=<repo> go run ./cmd/h5-metadata-fetch
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	halo5migrations "levelup/go-api/internal/games/halo_5/migrations"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

const officialMetaBase = "https://www.haloapi.com/metadata/h5/metadata/"

// defaultConfigRoot : racine portant config/titles/<slug>/mappings (worktree, où
// vivent les TOML versionnés — distinct de cfg.RepoRoot = clone runtime de données).
const defaultConfigRoot = "c:/Users/Guillaume/Downloads/Scripts/levelup-multititre"

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

	// Overrides de noms par langue (versionnés worktree). Arg positionnel 1 = config root (défaut worktree).
	configRoot := defaultConfigRoot
	if len(positional) > 0 {
		configRoot = positional[0]
	}
	labels, err := loadAssetLabels(filepath.Join(configRoot, "config", "titles", halo5.TitleSlug, "mappings", "asset_labels.toml"))
	if err != nil {
		fatal("asset_labels: %v", err)
	}
	fmt.Printf("overrides de noms : %d médailles, %d maps (nom EN), %d maps (par id), %d commendations\n",
		len(labels.Medals), len(labels.Maps), len(labels.MapsByID), len(labels.Commendations))
	ctx := context.Background()

	if !overridesOnly {
		seedMedals(ctx, db, key, labels)
		seedMaps(db, key)
		seedWeapons(db, key)
		seedCSRDesignations(db, key)
		seedTeamColors(ctx, db, key)
		seedCommendations(ctx, db, key, labels)
		seedPlaylists(db, key)

		// PONT catalogue → résolveur de noms. Sans cette étape, `asset_translations`
		// reste VIDE pour Halo 5 → ResolveAssetNamesBulk (home tile, match-view,
		// playlist favorite) renvoie 0 ligne → mode/carte/playlist s'affichent vides
		// PARTOUT. On peuple asset_translations depuis les catalogues déjà seedés
		// (playlists, maps_catalog) + l'endpoint officiel game-base-variants (modes).
		// À lancer EN DERNIER (dépend de playlists + maps_catalog seedés ci-dessus).
		seedAssetTranslations(db, key, labels)
	} else {
		fmt.Println("--overrides-only : fetch réseau sauté, application des overrides locaux uniquement")
	}

	// Overrides de map par asset_id — EN DERNIER (idempotent), pour survivre à un
	// re-fetch qui aurait réécrit name_canonical vide sur un canvas Forge.
	applyMapIDOverrides(db, labels.MapsByID)

	// Garde-fou : signale les maps référencées par le registre H5 sans nom résolu
	// dans asset_translations (retour silencieux du problème « carte vide »).
	logUnresolvedMaps(db, pr.SharedDBPath(halo5.TitleSlug))
}

// logUnresolvedMaps ouvre le registre H5 en lecture seule et WARN (slog) la liste
// des map_id référencés par match_registry qui n'ont PAS d'entrée asset_translations
// (map, langue de l'API) dans la metadata — le cas d'origine du bug « carte vide » (Tidal).
// Best-effort : registre indisponible (tenu RW par un writer) → skip loggé.
func logUnresolvedMaps(metaDB *sql.DB, registryPath string) {
	regDB, err := sql.Open("duckdb", registryPath+"?access_mode=read_only")
	if err != nil {
		slog.Warn("h5_metadata: garde-fou maps non résolues sauté (open registre)", "err", err)
		return
	}
	defer regDB.Close()

	resolved := readFallbackNames(metaDB, "map") // asset_id → nom seedé dans la langue de l'API
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
// "es-ES") et renvoie les noms localisés des assets (maps, playlists, modes,
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

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
