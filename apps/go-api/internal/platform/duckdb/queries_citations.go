package duckdb

import (
	"regexp"
	"strings"
)

const Q34CitationMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    COALESCE(citation_name_display_en, '') AS citation_name_display_en,
    mapping_type,
    COALESCE(category, 'misc')    AS category,
    image_path,
    description,
    tier_targets,
    composite_children
FROM citation_mappings
WHERE enabled IS NOT FALSE
ORDER BY category, citation_name_display`

// Q35 : Citations â€” totaux agrÃ©gÃ©s depuis match_citations (player stats.duckdb).
// ParamÃ¨tre : aucun.
const Q35CitationTotals = `
SELECT
    citation_name_norm,
    SUM(value) AS total
FROM match_citations_latest
WHERE citation_name_norm NOT LIKE '\_%%' ESCAPE '\'
GROUP BY citation_name_norm
ORDER BY total DESC`

// Q36a : Commendations — total de médailles gagnées par medal_id (xuid du joueur).
// Paramètre : ?1 = xuid du joueur.
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
const Q36aMedalTotals = `
SELECT
    medal_id,
    SUM(count) AS total_count
FROM medals_earned
WHERE xuid = ?
GROUP BY medal_id
ORDER BY total_count DESC`

// Q36b : Commendations â€” mappings mÃ©dailleâ†’citation depuis metadata.duckdb.
// ParamÃ¨tre : aucun. RequÃªte sur pdb.Metadata.
const Q36bMedalCitationMappings = `
SELECT
    medal_id,
    citation_name_display,
    COALESCE(category, 'misc')  AS category,
    image_path
FROM citation_mappings
WHERE mapping_type = 'medal'
  AND enabled IS NOT FALSE
  AND medal_id IS NOT NULL`

// Q38 : Match view — top citations gagnées dans un match.
//
// IMPORTANT — ADR 0016 (post 2026-05-26) : citation_mappings est dans
// metadata.duckdb, pas attachée aux conn player. Q38 a été splitée en :
//   - Q38MatchViewCitationsPlayer : top N citations brutes sur player DB
//   - Q26jCitationMappingsForNormsTemplate (existant) : lookup display côté
//     metadata.duckdb pour la liste de norms
//   - merge en Go (cf. CitationsRepo.LoadMatchCitationsForView)
//
// L'ancienne version Q38MatchViewCitations (LEFT JOIN cross-DB) levait
// "Catalog Error: Table with name citation_mappings does not exist" sur les
// conn player depuis ADR 0016. Le caller capturait nil silencieusement →
// page Match detail affichait des citations vides (incident 2026-05-26).
//
// Q38MatchViewCitationsPlayer — paramètre : ?1 = match_id.
// Retourne 2 colonnes : citation_name_norm, value.
// LIMIT 4 sur player DB → garantit les top 4 par value (même si certains
// norms n'ont pas de mapping côté metadata, ils restent dans le top et
// auront display=norm via COALESCE en Go).
const Q38MatchViewCitationsPlayer = `
SELECT
    citation_name_norm,
    value
FROM match_citations_latest
WHERE match_id = ?
  AND citation_name_norm IS NOT NULL
  AND value > 0
ORDER BY value DESC
LIMIT 4`

// Q40 : Moteur citations complet â€” tous les champs de citation_mappings.
// ParamÃ¨tre : aucun. RequÃªte sur metadata.duckdb (passÃ© comme DB racine dans le sync).
// Retourne 11 colonnes pour le dispatch par mapping_type.
const Q40CitationFullMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    COALESCE(mapping_type, 'medal') AS mapping_type,
    COALESCE(category, 'misc')     AS category,
    medal_id,
    medal_ids,
    stat_name,
    award_name,
    custom_function,
    composite_children,
    tier_targets
FROM citation_mappings
WHERE enabled IS NOT FALSE
ORDER BY citation_name_norm`

// Q39 : Moteur citations â€” mappings citationâ†’medal depuis metadata.duckdb.
// ParamÃ¨tre : aucun. RequÃªte sur pdb.Metadata.
// Retourne 4 colonnes : citation_name_norm, citation_name_display, medal_id, mapping_type.
const Q39CitationMedalMappings = `
SELECT
    citation_name_norm,
    citation_name_display,
    medal_id,
    COALESCE(mapping_type, 'medal') AS mapping_type
FROM citation_mappings
WHERE enabled IS NOT FALSE
  AND medal_id IS NOT NULL
ORDER BY citation_name_norm`

