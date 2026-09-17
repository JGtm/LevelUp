/**
 * ExplorerTargetFragRange.test — le bloc « Portée des frags » de l'encart cible.
 *
 * Ce que ces tests verrouillent :
 *
 *  1. DEUX BANDES NOMMÉES, toi et la cible — c'est ce que le bloc promet ; une légende à un
 *     seul nom laisserait croire qu'on ne regarde qu'un joueur.
 *  2. RIEN D'AUTRE QUE LES BANDES (demande utilisateur du 2026-09-17) : ni couverture
 *     « N frags mesurés sur M », ni liste des rôles écartés par le seuil.
 *  3. AUCUNE MESURE → état vide titré, jamais un graphe à zéro ligne ni une portée inventée.
 *
 * ECharts est mocké (jsdom ne peint pas de canvas) et CHARGÉ EN DIFFÉRÉ par ChartCard
 * (Suspense + import dynamique) : les assertions sur le graphe utilisent `findByTestId`, pas
 * `getByTestId`. La géométrie du graphe est testée PURE ailleurs
 * (`components/charts/weaponRangeChart.test.ts`), l'axe et l'ordre aussi
 * (`components/charts/weaponRangeRoles.test.ts`).
 */
import { describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerEncounterStats, SynthesisWeaponRange, WeaponRangeSide } from '@/lib/api/types'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

import { ExplorerTargetFragRange } from './ExplorerTargetFragRange'

const side = (o: Partial<WeaponRangeSide>): WeaponRangeSide => ({
  measured: 12,
  p10: 5,
  median: 9,
  p90: 18,
  min_m: 2,
  max_m: 30,
  above_pct: 30,
  level_pct: 50,
  below_pct: 20,
  ...o,
})

const bloc = (weapons: SynthesisWeaponRange['weapons']): SynthesisWeaponRange => ({
  weapons,
  median_kills_m: 9,
  median_deaths_m: 11,
  measured_kills: 40,
  total_kills: 55,
  measured_deaths: 30,
  total_deaths: 44,
})

const BASE: ExplorerEncounterStats = { count_together: 12 }

/** La cible a un rôle de plus (`sniper`) : l'axe est l'union, pas l'intersection. */
const AVEC_MESURES: ExplorerEncounterStats = {
  ...BASE,
  frag_range_self: bloc([{ weapon_key: 'precision', kills: side({ median: 22 }) }]),
  frag_range_target: bloc([
    { weapon_key: 'precision', kills: side({ median: 18 }) },
    { weapon_key: 'sniper', kills: side({ median: 45 }) },
  ]),
}

describe('ExplorerTargetFragRange', () => {
  it('rend le graphe et nomme les deux bandes : toi et la cible', async () => {
    renderWithProviders(<ExplorerTargetFragRange encounterStats={AVEC_MESURES} gamertag="Adversaire7" />)
    expect(screen.getByText('Portée des frags')).toBeInTheDocument()
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    const carte = screen.getByTestId('explorer-target-frag-range')
    expect(carte).toHaveTextContent('Toi')
    expect(carte).toHaveTextContent('Adversaire7')
  })

  it('ne montre QUE les bandes : ni couverture, ni rôles sous le seuil', async () => {
    renderWithProviders(<ExplorerTargetFragRange encounterStats={AVEC_MESURES} gamertag="Adversaire7" />)
    await screen.findByTestId('echarts-mock')
    const carte = screen.getByTestId('explorer-target-frag-range')
    // Les dénominateurs existent dans le contrat (40 mesurés sur 55) mais ne s'affichent pas.
    expect(carte).not.toHaveTextContent('55')
    expect(carte).not.toHaveTextContent('mesurés sur')
    expect(carte).not.toHaveTextContent('seuil')
  })

  it('un rôle mesuré par un seul joueur n’envoie pas le bloc en état vide', async () => {
    renderWithProviders(<ExplorerTargetFragRange encounterStats={AVEC_MESURES} gamertag="Adversaire7" />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.queryByText('Aucun frag mesuré sur vos matchs communs')).not.toBeInTheDocument()
  })

  it('aucune mesure des deux côtés : état vide titré, pas de graphe', () => {
    renderWithProviders(<ExplorerTargetFragRange encounterStats={BASE} gamertag="Adversaire7" />)
    expect(screen.getByText('Portée des frags')).toBeInTheDocument()
    expect(screen.getByText('Aucun frag mesuré sur vos matchs communs')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-mock')).not.toBeInTheDocument()
  })

  it('sans stats de rencontre du tout : le bloc reste rendu en état vide', () => {
    renderWithProviders(<ExplorerTargetFragRange encounterStats={null} gamertag="Adversaire7" />)
    expect(screen.getByTestId('explorer-target-frag-range')).toHaveTextContent(
      'Aucun frag mesuré sur vos matchs communs',
    )
  })
})
