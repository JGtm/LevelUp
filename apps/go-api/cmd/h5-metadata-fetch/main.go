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
	"net/http"
	"os"
	"path/filepath"
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
type frLabels struct {
	Weapons           map[string]string `toml:"weapons"`
	Medals            map[string]string `toml:"medals"`
	Maps              map[string]string `toml:"maps"`
	MedalDescriptions map[string]string `toml:"medal_descriptions"`
	Commendations     map[string]string `toml:"commendations"`
}

// loadFRLabels lit les overrides FR. Fichier absent / illisible → maps vides
// (name_fr = name_en, dégradation propre). Best-effort.
func loadFRLabels(path string) frLabels {
	fr := frLabels{Weapons: map[string]string{}, Medals: map[string]string{}, Maps: map[string]string{}, MedalDescriptions: map[string]string{}, Commendations: map[string]string{}}
	b, err := os.ReadFile(path)
	if err != nil {
		return fr
	}
	_ = toml.Unmarshal(b, &fr)
	if fr.Weapons == nil {
		fr.Weapons = map[string]string{}
	}
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
	return lookupOr(m, en, en)
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
	key := os.Getenv("LEVELUP_HALOAPI_KEY")
	if key == "" {
		fatal("LEVELUP_HALOAPI_KEY absent de l'env (clé d'abonnement www.haloapi.com)")
	}
	cfg, err := config.Load()
	if err != nil {
		fatal("config.Load: %v", err)
	}
	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(halo5.TitleSlug)

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

	// Overrides FR (versionnés worktree). Arg1 = config root (défaut worktree).
	configRoot := defaultConfigRoot
	if len(os.Args) > 1 {
		configRoot = os.Args[1]
	}
	fr := loadFRLabels(filepath.Join(configRoot, "config", "titles", halo5.TitleSlug, "mappings", "asset_labels_fr.toml"))
	fmt.Printf("FR overrides: %d armes, %d médailles, %d maps, %d commendations\n", len(fr.Weapons), len(fr.Medals), len(fr.Maps), len(fr.Commendations))

	seedMedals(db, key, fr.Medals, fr.MedalDescriptions)
	seedMaps(db, key)
	seedWeapons(db, key, fr.Weapons)
	seedCSRDesignations(db, key)
	seedCommendations(db, key, fr.Commendations)
	seedPlaylists(db, key)
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

// fetchMeta récupère un type de métadonnée officiel (corps JSON brut).
func fetchMeta(key, typ string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, officialMetaBase+typ, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", key)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d sur %s: %.200s", resp.StatusCode, typ, body)
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

func seedWeapons(db *sql.DB, key string, fr map[string]string) {
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
	n := 0
	for _, w := range weapons {
		// id officiel = numérique (tient dans weapon_labels.weapon_id UBIGINT).
		id, perr := strconv.ParseInt(w.ID, 10, 64)
		if perr != nil {
			continue
		}
		_, err := db.Exec(`INSERT OR REPLACE INTO weapon_labels
			(weapon_id, name_en, name_fr, icon_url, weapon_type) VALUES (?,?,?,?,?)`,
			id, w.Name, frOr(fr, w.Name), w.LargeIconImageURL, w.Type)
		if err != nil {
			fmt.Printf("weapons: insert %s: %v\n", w.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("weapons: %d seedées (sur %d)\n", n, len(weapons))
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

// apiCommendation — élément de l'API Metadata officielle /commendations. `id` (UUID)
// = la clé naturelle référencée par carnage ProgressiveCommendationDeltas[].Id.
type apiCommendation struct {
	Type         string `json:"type"` // Progressive | Meta | Daily
	Name         string `json:"name"`
	Description  string `json:"description"`
	IconImageURL string `json:"iconImageUrl"`
	ID           string `json:"id"`
	Category     struct {
		Name string `json:"name"`
	} `json:"category"`
}

// seedCommendations peuple commendation_definitions depuis l'API Metadata officielle
// (/commendations) → nom/description/icône CDN + type + catégorie, par UUID. La clé
// `id` est exactement le ProgressiveCommendationDeltas[].Id du carnage (jointure
// read-time pour peupler canonical.Commendation Name/IconURL). Tous les types
// (Progressive/Meta/Daily) sont seedés. name_fr = override FR sinon EN. Les noms de
// l'API portent des espaces parasites en fin → TrimSpace.
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
	n := 0
	for _, c := range comms {
		if c.ID == "" {
			continue // pas de clé naturelle → ignoré
		}
		nameEN := strings.TrimSpace(c.Name)
		_, err := db.Exec(`INSERT OR REPLACE INTO commendation_definitions
			(commendation_id, name_en, name_fr, description_en, description_fr,
			 commendation_type, category, icon_url)
			VALUES (?,?,?,?,?,?,?,?)`,
			c.ID, nameEN, frOr(fr, nameEN), strings.TrimSpace(c.Description), "",
			c.Type, c.Category.Name, c.IconImageURL)
		if err != nil {
			fmt.Printf("commendations: insert %s: %v\n", c.ID, err)
			continue
		}
		n++
	}
	fmt.Printf("commendations: %d seedées (sur %d)\n", n, len(comms))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
