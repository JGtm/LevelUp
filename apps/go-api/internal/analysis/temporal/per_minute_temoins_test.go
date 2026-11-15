package temporal

import (
	"strconv"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

func temoinMS(t *testing.T, label string, v float64, want string) {
	t.Helper()
	got := strconv.FormatFloat(v, 'g', -1, 64)
	t.Logf("%s = %s", label, got)
	if got != want {
		t.Errorf("%s = %s, attendu %s", label, got, want)
	}
}

// Témoins de caractérisation de l'intensité du match (événements par minute et par joueur sur sa durée) : valeur
// au bit près sur une durée non multiple de 60 000 ms, et 0 hors durée positive.
func TestTemoinMinute_IntensiteDuMatch(t *testing.T) {
	events := make([]canonical.HighlightEvent, 37)
	temoinMS(t, "intensite.937123", computeMatchIntensity(events, 937123, 7), "0.33842180497422125")
	temoinMS(t, "intensite.1800000", computeMatchIntensity(events, 1800000, 8), "0.15416666666666667")
	temoinMS(t, "intensite.zero", computeMatchIntensity(events, 0, 7), "0")
	temoinMS(t, "intensite.negatif", computeMatchIntensity(events, -5, 7), "0")
}

// Témoins de caractérisation de la courbe d'engagement : pace du joueur, de l'équipe et du lobby par minute de
// fenêtre, au bit près, fenêtre non multiple de 60 000 ms.
func TestTemoinMinute_CourbeDEngagement(t *testing.T) {
	ev := func(ms int64) canonical.HighlightEvent {
		return canonical.HighlightEvent{EventType: "kill", TimeMS: ms}
	}
	curve := buildEngagementCurve(buildCurveParams{
		PlayerEvents: []canonical.HighlightEvent{ev(10000), ev(20000), ev(45000), ev(70000)},
		TeamEvents:   []canonical.HighlightEvent{ev(10000), ev(15000), ev(20000), ev(45000), ev(50000), ev(70000)},
		LobbyEvents:  []canonical.HighlightEvent{ev(5000), ev(10000), ev(15000), ev(20000), ev(25000), ev(45000), ev(50000), ev(70000), ev(80000)},
		NTeam:        4,
		NHumansLobby: 8,
		MatchStartMS: 0,
		MatchEndMS:   90000,
		WindowMS:     37123,
		SamplingMS:   15000,
		Weights:      DefaultEventWeights(),
	})
	if len(curve) != 7 {
		t.Fatalf("points = %d, attendu 7", len(curve))
	}
	attendu := [7][3]string{
		{"1.616248686797942", "0.808124343398971", "0.6060932575492283"},
		{"3.232497373595884", "1.2121865150984565", "1.0101554292487136"},
		{"3.232497373595884", "1.2121865150984565", "0.808124343398971"},
		{"1.616248686797942", "0.808124343398971", "0.4040621716994855"},
		{"3.232497373595884", "1.2121865150984565", "0.6060932575492283"},
		{"1.616248686797942", "0.4040621716994855", "0.4040621716994855"},
		{"0", "0", "0.20203108584974275"},
	}
	for i, p := range curve {
		label := "courbe." + strconv.Itoa(i)
		temoinMS(t, label+".joueur", p.PaceJoueur, attendu[i][0])
		temoinMS(t, label+".team", p.PaceTeam, attendu[i][1])
		temoinMS(t, label+".lobby", p.PaceLobby, attendu[i][2])
	}
	if got := buildEngagementCurve(buildCurveParams{MatchEndMS: 90000, SamplingMS: 15000, WindowMS: 0}); got != nil {
		t.Errorf("fenêtre nulle : courbe nil attendue, obtenu %d points", len(got))
	}
}
