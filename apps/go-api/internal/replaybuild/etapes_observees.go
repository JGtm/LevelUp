package replaybuild

import "slices"

// buildBytesStepsBefore et buildBytesStepsAfter sont les etapes que BuildBytes rend a
// l'observateur AVANT et APRES le decodage des positions (`replay.BuildFromFilmSteps`), dans
// l'ordre. Lues par le harnais d'equivalence (via les accesseurs), gardees par observe_test.go.
var (
	buildBytesStepsBefore = []string{
		"score", "objectives", "vip", "skull", "bomb", "flag", "zones", "zoneRoles",
		"killsource", "spawnPoints", "spawnPointsState", "neutralDeaths", "killRefs",
	}
	// `EtapeRejeuDepuisLesFaits` PRECEDE `artifact` : le harnais doit savoir QUELLE BRANCHE a
	// servi avant de comparer les octets qu elle a produits (lot 4.1.2).
	buildBytesStepsAfter = []string{EtapeRejeuDepuisLesFaits, "artifact"}
)

// BuildBytesStepsBefore rend une COPIE de la liste des etapes qui precedent le decodage des
// positions (J12.4 : plus de variable de paquet exportee modifiable).
func BuildBytesStepsBefore() []string { return slices.Clone(buildBytesStepsBefore) }

// BuildBytesStepsAfter rend une COPIE de la liste des etapes qui suivent le decodage des positions.
func BuildBytesStepsAfter() []string { return slices.Clone(buildBytesStepsAfter) }
