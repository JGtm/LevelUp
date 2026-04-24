// Package duckdb — pool_unit_test.go : tests unitaires de PlayerDB (sans accès DB).
// Pas de build tag intégration : teste uniquement la logique pure de ReadDB().
package duckdb

import "testing"

// TestReadDB_ReturnsPlayer vérifie que ReadDB retourne toujours Player.
func TestReadDB_ReturnsPlayer(t *testing.T) {
	sentinel := &DB{}
	pdb := &PlayerDB{Player: sentinel}
	if got := pdb.ReadDB(); got != sentinel {
		t.Errorf("ReadDB() doit retourner Player, obtenu %v", got)
	}
}
