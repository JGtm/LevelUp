//go:build research

// Commande cmd_mouvement — la passe du lot 5.3 sur `HaloInfinite.exe`, en LECTURE SEULE.
//
//	MOUV_EXE="<bibliotheque>/<jeux>/common/Halo Infinite/HaloInfinite.exe" \
//	  go run -tags=research ./internal/games/halo_infinite/film/research/cmd_mouvement
//
// Sans `MOUV_EXE`, la commande ne fait rien et le dit : un instrument de recherche ne devine
// jamais le chemin d'une installation de jeu.
//
//	-vocabulaire  balaye le pool COMPLET des chaines de l'image pour le vocabulaire du
//	              mouvement, au lieu de resoudre les cibles. C'est la forme du NEGATIF.
//	-plafond      nombre de chaines recopiees par mot (les totaux, eux, restent complets).
package main

import (
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/games/halo_infinite/film/research/mouvement"
)

func main() {
	vocab := flag.Bool("vocabulaire", false, "balayer le pool complet des chaines")
	plafond := flag.Int("plafond", 40, "chaines recopiees par mot")
	flag.Parse()

	exe := os.Getenv("MOUV_EXE")
	if exe == "" {
		fmt.Fprintln(os.Stderr, "MOUV_EXE absent : chemin de HaloInfinite.exe attendu")
		os.Exit(2)
	}
	var err error
	if *vocab {
		err = mouvement.Vocabulaire(exe, *plafond, os.Stdout)
	} else {
		err = mouvement.Passe(exe, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ECHEC :", err)
		os.Exit(1)
	}
}
