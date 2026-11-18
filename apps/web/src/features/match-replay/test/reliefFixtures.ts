/**
 * reliefFixtures.ts — LE RELIEF DU REJEU, FABRIQUÉ POUR LES TESTS du panneau de la vue : ce que
 * `useReplayRelief` rend, avec des défauts « sans relief » que chaque test surcharge. Rien ici n'asserte.
 */
import { vi } from 'vitest'

import type { FloorChip } from '../hooks/reliefLogic'
import type { ReplayRelief } from '../hooks/useReplayRelief'

/** Les trois étages de la carte factice de la maquette : « Bas » à 0, deux étages au-dessus. */
export const THREE_FLOORS: readonly FloorChip[] = [
  { rank: 0, z: 0, rings: 0 },
  { rank: 1, z: 2.2, rings: 1 },
  { rank: 2, z: 4.4, rings: 2 },
]

/** Un relief inerte : ni fond à proposer, ni traits, ni étage. */
export function makeReplayRelief(over: Partial<ReplayRelief> = {}): ReplayRelief {
  return {
    background: null,
    selected: 'rendu',
    options: [],
    meshPercent: null,
    lines: { available: false, show: false, onToggle: vi.fn() },
    floor: { state: { state: 'none' }, cut: null, selectAll: vi.fn(), selectRank: vi.fn(), setHeight: vi.fn(), follow: { available: false, on: false, onToggle: vi.fn() } },
    linesLayer: null,
    followStep: vi.fn(),
    ...over,
  }
}
