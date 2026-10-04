// Package signaux lit dans le film les SIGNAUX DE SCORE ET D'OBJECTIF que la marche ne lit pas : les
// enregistrements d'entite du statborg ([LireLeStatborg]), les evenements d'objectif du pied de
// film ([FooterEvents]) et les rafales de capture de drapeau ([CaptureBurstTimes]).
//
// # UNE METHODE DE RECUPERATION DE LA GRAMMAIRE (ADR 0037 IR-6)
//
// Ces lectures ne traversent pas la boucle de records : elles LOCALISENT ce qu'elles lisent, par
// les contraintes mesurees d'un en-tete (statborg), par un motif d'octets (xuid et marqueur de fin
// du pied, tetes de l'echelle des tiers). Ce qu'elles rendent est un type a part, jamais mele a ce
// que la marche a lu, et leurs replis se comptent dans ce qu'elles rendent. La couche des faits
// (`facts/objectives`) les consomme et ne lit aucun octet (ADR 0037, D-2 amende).
//
// # POURQUOI UN SOUS-PAQUET, ET POURQUOI UNE FEUILLE
//
// Il appartient a la couche `grammar` par son arbre : sa forme fait monter `grammar.Rev`, et la
// couche des objectifs le voit par cette valeur. Mais il n'importe que `source`, `film/types` et
// `film/finalise`, jamais `grammar` : des instruments de test de `grammar` importent
// `facts/objectives` pour leurs oracles, et un `facts/objectives` qui importerait `grammar` fermerait
// un cycle d'imports dans leurs binaires de test.
package signaux
