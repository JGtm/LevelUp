// Package games — interfaces title-aware pour les services produit.
//
// Phase B du plan multi-titres : sépare en 2 interfaces (SRP) la lecture
// canonique services et l'exposition des libellés sémantiques.
//
//   - TitleDataAdapter : transforme le stockage propre du titre (DuckDB,
//     fetcher live, etc.) vers le canonique services consommé par les
//     services produit (career, home, match view…).
//
//   - TitleSemanticAdapter : expose les FieldMappingSet/AssetMappingSet/
//     OutcomeMappingSet chargés depuis les TOML versionnés Git.
//
// Les deux interfaces sont injectées séparément dans les services produit.
// Un service ne dépend que de ce dont il a besoin (data, semantic, ou les deux).
package games

import (
	"context"
	"errors"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// ErrCapabilityNotSupported est retournée par les méthodes Load* d'un
// TitleDataAdapter quand la capability requise est absente du titre.
//
// Comportement attendu côté caller : ne pas considérer comme une erreur
// inattendue, plutôt comme un signal de dégradation gracieuse (cf. plan §5.7).
var ErrCapabilityNotSupported = errors.New("capability not supported by title")

// ErrTitleNotResolved est retournée par le Resolver quand un slug est inconnu.
var ErrTitleNotResolved = errors.New("title slug not resolved")

// CapabilityKey identifie une capability produit (au sens HALO_INFINITE_CAPABILITY_MAP.md).
type CapabilityKey string

const (
	CapMatchHistory       CapabilityKey = "match.history"
	CapMatchDetailCore    CapabilityKey = "match.detail.core"
	CapMatchSkillSnapshot CapabilityKey = "match.skill.snapshot"
	CapCareerProgression  CapabilityKey = "career.progression"
	CapPveFirefight       CapabilityKey = "pve.firefight_stats"
	CapTimeseries         CapabilityKey = "analytics.timeseries"
	CapScoreboardExtra    CapabilityKey = "match.scoreboard.extra" // champs étendus du scoreboard
	CapCitationsEngine    CapabilityKey = "citations.engine"       // moteur de citations
)

// CapabilityMap décrit l'état des capabilities produit d'un adapter à un instant T.
type CapabilityMap map[CapabilityKey]CapabilityStatus

// CapabilityStatus reflète la sémantique de HALO_INFINITE_CAPABILITY_MAP.md.
type CapabilityStatus string

const (
	CapSupported  CapabilityStatus = "supported"
	CapDegraded   CapabilityStatus = "degraded"
	CapNotExposed CapabilityStatus = "not_exposed"
)

// Has retourne vrai si la capability est supportée ou dégradée (mais utilisable).
func (cm CapabilityMap) Has(k CapabilityKey) bool {
	switch cm[k] {
	case CapSupported, CapDegraded:
		return true
	}
	return false
}

// TitleDataAdapter expose la lecture canonique services pour un titre donné.
//
// Toutes les méthodes Load* :
//   - retournent un schéma canonique stable (cf. internal/games/canonical/) ;
//   - retournent ErrCapabilityNotSupported si la capability requise est absente ;
//   - n'ont aucune dépendance à un endpoint provider natif, à du SQL externe
//     ou à un nom de colonne titre-spécifique.
type TitleDataAdapter interface {
	TitleSlug() string
	Capabilities() CapabilityMap

	LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error)
	LoadMatchDetail(ctx context.Context, matchID string) (*canonical.MatchDetail, error)
	LoadPlayerStats(ctx context.Context, xuid string, scope canonical.StatsScope) (*canonical.PlayerStats, error)
	LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error)
	LoadEncounters(ctx context.Context, xuid string) ([]canonical.EncounterRow, error)
	LoadTimeseries(ctx context.Context, xuid string, query canonical.TimeseriesQuery) (*canonical.MetricSeries, error)

	// Phase B+ : scoreboard étendu + événements + amis (CapScoreboardExtra).
	// Retournent ErrCapabilityNotSupported si la capability est absente.
	LoadMatchScoreboard(ctx context.Context, matchID string) ([]canonical.MatchParticipant, error)
	LoadHighlightEvents(ctx context.Context, matchID string) ([]canonical.HighlightEvent, error)
	LoadFriendsXUIDs(ctx context.Context, xuid string) ([]string, error)
}

// TitleSemanticAdapter expose les libellés et la présentation pour un titre.
//
// Ranks() retourne le catalog des rangs de carrière localisés. Peut retourner
// un catalog vide (Len() == 0) si la table career_rank_translations n'a pas
// encore été peuplée — les services consommateurs doivent alors dégrader
// proprement (ex: afficher uniquement le rank_id).
//
// Assets() / Outcomes() peuvent retourner nil si les TOML correspondants ne
// sont pas définis pour le titre. Les callers doivent dégrader gracieusement.
type TitleSemanticAdapter interface {
	TitleSlug() string
	SchemaVersion() int
	Fields() *mappings.FieldMappingSet
	Ranks() *mappings.RankCatalog
	Assets() *mappings.AssetMappingSet
	Outcomes() *mappings.OutcomeMappingSet
}

// TitleAssetURLAdapter expose la composition d'URLs d'assets statiques
// title-spécifiques.
//
// Phase 6 du plan de finition multi-titres : couche 3 (cf. SRP 4-couches du
// BACKLOG static FS migration). Convertit un identifiant métier (mapName,
// medalID, tier) en URL relative, en encapsulant les conventions de naming
// propres au titre (encodage espaces map names, format CSR rank, etc.).
//
// Délègue la composition path à `internal/assets/static/` (couche 2 pure).
// Les callers (home, home_repo, citation_snippets, frontend) reçoivent cet
// adapter via le Resolver et n'ont aucune connaissance du format path/encoding.
//
// Les méthodes retournent "" quand l'identifiant est invalide (UUID, vide,
// non reconnu) — les callers doivent dégrader gracieusement (cache-aside,
// fallback nil, etc.).
type TitleAssetURLAdapter interface {
	TitleSlug() string

	// MapImageURL retourne l'URL de l'image d'une map à partir de son nom métier.
	// Retourne "" si mapName est vide, est un UUID ou n'est pas reconnu.
	MapImageURL(mapName string) string

	// MedalImageURL retourne l'URL de l'icône d'une médaille à partir de son ID numérique.
	MedalImageURL(medalID uint64) string

	// CSRRankImageURL retourne l'URL du badge d'un rang CSR (tier + sub-tier).
	// Cas spécial : pour Onyx, utiliser CSRRankImageURLOnyx (pas de sub-tier).
	CSRRankImageURL(tier string, subTier int) string

	// CSRRankImageURLOnyx retourne l'URL du badge Onyx (sans sub-tier).
	CSRRankImageURLOnyx() string
}

// Resolver injecte les adapters d'un titre courant aux services produit.
//
// Il est construit au boot du serveur et exposé via la DI. Un service produit
// reçoit Resolver et appelle Data(slug), Semantic(slug) ou AssetURL(slug)
// selon son besoin.
//
// Il n'a aucune connaissance des slugs supportés en dur : c'est le constructeur
// (api/server.go) qui peuple le resolver via Register*().
type Resolver interface {
	Data(titleSlug string) (TitleDataAdapter, error)
	Semantic(titleSlug string) (TitleSemanticAdapter, error)
	AssetURL(titleSlug string) (TitleAssetURLAdapter, error)
	DefaultSlug() string
}
