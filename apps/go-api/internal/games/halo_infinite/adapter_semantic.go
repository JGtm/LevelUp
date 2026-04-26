package halo_infinite

import (
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// SemanticAdapter est l'implémentation HI de games.TitleSemanticAdapter.
//
// Il enveloppe :
//   - un *mappings.FieldMappingSet chargé depuis config/titles/halo_infinite/mappings/fields.toml
//   - un *mappings.RankCatalog chargé depuis metadata.duckdb (career_rank_translations)
//   - un *mappings.AssetMappingSet chargé depuis assets.toml (modes, tiers, cadences, statuses)
//   - un *mappings.OutcomeMappingSet chargé depuis outcomes.toml (win/loss/tie/dnf)
type SemanticAdapter struct {
	fields   *mappings.FieldMappingSet
	ranks    *mappings.RankCatalog
	assets   *mappings.AssetMappingSet
	outcomes *mappings.OutcomeMappingSet
}

// NewSemanticAdapter construit un semantic adapter HI.
//
// fields ne doit pas être nil — la validation au boot doit garantir qu'un
// FieldMappingSet a été chargé pour le slug HI avant d'instancier l'adapter.
// Si fields est nil, NewSemanticAdapter retourne nil (signal au boot que le
// titre n'a pas de mapping disponible).
//
// ranks peut être nil ou vide : le service home dégradera vers un libellé
// minimal si la table career_rank_translations n'a pas été peuplée.
//
// assets et outcomes peuvent être nil : les TOML correspondants sont optionnels
// (les surfaces produit qui les consomment doivent dégrader gracieusement).
func NewSemanticAdapter(
	fields *mappings.FieldMappingSet,
	ranks *mappings.RankCatalog,
	assets *mappings.AssetMappingSet,
	outcomes *mappings.OutcomeMappingSet,
) *SemanticAdapter {
	if fields == nil {
		return nil
	}
	if ranks == nil {
		ranks = mappings.NewRankCatalog(titlePkg.DefaultSlug, nil)
	}
	return &SemanticAdapter{
		fields:   fields,
		ranks:    ranks,
		assets:   assets,
		outcomes: outcomes,
	}
}

// TitleSlug retourne le slug HI canonique.
func (a *SemanticAdapter) TitleSlug() string { return titlePkg.DefaultSlug }

// SchemaVersion retourne la version du schéma TOML chargé.
func (a *SemanticAdapter) SchemaVersion() int { return a.fields.SchemaVersion() }

// Fields retourne le FieldMappingSet chargé pour HI.
func (a *SemanticAdapter) Fields() *mappings.FieldMappingSet { return a.fields }

// Ranks retourne le catalog des rangs de carrière localisés.
func (a *SemanticAdapter) Ranks() *mappings.RankCatalog { return a.ranks }

// Assets retourne l'AssetMappingSet (modes, tiers, cadences, statuses).
// Peut être nil si le titre n'a pas de assets.toml.
func (a *SemanticAdapter) Assets() *mappings.AssetMappingSet { return a.assets }

// Outcomes retourne l'OutcomeMappingSet (win/loss/tie/dnf).
// Peut être nil si le titre n'a pas de outcomes.toml.
func (a *SemanticAdapter) Outcomes() *mappings.OutcomeMappingSet { return a.outcomes }
