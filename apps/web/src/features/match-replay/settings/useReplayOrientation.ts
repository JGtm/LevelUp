/**
 * useReplayOrientation — L'ORIENTATION DE LA CARTE, retenue PAR CARTE.
 *
 * Le nombre de quarts de tour vit sous `replay-orientation.<mapId>` (`replayPreferences.ts`) : une
 * carte tournée l'est à chaque ouverture, les autres ne bougent pas. Sans identifiant de carte
 * (page en chargement, match sans carte connue) il n'y a rien sous quoi retenir un choix : 0 quart
 * de tour, et la commande n'est pas proposée (`available`) — même règle que les calques sans
 * matière, une commande qui ne commande rien tromperait plus qu'elle n'informerait.
 *
 * LE CHANGEMENT DE CARTE RELIT LE STOCKAGE PENDANT LE RENDU (jamais depuis un effet) : l'état
 * garde la clé pour laquelle il a été lu, et la relit dès qu'elle change.
 */
import { useCallback, useState } from 'react'

import type { QuarterTurns } from '../model/orientation'
import { persistOrientation, readOrientation } from './replayPreferences'

export interface ReplayOrientation {
  /** L'orientation courante, en quarts de tour antihoraires. */
  turns: QuarterTurns
  /** Retient une orientation pour la carte courante (sans effet sans carte). */
  setTurns: (turns: QuarterTurns) => void
  /** Vrai quand la carte est connue : la commande n'a de sens que pour elle. */
  available: boolean
}

export function useReplayOrientation(mapId: string | null | undefined): ReplayOrientation {
  const id = mapId || null
  const read = (key: string | null): QuarterTurns => (key ? readOrientation(key) : 0)
  const [state, setState] = useState<{ id: string | null; turns: QuarterTurns }>(() => ({ id, turns: read(id) }))
  let current = state
  if (state.id !== id) {
    current = { id, turns: read(id) }
    setState(current)
  }
  const setTurns = useCallback(
    (turns: QuarterTurns) => {
      if (!id) return
      setState({ id, turns })
      persistOrientation(id, turns)
    },
    [id],
  )
  return { turns: current.turns, setTurns, available: id !== null }
}
