// Package memguard — LA SENTINELLE MEMOIRE D'UN DECODAGE DE FILM.
//
// Un film lourd peut faire grimper le tas de plusieurs gibioctets en quelques secondes. Ce paquet
// borne UN decodage par deux plafonds : un plafond SOUPLE (`debug.SetMemoryLimit`, le GC travaille
// plus fort a son approche) et un plafond DUR, echantillonne, au-dela duquel la sentinelle rend la
// main a l'appelant ([Arm], [ArmBytes]). Il mesure l'empreinte ([Footprint]) et retient le pic
// ([Guard.Peak]).
//
// # CE QUE CE PAQUET NE DECIDE PAS
//
// Il ne decide ni de l'arret du processus (l'appelant choisit sa doctrine dans le callback), ni
// QUAND on decode, ni combien de decodages tournent a la fois sur la machine, ni a quelle priorite
// CPU : le lanceur de processus, le verrou « un decodage a la fois » et la priorite vivent chez
// l'hote (ADR 0034).
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
// Elle mene a un arret du processus. C'est acceptable dans un processus qui ne tient aucun handle
// d'ecriture (un enfant dont l'artefact s'ecrit atomiquement). Ne JAMAIS l'armer dans un processus
// qui tient une base en ecriture : une mort brutale au milieu d'une transaction est exactement ce
// que la doctrine anti-corruption de l'hote interdit.
package memguard
