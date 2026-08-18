package himap

// SONDE (2026-08-18, plan PLAN_EQUIPEMENTS_MANQUANTS_SONS phase 1.5) — la SECONDE TENTATIVE
// de casser les identifiants de chaine qui resistent, sur une cible ETROITE et avec un
// vocabulaire ELARGI.
//
// POURQUOI UN DICTIONNAIRE A PART, ET PAS UN ELARGISSEMENT DE `dicoJetons`. Le dictionnaire
// global sert un balayage de ~260 identifiants (`TestSondeSofdNommage`) : son esperance de
// collision fortuite vaut `candidats x cibles / 2^32`, et l'elargir ferait entrer du bruit
// dans un instrument dont les sorties alimentent le MANIFESTE. Ici les cibles sont CINQ,
// nommement designees, et le meme budget de collision autorise trente fois plus de candidats.
// Le vocabulaire vit donc a cote, et le global reste tel qu'il est.
//
// LA PROFONDEUR RESTE DEUX, et c'est mesure plutot que prudent : le lot precedent a deja
// passe la profondeur TROIS sur le vocabulaire d'origine (51 millions de candidats, esperance
// 0,012 sur une cible) et n'a rien rendu. Ce qui manque n'est donc pas de la profondeur, c'est
// du VOCABULAIRE — d'ou l'elargissement a deux jetons.
//
// LE TEMOIN EST OBLIGATOIRE. Un dictionnaire qui ne casse rien ne prouve rien : il peut aussi
// bien etre casse lui-meme. Le test exige donc que les noms DEJA ETABLIS par ailleurs sortent
// de ce dictionnaire-ci — sans quoi son negatif ne vaudrait pas.
//
// AUCUN FICHIER DE JEU : les identifiants cibles sont des PIECES, recopiees de la chaine
// `sofd -> sofa` mesuree le 2026-08-18. Ce test tourne partout.

