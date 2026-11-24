/**
 * soundFixtures.ts — LE SON DE LA PAGE, FABRIQUÉ POUR LES TESTS du panneau « Vue » : un son sans piste, ou un son
 * avec piste dont les gestes s'espionnent. Rien ici n'asserte.
 */
import { vi } from 'vitest'

import type { ReplaySound } from '../sound/useReplaySound'

/** Un son avec piste, coupé, dont les gestes s'espionnent. */
export function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
  return {
    available: true, on: false, toggle: vi.fn(), volume: 0.7, setVolume: vi.fn(),
    categories: { weapon: true, grenade: true, melee: true, equipment: true, objective: true },
    toggleCategory: vi.fn(),
    ...over,
  } as unknown as ReplaySound
}

/** Un son sans piste : ni commande de son, ni catégorie. */
export const NO_SOUND = makeSound({ available: false })
