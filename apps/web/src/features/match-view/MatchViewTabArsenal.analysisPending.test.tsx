/**
 * Tests — MatchViewTabArsenal, emplacements d'ATTENTE pendant l'analyse du film
 * (`header.analysis_pending`) : l'arme de chaque frag (« En cours d'analyse ») et le rejeu 2D
 * (« En préparation ») remplacent l'état vide ; hors analyse, l'état normal est inchangé.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MATCH_VIEW_TEXT } from './i18n'
import { arsenalPendingSlots } from './blockPredicates'
import { MatchViewTabArsenal } from './MatchViewTabArsenal'

const hoisted = vi.hoisted(() => ({ killDistanceGate: true }))

vi.mock('@/lib/replay/queries', () => ({ useMatchReplay: () => ({ data: undefined }) }))
vi.mock('@/features/match-replay/model/equipmentUsageLogic', () => ({
  buildEquipmentUsage: () => ({}),
  hasEquipmentUsage: () => false,
}))
vi.mock('@/features/match-replay/MatchEquipmentUsageSection', () => ({ MatchEquipmentUsageSection: () => null }))
vi.mock('./MatchFragCard', () => ({ MatchFragCard: () => null }))
vi.mock('./MatchKillDistanceSection', () => ({ MatchKillDistanceSection: () => <div data-testid="kill-distance" /> }))
vi.mock('./MatchPositionsHeatmap', () => ({
  MatchPositionsHeatmap: ({ positions }: { positions?: unknown[] }) => (positions?.length ? <div data-testid="positions-heatmap" /> : null),
}))
vi.mock('./blockPredicates', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./blockPredicates')>()),
  useHasKillDistanceSection: () => hoisted.killDistanceGate,
}))

interface Over {
  analysisPending?: boolean
  killDistance?: boolean
  positions?: boolean
  locale?: 'fr' | 'en'
}

function afficher({ analysisPending, killDistance = false, positions = false, locale = 'fr' }: Over = {}) {
  return render(
    <MatchViewTabArsenal
      playerSlug="jgtm"
      matchId="ab526724"
      replayAvailable={false}
      scoreboard={[]}
      roster={[]}
      fragDistribution={null}
      weaponTools={null}
      killDistance={killDistance ? [{ xuid: 'x1', weapons: [] }] : null}
      emprise={null}
      livesNearTeammate={null}
      analysisPending={analysisPending}
      meXUID={null}
      friendGamertags={[]}
      matchPositions={positions ? [{ timeMs: 0, x: 1, y: 2, z: 0, team: 0 }] : []}
      locale={locale}
      t={MATCH_VIEW_TEXT[locale]}
    />,
  )
}

describe('arsenalPendingSlots', () => {
  it('rien hors analyse en cours', () => {
    expect(arsenalPendingSlots({ analysisPending: false, hasKillDistance: false, hasTerrain: false })).toEqual({
      killsWeapons: false,
      terrain: false,
    })
  })
  it('un emplacement par section dont la donnée manque', () => {
    expect(arsenalPendingSlots({ analysisPending: true, hasKillDistance: false, hasTerrain: false })).toEqual({
      killsWeapons: true,
      terrain: true,
    })
    expect(arsenalPendingSlots({ analysisPending: true, hasKillDistance: true, hasTerrain: false })).toEqual({
      killsWeapons: false,
      terrain: true,
    })
    expect(arsenalPendingSlots({ analysisPending: true, hasKillDistance: true, hasTerrain: true })).toEqual({
      killsWeapons: false,
      terrain: false,
    })
  })
})

describe('MatchViewTabArsenal — analyse du film en cours', () => {
  it("met « En cours d'analyse » (arme par frag) et « En préparation » (rejeu 2D) à la place de l'état vide", () => {
    afficher({ analysisPending: true })
    const slots = screen.getAllByTestId('analysis-pending-slot')
    expect(slots.map((s) => s.textContent)).toEqual(["En cours d'analyse", 'En préparation'])
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.sectionKillsWeapons)).toBeInTheDocument()
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.sectionEquipmentTerrain)).toBeInTheDocument()
    expect(screen.queryByText(MATCH_VIEW_TEXT.fr.arsenalEmptyTitle)).toBeNull()
    expect(screen.queryByTestId('kill-distance')).toBeNull()
  })

  it('traduit en anglais', () => {
    afficher({ analysisPending: true, locale: 'en' })
    expect(screen.getAllByTestId('analysis-pending-slot').map((s) => s.textContent)).toEqual([
      'Analysis in progress',
      'Being prepared',
    ])
  })

  it('ne remplace que ce qui manque : distance par arme et positions déjà là = aucun emplacement', () => {
    afficher({ analysisPending: true, killDistance: true, positions: true })
    expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
    expect(screen.getByTestId('kill-distance')).toBeInTheDocument()
    expect(screen.getByTestId('positions-heatmap')).toBeInTheDocument()
  })

  it('état normal : aucun emplacement, la section garde sa lecture habituelle', () => {
    afficher({ analysisPending: false })
    expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
    expect(screen.getByTestId('kill-distance')).toBeInTheDocument()
  })

  it("état normal sans porte de titre : l'onglet garde son état vide nommé", () => {
    hoisted.killDistanceGate = false
    try {
      afficher({ analysisPending: false })
      expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
      expect(screen.getByText(MATCH_VIEW_TEXT.fr.arsenalEmptyTitle)).toBeInTheDocument()
    } finally {
      hoisted.killDistanceGate = true
    }
  })
})
