package temporal

import (
	"testing"
	"time"
)

func TestGranularity_IsValid(t *testing.T) {
	t.Parallel()
	for _, g := range []Granularity{GranDay, GranWeek, GranMonth, GranAdaptive} {
		if !g.IsValid() {
			t.Errorf("%q should be valid", g)
		}
	}
	for _, g := range []Granularity{"", "1h", "year"} {
		if g.IsValid() {
			t.Errorf("%q should not be valid", g)
		}
	}
}

func TestResolveAdaptive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		period Period
		want   Granularity
	}{
		{Period1W, GranDay},
		{Period1M, GranDay},
		{Period1Y, GranWeek},
		{Period2Y, GranMonth},
		{PeriodAll, GranMonth},
		{Period("invalid"), GranDay}, // defensive default
	}
	for _, c := range cases {
		if got := ResolveAdaptive(c.period); got != c.want {
			t.Errorf("ResolveAdaptive(%s) want %s got %s", c.period, c.want, got)
		}
	}
}

func TestBucketByGranularity_Day(t *testing.T) {
	t.Parallel()
	rows := []fakeRow{
		{time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 27, 18, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 28, 9, 0, 0, 0, time.UTC)},
	}
	buckets := BucketByGranularity(rows, GranDay, PeriodAll)
	if len(buckets) != 2 {
		t.Fatalf("want 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Label != "2026-04-27" || len(buckets[0].Items) != 2 {
		t.Errorf("first: label=%q items=%d", buckets[0].Label, len(buckets[0].Items))
	}
	if buckets[1].Label != "2026-04-28" || len(buckets[1].Items) != 1 {
		t.Errorf("second: label=%q items=%d", buckets[1].Label, len(buckets[1].Items))
	}
	// End exclusive : End du premier bucket = Start du second.
	if !buckets[0].End.Equal(buckets[1].Start) {
		t.Error("day buckets: End[0] should equal Start[1]")
	}
}

func TestBucketByGranularity_Week(t *testing.T) {
	t.Parallel()
	// 2026-04-27 = lundi, ISO week 18.
	rows := []fakeRow{
		{time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)}, // mardi meme semaine
		{time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)},  // lundi semaine suivante
	}
	buckets := BucketByGranularity(rows, GranWeek, PeriodAll)
	if len(buckets) != 2 {
		t.Fatalf("want 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Label != "2026-W18" {
		t.Errorf("first label: got %q", buckets[0].Label)
	}
	if buckets[1].Label != "2026-W19" {
		t.Errorf("second label: got %q", buckets[1].Label)
	}
}

func TestBucketByGranularity_WeekStartingSunday(t *testing.T) {
	t.Parallel()
	// 2026-04-26 est dimanche -> doit appartenir a la semaine 17 (lundi 20 avril).
	rows := []fakeRow{
		{time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)},
	}
	buckets := BucketByGranularity(rows, GranWeek, PeriodAll)
	if len(buckets) != 1 {
		t.Fatalf("want 1 bucket, got %d", len(buckets))
	}
	if buckets[0].Label != "2026-W17" {
		t.Errorf("Sunday should map to ISO week of preceding Monday, got %q", buckets[0].Label)
	}
}

func TestBucketByGranularity_Month(t *testing.T) {
	t.Parallel()
	rows := []fakeRow{
		{time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)},
	}
	buckets := BucketByGranularity(rows, GranMonth, PeriodAll)
	if len(buckets) != 2 {
		t.Fatalf("want 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Label != "2026-03" {
		t.Errorf("first: got %q", buckets[0].Label)
	}
	if buckets[1].Label != "2026-04" {
		t.Errorf("second: got %q", buckets[1].Label)
	}
}

func TestBucketByGranularity_Adaptive(t *testing.T) {
	t.Parallel()
	rows := []fakeRow{{time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)}}
	cases := []struct {
		period Period
		want   string // attendu = label
	}{
		{Period1W, "2026-04-27"},
		{Period1M, "2026-04-27"},
		{Period1Y, "2026-W18"},
		{Period2Y, "2026-04"},
		{PeriodAll, "2026-04"},
	}
	for _, c := range cases {
		c := c
		t.Run(string(c.period), func(t *testing.T) {
			t.Parallel()
			b := BucketByGranularity(rows, GranAdaptive, c.period)
			if len(b) != 1 || b[0].Label != c.want {
				t.Errorf("period=%s want label=%q got %v", c.period, c.want, b)
			}
		})
	}
}

func TestBucketByGranularity_Sorted(t *testing.T) {
	t.Parallel()
	rows := []fakeRow{
		{time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)},
		{time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)},
	}
	buckets := BucketByGranularity(rows, GranDay, PeriodAll)
	if len(buckets) != 3 {
		t.Fatalf("want 3, got %d", len(buckets))
	}
	if !buckets[0].Start.Before(buckets[1].Start) || !buckets[1].Start.Before(buckets[2].Start) {
		t.Error("buckets not sorted by Start ascending")
	}
}

func TestBucketByGranularity_Empty(t *testing.T) {
	t.Parallel()
	rows := []fakeRow{}
	buckets := BucketByGranularity(rows, GranDay, PeriodAll)
	if len(buckets) != 0 {
		t.Errorf("expected empty, got %d", len(buckets))
	}
}

func TestBucketByGranularity_PreservesLocation(t *testing.T) {
	t.Parallel()
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("Europe/Paris timezone not available")
	}
	rows := []fakeRow{
		{time.Date(2026, 4, 27, 1, 0, 0, 0, paris)}, // 1h Paris = 23h UTC veille
	}
	buckets := BucketByGranularity(rows, GranDay, PeriodAll)
	if len(buckets) != 1 {
		t.Fatalf("want 1 bucket, got %d", len(buckets))
	}
	if buckets[0].Label != "2026-04-27" {
		t.Errorf("expected day in row's location (Paris), got %q", buckets[0].Label)
	}
}
