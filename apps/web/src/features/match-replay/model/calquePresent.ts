/**
 * calquePresent.ts — « CE CALQUE A-T-IL ÉTÉ PRODUIT ? », en un seul point (lot 4.2.2, schéma 62).
 *
 * # LA QUESTION, ET POURQUOI ELLE AVAIT BESOIN D'UN POINT UNIQUE
 *
 * Un tableau vide ne dit pas s'il est vide parce que le film ne portait rien, ou parce que
 * personne n'a regardé. Jusqu'au schéma 61, la réponse se JOIGNAIT : tableau absent + bloc de
 * couverture présent = « lu, rien trouvé » ; les deux absents = « pas lu ». Cette jointure
 * demandait de connaître la carte des blocs, et les deux nomenclatures ne coïncident pas —
 * `coverage.groundWeapons` mesure `padPickups`, `coverage.zones` mesure `zoneStates`.
 *
 * Depuis le schéma 62 le producteur répond directement : `layers[nom]` porte la révision de la
 * couche qui a produit le calque. Entrée présente = la passe a tourné ; entrée absente = elle
 * n'a pas tourné (garde de mode fermée, balayage non abouti).
 *
 * # LES TROIS RÉPONSES, ET POURQUOI IL EN FAUT TROIS
 *
 * L'objet `layers` peut être ABSENT — un artefact cuit avant 62. La question n'a alors pas de
 * réponse, et une fonction booléenne mentirait dans un sens ou dans l'autre : `false` se lirait
 * « le producteur a regardé et n'a rien produit », `true` affirmerait une production jamais
 * déclarée. `inconnu` est donc un état à part entière, exactement comme `unknown` du badge de
 * version (`replaySchemaStatusLogic.ts`) l'est pour la comparaison de versions.
 *
 * PUR : aucune dépendance React, aucun accès réseau. Le rendu ne change pas — ce module est la
 * LECTURE que `coverage` rendait laborieuse, et rien d'autre.
 */
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

/** Les trois réponses possibles à « ce calque a-t-il été produit ? ». */
export type CalquePresence = 'produit' | 'nonProduit' | 'inconnu'

/**
 * calquePresent dit si le calque nommé a été produit par la cuisson de ce document.
 *
 *  - `produit`     l'entrée existe : la passe a tourné, sous la révision que porte l'entrée ;
 *  - `nonProduit`  `layers` est là et l'entrée manque : la passe n'a PAS tourné. C'est une
 *                  réponse, pas un trou ;
 *  - `inconnu`     `layers` est absent : artefact antérieur au schéma 62, le producteur ne
 *                  répondait pas encore à cette question.
 *
 * `nom` est la CLÉ JSON du calque à la racine du document (`zoneStates`, `padPickups`, ...),
 * jamais le nom de son bloc de couverture : c'est le producteur qui a choisi cette
 * nomenclature, précisément parce que c'est celle que le lecteur manipule.
 */
export function calquePresent(doc: ReplayDocumentReady, nom: string): CalquePresence {
  const layers = doc.layers
  if (!layers) {
    return 'inconnu'
  }
  return layers[nom] === undefined ? 'nonProduit' : 'produit'
}

/**
 * revisionDuCalque rend la révision de la couche qui a produit ce calque, ou `undefined` quand
 * la question n'a pas de réponse (objet absent) ou que le calque n'a pas été produit.
 *
 * DEUX FONCTIONS ET NON UNE, et c'est la leçon de `computeReplaySchemaStatus` : mêler le
 * VERDICT (trois états) et la DONNÉE (une chaîne) forcerait chaque appelant à distinguer
 * `undefined` de `''`, ce que personne ne fait deux fois de la même façon.
 */
export function revisionDuCalque(doc: ReplayDocumentReady, nom: string): string | undefined {
  return doc.layers?.[nom]
}

/**
 * couchesDesCalquesProduits rend les RÉVISIONS DISTINCTES portées par ce document, triées.
 *
 * C'est ce que le badge par couche consomme : une montée de `grammar.Rev` périme d'un coup tous
 * les calques que cette couche produit, donc ce qui se nomme à l'écran est la COUCHE, jamais le
 * calque. Vide quand `layers` est absent — l'appelant distingue alors « aucune couche connue »
 * de « aucun calque produit » par `calquePresent`, qui porte les trois états.
 */
export function couchesDesCalquesProduits(doc: ReplayDocumentReady): string[] {
  const layers = doc.layers
  if (!layers) {
    return []
  }
  return [...new Set(Object.values(layers))].sort()
}
