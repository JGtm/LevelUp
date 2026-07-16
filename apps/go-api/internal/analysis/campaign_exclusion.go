// Package analysis — campaign_exclusion.go : fragment SQL partagé de masquage
// READ-SIDE des matchs de mode Campagne (PvE hors scope de l'app), par titre.
//
// Contexte (item backlog H1) : Halo 5 a historiquement ingéré des matchs de
// Campagne (solo + coop, GameMode 2 de l'API). Ils ne sont plus collectés
// (cf. internal/games/halo_5.isExcludedH5GameMode), mais ~287 matchs historiques
// subsistent dans le shared DB h5 (avec lignes match_participants ET
// player_match_enrichment). Décision produit : on les MASQUE À LA LECTURE plutôt
// que de purger la base (interdit — historique de corruption d'index ART DuckDB,
// cf. ADR 0019/0026 : aucun DELETE/UPSERT sauvage).
//
// Identification : par game_variant_id (GUID stables de l'API Metadata Halo 5).
// game_variant_name / mode_category / is_firefight sont NULL/FALSE pour ces
// matchs → le game_variant_id est le SEUL discriminant fiable (vérifié sur base).
//
// Source UNIQUE des GUID : campaignExcludedVariantIDs ci-dessous. Ce fragment est
// hébergé dans `analysis` (comme SQLStartTimeCanonical) pour être importable SANS
// cycle par platform/duckdb, par analysis lui-même (match_filter) et par
// progression/*. La colonne game_variant_id vit sur match_registry (et ses vues
// v_match_full / mv_player_matches) ; match_participants ne la porte pas → un site
// qui lit match_participants seul doit d'abord joindre le registre.
package analysis

import (
	"fmt"
	"strings"
)

// campaignExcludedVariantIDs : game_variant_id masqués à la LECTURE, par titre.
// Title-agnostic par construction : un titre absent de la map (ex. halo_infinite,
// qui n'a aucun match Campagne — le PvE Firefight vit dans shared_pve.duckdb, pas
// dans match_registry) → aucune clause émise (no-op). On branche sur la DONNÉE du
// titre (ses variants), jamais sur `slug == "..."` en dur dans la logique métier.
var campaignExcludedVariantIDs = map[string][]string{
	"halo_5": {
		"00000003-0000-0010-8000-00aa00389b71", // Campaign
		"67ffc2ff-a50e-4e5d-ae08-b40e3d961061", // Campaign Score Attack
	},
}

// CampaignExcludedVariantIDs retourne la liste (copie) des game_variant_id de
// Campagne masqués pour le titre donné, ou nil si aucun. Exporté pour les tests
// et les rares consommateurs Go (jamais reconstruire la liste ailleurs).
func CampaignExcludedVariantIDs(titleSlug string) []string {
	ids := campaignExcludedVariantIDs[titleSlug]
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	copy(out, ids)
	return out
}

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
	ids := campaignExcludedVariantIDs[titleSlug]
	if len(ids) == 0 {
		return ""
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + strings.ReplaceAll(id, "'", "''") + "'"
	}
	return fmt.Sprintf(" AND COALESCE(%s.game_variant_id, '') NOT IN (%s)",
		alias, strings.Join(quoted, ","))
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
	ids := campaignExcludedVariantIDs[titleSlug]
	if len(ids) == 0 {
		return ""
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + strings.ReplaceAll(id, "'", "''") + "'"
	}
	return fmt.Sprintf(
		" AND %s NOT IN (SELECT match_id FROM match_registry WHERE COALESCE(game_variant_id, '') IN (%s))",
		matchIDCol, strings.Join(quoted, ","))
}

// allCampaignVariantIDs : union des game_variant Campagne de TOUS les titres.
// Les GUID étant globalement uniques (issus de l'API Metadata Halo), cette union
// sert la variante title-agnostic ci-dessous sans risque de faux positif sur un
// autre titre.
func allCampaignVariantIDs() []string {
	var out []string
	for _, ids := range campaignExcludedVariantIDs {
		out = append(out, ids...)
	}
	return out
}

// SQLExcludeAllCampaignByMatchID : variante TITLE-AGNOSTIC de
// SQLExcludeCampaignByMatchID, pour les rares lecteurs SANS contexte de titre
// (ex. recherche de gamertags cross-joueur — GamertagRepo n'ouvre le shared que
// via un SharedReader, sans slug). Exclut TOUS les game_variant Campagne connus ;
// comme ces GUID sont globalement uniques, la clause est un no-op sur un shared DB
// qui n'en contient aucun (ex. Infinite). NE PAS l'utiliser quand le titre est
// disponible : préférer la forme title-aware (SQLExcludeCampaignByMatchID).
func SQLExcludeAllCampaignByMatchID(matchIDCol string) string {
	ids := allCampaignVariantIDs()
	if len(ids) == 0 {
		return ""
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "'" + strings.ReplaceAll(id, "'", "''") + "'"
	}
	return fmt.Sprintf(
		" AND %s NOT IN (SELECT match_id FROM match_registry WHERE COALESCE(game_variant_id, '') IN (%s))",
		matchIDCol, strings.Join(quoted, ","))
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
