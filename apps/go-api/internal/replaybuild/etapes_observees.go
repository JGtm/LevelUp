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
	// `EtapeRejeuDepuisLesFaits` OUVRE la liste : le harnais doit savoir QUELLE BRANCHE a servi avant de
	// comparer ce qu elle a produit (lot 4.1.2). `scoreTimeline` (les joueurs du tableau des scores et leurs
	// compteurs) et `eliminations` (le calque des eliminations) se lisent sur le DOCUMENT assemble : leurs
	// lignes sortent de l assemblage, pas d un balayage, et ne figurent dans aucune etape de la marche.
	buildBytesStepsAfter = []string{EtapeRejeuDepuisLesFaits, "scoreTimeline", "eliminations", "artifact"}
)

// BuildBytesStepsBefore rend une COPIE de la liste des etapes qui precedent le decodage des
// positions (J12.4 : plus de variable de paquet exportee modifiable).
func BuildBytesStepsBefore() []string { return slices.Clone(buildBytesStepsBefore) }

// BuildBytesStepsAfter rend une COPIE de la liste des etapes qui suivent le decodage des positions.
func BuildBytesStepsAfter() []string { return slices.Clone(buildBytesStepsAfter) }
