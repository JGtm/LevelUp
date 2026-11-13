/**
 * useZoomResetOnTurns — UN CHANGEMENT D'ORIENTATION REVIENT À LA CARTE ENTIÈRE.
 *
 * Le palier et le centre du zoom d'une orientation ne désignent plus le même endroit dans la
 * suivante : le zoom raisonne dans le repère de la carte tournée. Le rejeu, l'onglet Tactique et
 * « Occupation du terrain » partagent donc cette règle, écrite ici une seule fois.
 *
 * L'état du zoom se corrige PENDANT LE RENDU (jamais depuis un effet) : le hook garde les quarts de
 * tour déjà vus et appelle `reset` — un `setState` du même composant — quand ils changent.
 */
import { useState } from 'react'

import type { QuarterTurns } from './orientation'

/** Appelle `reset` (le retour à 1x du zoom) dès que `turns` change ; sans effet tant qu'il est stable. */
export function useZoomResetOnTurns(turns: QuarterTurns, reset: () => void): void {
  const [seen, setSeen] = useState(turns)
  if (seen !== turns) {
    setSeen(turns)
    reset()
  }
}
