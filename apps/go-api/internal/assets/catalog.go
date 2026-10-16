package assets

import (
	"context"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/platform/duckdb"
)

// Verdict est la position d'une référence d'asset face au catalogue local.
type Verdict int

const (
	// VerdictUnknown : l'identifiant est absent du catalogue local ; aucune source
	// distante ne doit être sollicitée pour lui.
	VerdictUnknown Verdict = iota
	// VerdictKnown : les données locales référencent l'identifiant.
	VerdictKnown
	// VerdictUnverifiable : aucun catalogue local ne couvre ce kind (l'identifiant est
	// dérivé d'une charge utile amont : image d'un joueur, de défi, de passe de combat).
	VerdictUnverifiable
	// VerdictUnavailable : le catalogue local n'a pas pu être lu (base indisponible,
	// erreur de lecture). Aucune source distante n'est sollicitée, et le refus n'est pas
	// mémorisé : la requête suivante interroge de nouveau le catalogue.
	VerdictUnavailable
)

// Catalog classe une référence d'asset contre les données locales.
type Catalog interface {
	Classify(ctx context.Context, ref Ref) Verdict
}

// MetadataCatalog est le catalogue adossé à metadata.duckdb : référentiel des
// médailles, catalogue et traductions de cartes. Les kinds dont l'identifiant dérive
// d'une charge utile amont (images de joueur, de défi, de passe de combat) n'y ont pas
// de référentiel : ils sont VerdictUnverifiable.
type MetadataCatalog struct {
	store      *DuckDBIndexStore
	titleKnown func(titleID string) bool
	played     *PlayedMapVersions
}

// NewMetadataCatalog construit le catalogue sur store ; titleKnown dit si un
// identifiant de titre est enregistré (registre des titres : aucun titre en dur).
func NewMetadataCatalog(store *DuckDBIndexStore, titleKnown func(titleID string) bool) *MetadataCatalog {
	return &MetadataCatalog{store: store, titleKnown: titleKnown}
}

// WithPlayedMapVersions fait juger les versions de carte aussi par les matchs réellement
// joués du titre (base partagée). Sans elle, seule la version courante du catalogue est connue.
func (c *MetadataCatalog) WithPlayedMapVersions(played *PlayedMapVersions) *MetadataCatalog {
	c.played = played
	return c
}

// Requêtes de présence (presence) : une table absente laisse l'identifiant inconnu, une
// base illisible le rend indisponible ; les deux sont fermés par défaut.
const (
	qMedalDefinition  = `SELECT 1 FROM medal_definitions WHERE medal_name_id = ? LIMIT 1`
	qMedalWaypoint    = `SELECT 1 FROM waypoint_medals_raw WHERE title_id = ? AND medal_id = ? LIMIT 1`
	qMapCatalog       = `SELECT 1 FROM maps_catalog WHERE title_slug = ? AND map_asset_id = ? LIMIT 1`
	qAssetTranslation = `SELECT 1 FROM asset_translations WHERE asset_type = ? AND asset_id = ? LIMIT 1`
	qMapImage         = `SELECT 1 FROM map_images_registry WHERE title_id = ? AND map_id = ? LIMIT 1`
	qMapVersion       = `SELECT 1 FROM maps_catalog WHERE title_slug = ? AND map_asset_id = ? AND current_version_id = ? LIMIT 1`
)

// Classify implémente Catalog.
func (c *MetadataCatalog) Classify(ctx context.Context, ref Ref) Verdict {
	if c.titleKnown == nil || !c.titleKnown(ref.TitleID) {
		return VerdictUnknown
	}
	switch ref.Kind {
	case KindMedalImage:
		return c.classifyMedal(ctx, ref)
	case KindMapImage:
		return c.classifyMap(ctx, ref)
	default:
		return VerdictUnverifiable
	}
}

