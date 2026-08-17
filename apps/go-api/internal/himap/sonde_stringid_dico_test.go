package himap

// OUTIL DE SONDE (2026-08-18) — CASSER un identifiant de chaine Halo Infinite par
// dictionnaire.
//
// POURQUOI IL FAUT LE CASSER. Les modules du jeu n'ont plus de table de chaines
// (`stringsSize` = 0 sur les neuf modules indexes) : un nom de definition n'y est present que
// sous la forme de son murmur3 x86_32 (seed 0), le `string_id` — la fonction `FUN_140748a74`
// de l'executable. Un identifiant se retrouve donc en HACHANT des candidats, jamais en lisant.
//
// LE DICTIONNAIRE S'ARRETE A DEUX JETONS, ET C'EST UNE DECISION MESUREE. Une premiere version
// enumerait jusqu'a TROIS jetons : 25 millions de candidats contre 200 identifiants cibles,
// soit 25e6 x 200 / 2^32 = 1,2 collision fortuite attendue. Elle en a rendu exactement une, et
// elle etait visible a l'oeil nu (`ability_punch_detector_thruster`). A deux jetons le
// dictionnaire compte ~340 000 candidats, l'esperance tombe a 0,02, et il retrouve TOUS les
// noms etablis par ailleurs — parce qu'aucun nom reel du domaine ne depasse deux jetons apres
// son prefixe (`ability_location_sensor`, `ability_deployable_wall`, `quantum_translocator`,
// `mobility_sprint`...). On perd la capacite de casser un hypothetique nom a trois jetons ;
// on gagne un instrument dont les sorties n'ont pas besoin d'etre triees a la main.
//
// CE QUE CET OUTIL NE GARANTIT PAS. Un identifiant que le dictionnaire ne casse pas reste
// ANONYME : il ne se rapproche pas du voisin le plus proche, et l'objet qu'il designe garde sa
// famille `other`.
//
// La fonction de hachage n'est pas reimplementee : `mapvar.LabelHash` est le murmur3 canonique
// du depot (identifie sur les labels Forge). Cet import ne vaut que pour les tests.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// dicoPrefixes : les familles de nom observees dans le jeu. La chaine vide couvre les noms
// sans prefixe (`active_camo`, `quantum_translocator`, `repair_field`).
var dicoPrefixes = []string{
	"", "ability_", "powerup_", "equipment_", "mobility_", "melee_", "player_",
	"sandbox_", "weapon_", "item_", "eqip_", "spartan_", "armor_", "gadget_", "consumable_",
}

// dicoJetons : le vocabulaire du domaine. Un nom vaut un prefixe suivi d'un ou deux jetons
// joints par `_`.
var dicoJetons = []string{
	"active", "camo", "camouflage", "cloak", "cloaking", "invisibility", "stealth", "hologram",
	"overshield", "shield", "shields", "bubble", "drop", "dropwall", "wall", "barrier", "cover",
	"shroud", "screen", "smoke", "deployable", "deploy", "portable", "bandana",
	"translocator", "translocate", "translocation", "quantum", "teleport", "teleporter",
	"portal", "warp", "beacon", "marker", "recall", "blink", "phase", "tunnel", "anchor",
	"threat", "seeker", "sensor", "threatsensor", "tracker", "track", "radar", "detect",
	"detector", "detection", "location", "locator", "scanner", "motion", "reveal",
	"repair", "field", "heal", "healing", "regen", "regeneration", "medic", "revive",
	"grapple", "hook", "grappleshot", "hookshot", "swing", "zipline", "grapplehook",
	"evade", "thruster", "thrust", "dash", "dodge", "boost", "slide", "dive", "jump",
	"knockback", "repulsor", "repulse", "repulsion", "push", "blast", "shockwave", "kinetic",
	"sprint", "run", "default", "none", "null", "melee", "punch", "speed", "damage",
	"drain", "power", "pulse", "emp", "trap", "mine", "turret", "sentry", "decoy", "flare",
	"gravity", "lift", "pad", "volume", "zone", "aura", "link", "node", "relay", "tag",
	"grenade", "socket", "spawner", "resupply", "crate", "pod", "canister", "device",
	"ammo", "infinite", "battery", "energy", "charge", "core", "module", "panel", "plate",
	"unsc", "banished", "forerunner", "spartan", "elite", "brute", "vehicle", "personal",
	"mp", "campaign", "test", "debug", "dummy", "generic", "base", "common", "shared",
}