// Médias — les anciens builders SQL Q37 (BuildQ37MediaQuery/Count/Options)
// ont été supprimés en P1 (refactor pipeline Go via SharedReader, ADR 0016).
// Les expressions SQL `q37*LabelExpr` et leurs régressions cross-DB
// (`shared.match_registry`) sont remplacées par les helpers Go équivalents
// dans media_repo_q37_pipeline.go (computedMapLabel, computedModeLabel,
// computedPlaylistLabel) — chargés via loadMediaMatchRegistry sur SharedReader.

type mediaWhereConfig struct {
	includeMapFilter      bool
	includeModeFilter     bool
	includePlaylistFilter bool
}

// mediaKindEquivalents retourne les valeurs DB qui doivent matcher un filtre
// de type donné, en couvrant à la fois la convention legacy ("clip"/"screenshot")
// et la nouvelle ("video"/"image"). Les médias indexés par les anciennes
// versions ont l'une, les nouveaux uploads ont l'autre — sans cette translation
// le filtre type ne retourne 0 résultat.
func mediaKindEquivalents(kind string) []string {
	switch kind {
	case "clip", "video":
		return []string{"clip", "video"}
	case "screenshot", "image":
		return []string{"screenshot", "image"}
	default:
		return []string{kind}
	}
}

// Strips appliqués côté Go pour normaliser une valeur de filtre map.
// Pattern conservateur : ne touche pas " Annex", `:`, etc.
var (
	mediaMapForgeSuffixRe   = regexp.MustCompile(`(?i)\s*-\s*Forge.*$`)
	mediaMapRankedSuffixRe  = regexp.MustCompile(`(?i)\s*-\s*Ranked.*$`)
	mediaMapVersionSuffixRe = regexp.MustCompile(`(?i)\s+v\d+$`)
)

// normalizeMediaMapName strippe les suffixes de variante pour grouper
// "Recharge v3" / "Recharge - Forge" / "Recharge" sous le même nom canonique.
func normalizeMediaMapName(s string) string {
	s = mediaMapForgeSuffixRe.ReplaceAllString(s, "")
	s = mediaMapRankedSuffixRe.ReplaceAllString(s, "")
	s = mediaMapVersionSuffixRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// mediaQueryConfig porte les paramètres de scoping (player_slug) pour le
// pipeline Q37. Utilisé par media_repo_q37_pipeline.go.
type mediaQueryConfig struct {
	playerSlug string
}

func (cfg mediaQueryConfig) useSharedSocialSchema() bool {
	return cfg.playerSlug != ""
}

// baseWhereClause renvoie les contraintes de base + le filtre de section (ownership).
//
//	"" (vide)   → sources visibles : mine + teammate (pas de contrainte player_slug)
//	"mine"      → uniquement player_slug courant
//	"teammate"  → uniquement les autres (player_slug != courant)
//
// En schéma legacy (pas de player_slug), seul "mine" et "" donnent des résultats ;
// "teammate" force WHERE FALSE pour cohérence (rien à montrer).
func (cfg mediaQueryConfig) baseWhereClause(sectionFilter string) ([]string, []any) {
	if !cfg.useSharedSocialSchema() {
		switch sectionFilter {
		case "teammate":
			return []string{"FALSE"}, nil
		default:
			return []string{"mf.status = 'active'"}, nil
		}
	}
	switch sectionFilter {
	case "mine":
		return []string{"mf.player_slug = ?"}, []any{cfg.playerSlug}
	case "teammate":
		return []string{"mf.player_slug <> ?"}, []any{cfg.playerSlug}
	default:
		// Tous auteurs — pas de contrainte sur player_slug
		return nil, nil
	}
}

// Q41 : MatchView Summary — match_citations + cumul global pour un seul match (player DB).
// Les métadonnées (display, image_path, tier_targets, description) sont chargées
// séparément via Q26j sur la metadata DB et mergées en Go (citation_mappings n'est
// pas attachée au player DB — voir pool.go).
const Q41SummaryTabCitations = `
SELECT
    mc.citation_name_norm,
    mc.value                AS match_delta,
    cum.total               AS cumulative_total
FROM match_citations_latest mc
JOIN (
    SELECT citation_name_norm, SUM(value) AS total
    FROM match_citations_latest
    GROUP BY citation_name_norm
) cum ON cum.citation_name_norm = mc.citation_name_norm
WHERE mc.match_id = ?
  AND mc.value > 0
ORDER BY mc.value DESC`
