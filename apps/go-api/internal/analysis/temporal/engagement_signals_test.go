package temporal_test

import (
	"testing"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/games/canonical"
)

// intPtrSig : helper local pour construire un *int (masque de presence des signaux riches).
func intPtrSig(n int) *int { return &n }

// ─── Sufficiency : 3 niveaux ───────────────────────────────────────────────

func TestSufficiency_InsufficientWithoutTimedPlayerEvents(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{HasTimedPlayerEvents: false, HasLobbyPace: true, DurationMS: temporal.MinMatchDurationMS}
	if got := s.Sufficiency(); got != temporal.SufficiencyInsufficient {
		t.Fatalf("Sufficiency() = %v, want Insufficient (pas de frags/morts dates)", got)
	}
}

func TestSufficiency_InsufficientWithoutLobbyPace(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{HasTimedPlayerEvents: true, HasLobbyPace: false, DurationMS: temporal.MinMatchDurationMS}
	if got := s.Sufficiency(); got != temporal.SufficiencyInsufficient {
		t.Fatalf("Sufficiency() = %v, want Insufficient (pas d'ancre lobby)", got)
	}
}

func TestSufficiency_InsufficientWhenTooShort(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{HasTimedPlayerEvents: true, HasLobbyPace: true, DurationMS: temporal.MinMatchDurationMS - 1}
	if got := s.Sufficiency(); got != temporal.SufficiencyInsufficient {
		t.Fatalf("Sufficiency() = %v, want Insufficient (match trop court)", got)
	}
}

func TestSufficiency_PartialWithMinimalOnly(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{HasTimedPlayerEvents: true, HasLobbyPace: true, DurationMS: temporal.MinMatchDurationMS}
	if got := s.Sufficiency(); got != temporal.SufficiencyPartial {
		t.Fatalf("Sufficiency() = %v, want Partial (minimal sans signal riche)", got)
	}
}

func TestSufficiency_FullWithObjectiveSignal(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{
		HasTimedPlayerEvents: true, HasLobbyPace: true, DurationMS: temporal.MinMatchDurationMS,
		ObjectiveEvents: intPtrSig(3),
	}
	if got := s.Sufficiency(); got != temporal.SufficiencyFull {
		t.Fatalf("Sufficiency() = %v, want Full (signal objectif present)", got)
	}
}

func TestSufficiency_FullWithRichKillMechanics(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{
		HasTimedPlayerEvents: true, HasLobbyPace: true, DurationMS: temporal.MinMatchDurationMS,
		RichKillMechanics: intPtrSig(2),
	}
	if got := s.Sufficiency(); got != temporal.SufficiencyFull {
		t.Fatalf("Sufficiency() = %v, want Full (mecaniques de kill riches)", got)
	}
}

// Un signal riche PRESENT mais a 0 ne compte pas comme riche (Partial, pas Full).
func TestSufficiency_ZeroCountRichSignalStaysPartial(t *testing.T) {
	t.Parallel()
	s := temporal.EngagementSignals{
		HasTimedPlayerEvents: true, HasLobbyPace: true, DurationMS: temporal.MinMatchDurationMS,
		ObjectiveEvents: intPtrSig(0),
	}
	if got := s.Sufficiency(); got != temporal.SufficiencyPartial {
		t.Fatalf("Sufficiency() = %v, want Partial (objectif present mais 0)", got)
	}
}

func TestSignalSufficiency_String(t *testing.T) {
	t.Parallel()
	cases := map[temporal.SignalSufficiency]string{
		temporal.SufficiencyInsufficient: "insufficient",
		temporal.SufficiencyPartial:      "partial",
		temporal.SufficiencyFull:         "full",
	}
	for lvl, want := range cases {
		if got := lvl.String(); got != want {
			t.Errorf("String(%d) = %q, want %q", lvl, got, want)
		}
	}
}

// ─── SignalsFromEvents : derivation title-agnostic ─────────────────────────

func TestSignalsFromEvents_DerivesMinimalAndRich(t *testing.T) {
	t.Parallel()
	player := concatEvents(
		makeEvents(canonical.EventKill, 10_000, 20_000),
		makeEvents(canonical.EventDeath, 15_000),
		// objectif ("mode") : famille de signal riche, non couverte par une constante canonical.
		[]canonical.HighlightEvent{{EventType: "mode", TimeMS: 30_000}},
		makeEvents(canonical.EventFirstKill, 5_000),
	)
	lobby := makeEvents(canonical.EventKill, 1_000, 2_000, 3_000)

	sig := temporal.SignalsFromEvents(player, lobby, 300_000)

	if !sig.HasTimedPlayerEvents {
		t.Error("HasTimedPlayerEvents devrait etre vrai (kills/deaths presents)")
	}
	if !sig.HasLobbyPace {
		t.Error("HasLobbyPace devrait etre vrai (lobby non vide)")
	}
	if sig.DurationMS != 300_000 {
		t.Errorf("DurationMS = %d, want 300000", sig.DurationMS)
	}
	if sig.ObjectiveEvents == nil || *sig.ObjectiveEvents != 1 {
		t.Errorf("ObjectiveEvents = %v, want 1", sig.ObjectiveEvents)
	}
	// first_kill compte comme mecanique riche.
	if sig.RichKillMechanics == nil || *sig.RichKillMechanics != 1 {
		t.Errorf("RichKillMechanics = %v, want 1", sig.RichKillMechanics)
	}
	if sig.Sufficiency() != temporal.SufficiencyFull {
		t.Errorf("Sufficiency() = %v, want Full", sig.Sufficiency())
	}
}

