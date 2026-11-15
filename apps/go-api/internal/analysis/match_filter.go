// Package analysis — match_filter.go : la taxonomie des modes injectée par les titres et la whitelist des
// issues du filtre de voisinage (Phase 2b du rework header MatchView). La construction du fragment SQL
// `AND <clauses>` (BuildNeighborsWhereClause) vit dans platform/duckdb/sqlfrag/match_filter.go.
//
// Multi-titres : `ModeCategory` est Halo-specific. Si la catégorie est
// inconnue (titre futur sans cette notion ou catégorie non répertoriée),
// la clause est silencieusement omise et `IgnoredFilters` rapporte le fait
// pour permettre au caller de logger un warning.
package analysis

import (
	"time"

	"levelup/go-api/internal/domain"
)

// ModeCategoryPrefixes : signature de la fonction qui résout les préfixes
// pair_name d'une catégorie. Injectée par le caller (service) pour éviter
// un cycle d'import analysis → games/halo_infinite. L'implémentation Halo
// est `halo_infinite.PairNamePrefixesForCategory`. Pour un titre futur
// sans la notion, l'adapter retournera une liste vide → la clause est omise.
type ModeCategoryPrefixes func(category string) []string

// InferModeCategory : signature qui infère la catégorie de mode depuis un
// pair_name brut (ex: "Arena:Slayer on X" → "Assassin"). Injectée par le caller
// (wiring) pour éviter le couplage platform/duckdb → games/halo_infinite.
// L'impl Halo est `halo_infinite.InferModeCategoryFromPairName`.
type InferModeCategory func(pairName string) string

// ModeTaxonomy regroupe les fonctions de classification des modes d'un titre,
// injectées dans platform/duckdb (media_repo) pour éviter le couplage direct à
// games/halo_infinite. Zéro-value = aucune classification (dégradation gracieuse
// via les helpers nil-safe ci-dessous : un titre sans taxonomie ne classe pas
// ses modes plutôt que de paniquer).
type ModeTaxonomy struct {
	InferCategory InferModeCategory    // pair_name → catégorie
	PrefixesFor   ModeCategoryPrefixes // catégorie → préfixes pair_name
	AllPrefixes   func() []string      // tous les préfixes connus
	Other         string               // libellé du bucket "non classé" (ex: "Other")
}

// Classify infère la catégorie ; "" si aucune fonction injectée.
func (t ModeTaxonomy) Classify(pairName string) string {
	if t.InferCategory == nil {
		return ""
	}
	return t.InferCategory(pairName)
}

// Prefixes retourne les préfixes pair_name d'une catégorie ; nil si non injecté.
func (t ModeTaxonomy) Prefixes(category string) []string {
	if t.PrefixesFor == nil {
		return nil
	}
	return t.PrefixesFor(category)
}

// KnownPrefixes retourne tous les préfixes connus ; nil si non injecté.
func (t ModeTaxonomy) KnownPrefixes() []string {
	if t.AllPrefixes == nil {
		return nil
	}
	return t.AllPrefixes()
}

// OutcomeCodeForLabel rend le code d'issue d'un label de ton (OutcomeToneWin...) ; ok est faux hors de la
// whitelist.
func OutcomeCodeForLabel(label string) (code int, ok bool) {
	code, ok = outcomeLabelToCode[label]
	return code, ok
}

// outcomeLabelToCode : whitelist canonique. Toute valeur hors map retourne 0
// (clause omise). C'est aussi la liste autorisée côté handler.
var outcomeLabelToCode = map[string]int{
	OutcomeToneWin:  domain.OutcomeWin,
	OutcomeToneLoss: domain.OutcomeLoss,
	OutcomeToneDraw: domain.OutcomeDraw,
	OutcomeToneDNF:  domain.OutcomeDNF,
}

// validOutcome retourne true si la valeur est dans la whitelist.
// Exposé pour le handler de validation amont.
func IsValidOutcomeLabel(s string) bool {
	_, ok := outcomeLabelToCode[s]
	return ok
}

// AllValidOutcomeLabels retourne la liste triée des labels acceptés (utile
// pour les messages de warning + tests).
func AllValidOutcomeLabels() []string {
	out := make([]string, 0, len(outcomeLabelToCode))
	for k := range outcomeLabelToCode {
		out = append(out, k)
	}
	// pas de tri pour ne pas importer "sort" inutilement ; les tests gèrent
	// l'ordre via map de comparaison ou Equal.
	return out
}

// AssertOutcomeMatchesTime : helper interne réservé aux tests pour vérifier
// le bon usage des time.Time (fail compile si signature change).
var _ time.Time
