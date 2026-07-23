// Package duckdb - season_pass_repo_helpers.go : helpers (free rewards,
// currency, payload resolvers, localized text, status computation, ptr
// constructors). Decoupe de season_pass_repo.go (god-file split,
// refactor 2026-05-27).
package duckdb

import (
	"database/sql"
	"fmt"
	"path"
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
)

func selectTierPreview(
	rank seasonPassRankRaw,
	itemMap map[string]seasonPassItemMeta,
	preferPremium bool,
) (seasonPassItemMeta, bool) {
	buckets := []struct {
		bucket    seasonPassRewardBucket
		isPremium bool
	}{
		{bucket: rank.FreeRewards, isPremium: false},
		{bucket: rank.PaidRewards, isPremium: true},
	}
	if preferPremium {
		buckets[0], buckets[1] = buckets[1], buckets[0]
	}
	// 1. Inventory items (cosmétiques, armes…) avec méta résolue.
	for _, entry := range buckets {
		for _, reward := range entry.bucket.InventoryRewards {
			path := strings.TrimSpace(reward.InventoryItemPath)
			if path == "" {
				continue
			}
			meta, ok := itemMap[path]
			if ok {
				return meta, entry.isPremium
			}
		}
	}
	// 2. Fallback : currency rewards (cR, xpboost, rerollcurrency, softcurrency).
	// Pour les paliers « purement monnaie », on rend le tier visible avec un titre
	// localisé et, quand l'asset existe localement, une miniature.
	for _, entry := range buckets {
		for _, reward := range entry.bucket.CurrencyRewards {
			if meta, ok := currencyRewardMeta(reward); ok {
				return meta, entry.isPremium
			}
		}
	}
	return seasonPassItemMeta{}, false
}

// currencyImagePath mappe le slug d'une currency (basename lowercase du
// CurrencyPath GameCMS, ex: "xpboost") vers son image officielle GameCMS,
// telle que publiée dans /hi/Progression/file/metadata/metadata.json
// (cf. https://den.dev/blog/halo-infinite-exchange-spartan-points/).
//
// Ces chemins sont relayés par le proxy /api/v1/assets/battlepass/tracks/{path}
// qui les télécharge à la 1ère demande via le resolver puis sert depuis le
// cache fichier — aucun asset à committer dans static/.
var currencyImagePath = map[string]string{
	"cr":             "progression/Currencies/Credit_Coin-SM.png",
	"softcurrency":   "progression/StoreContent/ToggleTiles/SpartanPoints_Common_4x4.png",
	"rerollcurrency": "progression/Currencies/1104-000-data-pad-e39bef84-2x2.png",
	"xpboost":        "progression/Currencies/1103-000-xp-boost-5e92621a-2x2.png",
	"xpgrant":        "progression/Currencies/1102-000-xp-grant-c77c6396-2x2.png",
}

// currencyTitleFR retourne le libellé français d'une currency, par slug lowercase.
func currencyTitleFR(slug string, amount int) string {
	var label string
	switch slug {
	case "cr":
		label = "Crédits"
	case "softcurrency":
		label = "Crédits Spartan"
	case "xpboost":
		label = "Boost XP"
	case "rerollcurrency":
		label = "Relance défi"
	default:
		if slug == "" {
			return ""
		}
		label = slug
	}
	if amount > 0 {
		return fmt.Sprintf("%d × %s", amount, label)
	}
	return label
}

// currencyRewardMeta convertit une CurrencyReward en seasonPassItemMeta.
// Retourne ok=false si le CurrencyPath est vide.
func currencyRewardMeta(reward seasonPassCurrencyReward) (seasonPassItemMeta, bool) {
	cleanPath := strings.TrimSpace(reward.CurrencyPath)
	if cleanPath == "" {
		return seasonPassItemMeta{}, false
	}
	slug := currencySlug(cleanPath)
	title := currencyTitleFR(slug, reward.Amount)
	meta := seasonPassItemMeta{Title: title}
	if imgPath, ok := currencyImagePath[slug]; ok {
		meta.ImageURL = localBPImageURL(imgPath, "tracks")
	}
	return meta, true
}

// currencySlug extrait le slug lowercase d'un CurrencyPath
// (ex: "Currency/Currencies/xpboost.json" → "xpboost").
func currencySlug(currencyPath string) string {
	base := path.Base(strings.ReplaceAll(currencyPath, "\\", "/"))
	if idx := strings.LastIndexByte(base, '.'); idx > 0 {
		base = base[:idx]
	}
	return strings.ToLower(strings.TrimSpace(base))
}

func resolveTrackImageURL(row seasonPassTrackRow, payload *seasonPassTrackPayload) *string {
	p := coalesceNullString(row.battlepassImagePath)
	if p == "" && payload != nil {
		p = payload.BattlePassImage
		if p == "" {
			p = payload.SummaryImagePath
		}
	}
	return localBPImageURL(p, "tracks")
}

func resolveTrackBackgroundURL(row seasonPassTrackRow, payload *seasonPassTrackPayload) *string {
	p := coalesceNullString(row.backgroundImagePath)
	if p == "" && payload != nil {
		p = payload.BackgroundImagePath
	}
	return localBPImageURL(p, "background")
}

