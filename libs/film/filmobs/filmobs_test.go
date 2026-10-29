package filmobs

import (
	"sync"
	"testing"
)

func TestCompteLocalEtSnapshot(t *testing.T) {
	const nom = "filmobs_test_local"
	if got := Load(nom); got != 0 {
		t.Fatalf("compteur absent = %d, attendu 0", got)
	}
	IncCounter(nom)
	AddInt(nom, 4)
	AddInt(nom, -2)
	if got := Load(nom); got != 3 {
		t.Fatalf("Load = %d, attendu 3", got)
	}
	snap := Snapshot()
	if snap[nom] != 3 {
		t.Fatalf("Snapshot[%q] = %d, attendu 3", nom, snap[nom])
	}
	snap[nom] = 99
	if got := Load(nom); got != 3 {
		t.Fatalf("modifier la copie a touché le compteur : %d", got)
	}
}

func TestPuitsRecoitChaqueIncrement(t *testing.T) {
	const nom = "filmobs_test_puits"
	var mu sync.Mutex
	recus := map[string]int64{}
	SetSink(func(n string, d int64) {
		mu.Lock()
		defer mu.Unlock()
		recus[n] += d
	})
	t.Cleanup(func() { SetSink(nil) })

	IncCounter(nom)
	AddInt(nom, 5)
	mu.Lock()
	got := recus[nom]
	mu.Unlock()
	if got != 6 {
		t.Fatalf("puits a reçu %d, attendu 6", got)
	}
	if Load(nom) != 6 {
		t.Fatalf("le compte local doit rester tenu avec un puits : %d", Load(nom))
	}

	SetSink(nil)
	AddInt(nom, 1)
	mu.Lock()
	got = recus[nom]
	mu.Unlock()
	if got != 6 {
		t.Fatalf("puits retiré mais appelé encore : %d", got)
	}
}

func TestConcurrence(t *testing.T) {
	const nom = "filmobs_test_concurrence"
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				IncCounter(nom)
			}
		}()
	}
	wg.Wait()
	if got := Load(nom); got != 3200 {
		t.Fatalf("Load = %d, attendu 3200", got)
	}
}
