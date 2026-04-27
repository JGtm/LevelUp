package temporal

import (
	"testing"
	"time"
)

type fakeRow struct{ ts time.Time }

func (f fakeRow) GetStartTime() time.Time { return f.ts }

func TestPeriod_IsValid(t *testing.T) {
	t.Parallel()
	valid := []Period{PeriodAll, Period2Y, Period1Y, Period1M, Period1W}
	for _, p := range valid {
		if !p.IsValid() {
			t.Errorf("%q should be valid", p)
		}
	}
	invalid := []Period{"", "invalid", "3y", "1d"}
	for _, p := range invalid {
		if p.IsValid() {
			t.Errorf("%q should not be valid", p)
		}
	}
}

func TestPeriod_Since(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		period Period
		want   *time.Time
	}{
		{"all", PeriodAll, nil},
		{"2y", Period2Y, ptrTime(time.Date(2024, 4, 27, 12, 0, 0, 0, time.UTC))},
		{"1y", Period1Y, ptrTime(time.Date(2025, 4, 27, 12, 0, 0, 0, time.UTC))},
		{"1m", Period1M, ptrTime(time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC))},
		{"1w", Period1W, ptrTime(time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC))},
		{"invalid", Period("3y"), nil},
		{"empty", Period(""), nil},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := c.period.Since(now)
			if (got == nil) != (c.want == nil) {
				t.Fatalf("want nil=%v got nil=%v", c.want == nil, got == nil)
			}
			if got != nil && !got.Equal(*c.want) {
				t.Errorf("want %v got %v", *c.want, *got)
			}
		})
	}
}

func TestFilterByPeriod(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	rows := []fakeRow{
		{time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)}, // 1j en arriere
		{time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},  // ~57j en arriere
		{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},  // ~16m en arriere
		{time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},  // ~4y en arriere
	}
	cases := []struct {
		name   string
		period Period
		want   int
	}{
		{"all keeps all", PeriodAll, 4},
		{"1w keeps last day only", Period1W, 1},
		{"1m keeps last day only", Period1M, 1},
		{"1y keeps recent + 16m back", Period1Y, 2},
		{"2y keeps everything except 4y", Period2Y, 3},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := FilterByPeriod(rows, c.period, now)
			if len(got) != c.want {
				t.Errorf("want %d rows got %d", c.want, len(got))
			}
		})
	}
}

func TestFilterByPeriod_Empty(t *testing.T) {
	t.Parallel()
	rows := []fakeRow{}
	now := time.Now()
	got := FilterByPeriod(rows, Period1Y, now)
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d", len(got))
	}
}

func TestFilterByPeriod_BoundaryInclusive(t *testing.T) {
	t.Parallel()
	// Une row exactement a `since` doit etre conservee (filtre >=).
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	since := *Period1W.Since(now)
	rows := []fakeRow{{since}}
	got := FilterByPeriod(rows, Period1W, now)
	if len(got) != 1 {
		t.Errorf("boundary should be kept, got %d", len(got))
	}
}

func TestFilterByPeriod_AllReturnsSameSlice(t *testing.T) {
	t.Parallel()
	// PeriodAll ne doit pas allouer une copie.
	rows := []fakeRow{{time.Now()}}
	got := FilterByPeriod(rows, PeriodAll, time.Now())
	if &got[0] != &rows[0] {
		t.Error("PeriodAll should return the input slice without copying")
	}
}

func ptrTime(v time.Time) *time.Time { return &v }