func resolveXPPerRank(row seasonPassTrackRow, payload *seasonPassTrackPayload) *int {
	if row.xpPerRank.Valid {
		return intPtr(int(row.xpPerRank.Int64))
	}
	if payload != nil && payload.XpPerRank > 0 {
		return intPtr(payload.XpPerRank)
	}
	return nil
}

func resolveMaxRank(payload *seasonPassTrackPayload, state trackSnapshotState) *int {
	maxRank := 0
	if payload != nil {
		for _, rank := range payload.Ranks {
			if rank.Rank > maxRank {
				maxRank = rank.Rank
			}
		}
	}
	if maxRank == 0 {
		maxRank = state.Rank
	}
	if maxRank <= 0 {
		return nil
	}
	return intPtr(maxRank)
}

func computeCompletionPercent(state trackSnapshotState, maxRank *int) *float64 {
	if maxRank == nil || *maxRank <= 0 {
		return nil
	}
	if state.HasReachedMaxRank {
		return floatPtr(100)
	}
	percent := float64(state.Rank) / float64(*maxRank) * 100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return floatPtr(percent)
}

func computeActiveTierProgressPercent(
	state trackSnapshotState,
	xpPerRank *int,
	activeTierRank *int,
) *float64 {
	if activeTierRank == nil || xpPerRank == nil || *xpPerRank <= 0 {
		return nil
	}
	if state.HasReachedMaxRank {
		return floatPtr(100)
	}
	percent := float64(state.Partial) / float64(*xpPerRank) * 100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return floatPtr(percent)
}

func resolveActiveTierRank(state trackSnapshotState, maxRank int) int {
	if maxRank <= 0 {
		return 0
	}
	if state.HasReachedMaxRank {
		return maxRank
	}
	activeTierRank := state.Rank + 1
	if activeTierRank <= 0 {
		activeTierRank = 1
	}
	if activeTierRank > maxRank {
		activeTierRank = maxRank
	}
	return activeTierRank
}

func payloadNameValue(payload *seasonPassTrackPayload) any {
	if payload == nil {
		return nil
	}
	return payload.Name
}

func payloadDescription(payload *seasonPassTrackPayload, preferEN bool) *string {
	if payload == nil {
		return nil
	}
	value := localizedText(payload.Description, preferEN)
	if value == "" || isPlaceholderDescription(value) {
		return nil
	}
	return &value
}

// localizedText résout un texte Battle Pass (nom/description) potentiellement
// multilingue. preferEN (dérivé de la locale de requête, cf. bpPreferEN) ordonne les
// clés de langue : anglais d'abord en EN, français d'abord sinon — au lieu de l'ordre
// FR-first figé historique qui laissait les libellés de pass en français quand l'UI
// passait en anglais.
func localizedText(value any, preferEN bool) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		// Format résolu Halo : {"value": "Operation: Ground Zero", "status": "Resolved"}
		// La clé "value" contient le texte déjà localisé (côté serveur Halo) : elle prime
		// quelle que soit la locale demandée (pas de variante par langue à ce niveau).
		if v, ok := typed["value"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		// Clés de langue explicites (stockées par le système de traduction interne),
		// ordonnées selon la locale de requête.
		langOrder := []string{"fr", "en", "default"}
		if preferEN {
			langOrder = []string{"en", "default", "fr"}
		}
		for _, key := range langOrder {
			candidate, ok := typed[key].(string)
			if ok && strings.TrimSpace(candidate) != "" {
				return strings.TrimSpace(candidate)
			}
		}
		// Pas de fallback générique sur toutes les valeurs : évite de retourner des
		// champs méta Halo comme "Status": "Ready" ou "StringId": "...".
	}
	return ""
}

func coalesceNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullStringPtr(value sql.NullString) *string {
	text := strings.TrimSpace(coalesceNullString(value))
	if text == "" {
		return nil
	}
	return &text
}

// isPlaceholderDescription retourne true pour les valeurs de description
// que GameCMS (Halo Infinite) laisse comme placeholder sans contenu réel.
// Couvre : "Placeholder Text" littéral, strings de dev du type "S5 Large Op Pass Description".
func isPlaceholderDescription(s string) bool {
	t := strings.TrimSpace(s)
	switch strings.ToLower(t) {
	case "placeholder text", "placeholder", "tbd", "todo", "":
		return true
	}
	// Strings internes 343 : "S5 Large Op Pass Description", "S6 Medium Op Pass 3 Description"…
	return strings.HasSuffix(t, " Description") && len(t) < 60
}

func descriptionPtr(value sql.NullString) *string {
	p := nullStringPtr(value)
	if p == nil || isPlaceholderDescription(*p) {
		return nil
	}
	return p
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

// computeSeasonPassStatus détermine le statut d'un track depuis les indicateurs connus.
func computeSeasonPassStatus(state trackSnapshotState) domain.SeasonPassStatus {
	if state.HasReachedMaxRank {
		return domain.SeasonPassStatusCompleted
	}
	if state.IsActive {
		return domain.SeasonPassStatusActive
	}
	if state.Rank > 0 || state.Partial > 0 {
		return domain.SeasonPassStatusInProgress
	}
	return domain.SeasonPassStatusNotStarted
}
