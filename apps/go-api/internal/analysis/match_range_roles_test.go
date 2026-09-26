package analysis

// match_range_roles_test.go — les bandes de rôle d'une période.
//
// Ce que ces tests verrouillent :
//
//  1. SEULS les points pleins (>= MatchRangeRoleMinMeasured frags) entrent dans les
//     quantiles — un match à 2 frags déplacerait les bandes sans être une médiane ;
//  2. sous MatchRangeRoleMinPoints points pleins, les TROIS repères sont absents ENSEMBLE ;
//  3. le filtre par xuid ne retient que le joueur demandé (le bloc Escouade publie N
//     joueurs, la période de session n'en veut qu'un).

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

func mrrProfil(joueurs ...domain.MatchRangePlayer) domain.MatchRangeProfile {
	return domain.MatchRangeProfile{MatchID: "m", Players: joueurs}
}

func mrrJoueur(xuid string, ecart float64, mesures int) domain.MatchRangePlayer {
	return domain.MatchRangePlayer{XUID: xuid, LobbyDeltaM: ecart, Measured: mesures}
}

func TestMatchRangeRoleBands_TiersDesPointsPleinsSeulement(t *testing.T) {
	// Quatre points pleins (0, 3, 6, 9) et un point creux à 100 : le creux ne doit rien
	// déplacer. Quantiles linéaires sur [0,3,6,9] : 1/3 -> 3, 2/3 -> 6, médiane -> 4,5.
	profils := []domain.MatchRangeProfile{
		mrrProfil(mrrJoueur("a", 0, 5)),
		mrrProfil(mrrJoueur("a", 3, 8)),
		mrrProfil(mrrJoueur("a", 6, 5)),
		mrrProfil(mrrJoueur("a", 9, 12)),
		mrrProfil(mrrJoueur("a", 100, 4)),
	}
	got := MatchRangeRoleBands(profils, "a")
	if got.FullPoints != 4 {
		t.Fatalf("FullPoints = %d, want 4 (le point a 4 frags est creux)", got.FullPoints)
	}
	if got.LowM == nil || got.HighM == nil || got.MedianM == nil {
		t.Fatalf("reperes = %v/%v/%v, want trois valeurs", got.LowM, got.HighM, got.MedianM)
	}
	for _, c := range []struct {
		nom  string
		got  float64
		want float64
	}{
		{"role_low_m", *got.LowM, 3},
		{"role_high_m", *got.HighM, 6},
		{"period_median_delta_m", *got.MedianM, 4.5},
	} {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Errorf("%s = %v, want %v", c.nom, c.got, c.want)
		}
	}
}

func TestMatchRangeRoleBands_AbsentsSousTroisPointsPleins(t *testing.T) {
	profils := []domain.MatchRangeProfile{
		mrrProfil(mrrJoueur("a", 0, 5)),
		mrrProfil(mrrJoueur("a", 9, 9)),
		mrrProfil(mrrJoueur("a", 50, 1)),
		mrrProfil(mrrJoueur("a", 60, 4)),
	}
	got := MatchRangeRoleBands(profils, "a")
	if got.FullPoints != 2 {
		t.Fatalf("FullPoints = %d, want 2", got.FullPoints)
	}
	if got.LowM != nil || got.HighM != nil || got.MedianM != nil {
		t.Errorf("reperes = %v/%v/%v, want absents ENSEMBLE (deux matchs ne font pas un habituel)",
			got.LowM, got.HighM, got.MedianM)
	}
}

func TestMatchRangeRoleBands_FiltreSurLeJoueurDemande(t *testing.T) {
	profils := []domain.MatchRangeProfile{
		mrrProfil(mrrJoueur("a", 0, 5), mrrJoueur("b", 100, 20)),
		mrrProfil(mrrJoueur("a", 3, 5), mrrJoueur("b", 100, 20)),
		mrrProfil(mrrJoueur("a", 6, 5), mrrJoueur("b", 100, 20)),
	}
	got := MatchRangeRoleBands(profils, "a")
	if got.FullPoints != 3 || got.HighM == nil || *got.HighM > 6 {
		t.Fatalf("bandes = %+v, want les seuls points de « a » (haut <= 6)", got)
	}
	// xuid vide = tous les joueurs publiés (le cas de l'Escouade).
	tous := MatchRangeRoleBands(profils, "")
	if tous.FullPoints != 6 {
		t.Errorf("FullPoints (xuid vide) = %d, want 6", tous.FullPoints)
	}
}
