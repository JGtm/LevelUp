// Package service — career_live_partial.go : conversion live API
// (progress + custom) en CareerProgressionPartial pour l'INSERT field-aware.
//
// La règle d'or : on ne signale comme "frais" QUE les champs effectivement
// rendus non-vides par l'API. Le carry-forward depuis la DB ne participe
// pas à la persistance (il sert uniquement à la lecture/affichage front).
//
// Distinction critique pour rank/XP :
//   - progress == nil → tous les champs progress sont nil (API muette, ne pas écrire)
//   - progress.CurrentRank == 0 ET CurrentXP == 0 ET pas IsMaxRank → API a probablement
//     rendu vide, on n'écrit RIEN (pas même un faux 0)
//   - progress.CurrentRank > 0 → c'est un vrai rang, on écrit Rank + CurrentXP même si
//     CurrentXP = 0 (début de palier valide)
//   - progress.IsMaxRank == true → on écrit IsMaxRank=true (rank peut être max-rank)
//
// Pour custom : chaque URL est indépendante.
//   - custom == nil → tous champs nil
//   - custom.BannerImageURL == "" → BannerImageURL nil (carry-forward DB en lecture)
//   - custom.BannerImageURL != "" → BannerImageURL set
package service

import (
	"strings"

	"levelup/go-api/internal/domain"
)

// FetchStatus tags the outcome of the last live fetch attempt.
type FetchStatus string

const (
	// FetchStatusOK : data exploitable rendue par l'API.
	FetchStatusOK FetchStatus = "ok"
	// FetchStatusAPIEmpty : API a répondu sans erreur mais sans data (silencieux).
	FetchStatusAPIEmpty FetchStatus = "api_empty"
	// FetchStatusForbidden : 403 (privacy joueur ou token sans perm).
	FetchStatusForbidden FetchStatus = "forbidden_403"
	// FetchStatusAuthMissing : aucun token Spartan disponible.
	FetchStatusAuthMissing FetchStatus = "auth_missing"
	// FetchStatusFailed : erreur transport/parse autre.
	FetchStatusFailed FetchStatus = "failed"
)

// PartialFromLive convertit les retours live API en CareerProgressionPartial.
// progress et custom peuvent être nil indépendamment. Retourne un Partial
// dont IsEmpty() peut être true si l'API n'a rien rendu d'exploitable.
func PartialFromLive(
	progress *domain.CareerRankSnapshot,
	custom *domain.SpartanCustomizationData,
) *domain.CareerProgressionPartial {
	p := &domain.CareerProgressionPartial{}

	if progress != nil && progressHasRealData(progress) {
		// On set Rank/XP même si CurrentXP=0 (vraie valeur début palier),
		// dès lors qu'au moins UN signal positif confirme la réponse API.
		if progress.CurrentRank > 0 {
			r := progress.CurrentRank
			p.Rank = &r
		}
		xp := progress.CurrentXP
		p.CurrentXP = &xp
		if progress.XPForNextRank > 0 {
			x := progress.XPForNextRank
			p.XPForNextRank = &x
		}
		if progress.XPTotal > 0 {
			t := progress.XPTotal
			p.XPTotal = &t
		}
		if progress.IsMaxRank {
			b := true
			p.IsMaxRank = &b
		}
		if s := strings.TrimSpace(progress.CurrentRankName); s != "" {
			p.RankName = &s
		}
		if s := strings.TrimSpace(progress.CurrentRankTier); s != "" {
			p.RankTier = &s
		}
		if s := strings.TrimSpace(progress.AdornmentPath); s != "" {
			p.AdornmentPath = &s
		}
		// SpartanID est aussi dans progress (legacy) ; custom prend le pas plus bas.
		if s := strings.TrimSpace(progress.SpartanID); s != "" && p.SpartanID == nil {
			p.SpartanID = &s
		}
	}

	if custom != nil {
		if s := strings.TrimSpace(custom.SpartanID); s != "" {
			p.SpartanID = &s
		}
		if s := strings.TrimSpace(custom.BannerImageURL); s != "" {
			p.BannerImageURL = &s
		}
		if s := strings.TrimSpace(custom.EmblemImageURL); s != "" {
			p.EmblemImageURL = &s
		}
		if s := strings.TrimSpace(custom.BackdropImageURL); s != "" {
			p.BackdropImageURL = &s
		}
	}

	return p
}

// progressHasRealData retourne true si l'objet progress contient au moins UN
// signal positif (rank>0, XP réel, ou IsMaxRank). Sinon c'est probablement
// une réponse API muette qu'on ne veut pas convertir en INSERT bidon.
func progressHasRealData(p *domain.CareerRankSnapshot) bool {
	if p == nil {
		return false
	}
	if p.CurrentRank > 0 {
		return true
	}
	if p.IsMaxRank {
		return true
	}
	// CurrentXP > 0 sans Rank ne suffit pas (incohérent) — préfère ignorer.
	return false
}
