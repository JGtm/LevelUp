/**
 * Tests ExplorerTargetAssists — bloc « Part des assistances » de l'encart cible :
 *  - assistances mesurées → figure papillon + « total · part » des deux côtés ;
 *  - aucune assistance mesurée (ou pas de stats de rencontre) → « — » + la raison,
 *    JAMAIS « 0 assistance » ;
 *  - la borne d'échelle vient de l'API (assist_volume_max), pas de la paire affichée.
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

describe('ExplorerTargetAssists', () => {
  it('rend la figure papillon et les totaux · parts quand les assistances sont mesurées', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    expect(screen.getByText('Part des assistances')).toBeInTheDocument()
    expect(screen.getByTestId('assist-butterfly')).toBeInTheDocument()
    const block = screen.getByTestId('explorer-target-assists')
    expect(block).toHaveTextContent("Il t'a assisté")
    expect(block).toHaveTextContent("Tu l'as assisté")
    expect(block).toHaveTextContent('20 · 25 %')
    expect(block).toHaveTextContent('20 % · 14')
  })

  it('les demi-barres sont bornées par assist_volume_max, pas par la paire affichée', () => {
    // Largeur totale de la demi-barre « reçues » = somme de ses segments.
    const receivedWidth = (): number =>
      screen
        .getAllByTestId(/^assist-segment-left-/)
        .map((s) => Number.parseFloat((s.closest('span[style*="width"]') as HTMLElement).style.width))
        .reduce((a, b) => a + b, 0)

    // Borne globale (120) : 20 assistances ne remplissent pas la demi-barre.
    const wide = renderWithProviders(<ExplorerTargetAssists encounterStats={WITH_ASSISTS} />)
    const bounded = receivedWidth()
    expect(bounded).toBeLessThan(99)
    wide.unmount()

    // Borne réduite au volume de la paire : la demi-barre serait pleine — c'est
    // exactement le défaut que la borne servie par l'API évite.
    renderWithProviders(<ExplorerTargetAssists encounterStats={{ ...WITH_ASSISTS, assist_volume_max: 20 }} />)
    expect(receivedWidth()).toBeGreaterThan(bounded)
  })

  it('aucun match mesuré : « — » et la raison, pas de papillon', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={BASE} />)
    const block = screen.getByTestId('explorer-target-assists')
    expect(block).toHaveTextContent('—')
    expect(block).toHaveTextContent('Aucun match ensemble avec film analysé')
    expect(screen.queryByTestId('assist-butterfly')).not.toBeInTheDocument()
  })

  it('sans stats de rencontre du tout : le bloc reste rendu en état vide', () => {
    renderWithProviders(<ExplorerTargetAssists encounterStats={null} />)
    expect(screen.getByTestId('explorer-target-assists')).toHaveTextContent('—')
    expect(screen.queryByTestId('assist-butterfly')).not.toBeInTheDocument()
  })
})
