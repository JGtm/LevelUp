package games

import "levelup/go-api/internal/games/mappings"

// GenericSemanticAdapter est l'implementation PARTAGEE de TitleSemanticAdapter.
//
// Raison d'etre (DRY multi-titre) : le semantic adapter n'a AUCUNE logique
// title-specific — il ne fait qu'exposer en lecture les mappings TOML charges
// pour un titre (fields/assets/outcomes) + le RankCatalog. La divergence reelle
// entre titres vit dans (a) les fichiers TOML (donnees) et (b) le DataAdapter
// (logique de projection source->canonique). Dupliquer ce pass-through par titre
// (l'ancien pattern halo_infinite.SemanticAdapter / synthetic_title_b.Semantic
// Adapter) n'apporte rien : tous les titres partagent CETTE impl, parametree par
// slug. Un nouveau titre n'ecrit donc PAS de semantic adapter.
type GenericSemanticAdapter struct {
	slug     string
	fields   *mappings.FieldMappingSet
	ranks    *mappings.RankCatalog
	assets   *mappings.AssetMappingSet
	outcomes *mappings.OutcomeMappingSet
}

// NewGenericSemanticAdapter construit le semantic adapter partage d'un titre.
//
// fields ne doit pas etre nil (la validation boot garantit qu'un FieldMappingSet
// a ete charge pour le slug) ; nil -> retourne nil (signal boot : titre sans
// mapping). ranks nil -> RankCatalog vide pour le slug (les consommateurs
// degradent vers le rank_id). assets/outcomes peuvent rester nil (TOML optionnels).
func NewGenericSemanticAdapter(
	slug string,
	fields *mappings.FieldMappingSet,
	ranks *mappings.RankCatalog,
	assets *mappings.AssetMappingSet,
	outcomes *mappings.OutcomeMappingSet,
) *GenericSemanticAdapter {
	if fields == nil {
		return nil
	}
	if ranks == nil {
		ranks = mappings.NewRankCatalog(slug, nil)
	}
	return &GenericSemanticAdapter{slug: slug, fields: fields, ranks: ranks, assets: assets, outcomes: outcomes}
}

// TitleSlug retourne le slug porteur (identite de l'adapter, pas un gating).
func (a *GenericSemanticAdapter) TitleSlug() string { return a.slug }

// SchemaVersion retourne la version du schema TOML des fields.
func (a *GenericSemanticAdapter) SchemaVersion() int { return a.fields.SchemaVersion() }

// Fields retourne le FieldMappingSet charge pour le titre.
func (a *GenericSemanticAdapter) Fields() *mappings.FieldMappingSet { return a.fields }

// Ranks retourne le catalog des rangs de carriere localises (jamais nil).
func (a *GenericSemanticAdapter) Ranks() *mappings.RankCatalog { return a.ranks }

// Assets retourne l'AssetMappingSet (modes, tiers, cadences) ou nil.
func (a *GenericSemanticAdapter) Assets() *mappings.AssetMappingSet { return a.assets }

// Outcomes retourne l'OutcomeMappingSet (win/loss/tie/dnf) ou nil.
func (a *GenericSemanticAdapter) Outcomes() *mappings.OutcomeMappingSet { return a.outcomes }
