/**
 * useReplayOrientation — L'ORIENTATION DE LA CARTE, retenue PAR CARTE.
 *
 * Le nombre de quarts de tour vit sous `replay-orientation.<mapId>` (`replayPreferences.ts`) : une
 * carte tournée l'est à chaque ouverture, les autres ne bougent pas. Le choix a TROIS ÉTATS :
 * aucun choix retenu (`retained` vaut `null` : « Auto », la règle `autoTurns` que `useReplayView`
 * applique), ou 0, 1, 2 ou 3 quarts de tour. Ce hook ne dit que ce que l'utilisateur a retenu ; la
 * position EFFECTIVE de la carte, Auto comprise, se résout dans `useReplayView`.
 *
 * Sans identifiant de carte (page en chargement, match sans carte connue) il n'y a rien sous quoi
 * retenir un choix, donc aucun retour possible d'un plan tourné par l'Auto : `retained` vaut 0 (la
 * carte reste droite) et la commande n'est pas proposée (`available`) — même règle que les calques
 * sans matière, une commande qui ne commande rien tromperait plus qu'elle n'informerait.
 *
 * LE CHANGEMENT DE CARTE RELIT LE STOCKAGE PENDANT LE RENDU (jamais depuis un effet) : l'état
 * garde la clé pour laquelle il a été lu, et la relit dès qu'elle change.
 */
import { useCallback, useState } from 'react'

import type { QuarterTurns } from '@/lib/replay/orientation'
import { clearOrientation, persistOrientation, readOrientation } from './replayPreferences'

export interface ReplayOrientation {
  /** Le choix retenu pour la carte, en quarts de tour horaires ; `null` = aucun choix (Auto). */
  retained: QuarterTurns | null
  /** Retient une orientation pour la carte courante, ou `null` pour l'effacer (retour en Auto). Sans effet sans carte. */
  setTurns: (turns: QuarterTurns | null) => void
  /** Vrai quand la carte est connue : la commande n'a de sens que pour elle. */
  available: boolean
}

export function useReplayOrientation(mapId: string | null | undefined): ReplayOrientation {
  const id = mapId || null
  const read = (key: string | null): QuarterTurns | null => (key ? readOrientation(key) : 0)
  const [state, setState] = useState<{ id: string | null; retained: QuarterTurns | null }>(() => ({ id, retained: read(id) }))
  let current = state
  if (state.id !== id) {
    current = { id, retained: read(id) }
    setState(current)
  }
  const setTurns = useCallback(
    (turns: QuarterTurns | null) => {
      if (!id) return
      setState({ id, retained: turns })
      if (turns === null) clearOrientation(id)
      else persistOrientation(id, turns)
    },
    [id],
  )
  return { retained: current.retained, setTurns, available: id !== null }
}
