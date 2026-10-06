// Package coordination mesure ce qu'une equipe fait ENSEMBLE : l'echange (un coequipier
// abat le tueur de celui qui vient de tomber) et la couverture qui en decoule.
//
// PUR : aucune I/O, aucun SQL, aucun reseau, aucun etat global. L'entree est une liste de
// `domain.KillEvent` (matchID, tueur, victime, instant) et une table d'equipes par match,
// que l'appelant projette depuis ce qu'il a lu — `match_kill_events_latest` cote base. Ce
// paquet ne connait ni DuckDB ni le document de rejeu.
//
// PARTAGE PAR PLUSIEURS SURFACES, et c'est la raison de son existence : la page Escouade sert
// l'echange en graphes par session et par composition, la vue match le sert en bloc « Riposte »,
// le bloc Coordination porte l'appui recu sur les Sessions et les Series temporelles, et l'onglet
// Tactique y lit l'isolement de ses morts. Une seule mecanique — deux implementations
// divergeraient au premier ajustement de fenetre.
//
// UNE REGLE DE FORME : aucune fonction exportee de ce paquet ne rend un taux en float64 nu.
// Un taux voyage dans `domain.Couverture`, avec son compte brut, sa quantite par match et
// son drapeau d'echantillon faible. Garde-rail : `no_naked_rate_test.go`.
package coordination
