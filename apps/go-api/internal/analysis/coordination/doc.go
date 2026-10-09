// Package coordination mesure ce qu'une equipe fait ENSEMBLE : l'appui recu (les assistances
// entre coequipiers, bloc des pages Sessions et Series temporelles), l'isolement des morts
// (onglet Tactique) et les vies passees pres d'un coequipier ou seul (Series temporelles).
//
// PUR : aucune I/O, aucun SQL, aucun reseau, aucun etat global. L'entree est ce que
// l'appelant a lu et projete (univers des matchs, table d'equipes par match, appuis,
// contextes de mort) ; ce paquet ne connait ni DuckDB ni le document de rejeu.
//
// UNE REGLE DE FORME : aucune fonction exportee de ce paquet ne rend un taux en float64 nu.
// Un taux voyage dans `domain.Couverture`, avec son compte brut, sa quantite par match et
// son drapeau d'echantillon faible. Garde-rail : `no_naked_rate_test.go`.
package coordination
