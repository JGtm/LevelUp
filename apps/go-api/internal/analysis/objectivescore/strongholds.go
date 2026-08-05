package objectivescore

// strongholds.go — décodeur de la timeline de score Strongholds.
//
// team0 = varAtBit(token+24 bits) : séquence monotone PROUVÉE (cmd/tmp_scoreval).
// team1 = varAtBit(token+23 bits) : candidat monotone mais shape-fragile (offset
// adjacent → bits chevauchants). On l'expose quand même car aucune meilleure
// source n'a été trouvée ; le SHAPE est la vérité, calibré au final DB.
//
// Calibration PAR MATCH (le scaling n'est PAS un facteur fixe) :
//
//	affiché = round(vg * finalT / lastVg)   si lastVg > 0
//	affiché = 0                              si lastVg == 0 (séquence plate)
//
// → la dernière frame retombe EXACTEMENT sur finalT (continuité avec la DB), les
// frames intermédiaires gardent la forme monotone réelle.

const sourceStrongholds = "film_strongholds_t2"

// DecodeStrongholds décode la timeline de score Strongholds. finalT0/finalT1 =
// scores finaux DB (match_registry) pour calibrer le scaling par équipe.
// Retourne un ScoreFrame par chunk type-2 ancré, dans l'ordre des chunks.
func DecodeStrongholds(chunks []ChunkInput, finalT0, finalT1 int) []ScoreFrame {
	times, raw0, raw1 := collectStrongholdsRaw(chunks)
	if len(times) == 0 {
		return nil
	}
	scaled0 := calibrateByFinal(raw0, finalT0)
	scaled1 := calibrateByFinal(raw1, finalT1)

	frames := make([]ScoreFrame, len(times))
	for i := range times {
		frames[i] = ScoreFrame{
			TimeMS:     times[i],
			Team0:      scaled0[i],
			Team1:      scaled1[i],
			Source:     sourceStrongholds,
			Confidence: confidenceApprox,
		}
	}
	return frames
}

// collectStrongholdsRaw lit, pour chaque chunk type-2 ancré, le start_ms + les
// varints bruts team0 (+24) et team1 (+23).
func collectStrongholdsRaw(chunks []ChunkInput) (times, raw0, raw1 []int) {
	for _, c := range chunks {
		p, tb := anchoredPayload(c)
		if p == nil {
			continue
		}
		times = append(times, c.StartMS)
		raw0 = append(raw0, varAtBit(p, tb+shOffTeam0))
		raw1 = append(raw1, varAtBit(p, tb+shOffTeam1))
	}
	return times, raw0, raw1
}
