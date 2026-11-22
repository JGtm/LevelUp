package timeline

import (
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

func ptrInt(v int) *int       { return &v }
func ptrInt64(v int64) *int64 { return &v }

func TestBuildTimelinesFromPlayerMatches_ReadsSummaryT0Ms(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		{Summary: canonical.MatchSummary{
			MatchID: "m_t0", DurationSeconds: ptrInt(600), T0Ms: ptrInt64(28000),
		}},
		{Summary: canonical.MatchSummary{
			MatchID: "m_no_t0", DurationSeconds: ptrInt(600), T0Ms: nil,
		}},
	}
	got := BuildTimelinesFromPlayerMatches(rows)
	if got["m_t0"].T0Ms != 28000 {
		t.Errorf("m_t0: T0Ms want 28000, got %d", got["m_t0"].T0Ms)
	}
	if got["m_no_t0"].T0Ms != 0 {
		t.Errorf("m_no_t0: T0Ms want 0 (nil fallback), got %d", got["m_no_t0"].T0Ms)
	}
}

func TestBuildTimelinesFromSquadRows_ReadsT0Ms(t *testing.T) {
	rows := []domain.SquadMatchRow{
		{MatchID: "m_t0", DurationSeconds: 600, T0Ms: ptrInt64(28000)},
		{MatchID: "m_no_t0", DurationSeconds: 600, T0Ms: nil},
		// Même match, autre coéquipier : même T0, écrasement sans effet.
		{MatchID: "m_t0", DurationSeconds: 600, T0Ms: ptrInt64(28000)},
	}
	got := BuildTimelinesFromSquadRows(rows)
	if got["m_t0"].T0Ms != 28000 {
		t.Errorf("m_t0: T0Ms want 28000, got %d", got["m_t0"].T0Ms)
	}
	if got["m_t0"].DurationMs != 600000 {
		t.Errorf("m_t0: DurationMs want 600000, got %d", got["m_t0"].DurationMs)
	}
	if got["m_no_t0"].T0Ms != 0 {
		t.Errorf("m_no_t0: T0Ms want 0 (nil fallback), got %d", got["m_no_t0"].T0Ms)
	}
}

// ForMatch est la règle unique de lecture : coup d'envoi connu = la chronologie en part ; inconnu
// (nil) = elle part du début du match et la durée de jeu vaut la durée du match.
func TestForMatch_CoupDEnvoiConnuOuInconnu(t *testing.T) {
	t0 := int64(28000)
	tl := ForMatch(time.Time{}, 600000, &t0)
	if tl.T0Ms != 28000 || tl.GameplayDurationMs() != 572000 {
		t.Errorf("connu : T0=%d durée de jeu=%d, attendu 28000 et 572000", tl.T0Ms, tl.GameplayDurationMs())
	}
	if un := ForMatch(time.Time{}, 600000, nil); un.T0Ms != 0 || un.GameplayDurationMs() != 600000 {
		t.Errorf("inconnu : T0=%d durée de jeu=%d, attendu 0 et 600000", un.T0Ms, un.GameplayDurationMs())
	}
	zero := int64(0)
	if z := ForMatch(time.Time{}, 600000, &zero); z.GameplayDurationMs() != 600000 {
		t.Errorf("coup d'envoi à zéro : durée de jeu=%d, attendu 600000", z.GameplayDurationMs())
	}
}
