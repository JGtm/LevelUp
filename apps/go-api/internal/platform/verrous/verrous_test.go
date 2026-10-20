package verrous

import (
	"sync"
	"testing"
)

// TestRegistreUnVerrouParCle — même clé, même verrou ; clés distinctes, verrous indépendants ;
// la valeur zéro est utilisable.
func TestRegistreUnVerrouParCle(t *testing.T) {
	var r Registre
	a, b := r.De("a"), r.De("b")
	if a == b {
		t.Fatal("deux clés distinctes partagent un verrou")
	}
	if r.De("a") != a {
		t.Fatal("la même clé rend deux verrous : l'exclusion tomberait")
	}
	a.Lock()
	defer a.Unlock()
	if !b.TryLock() {
		t.Fatal("le verrou d'une clé bloque celui d'une autre")
	}
	b.Unlock()
	if r.De("a").TryLock() {
		t.Fatal("le verrou tenu de « a » a été repris")
	}
}

// TestRegistreSurEnConcurrence — cent appelants simultanés sur la même clé obtiennent tous le
// même verrou (go test -race le vérifie en plus quand il est disponible).
func TestRegistreSurEnConcurrence(t *testing.T) {
	var r Registre
	const appelants = 100
	vus := make([]*sync.Mutex, appelants)
	var wg sync.WaitGroup
	for i := range appelants {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vus[i] = r.De("titre")
		}()
	}
	wg.Wait()
	for i := 1; i < appelants; i++ {
		if vus[i] != vus[0] {
			t.Fatalf("appelant %d : verrou différent du premier", i)
		}
	}
}
