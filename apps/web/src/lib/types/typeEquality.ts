/**
 * typeEquality.ts — L'ÉGALITÉ STRICTE DE DEUX TYPES, une fois pour toutes.
 *
 * POURQUOI CE FICHIER EXISTE (2026-09-13, lot 0.B du chantier décodeur de film). Le motif
 * `(<T>() => T extends A ? 1 : 2) extends (<T>() => T extends B ? 1 : 2)` — le seul qui
 * distingue vraiment deux types en TypeScript, `any` et les modificateurs optionnels compris —
 * était écrit deux fois (`model/replayContract.test.ts`, `layers/deltaLayersContract.guard.test.ts`).
 * Le contrat à l'exécution du document de rejeu en réclamait une TROISIÈME : c'est le seuil de
 * la règle n° 6 du dépôt (« ≤ 2 copies ; à la 3e, centraliser ET poser un garde-rail »).
 *
 * LE GARDE-RAIL EST `typeEquality.guard.test.ts` : il interdit de réécrire le motif ailleurs.
 * Sans lui, la factorisation re-diverge — c'est la leçon écrite du dépôt (un prédicat passé de
 * 8 à 36 copies APRÈS sa centralisation).
 *
 * CE QUE CES DEUX TYPES FONT. `Equals<A, B>` vaut `true` si les deux types sont strictement
 * identiques ; `Expect<T extends true>` transforme cette valeur en CONTRAINTE DE COMPILATION :
 * `type _ = Expect<Equals<A, B>>` fait tomber `tsc -b`, donc la CI, avant même qu'un test ne
 * s'exécute. C'est ce qui permet de prouver une exhaustivité contre un contrat généré.
 */

/** Égalité STRICTE de deux types (le double conditionnel différé est ce qui la rend stricte). */
export type Equals<A, B> = (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
  ? true
  : false

/** Transforme une égalité de types en contrainte de compilation. */
export type Expect<T extends true> = T
