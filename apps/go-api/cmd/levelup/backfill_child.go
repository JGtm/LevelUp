package main

// backfill_child.go — CE QUI RESTE DU RUNNER PARENT/ENFANT DES PASSES DE CUISSON : rien, et
// c'est le but.
//
// # POURQUOI CE MOTIF EXISTE : LE 2026-08-20, UNE PASSE A SATURE LA MACHINE
//
// `backfill-replay --only-existing` (29 films) a cuit QUATRE petits films (8 a 13 chunks, tous
// valides) puis s'est effondree sur le cinquieme : plus de six heures de spirale GC, puis
// `runtime.preemptM: duplicatehandle failed; errno=1450` — ERROR_NO_SYSTEM_RESOURCES, le
// runtime Go n'obtenait plus meme un handle de thread de Windows. Ce n'est pas UN film qui est
// trop gros : c'est le PROCESSUS UNIQUE qui empile les pics de tous les films et ne rend jamais
// rien a l'OS. La regle (doctrine machine D17) : UN FILM = UN PROCESSUS.
//
// # LE LANCEUR EST DESORMAIS `internal/filmproc` (PLAN_CUISSON_PERF item 5.4, 2026-09-03)
//
// Ce fichier portait sa PROPRE copie du motif : codes de sortie, categories d'issue, marqueur
// de pic memoire, relais de sortie, environnement de l'enfant. `internal/filmproc` porte
// EXACTEMENT le meme protocole, aux memes valeurs (0 / 10 / 11 / 12 / 13, marqueur
// `__levelup_pic_octets__=`), et il est deja le lanceur du post-sync, du harnais d'equivalence
// et de la passe d'attribution de zones. Deux copies d'un protocole finissent par diverger sur
// un code, et la divergence se lit alors comme une « mort subite » — la categorie fourre-tout.
//
// LA MIGRATION APPORTE EN PLUS LA PRIORITE CPU BASSE, qui manquait ici : `filmproc.Runner`
// lance ses enfants en priorite basse (lecon du 2026-08-26 — une passe a priorite normale rend
// la machine de travail inutilisable meme quand sa memoire est bornee). La passe de backfill
// n'en beneficiait pas ; elle en beneficie maintenant sans qu'on ait rien a regler.
//
// Il ne reste donc ici que le drapeau repetable des identites de carte, qui n'a rien a voir
// avec le lancement de processus et n'avait nulle part de mieux ou vivre.

import "strings"

// listeDrapeau : un drapeau REPETABLE (`--map-name A --map-name B`).
//
// Un drapeau unique a separateur serait plus court, mais les identites de carte sont des
// libelles libres : aucun separateur n'est sur. La repetition, elle, l'est.
type listeDrapeau []string

func (l *listeDrapeau) String() string { return strings.Join(*l, ", ") }

func (l *listeDrapeau) Set(v string) error {
	*l = append(*l, v)
	return nil
}
