/**
 * Tests LeverList — composition des phrases de leviers via gabarits i18n (F3).
 *
 * Le backend ne sert plus de phrase : chaque levier porte un axe + des données
 * structurées de contexte (context_key / context_label). Le front compose la
 * phrase FR/EN par axe. On vérifie :
 *   - la composition FR ET EN par axe (contextuel + comportemental) ;
 *   - qu'un GUID de carte (context_key by_map) n'est JAMAIS rendu — seul le
 *     context_label résolu (title-agnostic) apparaît.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { LeverList } from './LeverList'
import { getAscensionText } from './i18n'
import type { PatternLever } from './types'

afterEach(() => cleanup())

// Levier minimal — rank <= 3 et impact > 0.3 pour passer le filtre de LeverList.
function lever(partial: Partial<PatternLever>): PatternLever {
  return {
    rank: 1,
    axis: 'accuracy',
    current_val: 0,
    target_val: 0,
    horizon: 20,
    impact: 0.4,
    ...partial,
  }
}

const MAP_GUID = 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'

describe('LeverList — composition i18n des phrases (F3)', () => {
  it('by_map : rend le nom de carte résolu (context_label), jamais le GUID [FR]', () => {
    const t = getAscensionText('fr')
    render(
      <LeverList
        t={t}
        levers={[lever({ axis: 'map_avoidance', context_key: MAP_GUID, context_label: 'Aquarius' })]}
      />,
    )
    expect(screen.getByText('Améliore ton taux de victoire sur Aquarius')).toBeInTheDocument()
    expect(screen.queryByText(new RegExp(MAP_GUID))).not.toBeInTheDocument()
  })

  it('by_map : phrase EN avec « on » + nom résolu', () => {
    const t = getAscensionText('en')
    render(
      <LeverList
        t={t}
        levers={[lever({ axis: 'map_avoidance', context_key: MAP_GUID, context_label: 'Aquarius' })]}
      />,
    )
    expect(screen.getByText('Improve your win rate on Aquarius')).toBeInTheDocument()
    expect(screen.queryByText(new RegExp(MAP_GUID))).not.toBeInTheDocument()
  })

  it('by_mode : interpole le libellé de mode (déjà lisible) [FR/EN]', () => {
    render(<LeverList t={getAscensionText('fr')} levers={[lever({ axis: 'mode_selection', context_key: 'CTF' })]} />)
    expect(screen.getByText('Améliore ton taux de victoire en CTF')).toBeInTheDocument()
    cleanup()
    render(<LeverList t={getAscensionText('en')} levers={[lever({ axis: 'mode_selection', context_key: 'CTF' })]} />)
    expect(screen.getByText('Improve your win rate in CTF')).toBeInTheDocument()
  })

  it('by_squad : mappe with_friends → escouade / squad selon la locale', () => {
    render(<LeverList t={getAscensionText('fr')} levers={[lever({ axis: 'squad_play', context_key: 'with_friends' })]} />)
    expect(screen.getByText(/Améliore ton taux de victoire en/)).toHaveTextContent(
      getAscensionText('fr').squadVsSoloSquad,
    )
    cleanup()
    render(<LeverList t={getAscensionText('en')} levers={[lever({ axis: 'squad_play', context_key: 'solo' })]} />)
    expect(screen.getByText(/Improve your win rate in/)).toHaveTextContent(
      getAscensionText('en').squadVsSoloSolo,
    )
  })

  it('axe comportemental (accuracy) : phrase fixe sans contexte [FR/EN]', () => {
    render(<LeverList t={getAscensionText('fr')} levers={[lever({ axis: 'accuracy' })]} />)
    expect(screen.getByText('Améliore ta précision')).toBeInTheDocument()
    cleanup()
    render(<LeverList t={getAscensionText('en')} levers={[lever({ axis: 'accuracy' })]} />)
    expect(screen.getByText('Improve your accuracy')).toBeInTheDocument()
  })

  it('axe sans gabarit : repli sur le libellé d’axe, jamais la clé brute', () => {
    const t = getAscensionText('fr')
    render(<LeverList t={t} levers={[lever({ axis: 'csr_ranked' })]} />)
    // Repli : le libellé d'axe sert de phrase (title + sous-titre) — jamais « csr_ranked ».
    expect(screen.getAllByText(t.leverAxis.csr_ranked).length).toBeGreaterThan(0)
    expect(screen.queryByText('csr_ranked')).not.toBeInTheDocument()
  })
})
