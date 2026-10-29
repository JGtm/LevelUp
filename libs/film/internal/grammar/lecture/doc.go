// Package lecture porte les TYPES DE LA REPRÉSENTATION INTERMÉDIAIRE DU FILM : ce que la marche
// de la grammaire a lu, rangé aux niveaux stables du format — paquet, vues, records, composants —
// chacun avec son étendue en bits (ADR 0037).
//
// # CE QUE LE PAQUET EST
//
// Une FEUILLE de la couche `grammar` : des types et des constantes, aucune fonction, aucune
// variable, et aucun import du dépôt sauf `film/types`. La marche qui remplit ces types vit dans
// `grammar` ; les canaux qui les lisent vivent chez eux. Parce qu'il est dans l'arbre de la
// grammaire, un changement de sa forme fait monter `grammar.Rev`, et elle seule (ADR 0037 IR-9).
// Ni la couche de publication (`replay`) ni la façade (`decfilm`) ne l'importent.
// Garde-rails : `internal/archlint/film_lecture_test.go` et
// `internal/archlint/film_layers_deps_test.go`.
//
// # DURÉE DE VIE : UNE ARÈNE PAR PAQUET
//
// Un [Paquet] est rendu par la marche le temps d'un tour d'itération, puis RÉUTILISÉ pour le
// paquet suivant : ses tranches (records, composants, entrées de la vue C, genres de la vue A)
// sont remises à longueur nulle et réécrites, et son payload est une sous-tranche du chunk
// décompressé, jamais une copie. Ce qu'un consommateur veut garder au-delà du tour, il le copie.
//
// # LES ÉTENDUES
//
// Une position se compte en bits depuis le début du payload de SON paquet. Dans la structure, un
// élément porte son début et sa longueur ; le paquet porte le chunk et le rang. [Etendue] réunit
// les quatre pour CITER une lecture hors de la structure (provenance, ADR 0037 IR-1).
//
// # LES VALEURS ZÉRO
//
// Pour une énumération que la marche renseigne à chaque occurrence ([Etat], [Genre], [Preuve],
// [ProvenanceLargeur]), la valeur zéro est une sentinelle « non renseigné » : un test peut ainsi
// voir l'oubli. Pour les autres, la valeur zéro a un sens écrit à sa constante (aucune cause,
// vue non lue, verdict non rendu, liaison absente).
//
// # LES RANGS DE VUE
//
// Les rangs sont ceux DU FILM (le registraire `FUN_141f855b4`) : 0 la vue A des messages, 1 la vue
// B des entités, 2 la vue C de contrôle. La table d'entités de la marche hors ligne numérote la
// vue B 0 ; la conversion se fait dans `grammar`, jamais ici.
package lecture
