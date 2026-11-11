package main

// regions.go : la TABLE PAR INDEX DE RÉGION d'une entrée du catalogue.
//
// Un film nomme la région d'une position par un index, et `FUN_14076e524` lit les trois axes aux
// largeurs de CETTE région. L'entrée porte donc toutes les régions de compression de la carte,
// dans l'ordre du bloc structure-BSP du tag de niveau (`himap.RegionsBSPLocales` pour un module
// qui porte ses sbsp, `himap.RegionsBSPExternes` sinon), chacune avec ses bornes et ses largeurs.
// Les champs historiques de l'entrée (`min`, `max`, `axisWidths`, `region`) restent ceux de la
// région jouée.

import (
	"fmt"
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"

	"github.com/JGtm/halo-infinite-map-images/himap"
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
	poserLaLargeurDIndex(e, len(regions))
	j := e.Regions[jouee]
	if j.Min != e.Min || j.Max != e.Max || j.AxisWidths != e.AxisWidths {
		return fmt.Errorf("la région jouée %d de la table diffère de l'entrée", jouee)
	}
	slog.Info("table des régions", "module", e.Module, "regions", len(regions), "invalides", invalides,
		"jouee", jouee, "indexBits", decfilm.LargeurIndexDePlage(len(regions)))
	return nil
}

// poserLaLargeurDIndex écrit la largeur d'index de région que la loi du moteur tire du compte brut
// de régions ([decfilm.LargeurIndexDePlage]), sauf quand elle vaut la valeur implicite (1, une ou deux
// régions) : le champ reste alors absent du catalogue, comme l'entrée d'avant la table par région.
// Une carte de trois régions ou plus l'exige, faute de quoi le lecteur lirait un bit d'index de trop
// peu ([profile.MapQuantEntry.EffectiveRegionIndexBits]).
func poserLaLargeurDIndex(e *decfilm.MapQuantEntry, nbRegions int) {
	if bits := decfilm.LargeurIndexDePlage(nbRegions); bits != 1 {
		e.RegionIndexBits = bits
	}
}

// poserLesRegionsLocales range la table d'un module qui porte ses propres sbsp ; `q` est le BSP de
// déquantification que rend `himap.BSPQuantification`, qui doit être la région 0 de la table.
func poserLesRegionsLocales(e *decfilm.MapQuantEntry, modulePath string, q himap.BSP) error {
	regions, err := himap.RegionsBSPLocales(modulePath)
	if err != nil {
		return err
	}
	if regions[0].GlobalID != q.GlobalID {
		return fmt.Errorf("la région 0 de la table (%08x) n'est pas le BSP de déquantification (%08x)",
			regions[0].GlobalID, q.GlobalID)
	}
	return poserLesRegions(e, regions, 0)
}

// poserLesRegionsExternes range la table d'une carte dont les régions vivent hors de son module.
func poserLesRegionsExternes(e *decfilm.MapQuantEntry, regions []himap.RegionExterne, jouee int) error {
	bsps := make([]himap.BSP, len(regions))
	for i, r := range regions {
		bsps[i] = r.BSP
	}
	return poserLesRegions(e, bsps, jouee)
}

// avertitSiEcarte publie, en AVERTISSEMENT, les cas où le critère de région a écarté le plus
// gros tag sbsp. C'est la trace qui rend le catalogue relisible : sur les six canevas Forge de
// l'installation, cet avertissement est la ligne qui dit que les bornes ont changé de BSP.
func avertitSiEcarte(name, mod string, q himap.BSP, candidats []himap.BSP) {
	if len(candidats) < 2 || q.GlobalID == candidats[0].GlobalID {
		return
	}
	w, wPlusGros := q.Bounds.AxisWidths(), candidats[0].Bounds.AxisWidths()
	slog.Warn("le plus gros tag sbsp n'est PAS la région 0 — bornes prises sur la région 0",
		"carte", name, "module", mod, "candidats", len(candidats),
		"region0", fmt.Sprintf("gid=%08x W=%d/%d/%d étendue=%.1f/%.1f/%.1f",
			q.GlobalID, w[0], w[1], w[2], q.Bounds.Extent(0), q.Bounds.Extent(1), q.Bounds.Extent(2)),
		"plusGrosTag", fmt.Sprintf("gid=%08x W=%d/%d/%d étendue=%.1f/%.1f/%.1f",
			candidats[0].GlobalID, wPlusGros[0], wPlusGros[1], wPlusGros[2],
			candidats[0].Bounds.Extent(0), candidats[0].Bounds.Extent(1), candidats[0].Bounds.Extent(2)))
}

// entreeRegionLocale construit l'entrée d'une carte dont le module porte ses propres sbsp : les
// champs historiques viennent de `q` (le BSP de déquantification), la table par région du module.
func entreeRegionLocale(name, mod, modulePath string, q himap.BSP, candidats []himap.BSP) (decfilm.MapQuantEntry, error) {
	avertitSiEcarte(name, mod, q, candidats)
	e := decfilm.MapQuantEntry{Module: mod}
	w := q.Bounds.AxisWidths()
	for ax := 0; ax < 3; ax++ {
		e.Min[ax] = float32(q.Bounds.Min[ax])
		e.Max[ax] = float32(q.Bounds.Max[ax])
		e.AxisWidths[ax] = uint(w[ax])
	}
	if err := poserLesRegionsLocales(&e, modulePath, q); err != nil {
		return decfilm.MapQuantEntry{}, err
	}
	return e, nil
}
