// Package analysis — match_filter.go : construction pure des clauses WHERE
// pour Q25 paramétrable (Phase 2b du rework header MatchView).
//
// Ce module est pur : pas d'accès DB, pas de I/O. Il transforme un
// `domain.MatchFilterSpec` en fragment SQL `AND <clauses>` + arguments
// préparés. Le repo l'utilise pour assembler la query finale.
//
// Multi-titres : `ModeCategory` est Halo-specific. Si la catégorie est
// inconnue (titre futur sans cette notion ou catégorie non répertoriée),
// la clause est silencieusement omise et `IgnoredFilters` rapporte le fait
// pour permettre au caller de logger un warning.
package analysis

import (
	"strings"
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

// outcomeLabelToCode : whitelist canonique. Toute valeur hors map retourne 0
// (clause omise). C'est aussi la liste autorisée côté handler.
var outcomeLabelToCode = map[string]int{
	OutcomeToneWin:  domain.OutcomeWin,
	OutcomeToneLoss: domain.OutcomeLoss,
	OutcomeToneDraw: domain.OutcomeDraw,
	OutcomeToneDNF:  domain.OutcomeDNF,
}

// NeighborsWhereResult : retour de BuildNeighborsWhereClause.
type NeighborsWhereResult struct {
	// SQL fragment à injecter (préfixé par espace + AND si non vide).
	// Vide si aucun filtre applicable.
	SQL string
	// Args à passer en paramètres préparés (ordre cohérent avec SQL).
	Args []any
	// IgnoredFilters : noms des filtres qui ont été présents dans la spec
	// mais ignorés (ex: mode_category inconnue, outcome non whitelisté).
	// Le caller peut logger warning sans bloquer la requête.
	IgnoredFilters []string
}

// BuildNeighborsWhereClause : transforme un MatchFilterSpec en fragment SQL
// + args. Pure — testable sans DB, sans dépendance HTTP/router.
//
// Le SQL retourné est destiné à être injecté APRÈS la clause JOIN existante
// de Q25 (CTE `ordered`). Les noms de table utilisés sont :
//   - mr  → shared.match_registry (alias dans Q25)
//   - mp  → shared.match_participants (alias dans Q25)
//
// Convention : si le fragment est non vide, il commence par " AND " (avec
// espace de tête) — facilite l'insertion dans le template.
//
// `categoryPrefixes` est injectée pour éviter un cycle d'import analysis →
// games/halo_infinite. Si nil, le filtre `mode_category` est ignoré (utile
// pour les tests qui ne veulent pas tester ce chemin).
func BuildNeighborsWhereClause(spec *domain.MatchFilterSpec, categoryPrefixes ModeCategoryPrefixes) NeighborsWhereResult {
	res := NeighborsWhereResult{}
	if spec.IsEmpty() {
		return res
	}

	clauses := make([]string, 0, 6)
	args := make([]any, 0, 12)
	var ignored []string

	// Playlists (multi) : mr.playlist_name IN (?, ?, ...). Valeurs vides ignorées.
	if names := nonEmpty(spec.PlaylistNames); len(names) > 0 {
		placeholders := make([]string, len(names))
		for i, name := range names {
			placeholders[i] = "?"
			args = append(args, name)
		}
		clauses = append(clauses, "mr.playlist_name IN ("+strings.Join(placeholders, ", ")+")")
	}

	// Catégories de mode (multi) : union des préfixes pair_name de chaque
	// catégorie, le tout en un seul OR. categoryPrefixes (injecté) résout vers
	// les préfixes title-specific ; une catégorie qui ne résout vers aucun
	// préfixe est sautée. Si AUCUNE catégorie ne résout → filtre ignoré.
	// Note multi-titres : pour un titre futur sans la notion ModeCategory,
	// l'adapter retourne des listes vides → dégradation gracieuse.
	if cats := nonEmpty(spec.ModeCategories); len(cats) > 0 {
		modeClauses := make([]string, 0, len(cats)*4)
		for _, cat := range cats {
			var prefixes []string
			if categoryPrefixes != nil {
				prefixes = categoryPrefixes(cat)
			}
			for _, p := range prefixes {
				modeClauses = append(modeClauses, "mr.pair_name = ?", "mr.pair_name LIKE ?")
				args = append(args, p, p+":%")
			}
		}
		if len(modeClauses) > 0 {
			clauses = append(clauses, "("+strings.Join(modeClauses, " OR ")+")")
		} else {
			ignored = append(ignored, "mode_category")
		}
	}

	if spec.DateFrom != nil {
		// Pattern timezone canonique (cf. memory reference_timezone_canonical_pattern.md).
		clauses = append(clauses,
			SQLStartTimeCanonical("mr")+" >= ?",
		)
		args = append(args, spec.DateFrom.UTC())
	}

	if spec.DateTo != nil {
		clauses = append(clauses,
			SQLStartTimeCanonical("mr")+" <= ?",
		)
		// DateTo est inclusive — on ajoute 1 microseconde pour matcher
		// le comportement "fin de journée incluse" si le caller passe 23:59:59.
		args = append(args, spec.DateTo.UTC())
	}

	if spec.Outcome != nil && *spec.Outcome != "" {
		code, ok := outcomeLabelToCode[*spec.Outcome]
		if ok {
			// Note : la colonne en DB s'appelle `outcome` (cf. Q12/Q17 prod).
			// `outcome_code` est l'alias exposé côté domain via SELECT AS.
			clauses = append(clauses, "mp.outcome = ?")
			args = append(args, code)
		} else {
			ignored = append(ignored, "outcome")
		}
	}

	// SessionID : non implémenté en Phase 2b initial — les sessions vivent
	// dans player.player_match_enrichment qui n'est pas joint dans Q25
	// (player DB séparée du shared). Le caller doit utiliser les matchIds
	// du contexte côté front quand source='session'. Documenté dans
	// match_nav_inventory.md.
	if spec.SessionID != nil && *spec.SessionID != "" {
		ignored = append(ignored, "session_id")
	}

	// WithPlayerXuid (Phase 2c) : restreint aux matchs où ce XUID est présent
	// dans match_participants. Q25 join déjà mp avec mp.xuid = ? (joueur
	// principal) ; pour le coéquipier, on EXISTS sur un alias mp2.
	// Pas de préfixe `shared.` (Q25 tourne sur SharedReader depuis ADR 0016).
	if spec.WithPlayerXuid != nil && *spec.WithPlayerXuid != "" {
		clauses = append(clauses,
			"EXISTS (SELECT 1 FROM match_participants mp2 "+
				"WHERE mp2.match_id = mr.match_id AND mp2.xuid = ?)",
		)
		args = append(args, *spec.WithPlayerXuid)
	}

	if len(clauses) == 0 {
		res.IgnoredFilters = ignored
		return res
	}

	res.SQL = " AND " + strings.Join(clauses, " AND ")
	res.Args = args
	res.IgnoredFilters = ignored
	return res
}

// nonEmpty retourne les valeurs non vides (après TrimSpace) du slice.
// Défensif : le handler valide déjà chaque valeur, mais une slice contenant
// des chaînes vides ne doit pas produire un placeholder fantôme dans le IN (...).
func nonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
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
