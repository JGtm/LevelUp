/**
 * useNowWhile — l'heure courante (ms), remise à jour chaque seconde jusqu'à l'instant `untilMs` : sert un
 * compte à rebours. Sans échéance (`undefined`) aucune minuterie ne tourne ; la minuterie s'arrête seule à
 * l'échéance, après un dernier relevé qui la dépasse.
 *
 * L'heure est un magasin externe lu par `useSyncExternalStore` : l'abonnement relève l'heure au moment où il
 * s'établit (la valeur rendue n'est pas celle d'avant l'activation) puis à chaque seconde.
 */
import { useCallback, useSyncExternalStore } from 'react'

let snapshot = Date.now()

function getSnapshot(): number {
  return snapshot
}

export function useNowWhile(untilMs: number | undefined): number {
  const subscribe = useCallback(
    (onChange: () => void) => {
      snapshot = Date.now()
      if (untilMs === undefined) return () => undefined
      const id = window.setInterval(() => {
        snapshot = Date.now()
        onChange()
        if (snapshot >= untilMs) window.clearInterval(id)
      }, 1000)
      return () => window.clearInterval(id)
    },
    [untilMs],
  )
  return useSyncExternalStore(subscribe, getSnapshot)
}
