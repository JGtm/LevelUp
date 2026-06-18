package games

import (
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// OutcomeResolver résout l'expression SQL title-aware « la colonne d'issue vaut
// le code brut de l'outcome `o` pour ce titre » (MT-06 / PMT-5). Port consommé par
// les repos platform/duckdb pour bâtir les agrégats wins/losses/draws sans
// littéral `outcome = 2` codé en dur. La résolution se fait par code canonique,
// jamais par comparaison de slug (archlint no_slug_comparison). SQLEq retourne
// ok=false si le titre ne déclare pas de raw_code pour cette issue → le caller
// retombe sur son littéral legacy (byte-identique pour Halo), jamais sur le code
// d'un autre titre.
type OutcomeResolver interface {
	SQLEq(slug, col string, o canonical.Outcome) (string, bool)
}

// MappingsOutcomeResolver adapte une mappings.Registry au port OutcomeResolver.
type MappingsOutcomeResolver struct {
	reg         *mappings.Registry
	defaultSlug string
}

// NewMappingsOutcomeResolver construit le resolver autour d'une Registry chargée.
// defaultSlug ("" → "halo_infinite") résout un slug vide (ctx sans titre) vers le
// titre par défaut ; un slug NON vide mais inconnu retourne ok=false (pas de
// fallback cross-titre).
func NewMappingsOutcomeResolver(reg *mappings.Registry, defaultSlug string) *MappingsOutcomeResolver {
	if defaultSlug == "" {
		defaultSlug = "halo_infinite"
	}
	return &MappingsOutcomeResolver{reg: reg, defaultSlug: defaultSlug}
}

// SQLEq résout l'expression SQL d'égalité d'issue pour un titre. Voir OutcomeResolver.
func (r *MappingsOutcomeResolver) SQLEq(slug, col string, o canonical.Outcome) (string, bool) {
	if r == nil || r.reg == nil {
		return "", false
	}
	if slug == "" {
		slug = r.defaultSlug
	}
	set, ok := r.reg.GetOutcomes(slug)
	if !ok {
		return "", false
	}
	return set.SQLEqExpr(col, o)
}
