/**
 * HomeHeroKPIGrid — l'aide ⓘ de la tuile « Rendement / Résist. ».
 *
 * Cette tuile est la SEULE de la barre hero à porter DEUX indicateurs sous un libellé
 * abrégé, et deux pourcentages rapportés à une vie de Spartan ne se lisent pas sans leur
 * définition. L'aide est donc COMBINÉE — les deux définitions en un texte — là où les cartes
 * Escouade et les tuiles Match view en portent une chacune (retour utilisateur 2026-09-09).
 *
 * Ce que ce test verrouille : l'icône existe sur cette tuile-là, elle nomme les DEUX
 * indicateurs, et les autres tuiles n'en ont pas gagné une au passage.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { HeroKPIs } from '@/lib/api/types'

import { HomeHeroKPIGrid } from './HomeHeroKPIGrid'
import { getKPIText } from './kpi.i18n'

// La tuile compose `CombatYieldDisplay`, qui interroge la capability `damage_taken`.
vi.mock('@/lib/damage/effectiveHp', () => ({
  useProvidesDamageTaken: () => true,
}))

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (sel: (s: { locale: string }) => unknown) => sel({ locale: 'fr' }),
}))

const KPIS: HeroKPIs = {
  win_rate: 0.54,
  global_ratio: 1.1,
  avg_kda: 1.3,
  avg_accuracy: 47,
  total_matches: 320,
  wins: 170,
  draws: 4,
  dnfs: 2,
  losses: 144,
  total_playtime_secs: 360_000,
  favorite_weapon_name: 'BR75',
  favorite_weapon_kills: 900,
  favorite_playlist_name: 'Ranked Arena',
  favorite_playlist_count: 120,
  avg_offensive_conversion: 0.92,
  avg_defensive_resistance: 1.18,
  dmg_per_kill: 244,
  dmg_per_death: 265,
}

function renderGrid() {
  return render(
    <HomeHeroKPIGrid
      kpis={KPIS}
      labelOf={(key) => key}
      numberLocale="fr-FR"
      kpiText={getKPIText('fr')}
    />,
  )
}

describe('HomeHeroKPIGrid — aide ⓘ de la tuile Rendement / Résistance', () => {
  it('l’aide nomme les DEUX indicateurs, pas un seul', () => {
    renderGrid()
    const ligne = screen.getByText('Rendement / Résist.').closest('p') as HTMLElement
    fireEvent.mouseEnter(within(ligne).getByRole('button', { name: /info/i }))
    const aide = screen.getByRole('tooltip').textContent ?? ''
    expect(aide).toMatch(/un frag par vie dépensée/i)
    expect(aide).toMatch(/encaissés avant chaque mort/i)
  })

  it('c’est la SEULE tuile à porter une aide (les autres n’ont pas bougé)', () => {
    renderGrid()
    expect(screen.getAllByRole('button', { name: /info/i })).toHaveLength(1)
  })
})
