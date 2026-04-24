// Package duckdb — pool_unit_test.go : tests unitaires de PlayerDB (sans accès DB).
// Pas de build tag intégration : teste uniquement la logique pure de ReadDB().
package duckdb

import "testing"

// TestReadDB_FallbackToPlayer vérifie que ReadDB retourne Player quand PlayerRO est nil.
func TestReadDB_FallbackToPlayer(t *testing.T) {
	sentinel := &DB{}
	pdb := &PlayerDB{Player: sentinel, PlayerRO: nil}
	if got := pdb.ReadDB(); got != sentinel {
		t.Errorf("ReadDB() avec PlayerRO=nil doit retourner Player, obtenu %v", got)
	}
}

// TestReadDB_PrefersPlayerRO vérifie que ReadDB retourne PlayerRO quand il est non-nil.
func TestReadDB_PrefersPlayerRO(t *testing.T) {
	player := &DB{}
	playerRO := &DB{}
	pdb := &PlayerDB{Player: player, PlayerRO: playerRO}
	if got := pdb.ReadDB(); got != playerRO {
		t.Errorf("ReadDB() avec PlayerRO non-nil doit retourner PlayerRO, obtenu %v", got)
	}
}
