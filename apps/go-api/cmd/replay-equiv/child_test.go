package main

// child_test.go — CE QUE LE COLLECTEUR RETIENT, ET CE QU'IL EN JOURNALISE.
//
// Le collecteur ne decode rien : il hache ce qu'on lui donne et garde de quoi armer la garde
// anti-equivalence-vacuante. Les deux se testent sans film.

import (
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// TestCollecteurRetientLaValeurDesEtapesChaine — LE COMPTE NE DIT RIEN D'UNE CHAINE :
// `digest.Of("not_established")` rend 15, sa LONGUEUR. C'est la valeur qu'il faut retenir pour
// que le WARN des socles nomme l'etat au lieu d'un nombre d'octets.
func TestCollecteurRetientLaValeurDesEtapesChaine(t *testing.T) {
	var c collecteur
	c.etape("spawnPoints", []replay.MapSpawnPoint{})
	c.etape("spawnPointsState", replay.SpawnPointsNotEstablished)

	if got := c.etats["spawnPointsState"]; got != replay.SpawnPointsNotEstablished {
		t.Errorf("etat retenu %q, attendu %q", got, replay.SpawnPointsNotEstablished)
	}
	if got := c.comptes["spawnPointsState"]; got != len(replay.SpawnPointsNotEstablished) {
		t.Errorf("compte d'une chaine = sa longueur (%d), obtenu %d",
			len(replay.SpawnPointsNotEstablished), got)
	}
	if _, porte := c.etats["spawnPoints"]; porte {
		t.Error("une etape qui ne rend pas une chaine ne doit pas entrer dans les etats")
	}
	if n := len(c.lignes); n != 2 {
		t.Errorf("une ligne de digest par etape : 2 attendues, %d obtenues", n)
	}
}
