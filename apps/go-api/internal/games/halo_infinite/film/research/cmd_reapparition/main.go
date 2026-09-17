//go:build research

// Commande cmd_reapparition — la passe du lot 3.7 sur `HaloInfinite.exe`, en LECTURE SEULE.
//
//	REAP_EXE="D:/SteamLibrary/steamapps/common/Halo Infinite/HaloInfinite.exe" \
//	  go run -tags=research ./internal/games/halo_infinite/film/research/cmd_reapparition
//
// Sans `REAP_EXE`, la commande ne fait rien et le dit : un instrument de recherche ne devine
// jamais le chemin d'une installation de jeu.
//
//	-inventaire   balaye l'univers des noms de composant de l'image pour le vocabulaire de la
//	              reapparition, au lieu de resoudre les cibles. C'est la forme du NEGATIF.
package main

import (
	"flag"
	"fmt"
	"os"

	"levelup/go-api/internal/games/halo_infinite/film/research/reapparition"
)

func main() {
	inventaire := flag.Bool("inventaire", false, "balayer l'univers des noms de composant")
	flag.Parse()

	exe := os.Getenv("REAP_EXE")
	if exe == "" {
		fmt.Fprintln(os.Stderr, "REAP_EXE absent : chemin de HaloInfinite.exe attendu")
		os.Exit(2)
	}
	var err error
	if *inventaire {
		mots := []string{"timer", "respawn", "spawn", "dispenser", "return", "reset", "duration", "delay", "objective", "vehicle"}
		err = reapparition.Inventaire(exe, mots, os.Stdout)
	} else {
		err = reapparition.Passe(exe, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ECHEC :", err)
		os.Exit(1)
	}
}
