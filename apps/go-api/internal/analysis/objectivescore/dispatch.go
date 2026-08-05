package objectivescore

import "strings"

// dispatch.go — sélection mode + helpers transverses (calibration, classif mode).

// confidenceApprox = seule confiance émise par ce package : la granularité du
// snapshot type-2 est ~20s et les valeurs Strongholds sont calibrées sur le
// final DB. Aucune prétention ms-exact.
const confidenceApprox = "approx"

// DecodeScoreTimeline est le point d'entrée title-agnostique : il dispatch selon
// le nom de game-variant. Pour KOTH, il auto-sélectionne la variante via l'ordre
// de grandeur du final (≤ kothPointsThreshold → variante B meters/5, sinon A
// points). finalT0/finalT1 servent à calibrer Strongholds ET à choisir la
// variante KOTH. Retourne nil pour les modes non supportés (CTF/Slayer/Oddball).
func DecodeScoreTimeline(gameVariantName string, chunks []ChunkInput, finalT0, finalT1 int) []ScoreFrame {
	switch ClassifyMode(gameVariantName) {
	case ModeStrongholds:
		return DecodeStrongholds(chunks, finalT0, finalT1)
	case ModeKOTH:
		return DecodeKOTH(chunks, pickKOTHVariant(finalT0, finalT1))
	default:
		return nil
	}
}

// Mode = classe de mode à objectif gérée par ce décodeur.
type Mode int

const (
	ModeUnsupported Mode = iota
	ModeStrongholds
	ModeKOTH
)

// kothPointsThreshold : au-delà de ce final, on suppose une partie KOTH "points"
// (variante A) ; en-dessous, une partie "meters/rounds" Ranked (variante B, la
// seule prouvée exacte). Heuristique dérivée des films de validation
// (Ranked 4-2 → B ; 105-8 → A).
const kothPointsThreshold = 20

// pickKOTHVariant choisit la variante KOTH selon l'ordre de grandeur du final.
func pickKOTHVariant(finalT0, finalT1 int) string {
	if finalT0 > kothPointsThreshold || finalT1 > kothPointsThreshold {
		return VariantA
	}
	return VariantB
}

// ClassifyMode mappe un game_variant_name (ex. "Arena:Strongholds",
// "KOTH:Arena", "Ranked:King of the Hill") vers une Mode gérée. Insensible à la
// casse, robuste aux préfixes/suffixes de catégorie.
func ClassifyMode(gameVariantName string) Mode {
	g := strings.ToLower(gameVariantName)
	switch {
	case strings.Contains(g, "stronghold"):
		return ModeStrongholds
	case strings.Contains(g, "koth"), strings.Contains(g, "king of the hill"):
		return ModeKOTH
	default:
		return ModeUnsupported
	}
}

// calibrateByFinal applique le scaling PAR MATCH : chaque valeur brute vg est
// remise à l'échelle pour que la dernière frame tombe exactement sur final.
//
//	out[i] = round(raw[i] * final / lastRaw)   si lastRaw > 0
//	out[i] = 0                                  si lastRaw == 0 (séquence plate)
//
// Le shape monotone de raw est préservé ; seule l'échelle est ajustée.
func calibrateByFinal(raw []int, final int) []int {
	out := make([]int, len(raw))
	last := 0
	if len(raw) > 0 {
		last = raw[len(raw)-1]
	}
	if last <= 0 {
		return out // séquence plate / vide → rien à calibrer
	}
	for i, v := range raw {
		// round(v*final/last) en arithmétique entière (v, final >= 0).
		out[i] = (v*final + last/2) / last
	}
	return out
}
