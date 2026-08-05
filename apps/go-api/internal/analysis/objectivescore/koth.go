package objectivescore

// koth.go — décodeur de la timeline de score KOTH (King of the Hill).
//
// Deux variantes byte-aligned ancrées sur le BYTE du token (ab = tokenBit/8) :
//
//	Variante B (meters / Ranked, finals bas ≤ ~5) — PROUVÉE EXACTE (0a247154
//	  Ranked 4-2 → décodé 4-2 monotone) :
//	    team0 = t2[ab+12] / 5 ; team1 = t2[ab+16] / 5
//	Variante A (points, finals hauts ~100) — APPROX (team0 non strictement
//	  monotone, off-by-one possible sur le final) :
//	    total = t2[ab+14] ; team1 = t2[ab+13] >> 4 ; team0 = total - team1
//
// Le choix de variante est passé par l'appelant (qui connaît le contexte
// Ranked / l'ordre de grandeur du final). DecodeScoreTimeline auto-sélectionne
// par défaut (cf. dispatch.go).

const (
	sourceKOTHVariantB = "film_koth_t2_b" // meters/5
	sourceKOTHVariantA = "film_koth_t2_a" // points
	// VariantB / VariantA : sélecteurs de variante KOTH passés à DecodeKOTH.
	VariantB = "B"
	VariantA = "A"
)

// DecodeKOTH décode la timeline de score KOTH selon la variante demandée
// ("B" meters/5 par défaut, "A" points). Un ScoreFrame par chunk type-2 ancré.
func DecodeKOTH(chunks []ChunkInput, variant string) []ScoreFrame {
	if variant == VariantA {
		return decodeKOTHVariantA(chunks)
	}
	return decodeKOTHVariantB(chunks)
}

// decodeKOTHVariantB : team0=byte(ab+12)/5, team1=byte(ab+16)/5.
func decodeKOTHVariantB(chunks []ChunkInput) []ScoreFrame {
	var frames []ScoreFrame
	for _, c := range chunks {
		p, tb := anchoredPayload(c)
		if p == nil {
			continue
		}
		ab := tb / 8
		frames = append(frames, ScoreFrame{
			TimeMS:     c.StartMS,
			Team0:      byteAt(p, ab+kothBOffT0) / kothBDivisor,
			Team1:      byteAt(p, ab+kothBOffT1) / kothBDivisor,
			Source:     sourceKOTHVariantB,
			Confidence: confidenceApprox,
		})
	}
	return frames
}

// decodeKOTHVariantA : total=byte(ab+14), team1=byte(ab+13)>>4, team0=total-team1.
// team0 est borné à >= 0 (le total-team1 peut sous-déborder sur du bruit de decode).
func decodeKOTHVariantA(chunks []ChunkInput) []ScoreFrame {
	var frames []ScoreFrame
	for _, c := range chunks {
		p, tb := anchoredPayload(c)
		if p == nil {
			continue
		}
		ab := tb / 8
		total := byteAt(p, ab+kothAOffTot)
		team1 := byteAt(p, ab+kothAOffT1) >> 4
		team0 := total - team1
		if team0 < 0 {
			team0 = 0
		}
		frames = append(frames, ScoreFrame{
			TimeMS:     c.StartMS,
			Team0:      team0,
			Team1:      team1,
			Source:     sourceKOTHVariantA,
			Confidence: confidenceApprox,
		})
	}
	return frames
}

// byteAt lit l'octet à l'index i d'un payload (0 hors-bornes).
func byteAt(p []byte, i int) int {
	if i < 0 || i >= len(p) {
		return 0
	}
	return int(p[i])
}
