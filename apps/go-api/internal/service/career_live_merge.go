// Package service — career_live_merge.go : merge per-field cache+DB pour la
// home read path + overlay final identity vs DB fallback.
//
// Extrait de career_live_service.go (refactor V2 dette technique 2026-05-26).
//
// Responsabilités :
//   - mergeCareerRow : fusion live (progress + custom) + dbLast → CareerRankRow
//     unique servie au front. Carry-forward strict per-field (non-empty wins).
//   - overlayIdentityFromFallback : dernier patch de l'identity rendue par le
//     builder, depuis la row DB last-known-good — sécurise les cas où live a
//     produit identity != nil mais avec des champs vides.
//
// DIRECTIVE PRODUIT apparence (rappelée 2026-07-08) : bannière, emblème et
// backdrop sont des champs INDÉPENDANTS — chacun affiche TOUJOURS une valeur
// (l'actuelle si résoluble, sinon la dernière connue), jamais vide, et sans
// aucun couplage entre eux. Le carry-forward est donc inconditionnel,
// champ par champ. Contexte documenté : les emblèmes nouvelle génération
// (`<id>-SpartanEmblem`, ex. 3806589 équipé par JGtm le 2026-07-03) n'ont
// AUCUNE nameplate upstream (absents de mapping.json, aucune cfg positive,
// 404 CDN) — la bannière servie reste alors la dernière connue.
package service

import (
	"levelup/go-api/internal/domain"
)

// overlayIdentityFromFallback applique le filet DB last-known-good par-dessus
// le résultat live. Patché en place — l'objet identity retourné peut être :
//   - identity (live) avec champs vides remplis depuis fallback
//   - fallback (si identity était nil)
//   - nil (si les deux sont nil)
//
// Anti-régression « bannière qui va et vient » : un fetch live qui rend
// BannerImageURL=nil ne doit JAMAIS écraser la valeur DB historique
// (directive « jamais vide », cf. en-tête de fichier). Champs indépendants :
// chaque asset est patché pour lui-même, sans condition croisée.
func overlayIdentityFromFallback(identity, fallback *domain.HomeSpartanIdentityRow) *domain.HomeSpartanIdentityRow {
	if identity == nil {
		return fallback
	}
	if fallback == nil {
		return identity
	}
	if identity.SpartanID == nil && fallback.SpartanID != nil {
		identity.SpartanID = fallback.SpartanID
	}
	if identity.BannerImageURL == nil && fallback.BannerImageURL != nil {
		identity.BannerImageURL = fallback.BannerImageURL
	}
	if identity.EmblemImageURL == nil && fallback.EmblemImageURL != nil {
		identity.EmblemImageURL = fallback.EmblemImageURL
	}
	if identity.BackdropImageURL == nil && fallback.BackdropImageURL != nil {
		identity.BackdropImageURL = fallback.BackdropImageURL
	}
	if identity.AdornmentImageURL == nil && fallback.AdornmentImageURL != nil {
		identity.AdornmentImageURL = fallback.AdornmentImageURL
	}
	return identity
}

// mergeCareerRow fusionne progress (live) + custom (live) + dbLast (carry-
// forward) en une seule CareerRankRow. Ordre de priorité par champ :
//
//	live (si non-vide) → dbLast → zéro-valeur
//
// Si live retourne quelque chose pour un champ mais que la valeur est zéro
// ou chaîne vide, le DB l'écrase. C'est exactement le comportement « per-
// field fallback » attendu : on ne remplace jamais une valeur connue par
// une valeur vide remontée d'un fetch partiellement réussi.
//
// Retourne nil si toutes les sources sont vides.
func mergeCareerRow(
	progress *domain.CareerRankSnapshot,
	custom *domain.SpartanCustomizationData,
	dbLast *domain.CareerRankRow,
) *domain.CareerRankRow {
	if progress == nil && custom == nil && dbLast == nil {
		return nil
	}

	merged := &domain.CareerRankRow{}
	carryForward := mergeProgressInto(merged, progress, dbLast)
	carryForward = mergeCustomInto(merged, custom, dbLast) || carryForward
	carryForward = carryForwardDerivedFields(merged, dbLast) || carryForward

	if carryForward {
		careerLivePerFieldMerge.Add(1)
	}
	if merged.IsEmpty() {
		return nil
	}
	return merged
}