func TestSignalsFromEvents_NoRichSignalsLeavesPointersNil(t *testing.T) {
	t.Parallel()
	player := makeEvents(canonical.EventKill, 10_000, 20_000)
	lobby := makeEvents(canonical.EventKill, 1_000)

	sig := temporal.SignalsFromEvents(player, lobby, 240_000)

	if sig.ObjectiveEvents != nil {
		t.Errorf("ObjectiveEvents devrait etre nil (aucun event mode), got %v", *sig.ObjectiveEvents)
	}
	if sig.RichKillMechanics != nil {
		t.Errorf("RichKillMechanics devrait etre nil, got %v", *sig.RichKillMechanics)
	}
	if sig.Sufficiency() != temporal.SufficiencyPartial {
		t.Errorf("Sufficiency() = %v, want Partial", sig.Sufficiency())
	}
}

func TestSignalsFromEvents_EmptyIsInsufficientAndZero(t *testing.T) {
	t.Parallel()
	sig := temporal.SignalsFromEvents(nil, nil, 0)
	if !sig.IsZero() {
		t.Error("vecteur derive d'inputs vides devrait etre IsZero()")
	}
	if sig.Sufficiency() != temporal.SufficiencyInsufficient {
		t.Errorf("Sufficiency() = %v, want Insufficient", sig.Sufficiency())
	}
}

// ─── Poids nul : les signaux riches ne modifient PAS le score (DE-5) ───────

// Le score est calcule sur la courbe d'events ponderee ; input.Signals ne participe
// PAS au calcul (uniquement a la suffisance). Prouve que fournir des signaux riches
// distincts laisse EngagementScore et ResidualBrut identiques (byte-identical).
func TestComputeEngagement_RichSignalsDoNotChangeScore(t *testing.T) {
	t.Parallel()
	base := engagementInputFixture()

	// Sans signaux riches explicites (derive interne).
	base.Signals = temporal.EngagementSignals{}
	got1, err := temporal.ComputeEngagementScore(base)
	if err != nil {
		t.Fatalf("compute (derive) err = %v", err)
	}

	// Avec un vecteur riche explicite fourni par l'appelant.
	base.Signals = temporal.EngagementSignals{
		HasTimedPlayerEvents: true, HasLobbyPace: true, DurationMS: base.MatchEndMS - base.MatchStartMS,
		ObjectiveEvents: intPtrSig(5), RichKillMechanics: intPtrSig(4),
	}
	got2, err := temporal.ComputeEngagementScore(base)
	if err != nil {
		t.Fatalf("compute (riche) err = %v", err)
	}

	if got1.ResidualBrut != got2.ResidualBrut {
		t.Errorf("ResidualBrut change avec les signaux riches: %v vs %v (doit etre identique)", got1.ResidualBrut, got2.ResidualBrut)
	}
	if !floatPtrEqualSig(got1.EngagementScore, got2.EngagementScore) {
		t.Errorf("EngagementScore change avec les signaux riches: %v vs %v", got1.EngagementScore, got2.EngagementScore)
	}
	if len(got1.EngagementCurve) != len(got2.EngagementCurve) {
		t.Errorf("longueur de courbe differente: %d vs %d", len(got1.EngagementCurve), len(got2.EngagementCurve))
	}
	// Le SignalBasis, lui, reflete bien la difference de suffisance.
	if got2.SignalBasis != "full" {
		t.Errorf("SignalBasis (riche) = %q, want full", got2.SignalBasis)
	}
}

// engagementInputFixture construit un input minimal valide et exploitable (match
// suffisamment long, events joueur/lobby presents) pour les tests de compute.
func engagementInputFixture() temporal.EngagementScoreInput {
	player := concatEvents(
		makeEvents(canonical.EventKill, 30_000, 90_000, 150_000),
		makeEvents(canonical.EventDeath, 60_000, 120_000),
	)
	lobby := concatEvents(
		makeEvents(canonical.EventKill, 20_000, 40_000, 80_000, 100_000, 140_000, 200_000),
	)
	return temporal.EngagementScoreInput{
		PlayerEvents: player,
		TeamEvents:   nil,
		LobbyEvents:  lobby,
		NTeam:        4,
		NHumansLobby: 8,
		XUID:         "xuid-test",
		MatchStartMS: 0,
		MatchEndMS:   300_000,
		Mode:         "PvP_unranked",
		IsTeamMode:   true,
	}
}

func floatPtrEqualSig(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
