package halo_infinite

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"levelup/go-api/internal/assets/static"
)

// TitleSlug est le slug canonique d'Halo Infinite côté adapter.
const TitleSlug = "halo_infinite"

// EnvTitleScopedFlag est le nom de la variable d'environnement qui active
// le title-scoping des chemins /static/. Default OFF — les URLs émises restent
// au format flat (/static/maps/X.png) tant que la migration FS n'est pas
// activée. Phase 6.5 du plan flippe ce flag en même temps que git mv + UPDATE DB.
const EnvTitleScopedFlag = "STATIC_PATHS_TITLE_SCOPED"

// uuidRe matche un UUID v4 — utilisé pour rejeter les map names qui sont en
// fait des UUID bruts (non utilisables comme nom de fichier statique).
var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// mapPNGNames contient les noms de maps (EN) dont l'image locale est au format PNG.
// Tous les autres noms utilisent le format JPEG par défaut.
//
// Reflet du dict historique de internal/analysis/home.go:1036. Y demeure tant
// que home.go n'a pas été migré (Phase 6.3) — sera supprimé du package analysis
// à ce moment-là, l'adapter devient source de vérité unique.
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
// titleScoped est lu depuis ENV au boot — si true, les URLs émises sont
// title-scopées (/static/maps/halo_infinite/X.png). Sinon flat
// (/static/maps/X.png). Cf. Phase 6 du plan finition multi-titres.
type AssetURLAdapter struct {
	titleSlug   string
	titleScoped bool
}

// NewAssetURLAdapter construit un AssetURLAdapter en lisant le flag depuis ENV.
func NewAssetURLAdapter() *AssetURLAdapter {
	return &AssetURLAdapter{
		titleSlug:   TitleSlug,
		titleScoped: os.Getenv(EnvTitleScopedFlag) == "true",
	}
}

// NewAssetURLAdapterWithFlag permet d'injecter explicitement le flag (tests).
func NewAssetURLAdapterWithFlag(titleScoped bool) *AssetURLAdapter {
	return &AssetURLAdapter{titleSlug: TitleSlug, titleScoped: titleScoped}
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
	encoded := encodeSpaces(mapName)
	if a.titleScoped {
		return static.URL(static.KindMap, a.titleSlug, encoded, ext)
	}
	return path.Join(static.MountPoint, static.Folder(static.KindMap), encoded+ext)
}

// MedalImageURL retourne l'URL de l'icône d'une médaille à partir de son ID numérique.
func (a *AssetURLAdapter) MedalImageURL(medalID uint64) string {
	id := strconv.FormatUint(medalID, 10)
	if a.titleScoped {
		return static.URL(static.KindMedal, a.titleSlug, id, ".png")
	}
	return path.Join(static.MountPoint, static.Folder(static.KindMedal), id+".png")
}

// CSRRankImageURL retourne l'URL du badge d'un rang CSR.
// Format de l'id : 120px-HINF-CSR_{Tier}{SubTier} (ex: 120px-HINF-CSR_Gold3).
func (a *AssetURLAdapter) CSRRankImageURL(tier string, subTier int) string {
	tier = strings.TrimSpace(tier)
	if tier == "" {
		return ""
	}
	id := fmt.Sprintf("120px-HINF-CSR_%s%d", tier, subTier)
	if a.titleScoped {
		return static.URL(static.KindCSRRank, a.titleSlug, id, ".png")
	}
	return path.Join(static.MountPoint, static.Folder(static.KindCSRRank), id+".png")
}

// CSRRankImageURLOnyx retourne l'URL du badge Onyx (sans sub-tier).
func (a *AssetURLAdapter) CSRRankImageURLOnyx() string {
	id := "120px-HINF-CSR_Onyx"
	if a.titleScoped {
		return static.URL(static.KindCSRRank, a.titleSlug, id, ".png")
	}
	return path.Join(static.MountPoint, static.Folder(static.KindCSRRank), id+".png")
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
