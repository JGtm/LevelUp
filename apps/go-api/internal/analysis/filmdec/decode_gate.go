package filmdec

// decode_gate.go — UN SEUL decodage filmdec a la fois par process.
//
// POURQUOI CE VERROU VIT ICI, et pas chez un appelant : les parametres de replication du
// decodeur de bits sont des GLOBAUX DE PAQUET (SetAbsoluteAxisW, TraversalPrecision,
// SetRecordStateParam, ...), et la table d'observation compWidthObs
// (frame_chain_infer.go) est ecrite SANS verrou pendant un balayage. Deux decodages
// simultanes dans le meme process — un killsource et un rejeu 2D, ou deux rejeux — se
// contamineraient : c'est une course reelle, pas theorique.
//
// Chaque consommateur serialisait jusqu'ici DANS SON COIN (killsource.decodeMu, retire le
// 2026-08-13) : deux verrous locaux ne protegent rien l'un de l'autre. Ce verrou de
// paquet est LE point unique, partage par killsource.Decode et replay.BuildFromFilm.
//
// CONTRAT : tout chemin qui enchaine les balayages de ce paquet (Scan*, walk killsource)
// acquiert ce verrou pour TOUTE la duree du decodage d'un film — jamais par sous-appel,
// sinon deux films s'entrelacent entre deux sous-appels.

import "sync"

// processDecodeMu serialise tout decodage filmdec du process.
var processDecodeMu sync.Mutex

// LockProcessDecode acquiert le verrou process de decodage filmdec et rend la fonction de
// liberation. Usage : release := filmdec.LockProcessDecode(); defer release().
func LockProcessDecode() (release func()) {
	processDecodeMu.Lock()
	return processDecodeMu.Unlock
}
