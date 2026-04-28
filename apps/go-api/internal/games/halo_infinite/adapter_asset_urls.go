package halo_infinite

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"levelup/go-api/internal/assets/static"
)

// TitleSlug est le slug canonique d'Halo Infinite côté adapter.
const TitleSlug = "halo_infinite"

// uuidRe matche un UUID v4 — utilisé pour rejeter les map names qui sont en
// fait des UUID bruts (non utilisables comme nom de fichier statique).
var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// mapPNGNames contient les noms de maps (EN) dont l'image locale est au format PNG.
// Tous les autres noms utilisent le format JPEG par défaut.
var mapPNGNames = map[string]struct{}{
	"Aquarius":                 {},
	"Aquarius - Ranked":        {},
	"Bazaar":                   {},
	"Behemoth":                 {},
	"Breaker":                  {},
	"Breaker Heavies":          {},
	"Catalyst":                 {},
	"Deadlock":                 {},
	"Deadlock Heavies":         {},
	"Highpower":                {},
	"Highpower Heavies":        {},
	"Highpower Sentry Defense": {},
	"Launch Site":              {},
	"Recharge":                 {},
	"Recharge - Ranked":        {},
	"Streets":                  {},
	"Streets - Ranked":         {},
}

// AssetURLAdapter implémente games.TitleAssetURLAdapter pour Halo Infinite.
//
// Composition path déléguée à internal/assets/static (couche 2 SRP). Les URLs
// émises sont au format /static/{folder}/halo_infinite/X.{ext} en cohérence
// avec l'arborescence FS title-scopée (post-Phase 6.5 du plan finition
// multi-titres).
type AssetURLAdapter struct {
	titleSlug string
}

// NewAssetURLAdapter construit un AssetURLAdapter pour Halo Infinite.
func NewAssetURLAdapter() *AssetURLAdapter {
	return &AssetURLAdapter{titleSlug: TitleSlug}
}

// TitleSlug retourne "halo_infinite".
func (a *AssetURLAdapter) TitleSlug() string { return a.titleSlug }

// MapImageURL retourne l'URL de l'image d'une map.
// Encode les espaces du nom en %20 manuellement (pas via net/url.PathEscape
// pour préserver les "/" éventuels). Retourne "" si nom vide ou UUID.
func (a *AssetURLAdapter) MapImageURL(mapName string) string {
	mapName = strings.TrimSpace(mapName)
	if mapName == "" || uuidRe.MatchString(mapName) {
		return ""
	}
	ext := ".jpg"
	if _, ok := mapPNGNames[mapName]; ok {
		ext = ".png"
	}
	return static.URL(static.KindMap, a.titleSlug, encodeSpaces(mapName), ext)
}

// MedalImageURL retourne l'URL de l'icône d'une médaille à partir de son ID numérique.
func (a *AssetURLAdapter) MedalImageURL(medalID uint64) string {
	return static.URL(static.KindMedal, a.titleSlug, strconv.FormatUint(medalID, 10), ".png")
}

// CSRRankImageURL retourne l'URL du badge d'un rang CSR.
// Format de l'id : 120px-HINF-CSR_{Tier}{SubTier} (ex: 120px-HINF-CSR_Gold3).
func (a *AssetURLAdapter) CSRRankImageURL(tier string, subTier int) string {
	tier = strings.TrimSpace(tier)
	if tier == "" {
		return ""
	}
	id := fmt.Sprintf("120px-HINF-CSR_%s%d", tier, subTier)
	return static.URL(static.KindCSRRank, a.titleSlug, id, ".png")
}

// CSRRankImageURLOnyx retourne l'URL du badge Onyx (sans sub-tier).
func (a *AssetURLAdapter) CSRRankImageURLOnyx() string {
	return static.URL(static.KindCSRRank, a.titleSlug, "120px-HINF-CSR_Onyx", ".png")
}

// encodeSpaces remplace les espaces par %20 sans encoder les autres caractères.
// Utilisé pour les map names qui peuvent contenir des espaces et tirets.
func encodeSpaces(s string) string {
	if !strings.ContainsRune(s, ' ') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 6)
	for _, c := range s {
		if c == ' ' {
			b.WriteString("%20")
		} else {
			b.WriteRune(c)
		}
	}
	return b.String()
}
