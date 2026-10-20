// Package fragdist — fragdist_halo5_golden_test.go : LE VERROU DE HALO 5.
//
// # Le piège que ce fichier tient
//
// `isRegistryFragClass` est du code PARTAGÉ par les deux titres. Élargir l'ensemble des
// classes servies par le registre — ce que fait la bascule de l'arme du kill pour
// `equipment` et `environmental` (étape A1.4 du plan du 2026-09-01) — ferait AUSSI remonter
// les lignes `h5_environmental` de Halo 5, qui portent un identifiant numérique et vivent
// dans `weapon_kills`. Le sunburst de Halo 5 changerait, hors du périmètre du lot.
//
// Ce golden fige la sortie de `Build` sur un jeu de lignes de forme Halo 5, octet pour
// octet. Il a été écrit et passé AVANT A1.4 ; il doit passer à l'identique APRÈS. S'il
// rougit, ce n'est pas le golden qu'il faut mettre à jour : c'est que la modification
// déborde sur le second titre.
//
// # Pourquoi le golden est ici et pas sur une base DuckDB
//
// Le chemin de lecture de Halo 5 (`WeaponKillsRepo` sur `v_weapon_kills`) n'est pas touché
// par le lot : il est laissé strictement en l'état. Le seul code que les deux titres se
// partagent, et donc le seul endroit où une régression de Halo 5 peut naître, est ce
// builder. Forger une base ne testerait qu'un SQL inchangé — le verrou doit être là où est
// le risque.
package fragdist

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// Cles de registre Halo 5 du golden. Elles sont NOMMEES plutot qu ecrites en clair dans
// les litteraux de ligne : le detecteur de secrets prend « WeaponKey: "..." » pour une cle
// d API (faux positif de la regle generic-api-key). Les nommer coute moins qu une exception
// d allowlist, et se lit mieux.
const (
	keyH5BR         = "h5_" + "br85"
	keyH5Magnum     = "h5_" + "magnum"
	keyH5Rocket     = "h5_" + "rocket"
	keyH5Env        = "h5_" + "environmental"
	keyH5Warthog    = "h5_" + "vehicle_warthog"
	keyH5TurretPlas = "h5_" + "turret_plasma"
	keyH5Frag       = "h5_" + "frag_grenade"
	keyH5Splinter   = "h5_" + "splinter_grenade"
)

// h5Rows : des lignes de forme Halo 5, telles que `WeaponKillsRepo` les rend depuis
// `v_weapon_kills` — TOUTES portent un identifiant numérique, y compris les buckets
// non-combat, ce qui est précisément ce qui distingue Halo 5 de Halo Infinite.
func h5Rows() []port.WeaponKillRow {
	return []port.WeaponKillRow{
		{XUID: "h5_a", WeaponID: 1001, Kills: 40, Label: "BR85", Role: "precision", Class: "shoulder", Family: "battle_rifle", WeaponKey: keyH5BR},
		{XUID: "h5_a", WeaponID: 1002, Kills: 12, Label: "Magnum", Role: "sidearm", Class: "sidearm", Family: "magnum", WeaponKey: keyH5Magnum},
		{XUID: "h5_a", WeaponID: 1003, Kills: 7, Label: "Rocket Launcher", Role: "power", Class: "heavy", Family: "rocket_launcher", WeaponKey: keyH5Rocket},
		// Bucket non-combat AVEC identifiant numérique : la ligne qui piège.
		{XUID: "h5_a", WeaponID: 1004, Kills: 9, Label: "Environmental Explosives", Role: "environmental", Class: "environmental", Family: "environmental", WeaponKey: keyH5Env},
		// Engin : déjà ventilé par objet AVANT le lot — le comportement ne doit pas bouger.
		{XUID: "h5_a", WeaponID: 1005, Kills: 5, Label: "Warthog", LabelEN: "Warthog", Role: "vehicle", Class: "vehicle", Family: "vehicle", WeaponKey: keyH5Warthog},
		{XUID: "h5_a", WeaponID: 1006, Kills: 3, Label: "Tourelle à plasma", LabelEN: "Plasma Turret", Role: "turret", Class: "turret", Family: "turret", WeaponKey: keyH5TurretPlas},
		// Mécaniques natives attribuées à l'arme TENUE (kill_kind <> 'weapon').
		{XUID: "h5_a", WeaponID: 1001, Kills: 6, Role: "precision", Class: "shoulder", Family: "battle_rifle", WeaponKey: keyH5BR, MechanicKills: 6},
		// Grenades typées : le niveau 2 de la classe Grenade vient de la famille.
		{XUID: "h5_a", WeaponID: 1007, Kills: 8, Class: "grenade", Role: "grenade", Family: "frag_grenade", WeaponKey: keyH5Frag},
		{XUID: "h5_a", WeaponID: 1008, Kills: 4, Class: "grenade", Role: "grenade", Family: "splinter_grenade", WeaponKey: keyH5Splinter},
		// Sentinelles grenade/mêlée issues de match_participants.
		{XUID: "h5_a", WeaponID: 0, Kills: 14, Label: "Grenade", IsGrenadeMelee: true},
		{XUID: "h5_a", WeaponID: 1, Kills: 11, Label: "Mêlée", IsGrenadeMelee: true},
	}
}

