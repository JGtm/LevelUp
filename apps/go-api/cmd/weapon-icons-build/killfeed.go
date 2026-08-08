package main

// killfeed.go — L'ATLAS « SANDBOX » EST CELUI DU KILL FEED, et il porte sa propre table de
// nommage.
//
// CE QUI L'A ÉTABLI. Le tag `bitd` déclare un bloc `entries` dont chaque enregistrement est
// exactement le triplet cherché : `identifier` (StringID), `bitmap` (référence vers l'atlas)
// et `bitmap index`. Le tag `8646f61a` en porte 85, toutes vers l'atlas `0302cad3`. Une fois
// les identifiants craqués, le motif se lit sans ambiguïté : `killfeed_battle_rifle`,
// `killfeed_warthog`, `killfeed_headshot`… Ce n'est donc pas un atlas « sandbox » générique,
// c'est LE jeu d'icônes que le jeu affiche dans son kill feed.
//
// Layout d'une entrée, déduit du plugin bitd.xml et non d'offsets en dur :
//
//	+0  _2  identifier    (4)
//	+4  _41 bitmap        (28 — identifiant de tag à +8)
//	+32 _5  bitmap index  (2)
//	+34 padding           (2)   → 36 octets
//
// POURQUOI LE PRÉFIXE EST DANS LE CODE. Les identifiants ne sont NI dans le binaire du jeu
// (408 525 chaînes moissonnées, zéro correspondance) NI en clair dans les tags — tout est
// haché en release. Le motif `killfeed_<nom>` a été trouvé en essayant des préfixes plausibles
// sur un vocabulaire connu : 4 correspondances immédiates, puis 43 sur 85 en préfixant tout le
// vocabulaire du binaire. Chaque nom rendu est une ÉGALITÉ DE HACHAGE, pas une ressemblance.

