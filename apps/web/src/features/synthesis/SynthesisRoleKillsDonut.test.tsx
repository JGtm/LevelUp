import { describe, it, expect, afterEach, vi } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { SynthesisRoleKillsDonut } from './SynthesisRoleKillsDonut'

// Locale figée FR pour asserter les libellés i18n déportés du donut.
const mockShellState = { locale: 'fr' as 'fr' | 'en' }

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) => selector(mockShellState),
}))

describe('SynthesisRoleKillsDonut', () => {
  afterEach(cleanup)

  it('rend les libellés des rôles hors-arsenal (véhicule/tourelle/non-attribué/autres/environnement)', () => {
    render(
      <SynthesisRoleKillsDonut
        roles={[
          { role: 'automatic', kills: 120 },
          { role: 'vehicle', kills: 40 },
          { role: 'turret', kills: 20 },
          { role: 'unattributed', kills: 90 },
          { role: 'other', kills: 10 },
          { role: 'environmental', kills: 6 },
        ]}
      />,
    )
    expect(screen.getByText('Véhicule')).toBeInTheDocument()
    expect(screen.getByText('Tourelle')).toBeInTheDocument()
    expect(screen.getByText('Non attribué')).toBeInTheDocument()
    expect(screen.getByText('Autres')).toBeInTheDocument()
    expect(screen.getByText('Environnement')).toBeInTheDocument()
  })

  it('ne casse pas le rendu sur un rôle inconnu (fallback : clé affichée)', () => {
    // Rôle absent du manifest → formatMessage dégrade en affichant la clé, sans crash.
    const { container } = render(
      <SynthesisRoleKillsDonut
        roles={[
          { role: 'precision', kills: 100 },
          { role: 'mystery_role', kills: 30 },
        ]}
      />,
    )
    expect(container.querySelector('svg')).not.toBeNull()
    expect(screen.getByText('synthesis.charts.role_mystery_role')).toBeInTheDocument()
  })

  it('rend null quand aucune part n\'a de frags', () => {
    const { container } = render(<SynthesisRoleKillsDonut roles={[]} />)
    expect(container.firstChild).toBeNull()
  })
})
