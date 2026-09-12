package mapvar

// vehicules_noms_test.go — CRAQUAGE DES NOMS DE LA FAMILLE `vehi` DE LA PALETTE FORGE.
//
// La palette Forge expose 21 `type_id` de groupe de tag `vehi`
// (`.ai/V7.5/dumps/forge_zones/palette_complete_groupes.csv`) ; 15 portent deja un nom craque
// (`palette_noms.csv`). Les 6 restants sont des hachages murmur3 nus. Ce test rejoue
// [LabelHash] sur un vivier de noms de vehicules Halo — la MEME methode que le craquage des
// libelles de zone — et imprime les correspondances.
//
// Le vivier vient de trois sources, aucune inventee : les noms deja craques de la palette (qui
// donnent la CONVENTION d'ecriture : minuscules, souligne, prefixe de faction pour certains),
// les identifiants de vehicule du binaire (`MP_VEHICLE_IDENTIFIER`), et les variantes connues
// du jeu.

import (
	"fmt"
	"testing"
)

// vehiInconnus : les 6 hachages de type `vehi` sans nom craque, releves le 2026-08-31.
var vehiInconnus = map[int32]string{
	1029649325:  "-1751154772 — 20 instances / 10 cartes, le plus pose des inconnus",
	-1773333388: "-1362694062",
	-313705621:  "-105823600 — emprise identique au Warthog (2,244 x 1,014 x 0,832 wu)",
	-2002047233: "199265464",
	-303468340:  "2128426546",
	1161655938:  "-1430390016 — variante de caisse 81f1fc67, comme shade_turret",
}

// vehiVivier : les candidats. Convention lue sur les 15 noms deja craques (`shade_turret`,
// `brute_chopper`, `warthog_gauss`, `unsc_turret`, `auto_turret`, `plasma_turret`...).
var vehiVivier = []string{
	// Variantes de Warthog
	"warthog_razorback", "razorback", "warthog_rocket", "rockethog", "warthog_gauss_turret",
	"warthog_troop", "warthog_transport", "warthog_flatbed", "gausshog", "warthog_m12",
	// Mongoose et derives
	"gungoose", "mongoose_gungoose", "mongoose_gun", "goose", "mongoose_turret",
	// Vehicules de la banque MP
	"komodo", "falcon_unsc", "hornet", "sparrowhawk", "pelican", "condor",
	"chopper", "elite_ghost", "banshee_elite", "wraith_anti_air", "aa_wraith",
	"revenant", "spectre", "prowler", "locust", "seraph", "lich", "harvester",
	// Tourelles
	"turret", "unsc_turret_gun", "covenant_turret", "banished_turret", "grunt_turret",
	"shade", "machine_gun_turret", "gun_turret", "stationary_turret", "aa_turret",
	// Formes generiques et prefixes de faction
	"vehicle", "generic_vehicle", "banished_chopper", "banished_ghost", "banished_wraith",
	"unsc_warthog", "unsc_mongoose", "unsc_scorpion", "unsc_wasp", "unsc_falcon",
	"forerunner_vehicle", "phaeton", "wasp_unsc", "scorpion_tank", "tank",
	// Deux formes attestees ailleurs dans le depot
	"warthog_razorback_transport", "mongoose_gungoose_turret",
	// Seconde passe : les deux vehicules du roster encore absents (Rockethog, Komodo) et les
	// prefixes de faction attestes par `brute_chopper`.
	"warthog_rockets", "warthog_missile", "warthog_rocket_launcher", "warthog_at",
	"komodo", "wasp_komodo", "komodo_tank", "banished_komodo", "wasp_banished",
	"brute_ghost", "brute_banshee", "brute_wraith", "brute_scorpion", "brute_turret",
	"banished_banshee", "banished_wasp", "banished_scorpion", "banished_shade",
	"machinegun_turret", "unsc_machine_gun", "unsc_gun_turret", "unsc_aa_turret",
	"vehicle_spawner", "vehicle_pad", "vehicle_generic", "generic_vehi",
	"cov_turret", "covenant_shade", "grunt_shade", "elite_banshee", "elite_wraith",
	"mongoose_m274", "warthog_m12g", "warthog_m12r", "scorpion_m808",
	"wasp_aa", "falcon_uh144", "hornet_unsc", "chopper_brute", "ghost_brute",
	"turret_shade", "turret_plasma", "turret_auto", "turret_unsc",
	"rocket_warthog", "gauss_warthog", "razorback_warthog", "gungoose_mongoose",
}

// TestVehiculesNomsCraques imprime les correspondances trouvees.
func TestVehiculesNomsCraques(t *testing.T) {
	trouves := map[int32]string{}
	for _, nom := range vehiVivier {
		h := LabelHash(nom)
		if quoi, ok := vehiInconnus[h]; ok {
			trouves[h] = fmt.Sprintf("%s = %q (type_id %s)", "hash", nom, quoi)
			t.Logf("CRAQUE : hash %d = %q  [%s]", h, nom, quoi)
		}
	}
	// Controle : les 15 noms deja craques doivent redonner leur hachage, sinon la fonction de
	// hachage n'est pas celle de la palette et le negatif ne vaudrait rien.
	controle := map[string]int32{
		"shade_turret": 1028518636, "banshee": 419783896, "mongoose": 1063919886,
		"wasp": -1087066335, "unsc_turret": 2072906399, "warthog": -266450505,
		"wraith": 1206711506, "brute_chopper": 1403109065, "auto_turret": 1455922937,
		"ghost": -1284820930, "warthog_gauss": 1296991821, "plasma_turret": -262278708,
		"scorpion": 1730553442, "phantom": 1977724336, "falcon": 1953167207,
	}
	for nom, attendu := range controle {
		if got := LabelHash(nom); got != attendu {
			t.Fatalf("CONTROLE ROMPU : LabelHash(%q) = %d, attendu %d — le vivier ne prouverait rien",
				nom, got, attendu)
		}
	}
	t.Logf("controle : 15/15 noms deja craques redonnent leur hachage")
	t.Logf("%d hachage(s) craque(s) sur %d inconnus, vivier de %d noms",
		len(trouves), len(vehiInconnus), len(vehiVivier))
}
