package relations

import "testing"

func d(out DuelOutcome, kills, deaths int) Duel {
	return Duel{Outcome: out, KillsOnRival: kills, DeathsByRival: deaths}
}

func TestComputeRivalryMetrics_FragGapAndStreak(t *testing.T) {
	// Ancien→récent : L(2/5), L(3/4), W(6/3), W(7/2), W(5/4).
	duels := []Duel{
		d(DuelLoss, 2, 5),
		d(DuelLoss, 3, 4),
		d(DuelWin, 6, 3),
		d(DuelWin, 7, 2),
		d(DuelWin, 5, 4),
	}
	m := ComputeRivalryMetrics(duels)

	// FragGap cumulé : (2-5)+(3-4)+(6-3)+(7-2)+(5-4) = -3-1+3+5+1 = 5.
	if m.FragGap != 5 {
		t.Fatalf("FragGap=%d want 5", m.FragGap)
	}
	// Série en cours : 3 victoires consécutives (positif).
	if m.CurrentStreak != 3 {
		t.Fatalf("CurrentStreak=%d want 3", m.CurrentStreak)
	}
	// GlobalWinRate = 3/5.
	if m.GlobalWinRate == nil || *m.GlobalWinRate != 0.6 {
		t.Fatalf("GlobalWinRate=%v want 0.6", m.GlobalWinRate)
	}
	if m.DecisiveCount != 5 {
		t.Fatalf("DecisiveCount=%d want 5", m.DecisiveCount)
	}
}

func TestComputeRivalryMetrics_RollingWindow(t *testing.T) {
	// 6 duels : L L W W W W (ancien→récent). Fenêtre = 5.
	duels := []Duel{
		d(DuelLoss, 0, 1),
		d(DuelLoss, 0, 1),
		d(DuelWin, 1, 0),
		d(DuelWin, 1, 0),
		d(DuelWin, 1, 0),
		d(DuelWin, 1, 0),
	}
	m := ComputeRivalryMetrics(duels)
	if len(m.RollingWinRate) != 6 {
		t.Fatalf("rolling len=%d want 6", len(m.RollingWinRate))
	}
	// Point final : fenêtre des 5 derniers = [L W W W W] = 4/5 = 0.8.
	last := m.RollingWinRate[5]
	if last == nil || *last != 0.8 {
		t.Fatalf("rolling[last]=%v want 0.8", last)
	}
	// RecentWinRate doit égaler le dernier point glissant.
	if m.RecentWinRate == nil || *m.RecentWinRate != 0.8 {
		t.Fatalf("RecentWinRate=%v want 0.8", m.RecentWinRate)
	}
	// Premier point : fenêtre = [L] = 0/1 = 0.
	first := m.RollingWinRate[0]
	if first == nil || *first != 0.0 {
		t.Fatalf("rolling[0]=%v want 0.0", first)
	}
}

func TestComputeRivalryMetrics_NonDecisiveOnly(t *testing.T) {
	duels := []Duel{d(DuelOther, 1, 1), d(DuelOther, 2, 0)}
	m := ComputeRivalryMetrics(duels)
	if m.GlobalWinRate != nil {
		t.Fatalf("GlobalWinRate=%v want nil (no decisive)", m.GlobalWinRate)
	}
	if m.CurrentStreak != 0 {
		t.Fatalf("CurrentStreak=%d want 0 (last non-decisive)", m.CurrentStreak)
	}
	// FragGap reste calculé : (1-1)+(2-0)=2.
	if m.FragGap != 2 {
		t.Fatalf("FragGap=%d want 2", m.FragGap)
	}
	if m.RollingWinRate[0] != nil || m.RollingWinRate[1] != nil {
		t.Fatalf("rolling should be nil for non-decisive windows")
	}
}

func TestComputeRivalryMetrics_LossStreakNegative(t *testing.T) {
	duels := []Duel{d(DuelWin, 1, 0), d(DuelLoss, 0, 1), d(DuelLoss, 0, 2)}
	m := ComputeRivalryMetrics(duels)
	if m.CurrentStreak != -2 {
		t.Fatalf("CurrentStreak=%d want -2 (2 losses)", m.CurrentStreak)
	}
}

func TestComputeRivalryMetrics_Empty(t *testing.T) {
	m := ComputeRivalryMetrics(nil)
	if m.FragGap != 0 || m.CurrentStreak != 0 || m.GlobalWinRate != nil || m.RecentWinRate != nil {
		t.Fatalf("empty metrics=%+v", m)
	}
	if len(m.RollingWinRate) != 0 {
		t.Fatalf("rolling len=%d want 0", len(m.RollingWinRate))
	}
}

func TestResultToDuel(t *testing.T) {
	if ResultToDuel(ResultWin) != DuelWin {
		t.Fatal("ResultWin → DuelWin")
	}
	if ResultToDuel(ResultLoss) != DuelLoss {
		t.Fatal("ResultLoss → DuelLoss")
	}
	if ResultToDuel(0) != DuelOther {
		t.Fatal("0 → DuelOther")
	}
}

func TestDaypartFromHour(t *testing.T) {
	cases := map[int]Daypart{
		0: DaypartNight, 5: DaypartNight,
		6: DaypartMorning, 10: DaypartMorning,
		11: DaypartNoon, 13: DaypartNoon,
		14: DaypartAfternoon, 17: DaypartAfternoon,
		18: DaypartEvening, 21: DaypartEvening,
		22: DaypartLateNight, 23: DaypartLateNight,
	}
	for hour, want := range cases {
		if got := DaypartFromHour(hour); got != want {
			t.Errorf("DaypartFromHour(%d)=%d want %d", hour, got, want)
		}
	}
}
