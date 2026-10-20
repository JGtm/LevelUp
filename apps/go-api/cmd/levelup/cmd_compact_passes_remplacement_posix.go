//go:build !windows

package main

// cmd_compact_passes_remplacement_posix.go — le remplacement de la base par le fichier réécrit,
// hors Windows (POSIX ; cf. cmd_compact_passes_rewrite.go, « LA MÉTHODE »).
// Contrainte de plateforme EXPLICITE (ligne `//go:build`), jamais par le nom : un suffixe `_windows`
// / `_linux` serait une contrainte implicite (archlint `TestAucunSuffixeDePlateformeAccidentel`).
//
// `rename(2)` remplace le NOM de façon atomique pendant que la connexion est encore ouverte :
// l'ancien inode reste verrouillé par elle jusqu'à la fermeture, et un processus qui ouvre le
// chemin après le rename ouvre le fichier neuf. Aucun intervalle sans verrou sur l'ancienne base.

import (
	"errors"
	"fmt"
)

// remplacerSousVerrou rend (true, …) dès que la base a été remplacée.
func remplacerSousVerrou(r *reecriture) (bool, error) {
	if err := verifierWALAvantRemplacement(r); err != nil {
		return false, err
	}
	if err := renommer(r.neuf, r.path); err != nil {
		return false, fmt.Errorf("réécriture refusée : remplacement de %s: %w", r.path, err)
	}
	if err := passerEtape(etapeApresRemplacement); err != nil {
		return true, errors.Join(err, r.fermerSource())
	}
	return true, r.fermerSource()
}
