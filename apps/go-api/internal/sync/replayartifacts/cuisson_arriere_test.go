package replayartifacts

// cuisson_arriere_test.go — ce que la passe laisse a reprendre, et l'arret des derivations sur le
// budget de temps (et non sur un nombre).

import (
	"context"
	"testing"
	"time"
)

// Un match essaye et non resolu (echec, ecarte, sans film) n'est PAS a reprendre : la cadence courte
// qui le reprendrait sans fin ne le ferait pas avancer. Sont a reprendre : ce que la passe n'a pas
// atteint, et ce qu'elle a reporte (film non finalise, partie en cours, autre decodage).
func TestBilan_AReprendre(t *testing.T) {
	b := bilanCuisson{construits: 3, echecs: 2, sansFilm: 1, reportes: 1, ecartes: 1}
	// 10 selectionnes ; traites = 3 + 2 + 1 + 1 = 7, ecartes = 1 : 2 pas atteints ; + 1 reporte.
	if got := b.aReprendre(10); got != 3 {
		t.Errorf("a reprendre = %d, attendu 3 (2 pas atteints + 1 reporte)", got)
	}
	if got := (bilanCuisson{construits: 4}).aReprendre(4); got != 0 {
		t.Errorf("a reprendre = %d, attendu 0 : tout le lot est traite", got)
	}
	if got := (bilanCuisson{budgetEpuise: true, construits: 1}).aReprendre(5); got != 4 {
		t.Errorf("a reprendre = %d, attendu 4 : le budget a coupe la passe apres un match", got)
	}
	if got := (bilanCuisson{reportesModeJeu: 2, reportesVerrou: 1}).aReprendre(3); got != 3 {
		t.Errorf("a reprendre = %d, attendu 3 : partie en cours et verrou tenu se reprennent", got)
	}
}

func lotsDe(n int) []ArtefactRange { return make([]ArtefactRange, n) }

func TestEnLots_ToutLeTravailSousUnBudgetLarge(t *testing.T) {
	var tailles []int
	enLots(context.Background(), Deps{Budget: time.Minute}, lotsDe(12), func(lot []ArtefactRange) {
		tailles = append(tailles, len(lot))
	})
	if len(tailles) != 3 || tailles[0] != lotDeDerivations || tailles[2] != 12-2*lotDeDerivations {
		t.Errorf("lots = %v, attendu des lots de %d puis le reste : aucune borne en nombre", tailles, lotDeDerivations)
	}
}

func TestEnLots_ArretSurLeBudgetDeTemps(t *testing.T) {
	var lots int
	// Budget deja epuise : le premier lot part, aucun autre.
	enLots(context.Background(), Deps{Budget: -1}, lotsDe(12), func([]ArtefactRange) { lots++ })
	if lots != 1 {
		t.Errorf("lots appliques = %d, attendu 1 : le budget epuise arrete entre deux lots", lots)
	}

	// Horloge reelle : chaque lot dure 30 ms, le budget 50 ms.
	lots = 0
	enLots(context.Background(), Deps{Budget: 50 * time.Millisecond}, lotsDe(100), func([]ArtefactRange) {
		lots++
		time.Sleep(30 * time.Millisecond)
	})
	if lots < 2 || lots >= 20 {
		t.Errorf("lots appliques = %d : la passe devait s'arreter sur le budget de temps, avec du solde", lots)
	}
}

func TestEnLots_ContexteAnnule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var lots int
	enLots(ctx, Deps{Budget: time.Minute}, lotsDe(12), func([]ArtefactRange) { lots++; cancel() })
	if lots != 1 {
		t.Errorf("lots appliques = %d, attendu 1 : un contexte annule arrete la passe", lots)
	}
}
