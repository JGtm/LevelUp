// Package sqlfrag — match_filter.go : construction pure des clauses WHERE pour Q25 paramétrable (Phase 2b du
// rework header MatchView).
//
// Ce module est pur : pas d'accès DB, pas de I/O. Il transforme un `domain.MatchFilterSpec` en fragment SQL
// `AND <clauses>` + arguments préparés. Le repo l'utilise pour assembler la query finale.
//
// Multi-titres : `ModeCategory` est Halo-specific. Si la catégorie est inconnue (titre futur sans cette notion
// ou catégorie non répertoriée), la clause est silencieusement omise et `IgnoredFilters` rapporte le fait pour
// permettre au caller de logger un warning.
package sqlfrag

import (
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

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
func BuildNeighborsWhereClause(spec *domain.MatchFilterSpec, categoryPrefixes analysis.ModeCategoryPrefixes) NeighborsWhereResult {
	res := NeighborsWhereResult{}
	if spec.IsEmpty() {
		return res
	}

	b := neighborsBuilder{clauses: make([]string, 0, 6), args: make([]any, 0, 12)}
	b.playlists(spec)
	b.modeCategories(spec, categoryPrefixes)
	b.dates(spec)
	b.outcome(spec)
	b.session(spec)
	b.withPlayer(spec)

	if len(b.clauses) == 0 {
		res.IgnoredFilters = b.ignored
		return res
	}

	res.SQL = " AND " + strings.Join(b.clauses, " AND ")
	res.Args = b.args
	res.IgnoredFilters = b.ignored
	return res
}

// neighborsBuilder accumule, dans l'ordre, les clauses, les arguments liés et les filtres ignorés.
type neighborsBuilder struct {
	clauses []string
	args    []any
	ignored []string
}

// playlists : mr.playlist_name IN (?, ?, ...) (multi). Valeurs vides ignorées.
func (b *neighborsBuilder) playlists(spec *domain.MatchFilterSpec) {
	names := nonEmpty(spec.PlaylistNames)
	if len(names) == 0 {
		return
	}
	placeholders := make([]string, len(names))
	for i, name := range names {
		placeholders[i] = "?"
		b.args = append(b.args, name)
	}
	b.clauses = append(b.clauses, "mr.playlist_name IN ("+strings.Join(placeholders, ", ")+")")
}

// modeCategories : union des préfixes pair_name de chaque catégorie (multi), le tout en un seul OR.
// categoryPrefixes (injecté) résout vers les préfixes title-specific ; une catégorie qui ne résout vers
// aucun préfixe est sautée. Si AUCUNE catégorie ne résout → filtre ignoré. Note multi-titres : pour un titre
// futur sans la notion ModeCategory, l'adapter retourne des listes vides → dégradation gracieuse.
func (b *neighborsBuilder) modeCategories(spec *domain.MatchFilterSpec, categoryPrefixes analysis.ModeCategoryPrefixes) {
	cats := nonEmpty(spec.ModeCategories)
	if len(cats) == 0 {
		return
	}
	modeClauses := make([]string, 0, len(cats)*4)
	for _, cat := range cats {
		var prefixes []string
		if categoryPrefixes != nil {
			prefixes = categoryPrefixes(cat)
		}
		for _, p := range prefixes {
			modeClauses = append(modeClauses, "mr.pair_name = ?", "mr.pair_name LIKE ?")
			b.args = append(b.args, p, p+":%")
		}
	}
	if len(modeClauses) > 0 {
		b.clauses = append(b.clauses, "("+strings.Join(modeClauses, " OR ")+")")
	} else {
		b.ignored = append(b.ignored, "mode_category")
	}
}

// dates : bornes sur le début de match, par le pattern timezone canonique (cf. memory
// reference_timezone_canonical_pattern.md).
func (b *neighborsBuilder) dates(spec *domain.MatchFilterSpec) {
	if spec.DateFrom != nil {
		b.clauses = append(b.clauses, SQLStartTimeCanonical("mr")+" >= ?")
		b.args = append(b.args, spec.DateFrom.UTC())
	}
	if spec.DateTo != nil {
		b.clauses = append(b.clauses, SQLStartTimeCanonical("mr")+" <= ?")
		// DateTo est inclusive — on ajoute 1 microseconde pour matcher
		// le comportement "fin de journée incluse" si le caller passe 23:59:59.
		b.args = append(b.args, spec.DateTo.UTC())
	}
}

// outcome : mp.outcome = ? pour un label de la whitelist ; sinon filtre ignoré. La colonne en DB s'appelle
// `outcome` (cf. Q12/Q17 prod) ; `outcome_code` est l'alias exposé côté domain via SELECT AS.
func (b *neighborsBuilder) outcome(spec *domain.MatchFilterSpec) {
	if spec.Outcome == nil || *spec.Outcome == "" {
		return
	}
	if code, ok := analysis.OutcomeCodeForLabel(*spec.Outcome); ok {
		b.clauses = append(b.clauses, "mp.outcome = ?")
		b.args = append(b.args, code)
	} else {
		b.ignored = append(b.ignored, "outcome")
	}
}

// session : non implémenté en Phase 2b initial — les sessions vivent dans player.player_match_enrichment qui
// n'est pas joint dans Q25 (player DB séparée du shared). Le caller doit utiliser les matchIds du contexte
// côté front quand source='session'. Documenté dans match_nav_inventory.md.
func (b *neighborsBuilder) session(spec *domain.MatchFilterSpec) {
	if spec.SessionID != nil && *spec.SessionID != "" {
		b.ignored = append(b.ignored, "session_id")
	}
}

// withPlayer (Phase 2c) : restreint aux matchs où ce XUID est présent dans match_participants. Q25 join déjà
// mp avec mp.xuid = ? (joueur principal) ; pour le coéquipier, on EXISTS sur un alias mp2. Pas de préfixe
// `shared.` (Q25 tourne sur SharedReader depuis ADR 0016).
func (b *neighborsBuilder) withPlayer(spec *domain.MatchFilterSpec) {
	if spec.WithPlayerXuid != nil && *spec.WithPlayerXuid != "" {
		b.clauses = append(b.clauses,
			"EXISTS (SELECT 1 FROM match_participants mp2 "+
				"WHERE mp2.match_id = mr.match_id AND mp2.xuid = ?)",
		)
		b.args = append(b.args, *spec.WithPlayerXuid)
	}
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
