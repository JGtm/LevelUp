package main

// definitions_assets.go — commendations et couleurs d'équipe de Halo 5 : colonnes non textuelles
// dans commendation_definitions / team_colors, noms et descriptions dans asset_translations
// (types « commendation » et « team_color »), une ligne par langue de config/locales.toml pour
// laquelle le seed a un texte (ADR 0040 D-5). Médailles : definitions.go.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/ops"
	ddb "levelup/go-api/internal/platform/duckdb"
)

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

// fetchTeamColorNames refetch /team-colors dans la langue `l` (l'API HONORE Accept-Language,
// cf. fetchMetaLang) et indexe le nom par TeamId. Best-effort : échec ou parse KO → map vide
// (la langue n'écrit alors rien).
func fetchTeamColorNames(key string, l i18n.Locale) map[string]string {
	out := map[string]string{}
	body, err := fetchMetaLang(key, "team-colors", acceptLanguage(l))
	if err != nil {
		fmt.Printf("team-colors[%s]: SKIP (%v)\n", l, err)
		return out
	}
	var colors []apiTeamColor
	if err := json.Unmarshal(body, &colors); err != nil {
		fmt.Printf("team-colors[%s]: parse %v\n", l, err)
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
// TeamId → couleur (#RRGGBB) + icône, et le nom dans chaque langue de l'application. Source des
// libellés d'équipe « Rouge/Bleu » de la Match View H5 (jointure read-time sur team_id).
// Best-effort.
func seedTeamColors(ctx context.Context, db *sql.DB, key string) {
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
	localized := map[i18n.Locale]map[string]string{}
	for _, l := range i18n.Locales() {
		if l != i18n.Fallback {
			localized[l] = fetchTeamColorNames(key, l)
		}
	}
	n := persistTeamColors(ctx, db, colors, localized)
	fmt.Printf("team-colors: %d seedées (sur %d)\n", n, len(colors))
}

// teamColorNames : le nom d'une couleur d'équipe par langue — la langue de l'API depuis la
// réponse sans en-tête, les autres depuis leur propre passe (`localized`, indexé par l'id de
// l'API). Une langue sans nom n'écrit rien.
func teamColorNames(c apiTeamColor, localized map[i18n.Locale]map[string]string) i18n.Text {
	names := i18n.Text{}
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			setText(names, l, c.Name)
			continue
		}
		setText(names, l, localized[l][c.ID])
	}
	return names
}

