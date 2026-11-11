import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { LeadSharePoint } from '@/lib/charts/leadShare/model'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'
import { useAppShellStore } from '@/stores/appShellStore'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-stub" />,
}))

const { LeadShareChart } = await import('./LeadShareChart')

const POINTS: LeadSharePoint[] = [
  { key: 'a', label: '#1', share: 0.6 },
  { key: 'b', label: '#2', share: null },
]

describe('LeadShareChart', () => {
  it('titre par défaut, bouton d’information et résumé', () => {
    useAppShellStore.setState({ locale: 'fr' })
    const view = render(<LeadShareChart points={POINTS} summary="Moyenne 60 % sur 1 match mesuré" />)
    expect(view.getByText(LEAD_SHARE_TEXT.fr.name)).toBeTruthy()
    expect(view.getByRole('button', { name: /informations/i })).toBeTruthy()
    expect(view.getByText('Moyenne 60 % sur 1 match mesuré')).toBeTruthy()
  })

  it('aucun match mesuré : l’état vide dit la cause, aucun graphique', () => {
    useAppShellStore.setState({ locale: 'fr' })
    const view = render(<LeadShareChart points={[{ key: 'a', label: '#1', share: null }]} />)
    expect(view.getByText(LEAD_SHARE_TEXT.fr.emptyNoMeasure)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('EN : le nom et l’état vide passent en anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    const view = render(<LeadShareChart points={[]} />)
    expect(view.getByText('Lead share')).toBeTruthy()
    expect(view.getByText(LEAD_SHARE_TEXT.en.emptyNoMeasure)).toBeTruthy()
  })
})
