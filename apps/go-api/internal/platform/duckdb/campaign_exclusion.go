// Package duckdb — campaign_exclusion.go : alias package-local du masquage
// READ-SIDE des matchs de mode Campagne (item backlog H1).
//
// La SOURCE UNIQUE (liste des game_variant_id + construction de la clause SQL)
// vit dans internal/analysis (campaign_exclusion.go), au même titre que le
// fragment timezone canonique — ainsi elle est importable sans cycle par
// platform/duckdb, analysis (match_filter) et progression/*. Ce fichier n'expose
// que des alias package-local pour éviter d'importer `analysis` dans chaque
// fichier de constantes SQL (cf. StartTimeCanonicalSQL, même motif).
//
// Deux formes d'application :
//   - campaignExclusionToken : marqueur à INSÉRER dans une constante SQL au point
//     d'injection (après la dernière condition WHERE sur l'alias registre, avant
//     tout ORDER BY / GROUP BY), résolu au runtime par resolveCampaignExclusion
//     dans le caller (qui connaît le titre du joueur). Survit à un fmt.Sprintf.
//   - SQLExcludeCampaignVariants (via analysis) : clause littérale directement
//     concaténable quand le caller compose déjà la requête (ex. player_matches_repo).
package duckdb

import "levelup/go-api/internal/analysis"

// campaignExclusionToken : marqueur inséré dans les constantes SQL, résolu par
// resolveCampaignExclusion. Alias de analysis.CampaignExclusionToken.
const campaignExclusionToken = analysis.CampaignExclusionToken

// resolveCampaignExclusion remplace le token campagne d'une requête par la clause
// d'exclusion du titre (no-op si le titre n'a aucun mode masqué, ex. Infinite).
// Délègue à la source unique analysis.SQLResolveCampaignExclusion. `alias` =
// alias de match_registry / v_match_full / mv_player_matches dans la requête.
func resolveCampaignExclusion(query, titleSlug, alias string) string {
	return analysis.SQLResolveCampaignExclusion(query, titleSlug, alias)
}

// excludeCampaignClause retourne la clause littérale ` AND COALESCE(<alias>.
// game_variant_id,”) NOT IN (...)` (ou "" si le titre n'a aucun mode masqué), à
// concaténer directement quand le caller compose la requête en Go (SQL inline)
// ET que match_registry est déjà joint sous cet alias.
// Délègue à la source unique analysis.SQLExcludeCampaignVariants.
func excludeCampaignClause(titleSlug, alias string) string {
	return analysis.SQLExcludeCampaignVariants(titleSlug, alias)
}

// excludeCampaignByMatchID retourne la clause sous-requête ` AND <matchIDCol> NOT
// IN (SELECT match_id FROM match_registry WHERE game_variant_id IN (...))` (ou "" ),
// pour les requêtes participants-only / v_weapon_kills SANS jointure registre.
// Délègue à la source unique analysis.SQLExcludeCampaignByMatchID.
func excludeCampaignByMatchID(titleSlug, matchIDCol string) string {
	return analysis.SQLExcludeCampaignByMatchID(titleSlug, matchIDCol)
}

// excludeAllCampaignByMatchID : variante TITLE-AGNOSTIC pour les lecteurs sans
// contexte de titre (GamertagRepo). Délègue à analysis.SQLExcludeAllCampaignByMatchID.
func excludeAllCampaignByMatchID(matchIDCol string) string {
	return analysis.SQLExcludeAllCampaignByMatchID(matchIDCol)
}
