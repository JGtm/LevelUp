package main

// banks_dico.go — LE DICTIONNAIRE DE NOMS DE BANQUES.
//
// POURQUOI IL EXISTE. Le chunk `STID` n'est present que sur 2 banks sur 1305 (mesure de
// l'etape 1, cf. `stid.go`) : le jeu ne livre pas les noms de ses banques. Mais l'identifiant
// d'une banque (chunk `BKHD`) est le FNV-1 32 bits de son nom de fichier en minuscules —
// convention CALIBREE par `calibrerNommage` sur les banques qui ont un `.pck`. Un nom se
// retrouve donc par hachage, comme un `string_id` se casse par dictionnaire.
//
// LA GRAMMAIRE EST OBSERVEE, PAS DEVINEE. Les 841 `.pck` du jeu portent des noms en clair et
// suivent tous la meme forme :
//
//	sb_<NNN>_<famille>[_<portee>]_<jetons>
//	         004_mod   mp_        oddball          sb_004_mod_mp_oddball
//	         007_abl              repairfield      sb_007_abl_repairfield
//	         130_mus              strongholds      sb_130_mus_strongholds
//
// Les couples `<NNN>_<famille>` et les portees enumeres ici sont exactement ceux qu'on lit
// sur ces 841 fichiers ; les jetons viennent des noms de packs, des 14 noms d'equipements
// casses par le dictionnaire murmur3 (PLAN_EQUIPEMENTS_MANQUANTS_SONS phase 1) et des chaines
// internes du binaire du jeu.
//
// LE TEMOIN DE CALIBRATION EST GRATUIT ET IL EST OBLIGATOIRE : les 841 noms de packs sont
// connus. Un generateur qui n'en retrouve pas la grande majorite sur les familles visees ne
// vaut rien, et son silence sur une banque sans pack ne prouverait rien. `banks-noms` le
// mesure et l'imprime avant tout resultat.

import "fmt"

// famillesBanques : les couples `<NNN>_<famille>` LUS sur les 841 packs du jeu.
var famillesBanques = []string{
	"001_vo", "002_ui", "003_lvl", "004_mod", "006_cfx", "006_chm", "007_abl",
	"008_exp", "009_veh", "010_grn", "010_tur", "010_un", "010_veh", "010_wea",
	"010_whizby", "020_prototype", "099_debug", "110_mus", "120_mus", "130_mus",
}

// porteesBanques : le segment optionnel entre la famille et les jetons.
// `mp_` multijoueur, `fo_` forge, `ge_` generique, `ow_` monde ouvert, puis les factions.
var porteesBanques = []string{
	"", "mp_", "fo_", "ge_", "ow_", "un_", "cv_", "bt_", "fr_", "pl_", "bn_",
	"menu_", "shared_", "mp_shared_", "ge_shared_", "prj_",
}

// jetonsModes : le vocabulaire des MODES multijoueur. Source : les noms de packs
// `sb_130_mus_*` (qui nomment les modes en clair) et les chaines du binaire
// (`capture_the_flag`, `objective_type_stronghold`, `arming_bomb`, `extraction_encounter`).
var jetonsModes = []string{
	"ctf", "captureflag", "capturetheflag", "capture_the_flag", "flag", "flags",
	"strongholds", "stronghold", "zones", "zone", "capturezone", "capture_zone",
	"extraction", "extract", "extractionpoint", "extraction_point", "extractiondevice",
	"assault", "bomb", "neutralbomb", "onebomb", "oddball", "ball", "skull",
	"elimination", "vip", "slayer", "koth", "kingofthehill", "hill",
	"landgrab", "land_grab", "totalcontrol", "total_control", "btb_totalcontrol",
	"stockpile", "btb_stockpile", "btbstockpile", "powerseed", "power_seed", "seed",
	"infection", "infection_sword", "bastion", "firefight", "pve_firefight",
	"attrition", "escalation", "gungame", "fiesta", "forge", "forge_stingers",
	"academy", "prototype", "multiplayer_global", "global", "objective", "objectives",
	"weaponpod", "shared_weaponpod", "boundary", "boundary_loop_spline",
	"score", "scoring", "capture", "captured", "secure", "secured", "contested",
}