// casseIdentifiantsDeChaine cherche, pour chaque identifiant cible, un nom du dictionnaire
// dont le murmur3 lui est egal. Le second retour est le NOMBRE de candidats enumeres — il sert
// a chiffrer l'esperance de collision fortuite plutot qu'a la supposer negligeable.
func casseIdentifiantsDeChaine(cibles map[uint32]bool) (map[uint32]string, int) {
	out := map[uint32]string{}
	if len(cibles) == 0 {
		return out, 0
	}
	n := enumereCandidats(func(nom string, h uint32) {
		if !cibles[h] {
			return
		}
		if _, deja := out[h]; !deja {
			out[h] = nom
		}
	})
	return out, n
}

// tableCandidats rend le dictionnaire COMPLET indexe par hachage, et sa taille. Il sert a
// chercher un identifiant de chaine a une position INCONNUE d'une structure — a n'employer
// que sur des fenetres etroites, sous peine de collisions (l'esperance vaut
// taille_table x mots_balayes / 2^32).
func tableCandidats() (map[uint32]string, int) {
	out := make(map[uint32]string, 400000)
	n := enumereCandidats(func(nom string, h uint32) {
		if _, deja := out[h]; !deja {
			out[h] = nom
		}
	})
	return out, n
}

// enumereCandidats parcourt le dictionnaire a DEUX jetons et rend le nombre de candidats.
func enumereCandidats(vu func(nom string, h uint32)) int {
	return enumereJusqua(2, vu)
}

// enumereJusqua parcourt le dictionnaire jusqu'a `prof` jetons apres le prefixe.
//
// L'enumeration est faite sur un tampon reutilise : construire chaque candidat par
// concatenation de chaines couterait plus que le hachage lui-meme.
//
// LA PROFONDEUR 3 N'EST LEGITIME QUE SUR UNE CIBLE ETROITE. L'esperance de collision
// fortuite vaut `candidats x cibles / 2^32` : a 44 millions de candidats elle atteint 2 pour
// 200 cibles (inexploitable) mais 0,05 pour 5 (exploitable). C'est le PRODUIT qui decide, pas
// la taille du dictionnaire — d'ou ce parametre plutot qu'un dictionnaire unique.
func enumereJusqua(prof int, vu func(nom string, h uint32)) int {
	n := 0
	essaie := func(buf []byte) {
		n++
		vu(string(buf), uint32(mapvar.LabelHash(string(buf))))
	}
	buf := make([]byte, 0, 96)
	for _, p := range dicoPrefixes {
		buf = append(buf[:0], p...)
		lp := len(buf)
		if lp > 0 {
			essaie(buf[:lp-1]) // le prefixe seul, sans son souligne final
		}
		for _, a := range dicoJetons {
			buf = append(buf[:lp], a...)
			la := len(buf)
			essaie(buf)
			if prof < 2 {
				continue
			}
			for _, b := range dicoJetons {
				buf = append(buf[:la], '_')
				buf = append(buf, b...)
				lb := len(buf)
				essaie(buf)
				if prof < 3 {
					continue
				}
				for _, c := range dicoJetons {
					buf = append(buf[:lb], '_')
					buf = append(buf, c...)
					essaie(buf)
				}
			}
		}
	}
	return n
}

// casseLarge tente les cibles avec un dictionnaire a TROIS jetons. A n'employer que sur une
// poignee de cibles (cf. enumereJusqua) ; rend tous les candidats trouves par cible, pas
// seulement le premier, pour que l'implausibilite d'une collision soit visible.
func casseLarge(cibles map[uint32]bool) (map[uint32][]string, int) {
	out := map[uint32][]string{}
	n := enumereJusqua(3, func(nom string, h uint32) {
		if cibles[h] {
			out[h] = append(out[h], nom)
		}
	})
	return out, n
}

