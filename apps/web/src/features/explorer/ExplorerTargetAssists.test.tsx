/**
 * Tests ExplorerTargetAssists — bloc « Part des assistances » de l'encart cible, rendu
 * 1.A horizontal (D17) :
 *  - deux pistes épaisses (un sens chacune), tranches segmentées, comptes écrits ;
 *  - échelle LINÉAIRE commune bornée par le plus gros des deux totaux — plus de log,
 *    plus d'`assist_volume_max` : le sens majoritaire remplit sa piste ;
 *  - trait de parité présent sur les deux pistes ;
 *  - aucune assistance mesurée (ou pas de stats de rencontre) → « — » + la raison,
 *    JAMAIS « 0 assistance ».
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerEncounterStats } from '@/lib/api/types'

import { ExplorerTargetAssists } from './ExplorerTargetAssists'

const BASE: ExplorerEncounterStats = {
  count_together: 12,
  ally_count: 9,
  enemy_count: 3,
}

// 20 reçues sur 80 frags = 25 % ; 14 données sur 70 frags de la cible = 20 %.
const WITH_ASSISTS: ExplorerEncounterStats = {
  ...BASE,
  assist_volume_max: 120,
  assists: {
    matches_measured: 9,
    my_frags: 80,
    partner_frags: 70,
    received: { total: 20, low: 12, mid: 5, high: 3 },
    given: { total: 14, low: 9, mid: 3, high: 2 },
  },
}

/** Somme des largeurs (%) des segments d'une piste. */
function trackWidth(testId: string): number {
  return screen
    .getAllByTestId(new RegExp(`^${testId}-(low|mid|high)$`))
    .map((s) => {
      const holder = s.closest('div[style*="width"]') as HTMLElement
      // `calc(70% - 2px)` : seul le pourcentage porte l'échelle.
      return Number.parseFloat(/([\d.]+)%/.exec(holder.style.width)?.[1] ?? '0')
    })
    .reduce((a, b) => a + b, 0)
}

describe('ExplorerTargetAssists', () => {
  it('rend deux pistes épaisses, une par sens, avec les totaux · parts', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    expect(screen.getByText('Part des assistances')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-assist-track-received')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-assist-track-given')).toBeInTheDocument()
    const block = screen.getByTestId('explorer-target-assists')
    expect(block).toHaveTextContent('Il me sert')
    expect(block).toHaveTextContent('Je le sers')
    expect(block).toHaveTextContent('20 · 25 %')
    expect(block).toHaveTextContent('14 · 20 %')
  })

  it('écrit le compte de chaque tranche dans son segment', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    // Reçues : 12 / 5 / 3 sur une piste bornée à 20 → toutes les tranches pèsent assez.
    expect(screen.getByTestId('explorer-assist-track-received-low')).toHaveTextContent('12')
    expect(screen.getByTestId('explorer-assist-track-received-mid')).toHaveTextContent('5')
    expect(screen.getByTestId('explorer-assist-track-received-high')).toHaveTextContent('3')
    expect(screen.getByTestId('explorer-assist-track-given-low')).toHaveTextContent('9')
  })

  it('échelle linéaire bornée par le MAX des deux sens : le majoritaire remplit sa piste', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    const received = trackWidth('explorer-assist-track-received')
    const given = trackWidth('explorer-assist-track-given')
    // borne = max(20, 14) = 20 → reçues 100 %, données 70 %.
    expect(received).toBeCloseTo(100, 1)
    expect(given).toBeCloseTo(70, 1)
    // Le rapport des longueurs EST le rapport des volumes (ce que le log écrasait).
    expect(given / received).toBeCloseTo(14 / 20, 2)
  })

  it('`assist_volume_max` n’est plus lu : le rendu ne bouge pas quand il change', () => {
    const wide = renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    const before = trackWidth('explorer-assist-track-received')
    wide.unmount()
    renderWithProviders(
      <ExplorerTargetAssists encounterStats={{ ...WITH_ASSISTS, assist_volume_max: 20 }} />,
    )
    expect(trackWidth('explorer-assist-track-received')).toBeCloseTo(before, 3)
  })

  it('le trait de parité traverse les deux pistes, à la moitié du total des deux sens', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    const received = screen.getByTestId('explorer-assist-track-received-parity')
    const given = screen.getByTestId('explorer-assist-track-given-parity')
    // (20 + 14) / 2 = 17, sur une borne de 20 → 85 %.
    expect(received.style.left).toBe('85%')
    expect(given.style.left).toBe('85%')
  })

  it('aucun match mesuré : « — » et la raison, pas de piste', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={BASE} />)
    const block = screen.getByTestId('explorer-target-assists')
    expect(block).toHaveTextContent('—')
    expect(block).toHaveTextContent('Aucun match ensemble avec film analysé')
    expect(screen.queryByTestId('explorer-assist-bars')).not.toBeInTheDocument()
  })

  it('sans stats de rencontre du tout : le bloc reste rendu en état vide', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={null} />)
    expect(screen.getByTestId('explorer-target-assists')).toHaveTextContent('—')
    expect(screen.queryByTestId('explorer-assist-bars')).not.toBeInTheDocument()
  })
})
