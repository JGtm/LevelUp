package main

// noms.go — ETAPE 3 : retrouver les noms d'evenements Wwise par hachage.
//
// POURQUOI C'EST FAISABLE. Un identifiant d'objet Wwise est le FNV-1 32 bits du nom mis
// en minuscules. Le hachage n'est pas inversible, mais l'espace de recherche est ici
// minuscule : le nom de l'arme est deja connu (il est dans le nom du `.pck`), il ne reste
// qu'a essayer les verbes usuels. Quelques milliers de candidats suffisent, contre
// 2^32 pour une attaque aveugle.
//
// POURQUOI C'EST NECESSAIRE. Le chunk `STID`, qui porterait la table des noms, n'est
// present que sur 2 banks sur 1305 (mesure etape 1). Le jeu ne livre donc pas les noms.

import "strings"

// fnv1 rend le FNV-1 32 bits d'un nom, minuscules — la fonction d'identifiant de Wwise.
// Attention : FNV-1 (multiplication PUIS ou-exclusif), pas FNV-1a.
func fnv1(s string) uint32 {
	const offset = 2166136261
	const prime = 16777619
	h := uint32(offset)
	for _, c := range []byte(strings.ToLower(s)) {
		h *= prime
		h ^= uint32(c)
	}
	return h
}

// prefixesEvent : formes de nommage rencontrees dans les projets Wwise.
var prefixesEvent = []string{"play_", "", "start_", "stop_", "set_"}

// verbesEvent : suffixes candidats, du plus probable au plus rare. Le tir en premier :
// c'est la cible du chantier.
var verbesEvent = []string{
	"fire", "fire_1p", "fire_3p", "fire_player", "fire_npc", "fire_loop",
	"fire_start", "fire_stop", "fire_end", "fire_single", "fire_burst", "fire_auto",
	"fire_tail", "fire_first", "fire_last", "fire_alt", "altfire", "alt_fire",
	"shoot", "shot", "single", "burst", "auto", "loop", "start", "stop", "tail",
	"reload", "reload_start", "reload_end", "reload_empty", "reload_full",
	"equip", "unequip", "draw", "holster", "pickup", "drop", "swap",
	"melee", "dryfire", "dry_fire", "empty", "click", "trigger",
	"zoom_in", "zoom_out", "ads_in", "ads_out",
	"charge", "charge_start", "charge_stop", "overheat", "vent", "cooldown",
	"spinup", "spindown", "bolt", "cock", "impact", "whizby", "projectile",
}

// basesArme rend les racines de nom plausibles pour une arme donnee.
// `arme` vient du `.pck` (ex. « un_assaultrifle »).
func basesArme(arme string) []string {
	bases := []string{
		"wea_" + arme,
		arme,
		"sb_010_wea_" + arme,
		"weapon_" + arme,
	}
	// La faction (un_, cv_, bt_, fr_) n'est pas toujours dans le nom de l'evenement.
	if i := strings.Index(arme, "_"); i > 0 && i < 4 {
		nu := arme[i+1:]
		bases = append(bases, "wea_"+nu, nu, "weapon_"+nu)
	}
	return bases
}

// candidatsNoms rend, par hachage, tous les noms d'evenements plausibles pour une arme.
func candidatsNoms(arme string) map[uint32]string {
	out := map[uint32]string{}
	for _, base := range basesArme(arme) {
		for _, p := range prefixesEvent {
			for _, v := range verbesEvent {
				for _, nom := range []string{p + base + "_" + v, p + v + "_" + base} {
					out[fnv1(nom)] = nom
				}
			}
			out[fnv1(p+base)] = p + base
		}
	}
	return out
}

// resoudreNoms apparie les identifiants d'evenements observes aux candidats generes.
func resoudreNoms(arme string, ids []uint32) map[uint32]string {
	cands := candidatsNoms(arme)
	out := map[uint32]string{}
	for _, id := range ids {
		if nom, ok := cands[id]; ok {
			out[id] = nom
		}
	}
	return out
}
