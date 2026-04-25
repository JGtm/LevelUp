// Package metadata fournit les garde-fous d'import de métadonnées Waypoint.
//
// Sprint 54-D1.3 : fonctions de validation avant promotion de données brutes
// (médailles, assets) vers le schéma définitif.
package metadata

import (
	"fmt"
	"math"
	"strings"
)

// MedalEntry représente une entrée médaille candidate à l'import.
type MedalEntry struct {
	TitleID     string
	MedalID     int64
	Label       string
	Description string
	Category    string
	Rarity      string
	ImageURL    string
	SpriteIdx   int
	RawJSON     string
}

// GuardResult contient le verdict d'une vérification de garde-fous.
type GuardResult struct {
	Passed  bool
	Reason  string
	Details []string
}

// CheckCardinalityGuard vérifie que le nombre de médailles Waypoint est cohérent
// avec le nombre local existant (tolérance ± pct).
// localCount=0 est accepté (premier import).
func CheckCardinalityGuard(waypointCount, localCount int, tolerancePct float64) GuardResult {
	if localCount == 0 {
		return GuardResult{Passed: true, Reason: "premier import (local vide)"}
	}
	if waypointCount == 0 {
		return GuardResult{
			Passed: false,
			Reason: "Waypoint retourne 0 médailles — probable erreur API",
		}
	}

	ratio := math.Abs(float64(waypointCount-localCount)) / float64(localCount)
	threshold := tolerancePct / 100.0

	if ratio > threshold {
		return GuardResult{
			Passed: false,
			Reason: fmt.Sprintf(
				"cardinalité hors tolérance : Waypoint=%d, local=%d, écart=%.1f%% (seuil=%.0f%%)",
				waypointCount, localCount, ratio*100, tolerancePct,
			),
		}
	}
	return GuardResult{
		Passed: true,
		Reason: fmt.Sprintf("cardinalité OK : Waypoint=%d, local=%d, écart=%.1f%%", waypointCount, localCount, ratio*100),
	}
}

// CheckRequiredFieldsGuard vérifie que chaque entrée a les champs requis non vides.
func CheckRequiredFieldsGuard(entries []MedalEntry) GuardResult {
	var missing []string
	for i, e := range entries {
		if e.MedalID == 0 {
			missing = append(missing, fmt.Sprintf("[%d] medal_id=0", i))
		}
		if strings.TrimSpace(e.Label) == "" {
			missing = append(missing, fmt.Sprintf("[%d] label vide (medal_id=%d)", i, e.MedalID))
		}
		if strings.TrimSpace(e.Category) == "" {
			missing = append(missing, fmt.Sprintf("[%d] category vide (medal_id=%d)", i, e.MedalID))
		}
		if strings.TrimSpace(e.Rarity) == "" {
			missing = append(missing, fmt.Sprintf("[%d] rarity vide (medal_id=%d)", i, e.MedalID))
		}
	}
	if len(missing) > 0 {
		return GuardResult{
			Passed:  false,
			Reason:  fmt.Sprintf("%d champ(s) manquant(s) sur %d entrées", len(missing), len(entries)),
			Details: missing,
		}
	}
	return GuardResult{
		Passed: true,
		Reason: fmt.Sprintf("%d entrées, tous champs requis présents", len(entries)),
	}
}

// CheckImageGuard vérifie la cohérence des images : toutes ou aucune.
// Un import partiel d'images est interdit.
func CheckImageGuard(entries []MedalEntry) GuardResult {
	if len(entries) == 0 {
		return GuardResult{Passed: true, Reason: "aucune entrée"}
	}

	withImage := 0
	for _, e := range entries {
		if strings.TrimSpace(e.ImageURL) != "" {
			withImage++
		}
	}

	total := len(entries)
	if withImage == total {
		return GuardResult{Passed: true, Reason: fmt.Sprintf("%d/%d images présentes", withImage, total)}
	}
	if withImage == 0 {
		return GuardResult{Passed: true, Reason: "0 images — import sans assets visuels (accepté)"}
	}

	// Import partiel interdit.
	return GuardResult{
		Passed: false,
		Reason: fmt.Sprintf("images partielles : %d/%d — import partiel d'assets interdit", withImage, total),
	}
}

// RunAllGuards exécute les 3 garde-fous séquentiellement et retourne le premier échec.
// Si tous passent, retourne un résultat global positif.
func RunAllGuards(entries []MedalEntry, localCount int) GuardResult {
	if r := CheckCardinalityGuard(len(entries), localCount, 10.0); !r.Passed {
		return r
	}
	if r := CheckRequiredFieldsGuard(entries); !r.Passed {
		return r
	}
	if r := CheckImageGuard(entries); !r.Passed {
		return r
	}
	return GuardResult{
		Passed: true,
		Reason: fmt.Sprintf("tous les garde-fous passent (%d entrées)", len(entries)),
	}
}
