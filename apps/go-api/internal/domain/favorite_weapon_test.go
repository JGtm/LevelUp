package domain

import "testing"

// TestIsFavoriteWeaponCandidate — L ARME FAVORITE EST UNE ARME DE L ARSENAL (revue adverse du lot
// M6 des retours du rejeu, constat R1, 2026-09-24). La bobine a fusion UNSC porte depuis ce lot les
// identifiants de l objet tenu : son identifiant numerique n est plus nul, et c est la CLASSE, pas
// l absence d id, qui doit l ecarter. Une ligne qui n est PAS mesuree dans le film (Halo 5,
// `weapon_kills`) garde la regle d avant (id non nul), sans quoi le scoreboard du second titre
// bougerait.
func TestIsFavoriteWeaponCandidate(t *testing.T) {
	cas := []struct {
		nom      string
		id       int64
		class    string
		film     bool
		attendue bool
	}{
		{"arme de l arsenal mesuree", 42, FragClassShoulder, true, true},
		{"bobine a fusion : id de l objet tenu, classe environnement", 42, FragClassEnvironmental, true, false},
		{"equipement mesure", 42, FragClassEquipment, true, false},
		{"vehicule mesure, s il recevait un id", 42, FragClassVehicle, true, false},
		{"tourelle mesuree, s il recevait un id", 42, FragClassTurret, true, false},
		{"cle sans identifiant", 0, FragClassShoulder, true, false},
		{"Halo 5, bucket non attribue a id : inchange", 42, FragClassUnattributed, false, true},
		{"Halo 5, sans id : inchange", 0, FragClassShoulder, false, false},
	}
	for _, c := range cas {
		if got := IsFavoriteWeaponCandidate(c.id, c.class, c.film); got != c.attendue {
			t.Errorf("%s : %v, attendu %v", c.nom, got, c.attendue)
		}
	}
}
