package observability

import (
	"sync"
	"testing"
	"time"
)

// TestAllowThrottledLog_FirstOccurrenceAlwaysEmitted vérifie le contrat DC-B2 :
// la toute première occurrence d'une clé est toujours autorisée.
func TestAllowThrottledLog_FirstOccurrenceAlwaysEmitted(t *testing.T) {
	Reset()
	ResetThrottle()

	allow, suppressed := AllowThrottledLog("log_throttle_test_first", time.Minute)
	if !allow {
		t.Fatal("la première occurrence doit être émise")
	}
	if suppressed != 0 {
		t.Fatalf("aucune occurrence étouffée avant la première, got %d", suppressed)
	}
}

// TestAllowThrottledLog_SuppressesWithinWindow vérifie qu'un flood dans la
// fenêtre est étouffé tout en étant compté exactement (expvar), puis rapporté à
// l'émission suivante.
func TestAllowThrottledLog_SuppressesWithinWindow(t *testing.T) {
	Reset()
	ResetThrottle()

	fake := time.Unix(1_700_000_000, 0)
	throttleNow = func() time.Time { return fake }
	defer func() { throttleNow = time.Now }()

	const key = "log_throttle_test_window"
	window := 30 * time.Second

	// 1re occurrence : émise.
	if allow, _ := AllowThrottledLog(key, window); !allow {
		t.Fatal("1re occurrence doit être émise")
	}
	// 9 occurrences dans la même fenêtre : toutes étouffées.
	for i := 0; i < 9; i++ {
		if allow, _ := AllowThrottledLog(key, window); allow {
			t.Fatalf("occurrence #%d dans la fenêtre ne doit pas être émise", i+2)
		}
	}
	// Compteur expvar exact = 10 occurrences vues.
	if got := LoadCounter(key + "_total"); got != 10 {
		t.Fatalf("compteur total = %d, attendu 10", got)
	}

	// La fenêtre s'écoule : la prochaine occurrence est émise et rapporte les 9
	// étouffées.
	fake = fake.Add(31 * time.Second)
	allow, suppressed := AllowThrottledLog(key, window)
	if !allow {
		t.Fatal("après la fenêtre, l'occurrence doit être émise")
	}
	if suppressed != 9 {
		t.Fatalf("throttledSinceLast = %d, attendu 9", suppressed)
	}
	if got := LoadCounter(key + "_total"); got != 11 {
		t.Fatalf("compteur total = %d, attendu 11", got)
	}
}

// TestAllowThrottledLog_ConcurrentSingleEmit vérifie qu'un flood concurrent sous
// une même clé n'émet qu'une seule ligne mais compte toutes les occurrences.
func TestAllowThrottledLog_ConcurrentSingleEmit(t *testing.T) {
	Reset()
	ResetThrottle()

	fake := time.Unix(1_700_000_000, 0)
	throttleNow = func() time.Time { return fake }
	defer func() { throttleNow = time.Now }()

	const key = "log_throttle_test_concurrent"
	const n = 500
	var emitted int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if allow, _ := AllowThrottledLog(key, time.Minute); allow {
				mu.Lock()
				emitted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if emitted != 1 {
		t.Fatalf("émissions concurrentes = %d, attendu exactement 1", emitted)
	}
	if got := LoadCounter(key + "_total"); got != n {
		t.Fatalf("compteur total = %d, attendu %d (aucune occurrence avalée)", got, n)
	}
}