// persistTeamColors écrit les team_colors (colonnes non textuelles en INSERT OR REPLACE,
// idempotent) puis leurs noms par langue dans asset_translations. L'id string est converti en
// int pour team_id (INTEGER) ; un id non numérique est ignoré (best-effort par entrée).
// Retourne le nombre de couleurs écrites.
func persistTeamColors(ctx context.Context, db *sql.DB, colors []apiTeamColor, localized map[i18n.Locale]map[string]string) int {
	n := 0
	for _, c := range colors {
		// id officiel = entier sérialisé en string (spec API /team-colors) → team_id INTEGER.
		teamID, perr := strconv.Atoi(strings.TrimSpace(c.ID))
		if perr != nil {
			fmt.Printf("team-colors: id non numérique %q: %v\n", c.ID, perr)
			continue
		}
		if _, err := db.Exec(`INSERT OR REPLACE INTO team_colors
			(team_id, color, icon_url) VALUES (?,?,?)`,
			teamID, strings.TrimSpace(c.Color), strings.TrimSpace(c.IconURL)); err != nil {
			fmt.Printf("team-colors: insert %d: %v\n", teamID, err)
			continue
		}
		// Pas d'override TOML pour les couleurs d'équipe (contrairement aux armes/médailles).
		if err := ops.UpsertAssetTexts(ctx, db, ddb.AssetTypeTeamColor, strconv.Itoa(teamID), teamColorNames(c, localized), nil); err != nil {
			fmt.Printf("team-colors: textes %d: %v\n", teamID, err)
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

// commText — nom + description d'une commendation dans une langue, indexés par UUID.
type commText struct {
	name string
	desc string
}

// fetchCommendationTexts refetch /commendations dans la langue `l` (l'API Metadata HONORE
// Accept-Language, cf. fetchMetaLang) et indexe nom+description par UUID. Best-effort : endpoint
// en échec ou parse KO → map vide (la langue n'écrit alors que ses overrides TOML).
func fetchCommendationTexts(key string, l i18n.Locale) map[string]commText {
	out := map[string]commText{}
	body, err := fetchMetaLang(key, "commendations", acceptLanguage(l))
	if err != nil {
		fmt.Printf("commendations[%s]: SKIP (%v)\n", l, err)
		return out
	}
	var comms []apiCommendation
	if err := json.Unmarshal(body, &comms); err != nil {
		fmt.Printf("commendations[%s]: parse %v\n", l, err)
		return out
	}
	for _, c := range comms {
		if c.ID == "" {
			continue
		}
		out[c.ID] = commText{name: strings.TrimSpace(c.Name), desc: strings.TrimSpace(c.Description)}
	}
	return out
}

// commendationTexts rassemble le nom et la description d'une commendation par langue : la
// langue de l'API vient de la réponse sans en-tête ; une autre langue prend d'abord l'override
// TOML versionné (corrections manuelles, clé = nom de l'API) puis le texte localisé par l'API
// (`localized`, une passe par langue). Une langue sans texte n'écrit rien : la lecture se replie
// sur la langue de l'API, et la description d'une langue n'est jamais celle d'une autre.
func commendationTexts(c apiCommendation, labels assetLabels, localized map[i18n.Locale]map[string]commText) (names, descs i18n.Text) {
	names, descs = i18n.Text{}, i18n.Text{}
	nameAPI := strings.TrimSpace(c.Name)
	for _, l := range i18n.Locales() {
		if l == i18n.Fallback {
			setText(names, l, nameAPI)
			setText(descs, l, c.Description)
			continue
		}
		api := localized[l][c.ID]
		if v := strings.TrimSpace(labels.Commendations[nameAPI][l]); v != "" {
			setText(names, l, v)
		} else {
			setText(names, l, api.name)
		}
		setText(descs, l, api.desc)
	}
	return names, descs
}

// persistCommendation écrit une commendation : ses colonnes non textuelles (INSERT OR REPLACE,
// idempotent) puis ses textes par langue dans asset_translations.
func persistCommendation(ctx context.Context, db *sql.DB, c apiCommendation, labels assetLabels, localized map[i18n.Locale]map[string]commText) error {
	if _, err := db.Exec(`INSERT OR REPLACE INTO commendation_definitions
		(commendation_id, commendation_type, category, icon_url, tier_targets)
		VALUES (?,?,?,?,?)`,
		c.ID, c.Type, c.Category.Name, c.IconImageURL, tierTargetsCSV(c.Levels)); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	names, descs := commendationTexts(c, labels, localized)
	if err := ops.UpsertAssetTexts(ctx, db, ddb.AssetTypeCommendation, c.ID, names, descs); err != nil {
		return fmt.Errorf("textes: %w", err)
	}
	return nil
}

// seedCommendations peuple commendation_definitions depuis l'API Metadata officielle
// (/commendations) → icône CDN + type + catégorie + paliers, par UUID, et leurs noms et
// descriptions par langue dans asset_translations. La clé `id` est exactement le
// ProgressiveCommendationDeltas[].Id du carnage (jointure read-time pour peupler Name/IconURL
// des commendations, cf. halo_5.enrichCommendationTotals). Tous les types
// (Progressive/Meta/Daily) sont seedés. Les noms de l'API portent des espaces parasites en
// fin → TrimSpace.
func seedCommendations(ctx context.Context, db *sql.DB, key string, labels assetLabels) {
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
	localized := map[i18n.Locale]map[string]commText{}
	for _, l := range i18n.Locales() {
		if l != i18n.Fallback {
			localized[l] = fetchCommendationTexts(key, l)
		}
	}
	// Idempotent : garantit la colonne tier_targets même si la DB a été provisionnée
	// avant l'ajout de la migration (parité ALTER idempotent côté schéma).
	if _, err := db.Exec(`ALTER TABLE commendation_definitions ADD COLUMN IF NOT EXISTS tier_targets VARCHAR`); err != nil {
		fmt.Printf("commendations: ensure tier_targets column: %v\n", err)
	}
	n, withTiers := 0, 0
	for _, c := range comms {
		if c.ID == "" {
			continue // pas de clé naturelle → ignoré
		}
		if err := persistCommendation(ctx, db, c, labels, localized); err != nil {
			fmt.Printf("commendations: %s: %v\n", c.ID, err)
			continue
		}
		n++
		if tierTargetsCSV(c.Levels) != "" {
			withTiers++
		}
	}
	fmt.Printf("commendations: %d seedées (sur %d, %d avec paliers)\n", n, len(comms), withTiers)
}
