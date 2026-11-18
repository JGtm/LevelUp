/**
 * useAutoTurned — L'ÉTAT DE LA RÈGLE AUTO d'une vue de la carte : le plan est-il tourné d'un quart de tour par
 * l'Auto (`autoTurns`, `orientation.ts`) sur la boîte que la vue offre à la carte ?
 *
 * C'EST LE SEUL ENDROIT QUI APPLIQUE LA RÈGLE (`useAutoTurned.guard.test.ts`) : le rejeu (`useResolvedTurns`), la
 * Tactique et « Occupation du terrain » (`useMapViewPreferences`) l'appellent chacun avec SA boîte. Rien n'est
 * écrit dans les préférences : l'Auto se réévalue à chaque mesure.
 *
 * L'HYSTÉRÉSIS DE LA RÈGLE a besoin de savoir si le plan est DÉJÀ tourné : le hook en retient l'état, calculé
 * pendant le rendu (jamais depuis un effet, qui ferait peindre une image à la mauvaise orientation). Cet état
 * est RANGÉ AVEC SA PORTÉE (`scopeKey`) : une autre portée repart du plan droit. Un choix retenu l'efface : une
 * carte qui repasse en Auto repart du plan droit.
 *
 * `gele` fige la sortie sur l'état d'avant : la règle n'est pas évaluée et rien n'est rangé (le rejeu gèle
 * l'Auto pendant un export). Sans carte exploitable (`null`) ou sans boîte mesurée, la règle rend le plan droit.
 */
import { useState } from 'react'

import { autoTurns, type Dimensions, type QuarterTurns } from './orientation'

/** Ce que la règle lit d'une vue. */
export interface AutoTurnedInput {
  /** La portée de l'état retenu (un match, une carte) : une autre portée repart du plan droit. */
  scopeKey: string
  /** Le choix retenu pour la carte ; `null` = aucun choix : l'Auto décide. Un choix l'emporte toujours. */
  choice: QuarterTurns | null
  /** La carte posée DROITE, en unités monde ; `null` = inconnue (plan droit). */
  carte: Dimensions | null
  /** La boîte que la vue offre à la carte, en px CSS ; une dimension nulle = pas encore mesurée. */
  boite: Dimensions
  /** Vrai = sortie figée sur l'état d'avant. */
  gele?: boolean
}

/** L'état « tourné par l'Auto », rangé avec la portée où il a été décidé. */
interface Scoped {
  scopeKey: string
  turned: boolean
}

/** Vrai quand l'Auto tourne le plan d'un quart de tour ; jamais pour un choix retenu. */
export function useAutoTurned({ scopeKey, choice, carte, boite, gele = false }: AutoTurnedInput): boolean {
  const [stored, setStored] = useState<Scoped>({ scopeKey, turned: false })
  const turnedBefore = stored.scopeKey === scopeKey && stored.turned
  let turned = turnedBefore
  if (choice !== null) turned = false
  else if (!gele) turned = carte !== null && autoTurns(carte, boite, turnedBefore) === 1
  if (!gele && (stored.scopeKey !== scopeKey || stored.turned !== turned)) setStored({ scopeKey, turned })
  return turned
}
