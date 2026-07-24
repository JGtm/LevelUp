// Package halo_5 — career_sr.go : rang XP « Spartan Rank » (SR) Halo 5.
//
// Halo 5 a DEUX axes de progression : le CSR (compétence classée, palier
// Bronze..Onyx — cf. mapping_servicerecord) et le SR (rang d'expérience de compte,
// niveaux 1..152). Le SR n'est PAS dans le service record ni la liste de matchs :
// il vit dans la carnage (PlayerStats[].XpInfo). Ce fichier porte le RÉFÉRENTIEL
// statique des seuils XP (152 niveaux) + la projection vers CareerSnapshot.
//
// Données STATIQUES (Halo 5 est un jeu figé) : seuils issus du SpartanRankManifest
// (content-hacs.svc/contents/SpartanRankManifest), dump den.dev VÉRIFIÉ contre les
// jalons publiés (cf. .ai/refs/h5_spartan_rank_xp.csv). SR152 = MAXIMUM (pas un
// seuil « 0 », pas de rang suivant).
package halo_5

import (
	"fmt"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/mappings"
)

// h5MaxSpartanRank — niveau SR maximum Halo 5 (sommet de la progression XP).
const h5MaxSpartanRank = 152

// h5SRStartXP[n-1] = XP de compte CUMULÉ au DÉBUT du rang SR n. Index 0 = SR1
// (start 0), index 151 = SR152 (start 50 000 000 = MAX). Le « XP pour compléter le
// rang n » se dérive : h5SRStartXP[n] - h5SRStartXP[n-1] (indéfini au max).
var h5SRStartXP = [h5MaxSpartanRank]int{
	0, 300, 3600, 6600, 10700, 13700, 17500, 22500, 28500, 37000,
	41000, 47000, 54500, 63500, 74500, 87000, 101500, 118000, 137000, 160000,
	167000, 176000, 187500, 201000, 217000, 236000, 258000, 282500, 310000, 340000,
	349500, 361500, 376500, 394000, 414500, 438000, 464000, 493000, 525500, 562000,
	574000, 589000, 607500, 629000, 654000, 682000, 713500, 748500, 786500, 828000,
	873000, 922000, 975500, 1035000, 1100000, 1115000, 1135000, 1155000, 1180000, 1210000,
	1245000, 1280000, 1320000, 1365000, 1415000, 1465000, 1520000, 1580000, 1645000, 1720000,
	1735000, 1755000, 1780000, 1810000, 1845000, 1885000, 1930000, 1975000, 2025000, 2080000,
	2140000, 2205000, 2275000, 2355000, 2440000, 2465000, 2490000, 2520000, 2555000, 2595000,
	2640000, 2690000, 2745000, 2805000, 2870000, 2940000, 3015000, 3095000, 3180000, 3270000,
	3300000, 3335000, 3375000, 3420000, 3470000, 3530000, 3595000, 3665000, 3740000, 3820000,
	3905000, 3995000, 4090000, 4200000, 4320000, 4355000, 4395000, 4440000, 4495000, 4555000,
	4620000, 4690000, 4765000, 4845000, 4935000, 5025000, 5120000, 5220000, 5330000, 5475000,
	5520000, 5575000, 5640000, 5710000, 5790000, 5880000, 5980000, 6085000, 6200000, 6325000,
	6460000, 6615000, 6800000, 7050000, 7750000, 9000000, 11050000, 14000000, 18000000, 24000000,
	35000000, 50000000,
}

// h5SRAssetRef construit la référence d'asset d'un niveau SR. Pas d'image (le SR
// s'affiche en chiffre — aucun insigne par niveau dans le manifest, by design).
//
// ID = « SR N » (un LIBELLÉ, pas le numéro brut) : c'est cette valeur que le
// service recopie dans CareerRankData.RankLabel (cf. rankDataFromCanonical), donc
// la page Carrière affiche « SR 111 » et JAMAIS un libellé de career rank HINF
// (le catalogue HINF est neutralisé title-side, mais le label SR reste la source).
func h5SRAssetRef(sr int) *canonical.AssetReference {
	return &canonical.AssetReference{
		Kind:         "spartan_rank",
		ID:           fmt.Sprintf("SR %d", sr),
		DefaultLabel: fmt.Sprintf("SR %d", sr),
		Labels:       map[string]string{"en": fmt.Sprintf("SR%d", sr), "fr": fmt.Sprintf("SR %d", sr)},
	}
}

