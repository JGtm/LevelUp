package assets

import (
	"context"
	"log/slog"
	"strconv"
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
}

// NewMetadataCatalog construit le catalogue sur store ; titleKnown dit si un
// identifiant de titre est enregistré (registre des titres : aucun titre en dur).
func NewMetadataCatalog(store *DuckDBIndexStore, titleKnown func(titleID string) bool) *MetadataCatalog {
	return &MetadataCatalog{store: store, titleKnown: titleKnown}
}

// Requêtes de présence. Une table absente ou une base indisponible laisse
// l'identifiant inconnu (fermé par défaut), avec un journal.
const (
	qMedalDefinition = `SELECT 1 FROM medal_definitions WHERE medal_name_id = ? LIMIT 1`
	qMedalWaypoint   = `SELECT 1 FROM waypoint_medals_raw WHERE title_id = ? AND medal_id = ? LIMIT 1`
	qMapCatalog      = `SELECT 1 FROM maps_catalog WHERE title_slug = ? AND map_asset_id = ? LIMIT 1`
	qMapTranslation  = `SELECT 1 FROM asset_translations WHERE asset_type = 'map' AND asset_id = ? LIMIT 1`
	qMapImage        = `SELECT 1 FROM map_images_registry WHERE title_id = ? AND map_id = ? LIMIT 1`
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
	if c.any(ctx, ref,
		probe{qMedalDefinition, []any{medalID}},
		probe{qMedalWaypoint, []any{ref.TitleID, medalID}}) {
		return VerdictKnown
	}
	return VerdictUnknown
}

func (c *MetadataCatalog) classifyMap(ctx context.Context, ref Ref) Verdict {
	if c.any(ctx, ref,
		probe{qMapCatalog, []any{ref.TitleID, ref.ID}},
		probe{qMapTranslation, []any{ref.ID}},
		probe{qMapImage, []any{ref.TitleID, ref.ID}}) {
		return VerdictKnown
	}
	return VerdictUnknown
}

// probe est une requête de présence et ses arguments.
type probe struct {
	query string
	args  []any
}

// any dit si l'une des requêtes trouve une ligne. Une requête en erreur est journalisée
// et comptée comme sans résultat.
func (c *MetadataCatalog) any(ctx context.Context, ref Ref, probes ...probe) bool {
	for _, p := range probes {
		found, err := c.store.Exists(ctx, p.query, p.args...)
		if err != nil {
			slog.WarnContext(ctx, "assets: catalogue local illisible, identifiant traité comme inconnu",
				append(ref.LogAttrs(), "err", err)...)
			continue
		}
		if found {
			return true
		}
	}
	return false
}
