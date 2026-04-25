// Package duckdb — pool_unit_test.go : tests unitaires de PlayerDB (sans accès DB).
// Pas de build tag intégration : teste uniquement la logique pure de ReadDB().
package duckdb

import "testing"

// TestReadDB_ReturnsPlayer vérifie que ReadDB retourne la connexion Player (RW unique).
func TestReadDB_ReturnsPlayer(t *testing.T) {
	sentinel := &DB{}
	pdb := &PlayerDB{Player: sentinel}
	if got := pdb.ReadDB(); got != sentinel {
		t.Errorf("ReadDB() doit retourner Player, obtenu %v", got)
	}
}

// TestReadDB_NilPlayer vérifie que ReadDB retourne nil quand Player est nil.
func TestReadDB_NilPlayer(t *testing.T) {
	pdb := &PlayerDB{}
	if got := pdb.ReadDB(); got != nil {
		t.Errorf("ReadDB() avec Player nil doit retourner nil, obtenu %v", got)
	}
}
