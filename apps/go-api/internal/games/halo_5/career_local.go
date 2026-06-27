// Package halo_5 — career_local.go : projection du career LOCAL (DuckDB synchronisé)
// vers CareerSnapshot, pour servir le rang Halo 5 hors-ligne (démo, aucun token,
// l'API cryptum live échouerait). Parité de sémantique avec la voie live
// (mapping_servicerecord.mapCareerSnapshot) : mêmes paliers CSR FR + bornes SR.
package halo_5

import (
	"context"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// CareerLocalSource lit le career Halo 5 d'un joueur depuis le substrat LOCAL
// (DuckDB synchronisé) : meilleur CSR à vie + Spartan Rank. Implémentée
// STRUCTURELLEMENT par platform/duckdb.Halo5CareerSource (retour domain.H5CareerLocal,
// aucun import croisé → pas de cycle ; même pattern que MatchHistorySource).
type CareerLocalSource interface {
	GetLatestCareer(ctx context.Context) (*domain.H5CareerLocal, error)
}

// localCareerSnapshot projette le career local en CareerSnapshot canonique, avec la
// MÊME sémantique que la voie live (mapCareerSnapshot) : palier CSR FR (RankTier),
// « Palier N » (RankName, sauf Onyx), valeur brute seulement à Onyx ; bornes SR 152
// (jamais le fallback HINF 272) + SR réel. Retourne nil si la lecture DB échoue (le
// caller dégrade alors vers la voie live).
func (a *DataAdapter) localCareerSnapshot(ctx context.Context, gamertag string) *canonical.CareerSnapshot {
	data, err := a.careerLocal.GetLatestCareer(ctx)
	if err != nil {
		a.logger.DebugContext(ctx, "h5 career local: lecture échouée (dégradation)", "player", gamertag, "err", err)
		return nil
	}
	snap := &canonical.CareerSnapshot{Player: h5Identity(gamertag)}
	applyDefaultSpartanRankBounds(snap)
	pt := a.placementTotal
	if pt <= 0 {
		pt = h5DefaultPlacementMatches
	}
	snap.PlacementTotal = &pt
	if data != nil {
		if data.HasCSR {
			applyLocalCSR(snap, data)
		}
		if data.SpartanRank > 0 {
			applySpartanRank(snap, data.SpartanRank, data.TotalXP)
		}
	}
	return snap
}

// applyLocalCSR pose RankTier/RankName/HighestCSR depuis le meilleur CSR local, en
// réutilisant le référentiel de paliers h5Designations (parité mapCareerSnapshot).
func applyLocalCSR(snap *canonical.CareerSnapshot, d *domain.H5CareerLocal) {
	en, fr := designationFromTierEN(d.CSRTier)
	if fr == "" {
		return
	}
	tier := fr
	snap.RankTier = &tier
	name := fr // « Diamant 5 » (palier + sous-palier) ; Onyx sans sous-palier.
	if en != "onyx" && d.CSRSubTier > 0 {
		name = fr + " " + strconv.Itoa(d.CSRSubTier)
	}
	snap.RankName = &name
	// Valeur CSR brute significative QU'À Onyx (cf. invariant mapCareerSnapshot).
	if en == "onyx" && d.CSRValue > 0 {
		v := d.CSRValue
		snap.HighestCSR = &v
	}
}

// designationFromTierEN mappe un palier EN stocké (Bronze..Onyx, casse libre) vers
// (en normalisé, fr). ("","") si inconnu. Source unique : h5Designations.
func designationFromTierEN(tier string) (en, fr string) {
	t := strings.ToLower(strings.TrimSpace(tier))
	for i := range h5Designations {
		if h5Designations[i].en == t {
			return h5Designations[i].en, h5Designations[i].fr
		}
	}
	return "", ""
}