// SpartanRankLabel retourne le LIBELLÉ d'affichage du rang SR Halo 5 ("SR N"),
// title-aware. Source unique partagée par l'asset ref canonique (h5SRAssetRef) ET
// la persistance career_progression (rank_name, cf. livesync.writeCareerSR) — pour
// que la Home (qui lit career_progression.rank_name) affiche « SR 111 » au lieu du
// fallback générique HINF « Rang 111 » (career.rank_catalog = not_exposed pour h5,
// aucun catalogue ne résout le label côté Home).
func SpartanRankLabel(sr int) string {
	return fmt.Sprintf("SR %d", sr)
}

// BuildSpartanRankCatalog construit un mappings.RankCatalog title-aware Halo 5 à
// partir du référentiel SR statique (h5SRStartXP), pour que la Home résolve le
// libellé « SR N » au lieu du fallback générique HINF « Rang N ».
//
// Halo 5 n'expose PAS de career_rank_translations en metadata (career.rank_catalog
// = not_exposed), donc le SemanticAdapter h5 recevait un catalog vide et
// buildHomeCareerRank tombait sur « Rang N ». Ce builder donne au catalog les 152
// niveaux SR avec :
//   - ID = n (1..152)
//   - Title["en"]/Title["fr"] = "SR n" (réutilise SpartanRankLabel — source unique)
//   - XPRequired = XP pour COMPLÉTER le rang n = h5SRStartXP[n] - h5SRStartXP[n-1]
//     (0 au max SR152 : pas de rang suivant)
//
// Le catalog résout le label pour TOUTE la donnée (existante avec rank_name NULL en
// career_progression ET future) — même mécanisme que Halo Infinite (LoadRankCatalog),
// mais sans aucune écriture DB : le référentiel est calculé en mémoire au boot.
//
// XPRequired ne sert QUE de fallback quand la source (career_progression) ne porte
// pas xp_for_next (cf. buildHomeCareerRank) : un xp_for_next déjà peuplé en DB
// (ex. 2 950 000) n'est jamais écrasé. Next(152) absent → is_max dérivé correctement.
func BuildSpartanRankCatalog() *mappings.RankCatalog {
	entries := make([]mappings.RankEntry, 0, h5MaxSpartanRank)
	for n := 1; n <= h5MaxSpartanRank; n++ {
		label := SpartanRankLabel(n)
		var xpRequired int
		if n < h5MaxSpartanRank {
			if need := h5SRStartXP[n] - h5SRStartXP[n-1]; need > 0 {
				xpRequired = need
			}
		}
		entries = append(entries, mappings.RankEntry{
			ID:         n,
			Title:      map[string]string{mappings.LocaleEN: label, mappings.LocaleFR: label},
			Subtitle:   map[string]string{},
			Tier:       map[string]string{},
			XPRequired: xpRequired,
		})
	}
	return mappings.NewRankCatalog(TitleSlug, entries)
}

// applyDefaultSpartanRankBounds fixe les bornes de progression « Héros » Halo 5
// (RankMax = SR152, XPMax = XP cumulé au SR152) sur TOUT CareerSnapshot h5 qui a
// des stats arena, INDÉPENDAMMENT de l'enrichissement live du SR réel. C'est le
// filet déterministe d'AXE C : si enrichSpartanRank() échoue (pas de match récent,
// carnage indisponible), RankMax reste posé à 152 — le front affiche « X/152 » et
// jamais le fallback HINF « X/272 ». N'écrase JAMAIS un SR déjà enrichi : pose les
// bornes uniquement si elles sont absentes, et ne touche ni RankNumber ni le label.
func applyDefaultSpartanRankBounds(snap *canonical.CareerSnapshot) {
	if snap == nil {
		return
	}
	if snap.RankMax == nil {
		rankMax := h5MaxSpartanRank
		snap.RankMax = &rankMax
	}
	if snap.XPMax == nil {
		xpMax := h5SRStartXP[h5MaxSpartanRank-1]
		snap.XPMax = &xpMax
	}
	if snap.MaxRank == nil {
		snap.MaxRank = h5SRAssetRef(h5MaxSpartanRank)
	}
}