import (
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// stringIDsAnonymes : les identifiants de chaine NON CASSES qui portent un objet d'equipement
// PRESENT AU CORPUS. Ce sont les seuls dont le nom manque vraiment au produit.
var stringIDsAnonymes = map[uint32]string{
	0xb328c9fa: "sofa eb500815 · rang 10 famille A · eqip 4396db42 (corpus) + 4eebcb18",
	0x9bc48d0f: "sofa a0ed8dfc · rang 22 · eqip 72199cba — capteur par le MODELE",
	0xcb9f3095: "sofa fb80ca6f · rang 19 · eqip 8e2dc574 + 37c87a13 — mur par le MODELE",
	0x828e2a93: "sofa 8a7a9190 · rang 20 · eqip 8c77ffe7 — grappin par le MODELE",
	0x9de62e2f: "sofa e74de8a4 · rang 21 · eqip eef5d48d — propulseur par le MODELE",
}

// stringIDsTemoins : des noms ETABLIS PAR AILLEURS (chaine `sofa` cassee par le dictionnaire
// global, et RECETTE_LOADOUT §13). Ils doivent sortir de CE dictionnaire — c'est le controle
// qui donne sa valeur au negatif.
var stringIDsTemoins = map[uint32]string{
	0x00bef187: "repair_field",
	0x1f7c6a15: "quantum_translocator",
	0xc5bb0ba7: "threat_seeker",
	0x2edcb6ef: "active_camo",
	0x3ed01bf0: "powerup_overshield",
	0x566bb170: "ability_location_sensor",
	0xedebd7b7: "ability_deployable_wall",
	0x87b1d7a4: "ability_grapple_hook",
	0xed76a664: "ability_evade",
	0x2b5d5eac: "ability_knockback",
}

// dicoPrefixesLarges : les familles de nom, elargies. Les cinq premieres sont celles du
// dictionnaire global ; les suivantes viennent des conventions VUES ailleurs dans le jeu
// (`sb_007_abl_repairfield` pour `abl`, les categories d'objet pour `device`/`gadget`).
var dicoPrefixesLarges = []string{
	"", "ability_", "powerup_", "equipment_", "mobility_", "melee_", "player_",
	"sandbox_", "weapon_", "item_", "eqip_", "spartan_", "armor_", "gadget_", "consumable_",
	"abl_", "equip_", "eq_", "device_", "tool_", "utility_", "support_", "trait_",
	"deployable_", "deploy_", "mp_", "sof_", "pickup_", "unsc_", "banished_", "covenant_",
}

// dicoJetonsLarges : le vocabulaire du domaine, ELARGI aux champs lexicaux que le premier
// dictionnaire n'avait pas — l'occultation (voile, brume, rideau), la projection (emetteur,
// projecteur), le brouillage, et la lumiere dure. Les mots du dictionnaire global sont
// repris : l'elargissement ne doit rien PERDRE.
var dicoJetonsLarges = append(append([]string{}, dicoJetons...), []string{
	// occultation et dissimulation
	"veil", "fog", "mist", "cloud", "haze", "vapor", "vapour", "curtain", "obscure",
	"obscuring", "obscurant", "conceal", "concealment", "cover", "occlude", "occlusion",
	"blind", "blinding", "flash", "chaff", "dust", "ash", "cinder", "steam",
	"mirage", "illusion", "distortion", "refraction", "diffuse", "diffuser", "scatter",
	// projection et deploiement
	"emitter", "projector", "generator", "dispenser", "deployer", "launcher", "canister",
	"capsule", "cylinder", "tube", "dome", "sphere", "bubble", "curtainwall", "pylon",
	"post", "stake", "totem", "banner", "flagpole",
	// brouillage et perturbation
	"jammer", "jam", "jamming", "interference", "spoof", "static", "noise", "disrupt",
	"disruptor", "disruption", "scramble", "scrambler", "suppress", "suppressor",
	// lumiere et energie
	"hardlight", "photon", "prism", "beam", "ray", "glow", "lamp", "lantern", "strobe",
	// soins et soutien
	"regen", "restore", "restoration", "recovery", "support", "boost", "buff", "aura",
	// mouvement
	"ascend", "descend", "hover", "glide", "leap", "vault", "clamber", "wall_run",
	// mots de structure
	"screen", "shroud", "smoke", "shield", "barrier", "guard", "ward", "block",
}...)

// enumereLarge parcourt le dictionnaire ELARGI a deux jetons et rend le nombre de candidats.
func enumereLarge(vu func(nom string, h uint32)) int {
	n := 0
	essaie := func(buf []byte) {
		n++
		vu(string(buf), uint32(mapvar.LabelHash(string(buf))))
	}
	buf := make([]byte, 0, 96)
	for _, p := range dicoPrefixesLarges {
		buf = append(buf[:0], p...)
		lp := len(buf)
		if lp > 0 {
			essaie(buf[:lp-1]) // le prefixe seul, sans son souligne final
		}
		for _, a := range dicoJetonsLarges {
			buf = append(buf[:lp], a...)
			la := len(buf)
			essaie(buf)
			for _, b := range dicoJetonsLarges {
				buf = append(buf[:la], '_')
				buf = append(buf, b...)
				essaie(buf)
			}
		}
	}
	return n
}

// TestDicoLargeTemoinsPassent est le CONTROLE : le dictionnaire elargi doit retrouver les noms
// que d'autres chaines ont deja etablis. Sans lui, un negatif ne vaudrait rien.
func TestDicoLargeTemoinsPassent(t *testing.T) {
	trouves := map[uint32]string{}
	n := enumereLarge(func(nom string, h uint32) {
		if _, veut := stringIDsTemoins[h]; veut {
			if _, deja := trouves[h]; !deja {
				trouves[h] = nom
			}
		}
	})
	t.Logf("dictionnaire elargi : %d candidats · esperance de collision sur %d cibles = %.4f",
		n, len(stringIDsAnonymes),
		float64(n)*float64(len(stringIDsAnonymes))/4294967296.0)
	var manquants []string
	for h, nom := range stringIDsTemoins {
		got, ok := trouves[h]
		if !ok {
			manquants = append(manquants, nom)
			continue
		}
		if got != nom {
			t.Errorf("identifiant %#08x : le dictionnaire rend %q, attendu %q", h, got, nom)
		}
	}
	sort.Strings(manquants)
	if len(manquants) > 0 {
		t.Fatalf("le dictionnaire elargi NE RETROUVE PAS %d temoins (%s) : son negatif ne "+
			"vaudrait rien", len(manquants), strings.Join(manquants, ", "))
	}
}

// TestDicoLargeAnonymes est la MESURE : elle statue chaque identifiant anonyme. Un nom trouve
// est un nom du jeu ; aucun nom trouve est un NEGATIF, chiffre par le controle ci-dessus.
//
// Ce test n'ECHOUE PAS quand il ne trouve rien — le negatif est un resultat, pas une panne. Il
// echoue si un nom SORT sans qu'on l'ait vu : la table `stringIDsAnonymes` est alors perimee,
// et c'est le manifeste qu'il faut mettre a jour.
func TestDicoLargeAnonymes(t *testing.T) {
	trouves := map[uint32]string{}
	n := enumereLarge(func(nom string, h uint32) {
		if _, veut := stringIDsAnonymes[h]; veut {
			if _, deja := trouves[h]; !deja {
				trouves[h] = nom
			}
		}
	})
	t.Logf("== %d candidats contre %d identifiants anonymes ==", n, len(stringIDsAnonymes))
	ids := make([]uint32, 0, len(stringIDsAnonymes))
	for h := range stringIDsAnonymes {
		ids = append(ids, h)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, h := range ids {
		if nom, ok := trouves[h]; ok {
			t.Errorf("%#08x CASSE : %q — %s. Porter le nom au manifeste et retirer la ligne "+
				"de stringIDsAnonymes.", h, nom, stringIDsAnonymes[h])
			continue
		}
		t.Logf("  %#08x  NON CASSE  · %s", h, stringIDsAnonymes[h])
	}
}
