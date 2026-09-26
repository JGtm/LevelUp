package main

import "strings"

// vocabVehicules : les familles de chassis reconnues par leurs noms de maillage internes
// (chassis, roue, panneau...). Un `vehi` est classe par le mot-cle le PLUS present dans ses
// chaines ASCII. Attention : ce nom designe la FAMILLE DE CHASSIS, pas la variante d'armement
// (Rockethog/Razorback partagent le chassis « warthog » — leur difference vit dans les refs
// d'armement du vehi, pas dans les noms de maillage).
var vocabVehicules = []string{
	"warthog", "mongoose", "gungoose", "ghost", "banshee", "wasp",
	"scorpion", "wraith", "chopper", "pelican", "phantom", "falcon",
	"shade", "skiff", "turret", "cov",
}

// classeVehicule rend la famille de chassis d'un vehi, par vote de ses chaines ASCII, et le
// nombre d'occurrences du mot gagnant (0 = inconnu).
func classeVehicule(noms []string) (string, int) {
	compte := map[string]int{}
	for _, n := range noms {
		bas := strings.ToLower(n)
		for _, mot := range vocabVehicules {
			if strings.Contains(bas, mot) {
				compte[mot]++
			}
		}
	}
	meilleur, best := "", 0
	for _, mot := range vocabVehicules { // ordre stable, priorite au plus specifique en tete
		if compte[mot] > best {
			best, meilleur = compte[mot], mot
		}
	}
	if meilleur == "" {
		return "inconnu", 0
	}
	return meilleur, best
}
