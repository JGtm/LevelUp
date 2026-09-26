package main

// extras.go — deux lectures secondaires de l'artefact : les occupations de vehicule (pour
// l'axe 5, ou un porteur embarque n'a plus de position de bipede) et le nombre de MANCHES
// deduit du fil de score (independant de `coverage.score.rounds`).

// vehicule et ride : la forme minimale dont l'axe 5 a besoin.
type vehiculeArt struct {
	Rides []rideArt `json:"rides"`
}

type rideArt struct {
	T0   int    `json:"t0"`
	T1   int    `json:"t1"`
	XUID string `json:"xuid"`
}

// scoreTimeline : seules les manches sont lues.
type scoreTimelineArt struct {
	Teams []struct {
		Rounds []struct {
			Round int `json:"round"`
		} `json:"rounds"`
	} `json:"teams"`
}

// embarque dit si le xuid occupe un vehicule a cette image — c'est le repli que
// `carrierPosition.ts` (apps/web/src/features/match-replay/model/carrierPosition.ts) applique
// avant de rendre `null`.
func embarque(a *artefact, xuid string, frame int) bool {
	for _, v := range a.Vehicles {
		for _, r := range v.Rides {
			if r.XUID == xuid && frame >= r.T0 && frame <= r.T1 {
				return true
			}
		}
	}
	return false
}

// manchesDuFilScore rend le nombre de manches distinctes vues dans le fil de score de l'artefact.
func manchesDuFilScore(a *artefact) int {
	vues := map[int]bool{}
	for _, t := range a.ScoreTimeline.Teams {
		for _, r := range t.Rounds {
			vues[r.Round] = true
		}
	}
	return len(vues)
}
