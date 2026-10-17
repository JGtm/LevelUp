/**
 * Tests — MatchViewTabArsenal, emplacements d'ATTENTE pendant l'analyse du film
 * (`header.analysis_pending_parts`) : l'arme de chaque frag (« En cours d'analyse ») et le rejeu 2D
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

const TOUTES_CAUSES = ['kill_source', 'citations', 'replay']

interface Over {
  parts?: string[]
  state?: string
  killDistance?: boolean
  positions?: boolean
  locale?: 'fr' | 'en'
}

function afficher({ parts, state, killDistance = false, positions = false, locale = 'fr' }: Over = {}) {
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
      analysisCauses={{ analysis_pending_parts: parts, analysis_state: state }}
      meXUID={null}
      friendGamertags={[]}
      matchPositions={positions ? [{ timeMs: 0, x: 1, y: 2, z: 0, team: 0 }] : []}
      locale={locale}
      t={MATCH_VIEW_TEXT[locale]}
    />,
  )
}

describe('arsenalPendingSlots', () => {
  const base = { killSourcePending: false, replayPending: false, hasKillDistance: false, hasTerrain: false }
  it('rien sans cause', () => {
    expect(arsenalPendingSlots(base)).toEqual({ killsWeapons: false, terrain: false })
  })
  it('chaque section regarde sa cause', () => {
    expect(arsenalPendingSlots({ ...base, killSourcePending: true })).toEqual({ killsWeapons: true, terrain: false })
    expect(arsenalPendingSlots({ ...base, replayPending: true })).toEqual({ killsWeapons: false, terrain: true })
    expect(arsenalPendingSlots({ ...base, killSourcePending: true, replayPending: true })).toEqual({
      killsWeapons: true,
      terrain: true,
    })
  })
  it('ne remplace que ce qui manque', () => {
    const causes = { ...base, killSourcePending: true, replayPending: true }
    expect(arsenalPendingSlots({ ...causes, hasKillDistance: true })).toEqual({ killsWeapons: false, terrain: true })
    expect(arsenalPendingSlots({ ...causes, hasKillDistance: true, hasTerrain: true })).toEqual({
      killsWeapons: false,
      terrain: false,
    })
  })
})

describe('MatchViewTabArsenal — analyse du film en cours', () => {
  it("décodage sur ce match : « Analyse en cours » à la place de l'état vide, pour l'arme par frag et le rejeu 2D", () => {
    afficher({ parts: TOUTES_CAUSES, state: 'in_progress' })
    const slots = screen.getAllByTestId('analysis-pending-slot')
    expect(slots.map((s) => s.textContent)).toEqual(['Analyse en cours', 'Analyse en cours'])
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.sectionKillsWeapons)).toBeInTheDocument()
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.sectionEquipmentTerrain)).toBeInTheDocument()
    expect(screen.queryByText(MATCH_VIEW_TEXT.fr.arsenalEmptyTitle)).toBeNull()
    expect(screen.queryByTestId('kill-distance')).toBeNull()
  })

  it("en attente (aucun décodage sur ce match) : « Analyse en attente », jamais « en cours »", () => {
    for (const state of ['queued', undefined]) {
      const { unmount } = afficher({ parts: TOUTES_CAUSES, state })
      expect(screen.getAllByTestId('analysis-pending-slot').map((s) => s.textContent)).toEqual([
        'Analyse en attente',
        'Analyse en attente',
      ])
      unmount()
    }
  })

  it("causes isolées : kill_source seule n'annonce que l'arme par frag", () => {
    afficher({ parts: ['kill_source'], state: 'in_progress' })
    expect(screen.getAllByTestId('analysis-pending-slot').map((s) => s.textContent)).toEqual(['Analyse en cours'])
  })

  it("causes isolées : replay seul n'annonce que le rejeu 2D, la distance par arme garde sa lecture", () => {
    afficher({ parts: ['replay'] })
    expect(screen.getAllByTestId('analysis-pending-slot').map((s) => s.textContent)).toEqual(['Analyse en attente'])
    expect(screen.getByTestId('kill-distance')).toBeInTheDocument()
  })

  it('causes isolées : citations seules ne touchent pas cet onglet', () => {
    afficher({ parts: ['citations'] })
    expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
    expect(screen.getByTestId('kill-distance')).toBeInTheDocument()
  })

  it('traduit en anglais', () => {
    afficher({ parts: TOUTES_CAUSES, locale: 'en', state: 'in_progress' })
    expect(screen.getAllByTestId('analysis-pending-slot').map((s) => s.textContent)).toEqual([
      'Analysis in progress',
      'Analysis in progress',
    ])
  })

  it("traduit l'attente en anglais", () => {
    afficher({ parts: ['kill_source'], locale: 'en' })
    expect(screen.getAllByTestId('analysis-pending-slot').map((s) => s.textContent)).toEqual(['Analysis pending'])
  })

  it('ne remplace que ce qui manque : distance par arme et positions déjà là = aucun emplacement', () => {
    afficher({ parts: TOUTES_CAUSES, killDistance: true, positions: true })
    expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
    expect(screen.getByTestId('kill-distance')).toBeInTheDocument()
    expect(screen.getByTestId('positions-heatmap')).toBeInTheDocument()
  })

  it('état normal : aucun emplacement, la section garde sa lecture habituelle', () => {
    afficher({ parts: [] })
    expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
    expect(screen.getByTestId('kill-distance')).toBeInTheDocument()
  })

  it("état normal sans porte de titre : l'onglet garde son état vide nommé", () => {
    hoisted.killDistanceGate = false
    try {
      afficher({ parts: [] })
      expect(screen.queryByTestId('analysis-pending-slot')).toBeNull()
      expect(screen.getByText(MATCH_VIEW_TEXT.fr.arsenalEmptyTitle)).toBeInTheDocument()
    } finally {
      hoisted.killDistanceGate = true
    }
  })
})
