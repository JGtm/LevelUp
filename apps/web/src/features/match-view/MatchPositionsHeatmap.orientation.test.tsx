/**
 * MatchPositionsHeatmap — « Occupation du terrain » lit l'orientation et l'interrupteur du plan des murs
 * DU REJEU (`replay-orientation.<map_id>`, `replay-show-wall-plan`), sans réglage propre : le cadre prend le
 * rapport de la carte tournée, le tracé reçoit les quarts de tour et le plan des murs — ou `null` quand
 * la carte n'en a pas ou que le rejeu l'a éteint.
 */
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { MatchPlayerPosition } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { peindreLesPositions } from './_positionsPaint'
import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'

const background = vi.hoisted(() => vi.fn())
const image = vi.hoisted(() => vi.fn())
vi.mock('@/lib/replay/queries', () => ({
  useReplayMapBackground: () => background(),
  useReplayMapImage: (...args: unknown[]) => image(...args),
}))
vi.mock('./queries', () => ({ useMatchView: () => ({ data: { header: { map_id: 'map-streets' } } }) }))
vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  Link: ({ children }: { children: React.ReactNode }) => <a href="#tactique">{children}</a>,
}))
vi.mock('./_positionsPaint', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./_positionsPaint')>()),
  peindreLesPositions: vi.fn(),
}))

// Un fond de 20 x 10 m : une carte allongée, dont le quart de tour échange les dimensions.
const CAL = { metersPerPixel: 1, originX: 0, originY: 10, widthPx: 20, heightPx: 10, convention: 'test' }
const PLAN = { edges: [[0, 0, 5, 5], null], faces: 1, source: 'navmesh', vertices: 4 }
const SAMPLE: MatchPlayerPosition[] = [
  { timeMs: 0, x: 1, y: 5, z: 0, team: 0 },
  { timeMs: 0, x: 1.2, y: 5.1, z: 0, team: 0 },
  { timeMs: 20000, x: 8, y: 2, z: 0, team: 1 },
]

function rendre() {
  return render(<MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={SAMPLE} locale="fr" />)
}

/** Le dernier appel du tracé : la toile et ce que le composant lui donne. */
function dernierTrace() {
  const [, width, height, peinture] = vi.mocked(peindreLesPositions).mock.calls.at(-1) ?? []
  return { width, height, peinture }
}

beforeEach(() => {
  window.localStorage.clear()
  background.mockReset().mockReturnValue({ data: { calibration: CAL, plan: PLAN } })
  image.mockReset().mockReturnValue({ width: 20, height: 10 })
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(100)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn() } as never)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLesPositions).mockClear()
})

describe('MatchPositionsHeatmap — orientation et plan des murs du rejeu', () => {
  it('par défaut : carte droite, plan des murs normalisé donné au tracé, cadre au rapport du fond', () => {
    rendre()
    const { peinture } = dernierTrace()
    expect(peinture?.turns).toBe(0)
    expect(peinture?.plan?.edges).toEqual([[0, 0, 5, 5]])
    expect(screen.getByTestId('match-positions-frame').style.aspectRatio).toBe('20 / 10')
  })

  it('l’orientation retenue au rejeu pour CETTE carte est appliquée : cadre au rapport tourné, quarts de tour au tracé', () => {
    window.localStorage.setItem('replay-orientation.map-streets', '1')
    rendre()
    expect(dernierTrace().peinture?.turns).toBe(1)
    expect(screen.getByTestId('match-positions-frame').style.aspectRatio).toBe('10 / 20')
  })

  it('à trois quarts de tour : même rapport tourné ; à deux, le rapport d’origine', () => {
    window.localStorage.setItem('replay-orientation.map-streets', '3')
    const a = rendre()
    expect(screen.getByTestId('match-positions-frame').style.aspectRatio).toBe('10 / 20')
    a.unmount()
    window.localStorage.setItem('replay-orientation.map-streets', '2')
    rendre()
    expect(screen.getByTestId('match-positions-frame').style.aspectRatio).toBe('20 / 10')
  })

  it('l’orientation d’une autre carte ne tourne pas celle-ci', () => {
    window.localStorage.setItem('replay-orientation.ridgeline', '3')
    rendre()
    expect(dernierTrace().peinture?.turns).toBe(0)
  })

  it('plan des murs éteint au rejeu : aucun plan donné au tracé', () => {
    window.localStorage.setItem('replay-show-wall-plan', 'false')
    rendre()
    expect(dernierTrace().peinture?.plan).toBeNull()
  })

  it('fond rendu par défaut : l’image est demandée en variante rendu', () => {
    background.mockReturnValue({ data: { calibration: CAL, plan: PLAN, planImage: 'plan.png' } })
    rendre()
    expect(image.mock.calls.at(-1)?.[3]).toBe('rendu')
    expect(dernierTrace().peinture?.plan?.edges).toEqual([[0, 0, 5, 5]])
  })

  it('réglage plan et carte avec image plan : variante plan demandée, murs déjà dans l’image donc aucun plan au tracé', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    background.mockReturnValue({ data: { calibration: CAL, plan: PLAN, planImage: 'plan.png' } })
    rendre()
    expect(image.mock.calls.at(-1)?.[3]).toBe('plan')
    expect(dernierTrace().peinture?.plan).toBeNull()
  })

  it('réglage plan mais carte sans image plan : repli sur le rendu, plan des murs tracé', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    rendre()
    expect(image.mock.calls.at(-1)?.[3]).toBe('rendu')
    expect(dernierTrace().peinture?.plan?.edges).toEqual([[0, 0, 5, 5]])
  })

  it('carte sans plan : aucun plan donné au tracé', () => {
    background.mockReturnValue({ data: { calibration: CAL } })
    rendre()
    expect(dernierTrace().peinture?.plan).toBeNull()
  })

  it('aucune commande propre : ni interrupteur de plan, ni commande d’orientation sur le bloc', () => {
    rendre()
    expect(screen.queryByRole('switch')).toBeNull()
    expect(screen.queryByRole('radiogroup')).toBeNull()
  })
})
