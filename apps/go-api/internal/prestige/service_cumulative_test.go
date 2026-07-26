package prestige

import (
	"context"
	"testing"
	"time"
)

// service_cumulative_test.go — évaluation des défis CUMULATIFS (anomalie B,
// corrigée le 2026-07-26).
//
// Le défaut : `evaluateOne` (persistance) ET `computeCurrentValue` (jauge
// affichée) retombaient tous les deux sur EvaluateThreshold pour un
// EvalCumulative. Un objectif « 220 tirs à la tête » se voyait donc comparé à
// une MOYENNE par match (~1.25) au lieu d'un total. Corriger un seul des deux
// chemins aurait laissé l'autre faux : ces tests verrouillent les DEUX, et le
// fait qu'ils mesurent la même chose.

// cumulativeTestChallenge : un défi cumulatif actif, créé à `created`.
func cumulativeTestChallenge(created time.Time) Challenge {
	return Challenge{
		ID:          "c-cumul",
		UserID:      "demo-player",
		TitleSlug:   "halo_infinite",
		Metric:      "headshot_kills",
		Target:      220,
		EvalType:    EvalCumulative,
		WindowType:  WindowLastNMatches,
		WindowValue: "20",
		Status:      StatusActive,
		Tier:        TierHeroic,
		DataTier:    DataFull,
		CreatedAt:   created,
	}
}

// TestCumulativeChallenge_ProgressesBySumFromCreatedAt : la valeur mesurée est
// la SOMME rendue par CumulativeSince, et la borne transmise au provider est le
// created_at du défi — jamais plus tôt (invariant anti-complétion-rétroactive).
func TestCumulativeChallenge_ProgressesBySumFromCreatedAt(t *testing.T) {
	svc, chRepo, _, _, _, bp := buildCoverageService()
	created := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	c := cumulativeTestChallenge(created)
	chRepo.stored[c.ID] = c
	chRepo.listResult = []Challenge{c}

	// 176 tirs à la tête cumulés sur 40 matchs — soit BIEN plus que la fenêtre
	// baseline (20). Une somme plafonnée par la fenêtre serait donc détectable.
	bp.cumulative, bp.cumulativeN = 176, 40

	outcomes, err := svc.EvaluateForUser(context.Background(), "demo-player", "halo_infinite")
	if err != nil {
		t.Fatalf("EvaluateForUser: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("outcomes = %d, want 1", len(outcomes))
	}
	if outcomes[0].NewValue != 176 {
		t.Errorf("NewValue = %.2f, want 176 (le total, pas une moyenne)", outcomes[0].NewValue)
	}
	if outcomes[0].NewStatus != StatusActive {
		t.Errorf("status = %s, want active (176 < 220)", outcomes[0].NewStatus)
	}
	if !bp.sinceSeen.Equal(created) {
		t.Errorf("borne transmise au provider = %v, want created_at %v — un défi ne doit "+
			"jamais se compléter avec l'historique antérieur à sa création", bp.sinceSeen, created)
	}
}

// TestCumulativeChallenge_CompletesWhenTotalReachesTarget : franchissement de
// cible → transition persistée vers completed.
func TestCumulativeChallenge_CompletesWhenTotalReachesTarget(t *testing.T) {
	svc, chRepo, _, _, _, bp := buildCoverageService()
	c := cumulativeTestChallenge(time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	chRepo.stored[c.ID] = c
	chRepo.listResult = []Challenge{c}
	bp.cumulative, bp.cumulativeN = 231, 52

	outcomes, err := svc.EvaluateForUser(context.Background(), "demo-player", "halo_infinite")
	if err != nil {
		t.Fatalf("EvaluateForUser: %v", err)
	}
	if outcomes[0].NewStatus != StatusCompleted {
		t.Errorf("status = %s, want completed (231 >= 220)", outcomes[0].NewStatus)
	}
	if outcomes[0].Reason != EvalReasonTargetReached {
		t.Errorf("reason = %s, want target_reached", outcomes[0].Reason)
	}
	if len(chRepo.updateStatus) != 1 || chRepo.updateStatus[0] != StatusCompleted {
		t.Errorf("transition persistée = %v, want [completed]", chRepo.updateStatus)
	}
}

// TestCumulativeChallenge_GaugeMatchesEvaluation — RÉGRESSION CENTRALE. La jauge
// servie par ListChallenges (CurrentValue) DOIT être la même mesure que celle de
// l'évaluation. C'est computeCurrentValue qui produisait 1.25 au lieu du total :
// corriger evaluateOne seul aurait laissé l'écran faux.
func TestCumulativeChallenge_GaugeMatchesEvaluation(t *testing.T) {
	svc, chRepo, _, _, _, bp := buildCoverageService()
	c := cumulativeTestChallenge(time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC))
	chRepo.stored[c.ID] = c
	chRepo.listResult = []Challenge{c}
	bp.cumulative, bp.cumulativeN = 176, 40

	list, err := svc.ListActiveChallenges(context.Background(), "demo-player", "halo_infinite")
	if err != nil {
		t.Fatalf("ListActiveChallenges: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %d, want 1", len(list))
	}
	if list[0].CurrentValue != 176 {
		t.Errorf("CurrentValue affichée = %.2f, want 176 — la jauge doit servir le TOTAL "+
			"(elle affichait la moyenne par match avant le 2026-07-26)", list[0].CurrentValue)
	}
	// Et la mesure est bien la MÊME que celle du chemin qui persiste.
	outcomes, err := svc.EvaluateForUser(context.Background(), "demo-player", "halo_infinite")
	if err != nil {
		t.Fatalf("EvaluateForUser: %v", err)
	}
	if outcomes[0].NewValue != list[0].CurrentValue {
		t.Errorf("divergence affichage (%.2f) / évaluation (%.2f) — les deux chemins doivent "+
			"partager evaluateChallengeNow", list[0].CurrentValue, outcomes[0].NewValue)
	}
}

// TestThresholdChallenge_StillAveragesUnchanged — contrôle négatif : la
// correction cumulative ne touche pas au chemin threshold, qui reste une moyenne
// sur les N derniers matchs (et n'appelle PAS CumulativeSince).
func TestThresholdChallenge_StillAveragesUnchanged(t *testing.T) {
	svc, chRepo, _, _, _, bp := buildCoverageService()
	c := Challenge{
		ID: "c-thr", UserID: "demo-player", TitleSlug: "halo_infinite",
		Metric: "kda", Target: 2.0, EvalType: EvalThreshold,
		WindowType: WindowLastNMatches, WindowValue: "10",
		Status: StatusActive, Tier: TierNormal, DataTier: DataFull,
		CreatedAt: time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
	}
	chRepo.stored[c.ID] = c
	chRepo.listResult = []Challenge{c}
	// tenMatches() (fixture par défaut) = 10 matchs à 1.0 → moyenne 1.0.
	// Un cumul serait 10 : la distinction est donc observable.
	bp.cumulative, bp.cumulativeN = 10, 10

	outcomes, err := svc.EvaluateForUser(context.Background(), "demo-player", "halo_infinite")
	if err != nil {
		t.Fatalf("EvaluateForUser: %v", err)
	}
	if outcomes[0].NewValue != 1.0 {
		t.Errorf("NewValue = %.2f, want 1.0 (moyenne des 10 matchs, PAS leur somme)",
			outcomes[0].NewValue)
	}
	if !bp.sinceSeen.IsZero() {
		t.Errorf("CumulativeSince appelé pour un défi threshold (borne %v) — chemin croisé",
			bp.sinceSeen)
	}
}
