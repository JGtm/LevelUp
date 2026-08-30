// Package filmproc — L'EXECUTEUR CANONIQUE « UN FILM = UN PROCESSUS BORNE ».
//
// # POURQUOI CE PAQUET EXISTE : TROIS SINISTRES, LE MEME MECANISME
//
//	2026-08-20  `backfill-replay --only-existing` cuit quatre petits films puis s'effondre sur
//	            le cinquieme : six heures de spirale GC, puis `errno=1450`
//	            (ERROR_NO_SYSTEM_RESOURCES) — le runtime Go n'obtenait plus un handle de thread
//	            de Windows. Remede : un processus par film (cmd/levelup/backfill_child.go).
//	2026-08-24  `51101d1d` mesure 7,9 Go en 2,6 s a travers `replaybuild.BuildMatch`. Remede :
//	            une sentinelle memoire a deux plafonds (souple + dur).
//	2026-08-26  la mesure D3 de Total Control sature la machine DE TRAVAIL de l'utilisateur —
//	            deux fois. D'abord sept films BTB dans UN processus ; puis, apres correction,
//	            un film par processus mais SANS plafond ni priorite basse. Un seul film BTB a
//	            travers la construction d'artefact suffit a prendre le poste en otage.
//
// La lecon de 2026-08-26 est celle qui a fait naitre ce paquet : « un film = un processus » est
// NECESSAIRE ET PAS SUFFISANT. Il faut les trois ensemble — un processus par film, un PLAFOND
// DUR par processus, et une PRIORITE CPU BASSE pour que la machine reste utilisable pendant
// que la mesure tourne.
//
// # POURQUOI UN PAQUET PARTAGE PLUTOT QU'UNE TROISIEME COPIE
//
// La sentinelle existait en DEUX exemplaires — `cmd/levelup/backfill_memlimit.go` et
// `cmd/replay-worker/memlimit.go` — avec, en tete du second, la justification de la
// duplication : deux paquets `main`, Go n'en importe pas, et « ~80 lignes de mesure de tas »
// ne valaient pas un paquet interne. La regle du depot (CLAUDE.md n°6) tranche a la
// TROISIEME copie : `zone-attribution` en aurait eu besoin d'une, et c'est ce qui rend la
// centralisation due plutot qu'elegante. Les deux `main` importent desormais ce paquet ; leurs
// doctrines d'ARRET, elles, restent distinctes et le sont par un CALLBACK (cf. Guard).
//
// # CE QUE CE PAQUET NE PEUT PAS FAIRE
//
// La sentinelle ECHANTILLONNE : elle ne peut pas interrompre une allocation DEJA EN VOL. Un
// `make([]T, n)` de plusieurs gibioctets d'un seul tenant passe sous les deux plafonds — le
// souple ne refuse rien, la sentinelle ne voit le resultat qu'au tick suivant. Le processus
// meurt donc en QUELQUES SECONDES au lieu de quelques heures, et la machine survit, mais le
// pic transitoire a bien eu lieu. Le cran suivant, si un jour il le faut, n'est pas un
// reglage : c'est un Job Object Windows (`ProcessMemoryLimit`) pose sur l'enfant, qui fait
// ECHOUER l'allocation au lieu de la laisser aboutir.
//
// # LA SENTINELLE NE VIT QUE DANS UN PROCESSUS QUI NE TIENT AUCUNE ECRITURE
//
// Elle mene a un arret du processus. C'est acceptable dans un ENFANT qui ne tient aucun handle
// d'ecriture : la passe relache ses handles DuckDB AVANT tout decodage, et l'artefact s'ecrit
// atomiquement. Ne JAMAIS l'armer dans un processus qui tient une base en ecriture — une mort
// brutale au milieu d'une transaction est exactement ce que la doctrine anti-corruption
// interdit (ADR 0013/0019/0030).
package filmproc
