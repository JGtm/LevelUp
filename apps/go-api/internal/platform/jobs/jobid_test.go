// Package jobs — jobid_test.go : garde-rail sur newJobID (V3, sécurité).
//
// L'ID de job ne doit plus être énumérable (ancien format horodaté `job_<UnixNano>`).
// White-box (package jobs) car newJobID est non exporté.
package jobs

import (
	"regexp"
	"testing"
)

// jobIDPattern : préfixe `job_`, date YYYYMMDD lisible, puis 32 hex (16 octets
// crypto-aléatoires). Le fallback timestamp (crypto/rand en échec, jamais atteint
// en pratique) N'EST PAS couvert par ce motif : le test vérifie le chemin nominal.
var jobIDPattern = regexp.MustCompile(`^job_\d{8}_[0-9a-f]{32}$`)

// TestNewJobID_Format : le chemin nominal respecte le format attendu.
func TestNewJobID_Format(t *testing.T) {
	id := newJobID()
	if !jobIDPattern.MatchString(id) {
		t.Fatalf("newJobID = %q, ne correspond pas à %s", id, jobIDPattern.String())
	}
}

// TestNewJobID_UniqueNoCollision : 10 000 générations, 0 collision, 0 doublon.
// L'ancien format UnixNano pouvait collisionner à haute cadence (même ns dans une
// boucle serrée) ET était énumérable ; le suffixe 128 bits crypto rend les deux
// improbables/impossibles.
func TestNewJobID_UniqueNoCollision(t *testing.T) {
	const n = 10000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newJobID()
		if !jobIDPattern.MatchString(id) {
			t.Fatalf("itération %d : ID mal formé %q", i, id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("itération %d : collision sur %q", i, id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("attendu %d IDs uniques, obtenu %d", n, len(seen))
	}
}