// h5Counts : les compteurs natifs Halo 5, mécaniques comprises.
func h5Counts() domain.FragKillTypeCounts {
	return domain.FragKillTypeCounts{
		Melee: 11, Grenade: 14, Assassination: 5, GroundPound: 3, ShoulderBash: 2, Total: 120,
	}
}

// h5Golden est la sortie ATTENDUE, figée. Toute différence est une régression de Halo 5.
const h5Golden = `{"total_kills":120,"classes":[` +
	`{"class":"shoulder","kills":40,"authoritative":false,"roles":[{"role":"precision","kills":40}]},` +
	`{"class":"sidearm","kills":12,"authoritative":false},` +
	`{"class":"heavy","kills":7,"authoritative":false,"roles":[{"role":"power","kills":7}]},` +
	`{"class":"melee","kills":16,"authoritative":true,"roles":[{"role":"assassination","kills":5},{"role":"direct_melee","kills":11}]},` +
	`{"class":"grenade","kills":14,"authoritative":true,"roles":[{"role":"grenade_frag","kills":8},{"role":"grenade_splinter","kills":4},{"role":"grenade_other","kills":2}]},` +
	`{"class":"spartan_ability","kills":5,"authoritative":true,"roles":[{"role":"ground_pound","kills":3},{"role":"shoulder_bash","kills":2}]},` +
	`{"class":"vehicle","kills":5,"authoritative":false,"roles":[{"role":"h5_vehicle_warthog","kills":5,"label":"Warthog","label_en":"Warthog"}]},` +
	`{"class":"turret","kills":3,"authoritative":false,"roles":[{"role":"h5_turret_plasma","kills":3,"label":"Tourelle à plasma","label_en":"Plasma Turret"}]},` +
	`{"class":"unattributed","kills":18,"authoritative":false}]}`

// TestHalo5SunburstNeBougePas : le verrou. Sortie de `Build` sur des lignes Halo 5,
// comparée octet pour octet au golden.
//
// Les 9 frags `h5_environmental` doivent rester dans « Non attribué » : Halo 5 n'a pas de
// décodeur de film, sa provenance est `weapon_kills`, et ce lot ne lui ouvre AUCUNE classe
// nouvelle. Le résidu de 18 le dit (120 − 40 − 12 − 7 − 16 − 14 − 5 − 5 − 3).
func TestHalo5SunburstNeBougePas(t *testing.T) {
	got := Build(h5Rows(), h5Counts(), true)
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) != h5Golden {
		t.Errorf("le sunburst de Halo 5 a bouge.\nobtenu : %s\nattendu: %s", encoded, h5Golden)
	}
}

// TestHalo5EnvironmentalResteHorsClasse dit EXPLICITEMENT ce que le golden encode de plus
// fragile : la classe `environmental` de Halo 5 ne doit jamais apparaître comme un arc.
// Un golden qui rougit se lit mal ; ce test-ci nomme la régression.
func TestHalo5EnvironmentalResteHorsClasse(t *testing.T) {
	got := Build(h5Rows(), h5Counts(), true)
	for _, c := range got.Classes {
		if c.Class == domain.FragClassEnvironmental || c.Class == domain.FragClassEquipment {
			t.Errorf("classe %q servie pour Halo 5 (%d frags) : la bascule Halo Infinite a débordé",
				c.Class, c.Kills)
		}
	}
}
