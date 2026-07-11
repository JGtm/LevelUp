/**
 * Tests Vitest pour EngagementCurve (Phase 8).
 *
 * Smoke tests : verifier que le composant rend sans planter sur :
 * - donnees vides
 * - donnees nominales mode intra-match
 * - donnees nominales mode session
 *
 * On ne teste pas le rendu ECharts pixel par pixel (trop fragile) — on
 * verifie juste que la prop pass-through fonctionne et que les states
 * loading/empty/ready soient correctement propages a ChartCard.
 */
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { EngagementCurve, type EngagementPoint } from './EngagementCurve'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--${token})`,
}))

// Mock minimal d'echarts pour eviter le rendu reel.
vi.mock('echarts/core', () => ({
  use: vi.fn(),
  init: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
  })),
  registerTheme: vi.fn(),
}))

describe('EngagementCurve', () => {
  const samplePoints: EngagementPoint[] = [
    { x: 0, paceTeam: 10, paceAttendu: 11, paceJoueur: 9, paceLobby: 10.5 },
    { x: 60_000, paceTeam: 12, paceAttendu: 13, paceJoueur: 14, paceLobby: 12.2 },
    { x: 120_000, paceTeam: 11, paceAttendu: 12, paceJoueur: 13, paceLobby: 11.4 },
  ]

  it('rend sans erreur en mode intra-match', () => {
    const { container } = render(
      <EngagementCurve title="Engagement" points={samplePoints} granularity="intra" />,
    )
    expect(container).toBeTruthy()
  })

  it('rend sans erreur en mode session', () => {
    const { container } = render(
      <EngagementCurve title="Engagement session" points={samplePoints} granularity="session" />,
    )
    expect(container).toBeTruthy()
  })

  it('rend sans erreur avec points vides', () => {
    const { container } = render(<EngagementCurve title="Engagement" points={[]} />)
    expect(container).toBeTruthy()
  })

  it('rend sans erreur avec hideAttendu (cold_start) et labels lobby', () => {
    const { container } = render(
      <EngagementCurve
        title="Engagement"
        points={samplePoints}
        hideAttendu
        seriesLabels={{ team: 'Équipe', expected: 'Attendu', player: 'Joueur', lobby: 'Lobby' }}
      />,
    )
    expect(container).toBeTruthy()
  })

  it('propage state loading sans planter', () => {
    const { container } = render(
      <EngagementCurve title="Engagement" points={samplePoints} state="loading" />,
    )
    expect(container).toBeTruthy()
  })

  // Régression (2026-06-25) : state='error' rendait `new Error('error')` →
  // ChartCard affichait « error » brut (pas orienté end-user). On dégrade
  // désormais en état vide neutre avec message humain.
  it('sur erreur, dégrade en état vide avec message humain (jamais « Error » brut)', () => {
    const { getByText, queryByTestId } = render(
      <EngagementCurve
        title="Engagement"
        points={samplePoints}
        state="error"
        errorMessage="Engagement momentanément indisponible"
      />,
    )
    expect(getByText('Engagement momentanément indisponible')).toBeTruthy()
    // Aucun habillage d'erreur brut (le bug initial : ChartCardError « error »).
    expect(queryByTestId('chart-card-error')).toBeNull()
  })

  it('sur erreur sans errorMessage explicite, retombe sur un défaut FR propre', () => {
    const { getByText } = render(
      <EngagementCurve title="Engagement" points={samplePoints} state="error" />,
    )
    expect(getByText('Engagement momentanément indisponible')).toBeTruthy()
  })
})
