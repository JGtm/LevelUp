package dblease

import "time"

// LeakReporter est l'interface minimale acceptée par AssertNoLeasedWriters.
// *testing.T (et *testing.B) la satisfont nativement, mais elle reste assez
// restreinte pour être implémentable par un fake dans les tests du helper
// lui-même (sans avoir à reproduire toute testing.TB qui évolue avec les
// versions de Go).
type LeakReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}

// AssertNoLeasedWriters échoue le test si au moins un *LeasedWriter est encore
// tenu au moment de l'appel. À utiliser via t.Cleanup() dans tout test qui
// touche un writer (prestige, notifications, social, media, sync).
//
// Tolère un délai court (50 ms) pour absorber les Release() asynchrones via
// defer dans des goroutines fraîchement terminées.
//
// Usage :
//
//	func TestSomeWrite(t *testing.T) {
//	    t.Cleanup(func() { dblease.AssertNoLeasedWriters(t) })
//	    // ... corps du test ...
//	}
func AssertNoLeasedWriters(t LeakReporter) {
	t.Helper()
	deadline := time.Now().Add(50 * time.Millisecond)
	for {
		n := LeasedWritersInUse()
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("dblease: %d writer(s) still in use at end of test (leak)", n)
			return // au cas où le LeakReporter ne panic pas (tests du helper).
		}
		time.Sleep(2 * time.Millisecond)
	}
}
