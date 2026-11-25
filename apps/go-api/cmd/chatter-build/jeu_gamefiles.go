//go:build gamefiles

package main

// jeu_gamefiles.go — L'OUVERTURE DU JEU INSTALLE : la seule porte de la commande vers l'installation.
// Sans le tag `gamefiles`, `jeu_absent.go` la remplace par un refus qui dit quoi faire.

import (
	"github.com/JGtm/halo-infinite-files-reader/himodule"
	"github.com/JGtm/halo-infinite-map-images/himap"
)

// racineDuJeu rend la racine `deploy` : celle de `-deploy`, sinon l'installation détectée (variable
// d'environnement de la brique, puis emplacements connus).
func racineDuJeu(explicite string) (string, error) {
	if explicite != "" {
		return explicite, nil
	}
	return himap.DeployRoot()
}

// ouvrirModule ouvre un module du jeu (projeté en mémoire, jamais copié).
func ouvrirModule(chemin string) (sourceModule, error) {
	return himodule.Open(chemin)
}
