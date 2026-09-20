/**
 * MatchPositionsHeatmap.test.tsx — « Occupation du terrain » : ses trois portes et son filtre.
 *
 * Le tracé lui-même (grille, échelle, projection) est testé PUR dans `_positionsHeat.test.ts`
 * et dans `lib/replay/heatPaint.test.ts` ; ici on vérifie ce que le composant DÉCIDE : pas de
 * position, pas de fond de carte, ou rien qui tombe sur le plan -> la carte ne s'affiche pas.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { MatchPlayerPosition } from '@/lib/api/types'

import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'

const background = vi.hoisted(() => vi.fn())
const image = vi.hoisted(() => vi.fn())

vi.mock('@/lib/replay/queries', () => ({
  useReplayMapBackground: () => background(),
  useReplayMapImage: () => image(),
}))

const CAL = {
  metersPerPixel: 1,
  originX: 0,
  originY: 10,
  widthPx: 10,
  heightPx: 10,
  convention: 'test',
}

const sample: MatchPlayerPosition[] = [
  { timeMs: 0, x: 1, y: 5, z: 0, team: 0 },
  { timeMs: 0, x: 1.2, y: 5.1, z: 0, team: 0 },
  { timeMs: 20000, x: 8, y: 2, z: 0, team: 1 },
]

function withBackground() {
  background.mockReturnValue({ data: { calibration: CAL } })
  image.mockReturnValue(null)
}

beforeEach(() => {
  background.mockReset()
  image.mockReset()
  withBackground()
})

describe('MatchPositionsHeatmap', () => {
  it('rend la carte et son narratif quand le match a un fond et des positions', () => {
    render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={sample} locale="fr" />,
    )
    expect(screen.getByText('Occupation du terrain')).toBeTruthy()
    expect(screen.getByText(/plus c’est chaud/)).toBeTruthy()
    expect(screen.getByTestId('match-positions-canvas')).toBeTruthy()
  })

  it('ne publie AUCUNE note de méthode en pied (retrait du 2026-09-19)', () => {
    render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={sample} locale="fr" />,
    )
    // Le pas de grille, la part des positions dans le cadre et le mode d'attribution des
    // camps décrivaient la mesure, pas le match : ils ont été retirés (lot 2 du plan).
    expect(screen.queryByText(/Grille de|regroupement spatial/)).toBeNull()
  })

  it('borne la HAUTEUR du plan : la largeur du cadre ne dépasse jamais 60 % de la carte', () => {
    render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={sample} locale="fr" />,
    )
    // Le cadre gardait le rapport du monde sur TOUTE la largeur de la carte : un plan carré
    // y faisait un pavé aussi haut que large (lot 2, « hauteur réduite d'au moins 40 % »).
    const cadre = screen.getByTestId('match-positions-frame')
    expect(cadre.getAttribute('style')).toContain('min(60%')
  })

  it('propose le filtre par camp quand au moins une position porte un camp', () => {
    render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={sample} locale="fr" />,
    )
    expect(screen.getByText('Tous')).toBeTruthy()
    expect(screen.getByText('Camp A')).toBeTruthy()
    expect(screen.getByText('Camp B')).toBeTruthy()
  })

  it('masque le filtre quand aucune position ne porte de camp', () => {
    const unknown: MatchPlayerPosition[] = [{ timeMs: 0, x: 1, y: 2, z: 0, team: -1 }]
    render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={unknown} locale="fr" />,
    )
    expect(screen.queryByText('Camp A')).toBeNull()
  })

  // PORTE 1 — aucune position décodée.
  it('se masque (null) sans position', () => {
    for (const positions of [[] as MatchPlayerPosition[], undefined]) {
      const { container } = render(
        <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={positions} locale="fr" />,
      )
      expect(container.firstChild).toBeNull()
    }
  })

  // PORTE 2 — la carte du match n'a pas d'image figée : « Occupation du terrain » est un plan.
  it('se masque (null) quand la carte du match n’a pas de fond', () => {
    background.mockReturnValue({ data: undefined })
    const { container } = render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={sample} locale="fr" />,
    )
    expect(container.firstChild).toBeNull()
  })

  // PORTE 3 — rien ne tombe sur le plan.
  it('se masque (null) quand toutes les positions sont hors du cadre du fond', () => {
    const dehors: MatchPlayerPosition[] = [{ timeMs: 0, x: 900, y: 900, z: 0, team: 0 }]
    const { container } = render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={dehors} locale="fr" />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('rend le titre EN', () => {
    render(
      <MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={sample} locale="en" />,
    )
    expect(screen.getByText('Ground occupancy')).toBeTruthy()
  })
})