// TestDicoStringIDConnus est le CONTROLE de l'outil : les noms que la RECETTE §13 avait
// obtenus par un autre chemin doivent sortir du dictionnaire, aux identifiants lus dans les
// `sofa` de la palette de la famille A. Il ne touche pas aux fichiers du jeu, donc il tourne
// partout — c'est le garde-rail du dictionnaire, pas une sonde.
func TestDicoStringIDConnus(t *testing.T) {
	attendu := map[uint32]string{
		0xf08fa6e6: "mobility_sprint",
		0x566bb170: "ability_location_sensor",
		0xedebd7b7: "ability_deployable_wall",
		0x87b1d7a4: "ability_grapple_hook",
		0xed76a664: "ability_evade",
		0x2b5d5eac: "ability_knockback",
		0x17c1e79e: "melee_default",
		0x2edcb6ef: "active_camo",
		0x3ed01bf0: "powerup_overshield",
		0x1f7c6a15: "quantum_translocator",
		0xc5bb0ba7: "threat_seeker",
		0x00bef187: "repair_field",
	}
	cibles := map[uint32]bool{}
	for k := range attendu {
		cibles[k] = true
	}
	got, n := casseIdentifiantsDeChaine(cibles)
	t.Logf("%d candidats enumeres · esperance de collision fortuite sur %d cibles = %.4f",
		n, len(cibles), float64(n)*float64(len(cibles))/4294967296.0)
	for id, nom := range attendu {
		if got[id] != nom {
			t.Errorf("0x%08x : dictionnaire rend %q, attendu %q", id, got[id], nom)
		}
	}
}

// TestDicoLargeCiblesEtroites tente les identifiants de chaine que le dictionnaire a deux
// jetons ne casse pas et qui portent un objet OBSERVE dans les poses des films : les rangs
// 19 a 22 de la palette famille A (mesures comme mur / grappin / propulseur / capteur par la
// verite terrain Theater du 2026-07-27) et le rang 10 (le trou le plus frequent du corpus).
//
// Cinq cibles seulement : l'esperance de collision reste sous 0,1 (cf. enumereJusqua).
func TestDicoLargeCiblesEtroites(t *testing.T) {
	cibles := map[uint32]bool{
		0xcb9f3095: true, // sofa fb80ca6f — rang 19, eqip 8e2dc574 (MUR par la diagonale)
		0x828e2a93: true, // sofa 8a7a9190 — rang 20, eqip 8c77ffe7
		0x9de62e2f: true, // sofa e74de8a4 — rang 21, eqip eef5d48d
		0x9bc48d0f: true, // sofa a0ed8dfc — rang 22, eqip 72199cba (CAPTEUR par la diagonale)
		0xb328c9fa: true, // sofa eb500815 — rang 10, eqip 4396db42
		0x7f8e9891: true, // sofa ad9b4239 — rang 3, categorie nulle
	}
	got, n := casseLarge(cibles)
	t.Logf("%d candidats · %d cibles · esperance de collision %.4f",
		n, len(cibles), float64(n)*float64(len(cibles))/4294967296.0)
	for _, id := range []uint32{0xcb9f3095, 0x828e2a93, 0x9de62e2f, 0x9bc48d0f, 0xb328c9fa, 0x7f8e9891} {
		t.Logf("  0x%08x -> %v", id, got[id])
	}
}

// dicoBasesConnues : les noms de definition ETABLIS, tels que la chaine `sofa` les rend.
var dicoBasesConnues = []string{
	"mobility_sprint", "ability_location_sensor", "ability_deployable_wall",
	"ability_grapple_hook", "ability_evade", "ability_knockback", "melee_default",
	"active_camo", "powerup_overshield", "quantum_translocator", "threat_seeker",
	"repair_field", "regen_field", "ability_threat_sensor", "ability_repulsor",
	"ability_thruster", "ability_drop_wall", "ability_repair_field",
}

