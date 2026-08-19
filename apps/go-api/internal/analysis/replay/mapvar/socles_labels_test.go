package mapvar

// socles_labels_test.go — NOMMER LES LABELS INCONNUS DES SOCLES.
//
// D'OU VIENT CE FICHIER. Sur les cartes DEV (Catalyst, Cliffhanger), les objets de socle
// ne portent AUCUN label. Sur une carte Forge (Smallhalla), ils en portent — et parmi eux
// des filtres de mode deja nommes (`stockpile_include`, `infection_exclude`,
// `ctf_multi_exclude`) a cote de hashs que la table ne sait pas nommer, dont un revient
// trois a quatre fois sur le MEME objet. Si ce hash porte un nom, il dit ce qu'est
// l'objet, ou dans quel mode il s'allume.
//
// LE GARDE-FOU EST CELUI D'`objectives.go`, et il n'est pas negociable : une recherche par
// force brute PRODUIT des collisions. On ne retient un nom que s'il est semantiquement
// coherent avec le domaine Halo, et un hash non resolu reste INCONNU — on ne devine pas.
//
// Garde : `MVAR_LABELS` (liste de hashs a resoudre, separes par des virgules).

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

const soclesLabelsEnv = "MVAR_LABELS"

// soclesMots : le vocabulaire de recherche. Il est VOLONTAIREMENT restreint au domaine
// (modes de jeu Halo, objets de carte, verbes de placement) — elargir le vocabulaire
// multiplie les collisions fortuites sans rien apporter.
var soclesMots = []string{
	"weapon", "weapons", "spawn", "spawner", "spawns", "pad", "socket", "rack", "power",
	"powerup", "up", "item", "items", "drop", "ordnance", "equipment", "gadget", "ammo",
	"initial", "respawn", "static", "dynamic", "placed", "forge", "arena", "slayer",
	"koth", "hill", "king", "oddball", "ctf", "flag", "strongholds", "stronghold",
	"extraction", "stockpile", "assault", "infection", "elimination", "escalation",
	"fiesta", "firefight", "minigame", "land", "grab", "total", "control", "vip",
	"juggernaut", "race", "attrition", "tactical", "gungame", "grifball", "bomb", "ball",
	"zone", "neutral", "multi", "include", "exclude", "navpoint", "delivery", "objective",
	"vehicle", "grenade", "camo", "overshield", "shield", "sniper", "rocket", "sword",
	"hammer", "energy", "heavy", "light", "primary", "secondary", "team", "red", "blue",
	"all", "any", "none", "default", "custom", "map", "level", "variant", "mode", "game",
	"pickup", "loot", "cache", "crate", "box", "stand", "mount", "holder", "point",
}

// TestSoclesResoudreLabels cherche un nom snake_case dont le murmur3 retombe sur chaque
// hash demande, en une, deux ou trois parties.
func TestSoclesResoudreLabels(t *testing.T) {
	brut := strings.TrimSpace(os.Getenv(soclesLabelsEnv))
	if brut == "" {
		t.Skipf("%s absent — resolution de labels ignoree", soclesLabelsEnv)
	}
	cibles := map[int32]bool{}
	for _, s := range strings.Split(brut, ",") {
		v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
		if err != nil {
			t.Fatalf("%s: %q illisible: %v", soclesLabelsEnv, s, err)
		}
		cibles[int32(v)] = true
	}
	trouves := map[int32][]string{}
	essais := 0
	for _, a := range soclesMots {
		essais++
		soclesTeste(a, cibles, trouves)
		for _, b := range soclesMots {
			essais++
			soclesTeste(a+"_"+b, cibles, trouves)
			for _, c := range soclesMots {
				essais++
				soclesTeste(a+"_"+b+"_"+c, cibles, trouves)
			}
		}
	}
	t.Logf("== %d candidats essayes sur %d hashs cibles ==", essais, len(cibles))
	for h := range cibles {
		noms, ok := trouves[h]
		if !ok {
			t.Logf("  hash %12d : NON RESOLU — il reste inconnu", h)
			continue
		}
		t.Logf("  hash %12d : %d candidat(s) %v — a retenir seulement si semantiquement coherent",
			h, len(noms), noms)
	}
}

func soclesTeste(nom string, cibles map[int32]bool, trouves map[int32][]string) {
	h := LabelHash(nom)
	if cibles[h] {
		trouves[h] = append(trouves[h], nom)
	}
}