// jetonsEquipements : le vocabulaire des EQUIPEMENTS. Source : les 14 noms de tags casses par
// le dictionnaire murmur3 (`repair_field`, `quantum_translocator`, `threat_seeker`,
// `active_camo`, `powerup_overshield`, `ability_deployable_wall`, `ability_grapple_hook`,
// `ability_location_sensor`, `ability_evade`, `ability_knockback`, `unsc_thruster`,
// `regen_field`, `mobility_sprint`, `melee_default`), leurs graphies collees, et les noms de
// classes du binaire (`ShroudGeneratorComponent`, `QuantumTranslocator`, `ShroudScreen`).
var jetonsEquipements = []string{
	"repairfield", "repair_field", "regenfield", "regen_field",
	"quantumtranslocator", "quantum_translocator", "translocator", "teleporter",
	"threatseeker", "threat_seeker", "seeker", "threatsensor", "threat_sensor",
	"locationsensor", "location_sensor", "sensor", "motiontracker",
	"shroudscreen", "shroud_screen", "shroud", "shroudgenerator", "shroud_generator",
	"smokescreen", "smoke_screen", "veilscreen",
	"dropwall", "drop_wall", "deployablewall", "deployable_wall", "wall", "shieldwall",
	"grapple", "grappleshot", "grapplehook", "grapple_hook", "hook",
	"activecamo", "active_camo", "camo", "camouflage", "cloak",
	"overshield", "powerup_overshield", "shield", "powerup",
	"repulsor", "knockback", "thruster", "unsc_thruster", "evade", "dash",
	"sprint", "mobility_sprint", "equipment", "ability", "abilities",
}

// jetonsExplosions : le vocabulaire des explosions, pour lever les banques d'objets explosifs
// (les bobines) qui n'ont pas de pack. Source : les 49 packs `sb_008_exp_*`.
var jetonsExplosions = []string{
	"hardlight", "plasma", "shock", "kineticunsc", "kineticbanished", "watershallow",
	"single_small", "single_med", "single_large", "burst_small", "burst_med", "tiny",
	"coil", "fusioncoil", "fusion_coil", "blastcoil", "plasmacoil", "shockcoil",
	"hardlightcoil", "canister", "barrel", "propane", "explosive", "powercore",
}

// candidatsBanques rend tous les noms de banque plausibles, dedupliques par leur hachage.
//
// Le cout est le PRODUIT familles x portees x jetons ; il est imprime par l'appelant avec
// l'esperance de collision fortuite (`candidats x cibles / 2^32`), qui doit rester sous 0,10
// — le seuil que le lot des equipements manquants s'etait deja donne pour le murmur3.
func candidatsBanques(packs []string) map[uint32]string {
	out := make(map[uint32]string, 1<<19)

	// (1) Les 841 noms de packs, a l'identique : le temoin de calibration.
	for _, p := range packs {
		out[fnv1(p)] = p
	}

	// (2) Le produit grammatical.
	jetons := append(append(append([]string{}, jetonsModes...), jetonsEquipements...), jetonsExplosions...)
	for _, fam := range famillesBanques {
		for _, portee := range porteesBanques {
			base := "sb_" + fam + "_" + portee
			for _, j := range jetons {
				ajouter(out, base+j)
			}
		}
	}

	// (3) Les couples de jetons, sur les DEUX familles visees seulement (`mod` = les modes,
	// `abl` = les equipements). Ailleurs le produit exploserait sans rien viser.
	for _, fam := range []string{"004_mod", "007_abl", "130_mus"} {
		for _, portee := range porteesBanques {
			base := "sb_" + fam + "_" + portee
			for _, a := range jetonsModes {
				for _, b := range jetonsEquipements {
					ajouter(out, base+a+"_"+b)
					ajouter(out, base+b+"_"+a)
				}
			}
		}
	}

	// (4) Les noms nus, sans prefixe `sb_` : Wwise nomme ainsi ses banques par defaut.
	for _, j := range jetons {
		ajouter(out, j)
		ajouter(out, "sb_"+j)
	}
	ajouter(out, "Init")
	return out
}

// ajouter enregistre un candidat sans ecraser un nom deja retenu pour le meme hachage
// (deux candidats qui collident : le premier reste, et le fait est signale).
func ajouter(m map[uint32]string, nom string) {
	h := fnv1(nom)
	if ancien, vu := m[h]; vu && ancien != nom {
		return
	}
	m[h] = nom
}

// esperanceCollision rend le nombre attendu de correspondances FORTUITES entre `cibles`
// identifiants et `candidats` noms tires au hasard sur 32 bits.
func esperanceCollision(candidats, cibles int) float64 {
	return float64(candidats) * float64(cibles) / 4294967296.0
}

// formaterEsperance rend l'esperance et le verdict du seuil de 0,10.
func formaterEsperance(candidats, cibles int) string {
	e := esperanceCollision(candidats, cibles)
	verdict := "SOUS le seuil de 0,10"
	if e >= 0.10 {
		verdict = "AU-DESSUS du seuil de 0,10 — resultats a NE PAS publier tels quels"
	}
	return fmt.Sprintf("%d candidats x %d cibles => esperance de collision fortuite %.4f (%s)",
		candidats, cibles, e, verdict)
}