// dicoAffixesVariante : les marques de variante plausibles. Le jeu en emploie forcement une
// forme — les modes Fiesta servent des equipements AMELIORES (duree, portee, charges), et la
// structure montre qu'ils ont leur propre `sofa` ET leur propre `eqip`.
var dicoAffixesVariante = []string{
	"_fiesta", "_super", "_super_fiesta", "_superfiesta", "_upgraded", "_upgrade",
	"_improved", "_enhanced", "_mp", "_v2", "_02", "_2", "_alt", "_long", "_extended",
	"_big", "_large", "_plus", "_pro", "_advanced", "_elite", "_boosted", "_buffed",
	"_strong", "_max", "_ultra", "_mega", "_double", "_heavy", "_special", "_event",
	"_variant", "_b", "_tier2", "_t2", "_lvl2", "_level2", "_extra", "_bonus",
	"_powered", "_overcharged", "_infinite", "_unlimited", "_rapid", "_fast",
	"_s2", "_s3", "_s4", "_s5", "_s6", "_s7", "_new", "_old", "_legacy", "_test",
}

// TestDicoVariantesSuffixes — NEGATIF MESURE du 2026-08-18, et il faut le garder.
//
// L'HYPOTHESE, et elle etait bonne : la structure montre que les rangs 19 a 22 sont des objets
// qui PARTAGENT LEUR MODELE (`hlmt`) avec le capteur, le mur, le grappin et le propulseur — donc
// des VARIANTES du meme objet, avec leur propre `sofa` et leur propre `eqip`. Le mode Fiesta
// servant des equipements ameliores (duree, portee, charges — precision utilisateur du
// 2026-08-18), leur nom de definition devrait raisonnablement s'ecrire « base + marque de
// variante ».
//
// **IL NE S'ECRIT PAS AINSI.** 2 970 candidats (18 bases x 55 affixes, en suffixe, en prefixe et
// en remplacement de prefixe) contre 6 cibles : esperance de collision 4e-6, donc l'absence de
// resultat est une ABSENCE, pas un hasard. Les identifiants 0xcb9f3095, 0x828e2a93, 0x9de62e2f,
// 0x9bc48d0f, 0xb328c9fa et 0x7f8e9891 ne sont pas des derives des noms connus.
//
// Ce test ne touche pas aux fichiers du jeu : il tourne partout, et il ECHOUERA le jour ou l'un
// de ces identifiants se casse — ce qui est exactement ce qu'on veut savoir.
func TestDicoVariantesSuffixes(t *testing.T) {
	cibles := map[uint32]string{
		0xcb9f3095: "rang 19 (modele = mur)",
		0x828e2a93: "rang 20 (modele = grappin)",
		0x9de62e2f: "rang 21 (modele = propulseur)",
		0x9bc48d0f: "rang 22 (modele = capteur)",
		0xb328c9fa: "rang 10 (aucun modele commun)",
		0x7f8e9891: "rang 3 (categorie nulle)",
	}
	n := 0
	essaie := func(s string) {
		n++
		if note, ok := cibles[uint32(mapvar.LabelHash(s))]; ok {
			t.Errorf("0x%08x = %q [%s] — le negatif du 2026-08-18 est PERIME, mettre a jour le"+
				" manifeste et ce test", uint32(mapvar.LabelHash(s)), s, note)
		}
	}
	for _, b := range dicoBasesConnues {
		for _, a := range dicoAffixesVariante {
			essaie(b + a)
			essaie(strings.TrimPrefix(a, "_") + "_" + b)
			// La marque peut aussi REMPLACER le prefixe : `fiesta_deployable_wall`.
			if i := strings.Index(b, "_"); i > 0 {
				essaie(strings.TrimPrefix(a, "_") + b[i:])
			}
		}
	}
	t.Logf("%d candidats · %d cibles · esperance de collision %.8f",
		n, len(cibles), float64(n)*float64(len(cibles))/4294967296.0)
}
