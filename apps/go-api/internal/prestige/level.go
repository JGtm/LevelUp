package prestige

// level.go — calcul du niveau Prestige d'un joueur depuis son total PP.
//
// Référence : Axe 6 du plan conceptuel.
// Les seuils et noms sont configurables via tuning.Levels.

// LevelFromPP retourne le niveau Prestige correspondant au total PP donné.
//
// Le niveau est l'index du dernier seuil franchi.
// ProgressRatio est la fraction parcourue vers le seuil suivant ([0, 1]).
// NextThresholdPP = -1 si on est au niveau max.
//
// Précondition : tuning.Levels.Thresholds est trié, monotone, commence par 0.
// (Validé dans tuning.Validate())
func LevelFromPP(t Tuning, totalPP int) Level {
	thresholds := t.Levels.Thresholds
	names := t.Levels.Names
	if len(thresholds) == 0 || len(names) == 0 {
		return Level{}
	}

	// Trouver l'index du dernier seuil franchi.
	idx := 0
	for i, th := range thresholds {
		if totalPP >= th {
			idx = i
		}
	}

	current := thresholds[idx]
	var next int
	var progress float64
	if idx < len(thresholds)-1 {
		next = thresholds[idx+1]
		span := float64(next - current)
		if span > 0 {
			progress = float64(totalPP-current) / span
		}
	} else {
		next = -1
		progress = 1.0
	}

	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	name := ""
	if idx < len(names) {
		name = names[idx]
	}

	return Level{
		Index:           idx,
		Name:            name,
		ThresholdPP:     current,
		NextThresholdPP: next,
		ProgressRatio:   progress,
	}
}
