package observability

import (
	"sync"
	"testing"
)

func TestIncCounter_FirstAndSubsequent(t *testing.T) {
	Reset()
	IncCounter("test_counter_inc")
	if got := LoadCounter("test_counter_inc"); got != 1 {
		t.Errorf("after 1 inc, want 1, got %d", got)
	}
	IncCounter("test_counter_inc")
	IncCounter("test_counter_inc")
	if got := LoadCounter("test_counter_inc"); got != 3 {
		t.Errorf("after 3 inc, want 3, got %d", got)
	}
}

func TestAddInt_NegativeDelta(t *testing.T) {
	Reset()
	AddInt("test_counter_addint", 10)
	AddInt("test_counter_addint", -3)
	if got := LoadCounter("test_counter_addint"); got != 7 {
		t.Errorf("want 7, got %d", got)
	}
}

func TestLoadCounter_AbsentReturnsZero(t *testing.T) {
	Reset()
	if got := LoadCounter("never_set"); got != 0 {
		t.Errorf("absent counter should return 0, got %d", got)
	}
}

func TestIncCounter_Concurrent(t *testing.T) {
	Reset()
	const N = 1000
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			IncCounter("test_counter_concurrent")
		}()
	}
	wg.Wait()
	if got := LoadCounter("test_counter_concurrent"); got != N {
		t.Errorf("concurrent inc: want %d, got %d", N, got)
	}
}

func TestRecordDurationMS_BasicAggregation(t *testing.T) {
	Reset()
	RecordDurationMS("test_duration_basic", 100)
	RecordDurationMS("test_duration_basic", 200)
	RecordDurationMS("test_duration_basic", 50)

	count, sum, avg, max := LoadDurationStats("test_duration_basic")
	if count != 3 {
		t.Errorf("count: want 3, got %d", count)
	}
	if sum != 350 {
		t.Errorf("sum: want 350, got %d", sum)
	}
	if avg != 116 { // 350 / 3 = 116 (integer division)
		t.Errorf("avg: want 116, got %d", avg)
	}
	if max != 200 {
		t.Errorf("max: want 200, got %d", max)
	}
}

func TestRecordDurationMS_NegativeIgnored(t *testing.T) {
	Reset()
	RecordDurationMS("test_duration_neg", -50) // ignore
	RecordDurationMS("test_duration_neg", 100)

	count, _, _, max := LoadDurationStats("test_duration_neg")
	if count != 1 {
		t.Errorf("count: want 1 (negative ignored), got %d", count)
	}
	if max != 100 {
		t.Errorf("max: want 100, got %d", max)
	}
}

func TestRecordDurationMS_AbsentReturnsZeros(t *testing.T) {
	Reset()
	count, sum, avg, max := LoadDurationStats("never_recorded")
	if count != 0 || sum != 0 || avg != 0 || max != 0 {
		t.Errorf("absent metric should be all zero, got count=%d sum=%d avg=%d max=%d",
			count, sum, avg, max)
	}
}

func TestRecordDurationMS_Concurrent(t *testing.T) {
	Reset()
	const N = 1000
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			RecordDurationMS("test_duration_concurrent", int64(i))
		}()
	}
	wg.Wait()

	count, sum, _, max := LoadDurationStats("test_duration_concurrent")
	if count != N {
		t.Errorf("count: want %d, got %d", N, count)
	}
	expectedSum := int64(N) * int64(N-1) / 2 // 0..N-1
	if sum != expectedSum {
		t.Errorf("sum: want %d, got %d", expectedSum, sum)
	}
	if max != int64(N-1) {
		t.Errorf("max: want %d, got %d", N-1, max)
	}
}

func TestReset_ClearsAllMetrics(t *testing.T) {
	IncCounter("test_reset_counter")
	RecordDurationMS("test_reset_duration", 100)
	Reset()
	if got := LoadCounter("test_reset_counter"); got != 0 {
		t.Errorf("counter should be 0 after Reset, got %d", got)
	}
	count, _, _, _ := LoadDurationStats("test_reset_duration")
	if count != 0 {
		t.Errorf("duration count should be 0 after Reset, got %d", count)
	}
}