import (
	"encoding/binary"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

const (
	bitdEntrySize   = 36
	bitdOffBitmap   = 4
	bitdOffIndex    = 32
	killfeedPrefix  = "killfeed_"
	sandboxAtlasTag = 0x0302cad3
)

// resolveKillfeedNames rend, par index de l'atlas kill feed, le nom interne craqué.
func resolveKillfeedNames(ix *tagIndex) map[int]string {
	dict := killfeedDict()
	out := map[int]string{}
	seen := map[uint32]bool{}
	for _, r := range ix.byGroup["bitd"] {
		if seen[r.ID] {
			continue
		}
		seen[r.ID] = true
		data, err := ix.extract(r)
		if err != nil {
			continue
		}
		wt, err := openWeapTag(data)
		if err != nil {
			continue
		}
		root, err := wt.rootBlock()
		if err != nil {
			continue
		}
		for _, blk := range wt.tagBlocksOf(root) {
			abs, size := wt.blockAbs(blk)
			if abs < 0 || size < bitdEntrySize || abs+size > len(data) {
				continue
			}
			for i := 0; i+bitdEntrySize <= size; i += bitdEntrySize {
				o := abs + i
				if binary.LittleEndian.Uint32(data[o+bitdOffBitmap+8:]) != sandboxAtlasTag {
					continue
				}
				name, hit := dict[binary.LittleEndian.Uint32(data[o:])]
				if !hit {
					continue
				}
				idx := int(binary.LittleEndian.Uint16(data[o+bitdOffIndex:]))
				if _, dup := out[idx]; !dup {
					out[idx] = strings.TrimPrefix(name, killfeedPrefix)
				}
			}
		}
	}
	return out
}

// plausibleIdent : un identifiant du jeu s'écrit `[a-z0-9_]+`. Ce filtre n'est pas cosmétique.
//
// L'argument « une égalité de hachage sur 32 bits vaut certitude » ne tient que pour un
// vocabulaire PETIT. Mesuré : le paquet `gamecms.cms` rend 6 068 000 chaînes, soit ~24 M de
// hachages contre 42 cibles — l'espérance de collision fortuite y vaut ~0,2, et le seul
// « match » obtenu était `killfeed_3O1\`, du bruit. La moisson du binaire (408 525 chaînes,
// espérance ~0,07) reste sûre ; le filtre garantit qu'un faux positif ne PASSE PAS pour un nom.
func plausibleIdent(s string) bool {
	t := strings.TrimPrefix(s, killfeedPrefix)
	if len(t) < 3 || len(t) > 40 {
		return false
	}
	for _, c := range t {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}

// killfeedDict : hachage -> nom, à partir des chaînes du binaire du jeu ET du vocabulaire
// curaté, chacune essayée telle quelle et préfixée.
func killfeedDict() map[uint32]string {
	dict := make(map[uint32]string, 1<<20)
	add := func(s string) {
		for _, v := range []string{s, killfeedPrefix + s} {
			if !plausibleIdent(v) {
				continue
			}
			h := uint32(mapvar.LabelHash(v))
			if _, dup := dict[h]; !dup {
				dict[h] = v
			}
		}
	}
	if exe := gameBinary(); exe != "" {
		if strs, err := harvestStrings(exe, 3); err == nil {
			for s := range strs {
				add(strings.ToLower(s))
			}
		}
	}
	for _, w := range curatedVocabulary {
		add(w)
	}
	for _, w := range patternVocabulary() {
		add(w)
	}
	return dict
}

// patternVocabulary décline les MOTIFS observés dans les identifiants déjà craqués, au lieu
// de continuer à deviner mot par mot.
//
// Les motifs ne sont pas supposés : ils sont lus dans ce qui a déjà craqué —
// `killfeed_turret_plasma` et `killfeed_turret_chaingun` d'un côté, `killfeed_scorpion_turret`
// de l'autre, `killfeed_falcon_chaingun`, `killfeed_plasma_grenade_stick`,
// `killfeed_flag_melee` et `killfeed_bomb_melee`. Décliner coûte quelques centaines de
// hachages et ne peut rien inventer : seule une égalité exacte sort.
func patternVocabulary() []string {
	var out []string
	armements := []string{
		"chaingun", "gauss", "rocket", "plasma", "machinegun", "machine_gun", "grenade",
		"needler", "missile", "cannon", "shade", "aa", "at", "sentinel", "gunner",
		"grenadier", "flak", "beam", "laser", "mortar",
	}
	engins := []string{
		"warthog", "rockethog", "gungoose", "mongoose", "scorpion", "wraith", "banshee",
		"ghost", "chopper", "wasp", "pelican", "phantom", "falcon", "razorback", "shade",
	}
	for _, a := range armements {
		out = append(out, "turret_"+a, a+"_turret")
		for _, e := range engins {
			out = append(out, e+"_"+a)
		}
	}
	for _, e := range engins {
		out = append(out, e+"_turret")
	}
	for _, g := range []string{"frag", "plasma", "spike", "dynamo", "splinter"} {
		out = append(out, g+"_grenade_stick", g+"_stick", "stick_"+g)
	}
	// Schema `<qualifiant>_<classe>` : c est lui qui a rendu `commando_rifle`, `ma5k_smg` et
	// `plasma_blaster` la ou le seul nom commercial (`commando`, `ma5k`) echouait — et ces
	// mots-la SONT dans le binaire, donc bien essayes. Le jeu nomme par classe, pas par
	// marque.
	qualifiants := []string{
		"sidekick", "bulldog", "commando", "disruptor", "mangler", "ravager", "stalker",
		"cindershot", "shock", "pulse", "vestige", "avenger", "ma5k", "spnkr", "fuel",
		"fuelrod", "plasma", "energy", "gravity", "sentinel", "battle", "assault", "sniper",
		"heatwave", "skewer", "bandit", "needler", "hydra", "mutilator", "kinetic",
	}
	for _, q := range qualifiants {
		for _, c := range []string{
			"rifle", "pistol", "smg", "blaster", "shotgun", "launcher", "carbine", "cannon",
			"caster", "beam", "sword", "hammer", "magnum", "dmr",
		} {
			out = append(out, q+"_"+c, q+c)
		}
	}
	for _, m := range []string{
		"flag", "ball", "oddball", "sword", "hammer", "skull", "core", "seed", "bomb",
		"fist", "weapon", "shield", "back_smack",
	} {
		out = append(out, m+"_melee", "melee_"+m)
	}
	return out
}

// curatedVocabulary — les noms que le binaire ne contient PAS comme jeton isolé et qu'aucune
// moisson ne rend donc. Chaque entrée n'est retenue que si son hachage tombe EXACTEMENT sur
// un StringID cherché : sur un espace de 32 bits, une centaine de candidats rend une
// collision fortuite négligeable (~1 sur 40 millions). Ce n'est pas une liste de suppositions
// affichées telles quelles, c'est un vocabulaire soumis à un test qui échoue bruyamment.
var curatedVocabulary = []string{
	"sandwich", "mythic_sandwich", "fusion_coil", "power_seed", "machine_gun",
	"assault_rifle", "battle_rifle", "sniper", "sniper_rifle", "shotgun", "commando",
	"sidekick", "bulldog", "rocket_launcher", "spnkr", "fuel_rod", "skewer", "sword",
	"energy_sword", "gravity_hammer", "hammer", "avenger", "bandit", "bandit_evo",
	"carbine", "vestige_carbine", "disruptor", "mangler", "ravager", "stalker_rifle",
	"pulse_carbine", "shock_rifle", "sentinel_beam", "sentinelbeam", "needler",
	"plasma_pistol", "cindershot", "heatwave", "hydra", "mutilator", "plasma_turret",
	"shade_turret", "turret", "skull", "flag", "ball", "bomb", "oddball", "seed",
	"grenade", "frag_grenade", "plasma_grenade", "dynamo_grenade", "spike_grenade",
	"overshield", "active_camo", "camo", "repair_field", "drop_wall", "threat_sensor",
	"grappleshot", "thruster", "repulsor", "shroud_screen", "quantum", "quantum_translocator",
	"warthog", "rockethog", "gungoose", "mongoose", "scorpion", "wraith", "banshee", "ghost",
	"chopper", "wasp", "pelican", "phantom", "falcon", "razorback",
	"headshot", "melee", "suicide", "ricochet", "callout", "environment", "splatter",
	// Vocabulaire donne par l utilisateur en regardant les icones, puis valide par hachage.
	"perfect", "waterfall", "back_smack", "vip", "stockpile", "kinetic_barrel",
	"player_left", "player_joined", "player_rejoined", "assist", "betrayal",
}