func (c *MetadataCatalog) classifyMedal(ctx context.Context, ref Ref) Verdict {
	medalID, err := strconv.ParseInt(ref.ID, 10, 64)
	if err != nil {
		return VerdictUnknown
	}
	return c.presence(ctx, ref,
		probe{qMedalDefinition, []any{medalID}},
		probe{qMedalWaypoint, []any{ref.TitleID, medalID}})
}

// classifyMap juge la carte, puis sa version (Variant), qui entre dans l'appel distant
// (DiscoveryUGC). Une version est connue si c'est la version courante du catalogue
// (current_version_id) ou si le couple (carte, version) figure dans un match réellement joué
// du titre (WithPlayedMapVersions), même pour une carte encore absente des métadonnées. Une
// version jamais jouée d'une carte connue est VerdictUnverifiable (récupérée pour un appelant
// de confiance seulement) ; d'une carte inconnue, VerdictUnknown. Sans cela, chaque version
// inventée coûterait un appel signé aux API Halo.
func (c *MetadataCatalog) classifyMap(ctx context.Context, ref Ref) Verdict {
	mapVerdict := c.presence(ctx, ref,
		probe{qMapCatalog, []any{ref.TitleID, ref.ID}},
		probe{qAssetTranslation, []any{"map", ref.ID}},
		probe{qMapImage, []any{ref.TitleID, ref.ID}})
	if mapVerdict == VerdictUnavailable || ref.Variant == "" {
		return mapVerdict
	}
	if mapVerdict == VerdictKnown {
		switch c.presence(ctx, ref, probe{qMapVersion, []any{ref.TitleID, ref.ID, ref.Variant}}) {
		case VerdictKnown:
			return VerdictKnown
		case VerdictUnavailable:
			return VerdictUnavailable
		}
	}
	return c.classifyPlayedVersion(ctx, ref, mapVerdict)
}

// classifyPlayedVersion juge une version hors catalogue par les matchs joués du titre. Sans
// source, ou si la base partagée est illisible (journalisé), le jugement du catalogue seul
// s'applique, sauf qu'une carte inconnue devient VerdictUnavailable : un refus n'est pas
// mémorisé sur une lecture en échec.
func (c *MetadataCatalog) classifyPlayedVersion(ctx context.Context, ref Ref, mapVerdict Verdict) Verdict {
	fallback := VerdictUnverifiable
	if mapVerdict != VerdictKnown {
		fallback = VerdictUnknown
	}
	if c.played == nil {
		return fallback
	}
	played, err := c.played.Played(ctx, ref.TitleID, ref.ID, ref.Variant)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "assets: versions de carte jouées illisibles", append(ref.LogAttrs(), "err", err)...)
		if fallback == VerdictUnknown {
			return VerdictUnavailable
		}
		return fallback
	case played:
		return VerdictKnown
	default:
		return fallback
	}
}

// probe est une requête de présence et ses arguments.
type probe struct {
	query string
	args  []any
}

// presence rend VerdictKnown si l'une des requêtes trouve une ligne ; sinon
// VerdictUnavailable si l'une d'elles a échoué (base indisponible, erreur de lecture :
// journalisé), VerdictUnknown si toutes ont répondu sans ligne. Une table absente
// (duckdb.IsCatalogError : métadonnées d'un titre sans ce référentiel) vaut une réponse sans
// ligne.
func (c *MetadataCatalog) presence(ctx context.Context, ref Ref, probes ...probe) Verdict {
	failed := false
	for _, p := range probes {
		found, err := c.store.Exists(ctx, p.query, p.args...)
		switch {
		case err == nil && found:
			return VerdictKnown
		case err == nil:
		case duckdb.IsCatalogError(err):
			slog.DebugContext(ctx, "assets: référentiel absent du catalogue local",
				append(ref.LogAttrs(), "err", err)...)
		default:
			failed = true
			slog.WarnContext(ctx, "assets: catalogue local illisible", append(ref.LogAttrs(), "err", err)...)
		}
	}
	if failed {
		return VerdictUnavailable
	}
	return VerdictUnknown
}