// mergeProgressInto applique progress live + dbLast carry-forward sur les
// champs rank/current_xp/is_max_rank. Retourne true si au moins un champ a
// été carry-forward depuis dbLast.
func mergeProgressInto(merged *domain.CareerRankRow, progress *domain.CareerRankSnapshot, dbLast *domain.CareerRankRow) bool {
	if progress != nil {
		merged.Rank = progress.CurrentRank
		merged.CurrentXP = progress.CurrentXP
		merged.IsMaxRank = progress.IsMaxRank
	}
	if dbLast == nil {
		return false
	}
	carryForward := false
	if merged.Rank <= 0 && dbLast.Rank > 0 {
		merged.Rank = dbLast.Rank
		carryForward = true
	}
	// current_xp : on ne réécrit JAMAIS depuis dbLast quand progress live a
	// réussi — même un current_xp=0 live est l'état réel du joueur (palier
	// juste franchi). Le carry-forward ne sert qu'en l'absence totale de live.
	if progress == nil && merged.CurrentXP == 0 {
		merged.CurrentXP = dbLast.CurrentXP
	}
	if progress == nil {
		merged.IsMaxRank = dbLast.IsMaxRank
	}
	return carryForward
}

// mergeCustomInto applique custom live + dbLast carry-forward sur les champs
// spartan_id/banner/emblem/backdrop. Retourne true si au moins un champ a été
// carry-forward depuis dbLast. Chaque champ est indépendant : carry-forward
// inconditionnel par champ (directive « jamais vide », cf. en-tête de fichier).
func mergeCustomInto(merged *domain.CareerRankRow, custom *domain.SpartanCustomizationData, dbLast *domain.CareerRankRow) bool {
	if custom != nil {
		merged.SpartanID = custom.SpartanID
		merged.BannerImageURL = custom.BannerImageURL
		merged.EmblemImageURL = custom.EmblemImageURL
		merged.BackdropImageURL = custom.BackdropImageURL
	}
	if dbLast == nil {
		return false
	}
	carryForward := false
	if merged.SpartanID == "" && dbLast.SpartanID != "" {
		merged.SpartanID = dbLast.SpartanID
		carryForward = true
	}
	if merged.BannerImageURL == "" && dbLast.BannerImageURL != "" {
		merged.BannerImageURL = dbLast.BannerImageURL
		carryForward = true
	}
	if merged.EmblemImageURL == "" && dbLast.EmblemImageURL != "" {
		merged.EmblemImageURL = dbLast.EmblemImageURL
		carryForward = true
	}
	if merged.BackdropImageURL == "" && dbLast.BackdropImageURL != "" {
		merged.BackdropImageURL = dbLast.BackdropImageURL
		carryForward = true
	}
	return carryForward
}

// carryForwardDerivedFields hydrate les champs dérivés (rank_name, rank_tier,
// xp_for_next_rank, xp_total, adornment_path) depuis dbLast si encore vides.
// EnrichFromMetadata les recalculera depuis merged.Rank ; ce carry-forward
// est utile uniquement si metadata est indisponible. Retourne false car ces
// champs sont dérivés (pas du "vrai" carry-forward métier).
func carryForwardDerivedFields(merged *domain.CareerRankRow, dbLast *domain.CareerRankRow) bool {
	if dbLast == nil {
		return false
	}
	if merged.RankName == "" {
		merged.RankName = dbLast.RankName
	}
	if merged.RankTier == "" {
		merged.RankTier = dbLast.RankTier
	}
	if merged.XPForNextRank == 0 {
		merged.XPForNextRank = dbLast.XPForNextRank
	}
	if merged.XPTotal == 0 {
		merged.XPTotal = dbLast.XPTotal
	}
	if merged.AdornmentPath == "" {
		merged.AdornmentPath = dbLast.AdornmentPath
	}
	return false
}