// SpartanRankProgression dérive (current_xp, xp_for_next, xp_total, is_max) d'un
// couple (SpartanRank, TotalXP) Halo 5 — fonction PURE partagée par l'enrichissement
// de snapshot (applySpartanRank) ET la persistance career_progression au sync (C3).
// SR hors [1..152] → ok=false (à ignorer). Source unique de la dérivation SR→XP.
func SpartanRankProgression(spartanRank, totalXP int) (currentXP, xpForNext, xpTotal int, isMax, ok bool) {
	if spartanRank < 1 || spartanRank > h5MaxSpartanRank {
		return 0, 0, 0, false, false
	}
	xpTotal = totalXP
	if cur := totalXP - h5SRStartXP[spartanRank-1]; cur >= 0 {
		currentXP = cur
	}
	if spartanRank >= h5MaxSpartanRank {
		return currentXP, 0, xpTotal, true, true
	}
	if need := h5SRStartXP[spartanRank] - h5SRStartXP[spartanRank-1]; need > 0 {
		xpForNext = need
	}
	return currentXP, xpForNext, xpTotal, false, true
}

// applySpartanRank enrichit un CareerSnapshot avec le rang XP (SR) Halo 5 à partir
// du SpartanRank + TotalXP du joueur (lus dans la carnage XpInfo). SR152 = MAX :
// IsMaxRank, aucun rang suivant, aucun « XP avant le suivant ». Borne défensive : un
// SR hors [1..152] est ignoré (snapshot inchangé).
//
// DEUX AXES distincts : le SR (rang XP de compte) va dans CurrentRank/RankNumber/
// CurrentXP/XPTotal/XPForNextRank/NextRank/IsMaxRank ; le CSR (compétence classée,
// posé par mapCareerSnapshot dans RankTier/RankName/HighestCSR) n'est PAS touché —
// les deux rangs coexistent (le front affiche le SR comme rang de progression XP et
// le CSR comme rang compétitif).
func applySpartanRank(snap *canonical.CareerSnapshot, spartanRank, totalXP int) {
	if snap == nil || spartanRank < 1 || spartanRank > h5MaxSpartanRank {
		return
	}
	snap.RankNumber = spartanRank
	snap.CurrentRank = h5SRAssetRef(spartanRank)
	// Bornes de progression « Héros » Halo 5 (title-agnostic : le service les lit
	// au lieu des constantes HINF). RankMax = SR152 ; XPMax = XP cumulé au SR152 ;
	// MaxRank = libellé du rang sommet (« SR 152 ») pour le titre du gauge « path
	// to max rank » — le service le recopie dans HeroProgress.MaxRankName*.
	rankMax := h5MaxSpartanRank
	xpMax := h5SRStartXP[h5MaxSpartanRank-1]
	snap.RankMax = &rankMax
	snap.XPMax = &xpMax
	snap.MaxRank = h5SRAssetRef(h5MaxSpartanRank)
	if totalXP > 0 {
		xt := totalXP
		snap.XPTotal = &xt
	}
	// XP accumulé DANS le rang courant (au-delà de son seuil de départ).
	if cur := totalXP - h5SRStartXP[spartanRank-1]; cur >= 0 {
		snap.CurrentXP = &cur
	}

	if spartanRank >= h5MaxSpartanRank {
		snap.IsMaxRank = true
		return // SR152 = sommet : pas de rang suivant ni de progression résiduelle.
	}
	snap.NextRank = h5SRAssetRef(spartanRank + 1)
	if need := h5SRStartXP[spartanRank] - h5SRStartXP[spartanRank-1]; need > 0 {
		snap.XPForNextRank = &need
	}
}
