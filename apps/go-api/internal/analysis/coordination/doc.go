// Package coordination mesure ce qu'une equipe fait ENSEMBLE : l'echange (un coequipier
// abat le tueur de celui qui vient de tomber) et la couverture qui en decoule.
//
// PUR : aucune I/O, aucun SQL, aucun reseau, aucun etat global. L'entree est une liste de
// `domain.KillEvent` (matchID, tueur, victime, instant) et une table d'equipes par match,
// que l'appelant projette depuis ce qu'il a lu — `match_kill_events_latest` cote base. Ce
// paquet ne connait ni DuckDB ni le document de rejeu.
//
// PARTAGE PAR DEUX SURFACES, et c'est la raison de son existence : l'onglet Tactique sert
// l'echange en KPI par carte, la page Escouade le sert en graphes par session et par
// composition. Une seule mecanique, deux lectures — deux implementations divergeraient au
// premier ajustement de fenetre.
//
// UNE REGLE DE FORME : aucune fonction exportee de ce paquet ne rend un taux en float64 nu.
// Un taux voyage dans `domain.Couverture`, avec son compte brut, sa quantite par match et
// son drapeau d'echantillon faible. Garde-rail : `no_naked_rate_test.go`.
package coordination
