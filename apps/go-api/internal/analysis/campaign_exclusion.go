// Package analysis — campaign_exclusion.go : donnée partagée du masquage
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
// Source UNIQUE des GUID : campaignExcludedVariantIDs ci-dessous. Les fragments SQL qui
// l'emploient vivent dans platform/duckdb/sqlfrag (campaign_exclusion.go) ; la donnée reste ici, importable
// sans cycle par ce paquet et par progression/*. La colonne game_variant_id vit sur match_registry (et ses vues
// v_match_full / mv_player_matches) ; match_participants ne la porte pas → un site
// qui lit match_participants seul doit d'abord joindre le registre.
package analysis

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

// AllCampaignVariantIDs : union des game_variant Campagne de TOUS les titres (copie).
// Les GUID étant globalement uniques (issus de l'API Metadata Halo), cette union
// sert la variante title-agnostic ci-dessous sans risque de faux positif sur un
// autre titre.
func AllCampaignVariantIDs() []string {
	var out []string
	for _, ids := range campaignExcludedVariantIDs {
		out = append(out, ids...)
	}
	return out
}
