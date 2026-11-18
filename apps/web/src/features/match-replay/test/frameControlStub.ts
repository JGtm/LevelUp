/**
 * frameControlStub — UN `ReplayFrameControl` DE TEST : le cadre donné, des gestes qui sont des espions.
 * `peek` rend l'état du cadre donné (Plein cadre : ouvert depuis le Standard). Pas d'enchaînement des
 * transitions ici : ce qui s'enchaîne est le modèle (`replayFrame.test.ts`), pas ce double.
 */
import { vi } from 'vitest'

import type { ReplayFrameControl } from '../hooks/useReplayFrame'
import type { ReplayFrame } from '../model/replayFrame'

export function makeFrameControl(frame: ReplayFrame = 'standard', over: Partial<ReplayFrameControl> = {}): ReplayFrameControl {
  return {
    frame,
    disabled: false,
    toggleCinema: vi.fn(),
    toggleFullFrame: vi.fn(),
    exit: vi.fn(),
    peek: () => ({ frame, returnTo: frame === 'plein-cadre' ? 'standard' : frame }),
    ...over,
  }
}
