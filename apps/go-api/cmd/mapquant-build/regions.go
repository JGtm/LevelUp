package main

// regions.go : la TABLE PAR INDEX DE RÉGION d'une entrée du catalogue.

import (
	"fmt"
	"log/slog"

	"github.com/JGtm/LevelUp/libs/film/decfilm"

	"github.com/JGtm/LevelUp/libs/himap/himap"
)

// aabbDuJeu dit si le jeu remplit la ligne de la région (`FUN_140be9d1c` : min < max sur les trois
// axes). Une région qui échoue garde son index dans le tableau de la carte, sans ligne : zéro bit
// par axe à la lecture.
func aabbDuJeu(b himap.Bounds) bool {
	for ax := 0; ax < 3; ax++ {
		if !(b.Max[ax] > b.Min[ax]) {
			return false
		}
	}
	return true
}

// poserLesRegions range la table par index de région de l'entrée : une ligne par région, dans
// l'ordre du tableau de la carte (le bloc structure-BSP du tag de niveau), avec ses bornes et la
// ligne 0x10 que la loi du moteur en tire. Une région dont l'AABB échoue au test du jeu est écrite
// `invalid` (son index avance, sa ligne reste à zéro). La région jouée (`jouee`) doit être valide et
// avoir les valeurs que l'entrée porte déjà : le catalogue d'avant et celui d'après ne peuvent pas
// se contredire sur elle.
func poserLesRegions(e *decfilm.MapQuantEntry, regions []himap.BSP, jouee int) error {
	if jouee < 0 || jouee >= len(regions) {
		return fmt.Errorf("région jouée %d hors des %d régions", jouee, len(regions))
	}
	invalides := 0
	for i, r := range regions {
		var lo, hi [3]float32
		var w [3]uint
		for ax := 0; ax < 3; ax++ {
			lo[ax], hi[ax] = float32(r.Bounds.Min[ax]), float32(r.Bounds.Max[ax])
		}
		if !aabbDuJeu(r.Bounds) {
			if i == jouee {
				return fmt.Errorf("AABB dégénérée (région jouée %d)", i)
			}
			invalides++
			e.AjouterRegion(lo, hi, w, true)
			continue
		}
		aw := r.Bounds.AxisWidths()
		for ax := 0; ax < 3; ax++ {
			w[ax] = uint(aw[ax])
		}
		e.AjouterRegion(lo, hi, w, false)
	}
	j := e.Regions[jouee]
	if j.Min != e.Min || j.Max != e.Max || j.AxisWidths != e.AxisWidths {
		return fmt.Errorf("la région jouée %d de la table diffère de l'entrée", jouee)
	}
	slog.Info("table des régions", "module", e.Module, "regions", len(regions), "invalides", invalides,
		"jouee", jouee, "indexBits", decfilm.LargeurIndexDePlage(len(regions)))
	return nil
}
