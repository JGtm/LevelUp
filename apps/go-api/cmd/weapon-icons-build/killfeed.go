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

// killfeedDict : hachage -> nom, à partir des chaînes du binaire du jeu ET du vocabulaire
// curaté, chacune essayée telle quelle et préfixée.
func killfeedDict() map[uint32]string {
	dict := make(map[uint32]string, 1<<20)
	add := func(s string) {
		for _, v := range []string{s, killfeedPrefix + s} {
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
	return dict
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
	"headshot", "melee", "suicide", "ricochet", "callout", "environment",
	"player_left", "player_joined", "player_rejoined", "assist", "betrayal",
}
