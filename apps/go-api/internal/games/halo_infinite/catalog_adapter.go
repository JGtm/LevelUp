package halo_infinite

// catalog_adapter.go — Phase D du plan PLAN_PLAYLISTS_CATALOG.md.
//
// CatalogAdapter implémente games.TitleCatalogAdapter pour Halo Infinite :
//   - Wrap halo.HaloProvider.FetchAsset pour la lecture DiscoveryUGC
//   - Charge experience_rules.toml au boot et expose ClassifyExperience
//   - Délègue InferModeCategoryFromPairName (mode_category.go) pour mode_category
//
// Phase D minimal : single-lang fetch, skeleton viable. Le multi-langues (14
// langues) et le parsing CustomData.PlaylistEntries → PairLinks sont étendus
// en Phase F (drain queue) où le coût de 14 round-trips est acceptable.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
)

// AssetType identifie le type d'asset à fetcher (miroir local de halo.AssetType
// pour éviter le cycle d'import games/halo_infinite ↔ platform/halo).
type AssetType string

const (
	AssetTypePlaylist    AssetType = "playlist"
	AssetTypeMap         AssetType = "map"
	AssetTypePair        AssetType = "pair"
	AssetTypeGameVariant AssetType = "game_variant"
)

// DiscoveryAssetRaw est la projection minimale d'un asset DiscoveryUGC.
// Miroir local de halo.DiscoveryAsset (cf. AssetType).
type DiscoveryAssetRaw struct {
	AssetID     string
	VersionID   string
	PublicName  string
	Description string
}

// AssetFetcher abstrait l'accès DiscoveryUGC. Implémenté côté platform/halo
// par un wrapper léger qui mappe halo.AssetType ↔ AssetType. Permet à
// games/halo_infinite de rester découplé de platform/halo.
type AssetFetcher interface {
	FetchAsset(ctx context.Context, assetType AssetType, titleID, assetID, versionID, lang string) (*DiscoveryAssetRaw, error)
}

const (
	titleSlugHaloInfinite = "halo_infinite"
	titleIDHaloInfinite   = "hi" // path component DiscoveryUGC
	defaultLang           = "en"
)

// experienceRulesTOML est la projection brute de experience_rules.toml.
type experienceRulesTOML struct {
	SchemaVersion int                   `toml:"schema_version"`
	Rules         []experienceRuleEntry `toml:"rule"`
}

type experienceRuleEntry struct {
	Experience string             `toml:"experience"`
	MatchAny   experienceMatchAny `toml:"match_any"`
}

type experienceMatchAny struct {
	NamePrefix   []string `toml:"name_prefix"`
	NameContains []string `toml:"name_contains"`
	NameExact    []string `toml:"name_exact"`
	IsRanked     *bool    `toml:"is_ranked"`
}

// CatalogAdapter implémente games.TitleCatalogAdapter pour Halo Infinite.
type CatalogAdapter struct {
	fetcher         AssetFetcher
	experienceRules []experienceRuleEntry
}

// NewCatalogAdapter construit l'adapter avec un fetcher DiscoveryUGC injecté
// et charge les règles d'experience depuis le TOML versionné.
func NewCatalogAdapter(fetcher AssetFetcher, experienceRulesPath string) (*CatalogAdapter, error) {
	rules, err := loadExperienceRules(experienceRulesPath)
	if err != nil {
		return nil, fmt.Errorf("CatalogAdapter: load experience rules: %w", err)
	}
	return &CatalogAdapter{
		fetcher:         fetcher,
		experienceRules: rules,
	}, nil
}

// loadExperienceRules charge et valide les règles depuis un fichier TOML.
func loadExperienceRules(path string) ([]experienceRuleEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc experienceRulesTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.SchemaVersion != 1 {
		return nil, fmt.Errorf("schema_version=%d non supportée (attendu 1)", doc.SchemaVersion)
	}
	if len(doc.Rules) == 0 {
		return nil, fmt.Errorf("aucune règle dans %s", path)
	}
	return doc.Rules, nil
}

// TitleSlug retourne le slug du titre.
func (a *CatalogAdapter) TitleSlug() string {
	return titleSlugHaloInfinite
}

// FetchPlaylist enveloppe FetchAsset(AssetTypePlaylist) en mono-lang EN.
//
// Phase D minimal : ne parse pas CustomData.PlaylistEntries (donc PairLinks vide).
// Le parsing complet est en Phase F (drain queue).
func (a *CatalogAdapter) FetchPlaylist(ctx context.Context, assetID, versionID string) (canonical.CanonicalPlaylist, error) {
	if a.fetcher == nil {
		return canonical.CanonicalPlaylist{}, fmt.Errorf("CatalogAdapter: fetcher non injecté")
	}
	asset, err := a.fetcher.FetchAsset(ctx, AssetTypePlaylist, titleIDHaloInfinite, assetID, versionID, defaultLang)
	if err != nil {
		return canonical.CanonicalPlaylist{}, err
	}
	pl := canonical.CanonicalPlaylist{
		AssetID:       asset.AssetID,
		VersionID:     asset.VersionID,
		NameCanonical: asset.PublicName,
		Names:         map[string]string{defaultLang: asset.PublicName},
	}
	pl.Experience = a.ClassifyExperience(pl)
	pl.IsRanked = pl.Experience == canonical.ExperienceRanked
	return pl, nil
}

