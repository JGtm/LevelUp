package games

// catalog_adapter.go — interface TitleCatalogAdapter (Phase C plan catalogue).
//
// Sœur des interfaces TitleDataAdapter / TitleSemanticAdapter / TitleAssetURLAdapter.
// Encapsule la logique title-spécifique de fetch et classification des assets
// catalogue (playlists, pairs, maps, game variants).
//
// Le service CatalogFetcherService (Phase F) utilise cette interface pour
// drainer la queue catalog_fetch_queue de manière title-agnostic.

import (
	"context"

	"levelup/go-api/internal/games/canonical"
)

// TitleCatalogAdapter expose les opérations catalogue title-spécifiques.
//
// Les implémentations connues :
//   - games/halo_infinite/CatalogAdapter : enveloppe DiscoveryUGC (Phase D)
//   - games/synthetic_title_b/CatalogAdapter : fixtures pour tests d'isolation
//
// Toutes les méthodes Fetch* peuvent retourner une erreur réseau / parsing
// transitoire. Le caller (CatalogFetcherService) gère le retry / le marquage
// d'attempts dans catalog_fetch_queue.
type TitleCatalogAdapter interface {
	TitleSlug() string

	// FetchPlaylist récupère une playlist depuis l'API du titre et retourne sa
	// définition canonique multi-langues (avec ses pair_links).
	FetchPlaylist(ctx context.Context, assetID, versionID string) (canonical.CanonicalPlaylist, error)

	// FetchPair récupère un pair (map+mode) depuis l'API du titre.
	// La sortie inclut mode_category (sortie de InferModeCategoryFromPairName)
	// et mode_labels par langue (sortie de NormalizeModeLabel).
	FetchPair(ctx context.Context, assetID, versionID string) (canonical.CanonicalPair, error)

	// FetchMap récupère une map depuis l'API du titre, image URL incluse.
	FetchMap(ctx context.Context, assetID, versionID string) (canonical.CanonicalMap, error)

	// FetchGameVariant récupère un game variant depuis l'API du titre.
	// Inclut mode_canonical (mappage game_variant_category → enum ModeCanonical).
	FetchGameVariant(ctx context.Context, assetID, versionID string) (canonical.CanonicalGameVariant, error)

	// ClassifyExperience retourne l'experience d'une playlist (ranked/social/btb/...)
	// depuis ses attributs natifs (nom, tags, GameVariantCategory).
	// Implémenté via TOML config/titles/{slug}/catalog/experience_rules.toml.
	ClassifyExperience(playlist canonical.CanonicalPlaylist) canonical.Experience
}
