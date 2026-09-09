/**
 * SquadEfficiencyChart — chaque carte porte SA définition, pas celle de sa voisine.
 *
 * Les deux cartes partageaient un texte unique de 120 mots qui définissait les DEUX
 * indicateurs : sur la carte Rendement, on lisait la définition de la Résistance avant la
 * sienne. Découpé le 2026-09-09 (retour utilisateur). Sans ce test, échanger les deux aides
 * — ou en recâbler une sur l'autre — laisserait la suite verte.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import { SquadEfficiencyChart } from './SquadEfficiencyChart'
import { getSquadText } from './i18n'

// jsdom ne peint pas de canvas : le rendu ECharts n'a aucun intérêt ici.
vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const point = (order: number): SquadPerformanceSeriesPoint =>
  ({
    match_order: order,
    kills: 12,
    deaths: 9,
    assists: 4,
    damage_dealt: 3200,
    damage_taken: 2800,
    rendement_offensif: 1.1,
    resistance_defensive: 1.3,
  }) as unknown as SquadPerformanceSeriesPoint

function renderCards() {
  const t = getSquadText('fr')
  return renderWithProviders(
    <SquadEfficiencyChart
      rowsByPlayer={{ Me: [point(0), point(1)] }}
      playerOrder={['Me']}
      colorByPlayer={{ Me: '#aa3366' }}
      labels={t.efficiencySeries}
    />,
  )
}

/** L'icône ⓘ SŒUR d'un titre de carte donné — les deux cartes en portent une. */
function infoIconOf(title: string): HTMLElement {
  const entete = screen.getByText(title).closest('span') as HTMLElement
  return within(entete).getByRole('button', { name: /info/i })
}

describe('SquadEfficiencyChart — une aide ⓘ par carte', () => {
  it('« Rendement » définit ce qu’une vie de dégâts rapporte', () => {
    renderCards()
    fireEvent.mouseEnter(infoIconOf('Rendement'))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toMatch(/un frag par vie dépensée/i)
    expect(aide).not.toMatch(/encaissés avant chaque mort/i)
  })

  it('« Résistance » définit ce qui est encaissé avant chaque mort', () => {
    renderCards()
    fireEvent.mouseEnter(infoIconOf('Résistance'))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toMatch(/encaissés avant chaque mort/i)
    expect(aide).not.toMatch(/un frag par vie dépensée/i)
  })
})
