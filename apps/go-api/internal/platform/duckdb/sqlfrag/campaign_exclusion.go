// Package sqlfrag — campaign_exclusion.go : les fragments SQL de masquage READ-SIDE des matchs de Campagne
// (PvE hors scope de l'app), par titre. La source unique des GUID, analysis.CampaignExcludedVariantIDs, et le
// contexte du masquage sont décrits dans analysis/campaign_exclusion.go.
package sqlfrag

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
)

// SQLExcludeCampaignVariants retourne un fragment SQL prêt à concaténer dans un
// WHERE :
//
//	AND COALESCE(<alias>.game_variant_id, '') NOT IN ('id1','id2', ...)
//
// pour le titre donné, ou "" (no-op) si le titre n'a aucun mode masqué. `alias`
// est l'alias de match_registry / v_match_full / mv_player_matches dans la requête
// appelante (ex. "r", "mr", "reg").
//
// Les GUID sont INLINÉS comme littéraux SQL (aucun placeholder / argument) : ils
// proviennent EXCLUSIVEMENT de campaignExcludedVariantIDs (constantes compile-time,
// jamais d'entrée utilisateur) → aucune surface d'injection. Cette forme sans
// argument évite tout désalignement d'args et s'applique à n'importe quel site de
// lecture, y compris les requêtes multi-CTE ou à IN (%s) dynamique. Les quotes
// sont doublées par prudence (défense en profondeur) bien que les GUID n'en
// contiennent pas.
func SQLExcludeCampaignVariants(titleSlug, alias string) string {
	list := quotedIDList(analysis.CampaignExcludedVariantIDs(titleSlug))
	if list == "" {
		return ""
	}
	return fmt.Sprintf(" AND COALESCE(%s.game_variant_id, '') NOT IN (%s)", alias, list)
}

// quotedIDList quote chaque id en littéral SQL (quotes internes doublées — défense
// en profondeur, les GUID n'en contiennent pas) et les joint par ','. Retourne ""
// si la liste est vide. UNIQUE point de quoting du fichier : les ids proviennent
// EXCLUSIVEMENT de constantes compile-time (analysis.CampaignExcludedVariantIDs) → aucune
// surface d'injection.
func quotedIDList(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + strings.ReplaceAll(id, "'", "''") + "'"
	}
	return strings.Join(quoted, ",")
}

// sqlExcludeByMatchIDSubquery construit la clause SOUS-REQUÊTE d'exclusion par
// match_id pour un jeu de game_variant Campagne, ou "" si ids est vide. Forme
// PARTAGÉE par SQLExcludeCampaignByMatchID (title-aware) et
// SQLExcludeAllCampaignByMatchID (title-agnostic) — même SQL à la source des ids près.
func sqlExcludeByMatchIDSubquery(matchIDCol string, ids []string) string {
	list := quotedIDList(ids)
	if list == "" {
		return ""
	}
	return fmt.Sprintf(
		" AND %s NOT IN (SELECT match_id FROM match_registry WHERE COALESCE(game_variant_id, '') IN (%s))",
		matchIDCol, list)
}

// SQLExcludeCampaignByMatchID retourne un fragment SQL prêt à concaténer dans un
// WHERE :
//
//	AND <matchIDCol> NOT IN (SELECT match_id FROM match_registry
//	                         WHERE COALESCE(game_variant_id, '') IN ('id1','id2', ...))
//
// pour le titre donné, ou "" (no-op) sinon. Forme SOUS-REQUÊTE (self-contained) :
// destinée aux sites de lecture qui agrègent match_participants / v_weapon_kills
// SANS jointure sur match_registry (la colonne game_variant_id n'y est pas). Évite
// de restructurer le FROM/GROUP BY — on filtre par match_id. `matchIDCol` est la
// colonne match_id qualifiée de la requête (ex. "mp.match_id", "wk.match_id").
// Mêmes garanties que SQLExcludeCampaignVariants : GUID littéraux (source unique,
// aucun placeholder / arg), title-agnostic (no-op Infinite).
func SQLExcludeCampaignByMatchID(titleSlug, matchIDCol string) string {
	return sqlExcludeByMatchIDSubquery(matchIDCol, analysis.CampaignExcludedVariantIDs(titleSlug))
}

// SQLExcludeAllCampaignByMatchID : variante TITLE-AGNOSTIC de
// SQLExcludeCampaignByMatchID, pour les rares lecteurs SANS contexte de titre
// (ex. recherche de gamertags cross-joueur — GamertagRepo n'ouvre le shared que
// via un SharedReader, sans slug). Exclut TOUS les game_variant Campagne connus ;
// comme ces GUID sont globalement uniques, la clause est un no-op sur un shared DB
// qui n'en contient aucun (ex. Infinite). NE PAS l'utiliser quand le titre est
// disponible : préférer la forme title-aware (SQLExcludeCampaignByMatchID).
func SQLExcludeAllCampaignByMatchID(matchIDCol string) string {
	return sqlExcludeByMatchIDSubquery(matchIDCol, analysis.AllCampaignVariantIDs())
}

// CampaignExclusionToken est le marqueur inséré dans les requêtes SQL (constantes
// à trous) au point exact où la clause d'exclusion Campagne doit s'insérer — juste
// après la dernière condition WHERE portant sur l'alias registre, AVANT tout
// ORDER BY / GROUP BY. Le caller (qui connaît le titre du joueur) le résout via
// SQLResolveCampaignExclusion. Le token ne contient aucun `%` → il survit à un
// fmt.Sprintf préalable (résolution des placeholders IN (%s)).
const CampaignExclusionToken = "/*__EXCLUDE_CAMPAIGN__*/"

// SQLResolveCampaignExclusion remplace CampaignExclusionToken par la clause
// d'exclusion Campagne pour (titleSlug, alias), ou par une chaîne vide si le titre
// n'a aucun mode masqué (no-op — ex. Infinite). Idempotent si le token est absent.
// `alias` = alias de match_registry / v_match_full / mv_player_matches dans la
// requête (ex. "r").
func SQLResolveCampaignExclusion(query, titleSlug, alias string) string {
	return strings.Replace(query, CampaignExclusionToken, SQLExcludeCampaignVariants(titleSlug, alias), 1)
}

// SQLExcludeCampaignByMatchIDCtx : exclusion de la Campagne pour les lecteurs de sync qui n'ont
// que le contexte. Titre posé dans le contexte : forme title-aware. Titre ABSENT : forme
// title-agnostic (tous les GUID de Campagne connus, no-op sur une base qui n'en porte aucun) —
// jamais le repli silencieux de ctxkeys.TitleSlug sur Infinite, qui laisserait la Campagne d'un
// titre dont le contexte n'a pas été renseigné.
func SQLExcludeCampaignByMatchIDCtx(ctx context.Context, matchIDCol string) string {
	if slug, ok := ctxkeys.TitleSlugIfSet(ctx); ok {
		return SQLExcludeCampaignByMatchID(slug, matchIDCol)
	}
	return SQLExcludeAllCampaignByMatchID(matchIDCol)
}

// SQLResolveCampaignExclusionCtx : résout CampaignExclusionToken comme SQLExcludeCampaignByMatchIDCtx,
// sur la colonne `alias`.match_id (même règle de titre posé / absent).
func SQLResolveCampaignExclusionCtx(ctx context.Context, query, alias string) string {
	return strings.Replace(query, CampaignExclusionToken, SQLExcludeCampaignByMatchIDCtx(ctx, alias+".match_id"), 1)
}
