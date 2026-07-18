package external

import (
	"testing"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/progression/coach"
)

// TestDefaultForwardedCategories_MirrorsCoach est le GARDE-RAIL de cohérence :
// l'ensemble des catégories relayées par défaut DOIT être exactement l'ensemble
// des catégories émises par le coach (AlertType.NotificationCategory sur tous les
// AlertType). Si un nouvel AlertType coach est ajouté (ou une catégorie non-coach
// se glisse dans la liste par défaut), ce test échoue — évitant qu'un signal coach
// soit silencieusement non relayé ou qu'une notification non-coach parte à l'externe.
func TestDefaultForwardedCategories_MirrorsCoach(t *testing.T) {
	// Ensemble dérivé du coach (source de vérité).
	want := map[notifications.Category]struct{}{}
	for _, at := range coach.AllAlertTypes() {
		if c := at.NotificationCategory(); c != "" {
			want[c] = struct{}{}
		}
	}
	// Ensemble déclaré par défaut.
	got := map[notifications.Category]struct{}{}
	for _, c := range DefaultForwardedCategories() {
		got[c] = struct{}{}
	}

	for c := range want {
		if _, ok := got[c]; !ok {
			t.Errorf("catégorie coach %q émise mais absente de DefaultForwardedCategories()", c)
		}
	}
	for c := range got {
		if _, ok := want[c]; !ok {
			t.Errorf("catégorie %q relayée par défaut mais NON émise par le coach", c)
		}
	}
}

// TestDefaultForwardedCategories_NoDuplicates garantit l'absence de doublon.
func TestDefaultForwardedCategories_NoDuplicates(t *testing.T) {
	seen := map[notifications.Category]bool{}
	for _, c := range DefaultForwardedCategories() {
		if seen[c] {
			t.Errorf("doublon dans DefaultForwardedCategories() : %q", c)
		}
		seen[c] = true
	}
}
