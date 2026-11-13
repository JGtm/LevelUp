package main

// cadence_derives.go — LE PAS COMME MARQUE DE FRAÎCHEUR, VU DES PASSES QUI NE CUISENT RIEN.
//
// La source unique du pas de cuisson et de son critère de fraîcheur est `replaybuild` (cadence.go). Les
// passes qui ne font que PROJETER des artefacts déjà rangés (`tactical-rasters`) ne nomment pas ce
// paquet : leur ratchet (`TestAucuneCuissonDepuisLeTactique`) interdit tout fichier tactique qui le
// nomme, pour qu'aucune cuisson ne devienne atteignable depuis la page ou son rattrapage. Ces deux
// fonctions sont le seul pont : elles ne lisent que des nombres, et appellent la source unique.

import "levelup/go-api/internal/replaybuild"

// aLaCadenceCourante dit si un dérivé a été projeté d'un artefact au pas de cuisson courant.
func aLaCadenceCourante(frameIntervalMS int) bool {
	return replaybuild.CadenceUpToDate(frameIntervalMS)
}

// memeCadence dit si un dérivé et l'artefact dont il vient sont sur la même grille.
func memeCadence(derive, artefact int) bool { return replaybuild.SameCadence(derive, artefact) }