// FetchPair enveloppe FetchAsset(AssetTypePair).
func (a *CatalogAdapter) FetchPair(ctx context.Context, assetID, versionID string) (canonical.CanonicalPair, error) {
	if a.fetcher == nil {
		return canonical.CanonicalPair{}, fmt.Errorf("CatalogAdapter: fetcher non injecté")
	}
	asset, err := a.fetcher.FetchAsset(ctx, AssetTypePair, titleIDHaloInfinite, assetID, versionID, defaultLang)
	if err != nil {
		return canonical.CanonicalPair{}, err
	}
	return canonical.CanonicalPair{
		AssetID:       asset.AssetID,
		VersionID:     asset.VersionID,
		NameCanonical: asset.PublicName,
		Names:         map[string]string{defaultLang: asset.PublicName},
		ModeCategory:  InferModeCategoryFromPairName(asset.PublicName),
		ModeLabels:    map[string]string{defaultLang: analysis.NormalizeModeLabel(asset.PublicName)},
	}, nil
}

// FetchMap enveloppe FetchAsset(AssetTypeMap).
func (a *CatalogAdapter) FetchMap(ctx context.Context, assetID, versionID string) (canonical.CanonicalMap, error) {
	if a.fetcher == nil {
		return canonical.CanonicalMap{}, fmt.Errorf("CatalogAdapter: fetcher non injecté")
	}
	asset, err := a.fetcher.FetchAsset(ctx, AssetTypeMap, titleIDHaloInfinite, assetID, versionID, defaultLang)
	if err != nil {
		return canonical.CanonicalMap{}, err
	}
	return canonical.CanonicalMap{
		AssetID:       asset.AssetID,
		VersionID:     asset.VersionID,
		NameCanonical: asset.PublicName,
		Names:         map[string]string{defaultLang: asset.PublicName},
		// ImageURL : peuplée en Phase F via assetResolver (KindMapImage).
	}, nil
}

// FetchGameVariant enveloppe FetchAsset(AssetTypeGameVariant).
func (a *CatalogAdapter) FetchGameVariant(ctx context.Context, assetID, versionID string) (canonical.CanonicalGameVariant, error) {
	if a.fetcher == nil {
		return canonical.CanonicalGameVariant{}, fmt.Errorf("CatalogAdapter: fetcher non injecté")
	}
	asset, err := a.fetcher.FetchAsset(ctx, AssetTypeGameVariant, titleIDHaloInfinite, assetID, versionID, defaultLang)
	if err != nil {
		return canonical.CanonicalGameVariant{}, err
	}
	return canonical.CanonicalGameVariant{
		AssetID:       asset.AssetID,
		VersionID:     asset.VersionID,
		NameCanonical: asset.PublicName,
		Names:         map[string]string{defaultLang: asset.PublicName},
		ModeCanonical: classifyModeCanonical(asset.PublicName),
		// GameVariantCategory : peuplé en Phase F (parsing CustomData).
	}, nil
}

// ClassifyExperience applique les règles TOML séquentiellement et retourne
// la première experience qui match. Si aucune règle ne match, retourne ExperienceUnknown.
func (a *CatalogAdapter) ClassifyExperience(pl canonical.CanonicalPlaylist) canonical.Experience {
	name := pl.NameCanonical
	for _, rule := range a.experienceRules {
		if matchExperienceRule(rule.MatchAny, name, pl.IsRanked) {
			return canonical.Experience(rule.Experience)
		}
	}
	return canonical.ExperienceUnknown
}

// matchExperienceRule retourne true si l'un des critères match_any est satisfait.
func matchExperienceRule(m experienceMatchAny, name string, isRanked bool) bool {
	for _, p := range m.NamePrefix {
		if p != "" && strings.HasPrefix(name, p) {
			return true
		}
	}
	for _, c := range m.NameContains {
		if c == "" {
			// la règle fallback "" en fin de fichier match toujours
			return true
		}
		if strings.Contains(name, c) {
			return true
		}
	}
	for _, e := range m.NameExact {
		if e != "" && name == e {
			return true
		}
	}
	if m.IsRanked != nil && *m.IsRanked == isRanked {
		return true
	}
	return false
}

// classifyModeCanonical déduit ModeCanonical depuis un nom de game variant brut.
// Heuristique simple Phase D — peut être enrichie en Phase F via TOML mode_canonical_map.
func classifyModeCanonical(name string) canonical.ModeCanonical {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "slayer"):
		return canonical.ModeSlayer
	case strings.Contains(n, "ctf") || strings.Contains(n, "capture"):
		return canonical.ModeCTF
	case strings.Contains(n, "oddball"):
		return canonical.ModeOddball
	case strings.Contains(n, "king of the hill"), strings.Contains(n, "koth"):
		return canonical.ModeKOTH
	case strings.Contains(n, "stronghold"):
		return canonical.ModeStrongholds
	case strings.Contains(n, "extraction"):
		return canonical.ModeExtraction
	case strings.Contains(n, "fiesta"):
		return canonical.ModeFiesta
	case strings.Contains(n, "firefight") || strings.Contains(n, "kotr"):
		return canonical.ModeFirefightKOTR
	case strings.Contains(n, "attrition"):
		return canonical.ModeAttrition
	case strings.Contains(n, "stockpile"):
		return canonical.ModeStockpile
	case strings.Contains(n, "total control"):
		return canonical.ModeTotalControl
	}
	return canonical.ModeUnknown
}
