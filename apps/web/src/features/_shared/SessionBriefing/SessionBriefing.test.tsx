/**
 * SessionBriefing.test.tsx — tests Vitest (8 cas) du composant briefing.
 *
 * Couvre :
 *   - Solo : pas de bande verdict, pas de trends ▲/▼ visibles
 *   - Squad : bande verdict + N+1 cards
 *   - Drill-down : click sur card joueur → KpiGrid recalcule
 *   - Reset : click sur ✕ → retour à activeXuid
 *   - Trend kills (higher_is_better)
 *   - Trend deaths (lower_is_better — inversion)
 *   - Singulier/pluriel outcomes
 *   - Fallback : drill-down sur xuid manquant dans kpisByXuid
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as fieldMappingsModule from '@/lib/i18n/fieldMappings'
import type { KPIStats } from '@/lib/api/types'
import type { PlayerScoreCard, SquadScoreCard } from '@/features/squad/v2/types'

import { SessionBriefing } from './SessionBriefing'

// ─── Helpers fixtures ─────────────────────────────────────────────────────────

function makeKPIs(overrides: Partial<KPIStats> = {}): KPIStats {
  return {
    matches_count: 10,
    total_play_seconds: 6540,
    avg_match_seconds: 654,
    kills_per_game: 8.7,
    kills_per_minute: 1.0,
    deaths_per_game: 10.8,
    deaths_per_minute: 1.24,
    assists_per_game: 4.5,
    assists_per_minute: 0.52,
    avg_accuracy: 46.92,
    avg_life_seconds: 37,
    outcomes: { wins: 3, losses: 7, ties: 0, dnf: 0 },
    ...overrides,
  }
}

function makePlayerCard(xuid: string, gamertag: string, score: number, comparison: 'above' | 'below' | 'near'): PlayerScoreCard {
  return {
    xuid,
    gamertag,
    score,
    label: 'average',
    comparison,
    kd_ratio: 1.0,
    win_rate: 0.5,
    accuracy: 50,
    kills: 8,
  }
}

function makeSquadScore(): SquadScoreCard {
  return {
    score: 44,
    grade: 'C',
    base_avg: 44,
    bonus_win_rate: 0,
    bonus_min_kd: 0,
    bonus_balance: 0,
    team_win_rate: 0.5,
    min_kd: 1.0,
    kills_std_dev: 2.0,
  }
}

// ─── Setup ────────────────────────────────────────────────────────────────────

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  // Mock useFieldMappings : on retourne les libellés outcomes FR.
  vi.spyOn(fieldMappingsModule, 'useOutcomeLabel').mockImplementation((key: string) => {
    const map: Record<string, string> = {
      win: 'Victoire',
      loss: 'Défaite',
      tie: 'Égalité',
      dnf: 'Abandon',
    }
    return map[key] ?? key
  })
})

afterEach(() => {
  vi.restoreAllMocks()
})

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('SessionBriefing — mode solo', () => {
  it('rend KpiGrid sans team card ni player cards ni trends', () => {
    const kpis = makeKPIs()
    renderWithProviders(<SessionBriefing kpis={kpis} />)

    // Verdict band : la Results bar + mini-cards Matchs/Durée sont toujours
    // rendues (refacto 2026-05 : SquadVerdict toujours monté, sections team
    // card / player cards conditionnelles).
    expect(screen.getByText('Matchs joués')).toBeInTheDocument()
    expect(screen.getByText('Durée totale')).toBeInTheDocument()
    // La durée moyenne/match est désormais inline-sub de la card Matchs (10:54)
    expect(screen.getByText(/10min54\/match/)).toBeInTheDocument()
    // Pas de team card en solo (squadScore absent)
    expect(screen.queryByText(/Score d'équipe/)).not.toBeInTheDocument()
    // Pas de trend hint en solo (kpisByXuid absent → pas de comparaison équipe)
    expect(screen.queryByText(/vs moyenne d'équipe/)).not.toBeInTheDocument()
    // KPI évaluatifs affichés (KpiGrid)
    expect(screen.getByText('Frags par match')).toBeInTheDocument()
    expect(screen.getByText('8.70')).toBeInTheDocument()
  })
})

describe('SessionBriefing — mode squad', () => {
  it('rend verdict + N+1 cards joueurs', () => {
    const kpis = makeKPIs()
    const teamAvg = makeKPIs({ kills_per_game: 8.0, deaths_per_game: 10.0 })
    const squad = {
      score: makeSquadScore(),
      players: [
        makePlayerCard('xuid-me', 'Spartan-117', 50, 'above'),
        makePlayerCard('xuid-choco', 'Chocoboflor', 40, 'below'),
        makePlayerCard('xuid-ghost', 'Ghost', 51, 'above'),
      ],
      kpisByXuid: {
        'xuid-me': kpis,
        'xuid-choco': makeKPIs({ kills_per_game: 6.5 }),
        'xuid-ghost': makeKPIs({ kills_per_game: 11.0 }),
      },
      teamAvgKpis: teamAvg,
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)

    expect(screen.getByText(/Score d'équipe/)).toBeInTheDocument()
    expect(screen.getByText('Spartan-117')).toBeInTheDocument()
    expect(screen.getByText('Chocoboflor')).toBeInTheDocument()
    expect(screen.getByText('Ghost')).toBeInTheDocument()
    // Trend hint visible quand teamAvg fourni
    expect(screen.getByText(/vs moyenne d'équipe/)).toBeInTheDocument()
  })

  it('drill-down : click sur Chocoboflor → KpiGrid affiche ses stats + reset bar visible', () => {
    const kpis = makeKPIs()
    const chocoKpis = makeKPIs({ kills_per_game: 6.5 })
    const squad = {
      score: makeSquadScore(),
      players: [
        makePlayerCard('xuid-me', 'Spartan-117', 50, 'above'),
        makePlayerCard('xuid-choco', 'Chocoboflor', 40, 'below'),
      ],
      kpisByXuid: {
        'xuid-me': kpis,
        'xuid-choco': chocoKpis,
      },
      teamAvgKpis: makeKPIs({ kills_per_game: 8.0 }),
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)

    // Initial : 8.70 (kpis du main)
    expect(screen.getByText('8.70')).toBeInTheDocument()

    // Click sur Chocoboflor
    const chocoButton = screen.getByText('Chocoboflor').closest('button')
    expect(chocoButton).not.toBeNull()
    fireEvent.click(chocoButton!)

    // Désormais 6.50 (kpis de Chocoboflor)
    expect(screen.getByText('6.50')).toBeInTheDocument()
    // Reset bar visible
    expect(screen.getByText(/Vue active : Chocoboflor/)).toBeInTheDocument()
    expect(screen.getByText(/revenir à mes stats/)).toBeInTheDocument()
  })

  it('reset drill-down : click sur ✕ → retour à activeXuid', () => {
    const kpis = makeKPIs()
    const chocoKpis = makeKPIs({ kills_per_game: 6.5 })
    const squad = {
      score: makeSquadScore(),
      players: [
        makePlayerCard('xuid-me', 'Spartan-117', 50, 'above'),
        makePlayerCard('xuid-choco', 'Chocoboflor', 40, 'below'),
      ],
      kpisByXuid: {
        'xuid-me': kpis,
        'xuid-choco': chocoKpis,
      },
      teamAvgKpis: makeKPIs({ kills_per_game: 8.0 }),
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)

    // Drill-down
    fireEvent.click(screen.getByText('Chocoboflor').closest('button')!)
    expect(screen.getByText('6.50')).toBeInTheDocument()

    // Reset
    fireEvent.click(screen.getByText(/revenir à mes stats/))
    expect(screen.getByText('8.70')).toBeInTheDocument()
    expect(screen.queryByText(/Vue active : Chocoboflor/)).not.toBeInTheDocument()
  })
})

describe('SessionBriefing — trends', () => {
  it('kills > team_avg → trend "above" affiché en vert (▲)', () => {
    const kpis = makeKPIs({ kills_per_game: 12.0 }) // > team_avg 8.0
    const squad = {
      score: makeSquadScore(),
      players: [makePlayerCard('xuid-me', 'Me', 50, 'above')],
      kpisByXuid: { 'xuid-me': kpis },
      teamAvgKpis: makeKPIs({ kills_per_game: 8.0 }),
      activeXuid: 'xuid-me',
    }
    const { container } = renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)
    // Le symbole ▲ doit apparaître dans la grille (pas seulement dans la verdict band)
    expect(container.querySelectorAll(':scope :where(span)')).not.toHaveLength(0)
    expect(screen.getByText('12.00')).toBeInTheDocument()
    // Cherche un span avec ▲ près de la valeur kills
    const trendSymbols = screen.getAllByText('▲')
    expect(trendSymbols.length).toBeGreaterThan(0)
  })

  it('deaths > team_avg → trend "below" (lower_is_better, inversion)', () => {
    const kpis = makeKPIs({ deaths_per_game: 14.0 }) // > team_avg 10.0 = mauvais
    const squad = {
      score: makeSquadScore(),
      players: [makePlayerCard('xuid-me', 'Me', 50, 'above')],
      kpisByXuid: { 'xuid-me': kpis },
      teamAvgKpis: makeKPIs({ deaths_per_game: 10.0 }),
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)
    expect(screen.getByText('14.00')).toBeInTheDocument()
    // ▼ apparaît (deaths > team_avg = below pour lower_is_better)
    const trendSymbols = screen.getAllByText('▼')
    expect(trendSymbols.length).toBeGreaterThan(0)
  })
})

describe('SessionBriefing — outcomes pluralisation', () => {
  it('rend "1 Victoire" (singulier) et "7 Défaites" (pluriel) — squad mode', () => {
    // La Results bar avec libellés outcomes est désormais dans SquadVerdict
    // (squad mode uniquement) ; en solo, pas de Results bar.
    const kpis = makeKPIs({ outcomes: { wins: 1, losses: 7, ties: 0, dnf: 0 } })
    const squad = {
      score: makeSquadScore(),
      players: [makePlayerCard('xuid-me', 'Spartan-117', 50, 'above')],
      kpisByXuid: { 'xuid-me': kpis },
      teamAvgKpis: makeKPIs(),
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)
    expect(screen.getByText(/Victoire\b/)).toBeInTheDocument()
    expect(screen.getByText(/Défaites\b/)).toBeInTheDocument()
  })
})

describe('SessionBriefing — placement Matchs/Durée selon mode', () => {
  it('mode solo : Matchs joués + Durée totale visibles dans le KpiGrid', () => {
    const kpis = makeKPIs({ matches_count: 12, total_play_seconds: 6540, avg_match_seconds: 654 })
    renderWithProviders(<SessionBriefing kpis={kpis} />)
    expect(screen.getByText('Matchs joués')).toBeInTheDocument()
    expect(screen.getByText('Durée totale')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByText(/10min54\/match/)).toBeInTheDocument()
  })

  it('mode squad : Matchs joués + Durée totale présents UNE seule fois (dans SquadVerdict, pas dupliqués dans KpiGrid)', () => {
    const kpis = makeKPIs({ matches_count: 12, total_play_seconds: 6540, avg_match_seconds: 654 })
    const squad = {
      score: makeSquadScore(),
      players: [makePlayerCard('xuid-me', 'Spartan-117', 50, 'above')],
      kpisByXuid: { 'xuid-me': kpis },
      teamAvgKpis: makeKPIs(),
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)
    // Les labels existent (dans la verdict bar) mais pas en double.
    expect(screen.getAllByText('Matchs joués')).toHaveLength(1)
    expect(screen.getAllByText('Durée totale')).toHaveLength(1)
    // Valeurs visibles (rendues dans la verdict bar).
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByText(/10min54\/match/)).toBeInTheDocument()
  })
})

describe('SessionBriefing — Delta rang (card conditionnelle)', () => {
  it('rend "Delta CSR" + "+27" quand rank_delta.kind=csr et value=27 (entier)', () => {
    const kpis = makeKPIs({ rank_delta: { kind: 'csr', value: 27, count: 3 } })
    renderWithProviders(<SessionBriefing kpis={kpis} />)
    expect(screen.getByText('Delta CSR')).toBeInTheDocument()
    expect(screen.getByText('+27')).toBeInTheDocument()
  })

  it('rend "Delta LUSR" + "−0.02" quand rank_delta.kind=lusr et value<0 (2 décimales)', () => {
    const kpis = makeKPIs({ rank_delta: { kind: 'lusr', value: -0.02, count: 2 } })
    renderWithProviders(<SessionBriefing kpis={kpis} />)
    expect(screen.getByText('Delta LUSR')).toBeInTheDocument()
    expect(screen.getByText('−0.02')).toBeInTheDocument()
  })

  it('rend "±0" pour delta CSR à zéro (cas neutral)', () => {
    const kpis = makeKPIs({ rank_delta: { kind: 'csr', value: 0, count: 1 } })
    renderWithProviders(<SessionBriefing kpis={kpis} />)
    expect(screen.getByText('±0')).toBeInTheDocument()
  })

  it("n'ajoute PAS la card si rank_delta absent", () => {
    const kpis = makeKPIs() // pas de rank_delta
    renderWithProviders(<SessionBriefing kpis={kpis} />)
    expect(screen.queryByText(/Delta CSR/)).not.toBeInTheDocument()
    expect(screen.queryByText(/Delta LUSR/)).not.toBeInTheDocument()
  })
})

describe('SessionBriefing — fallback drill-down', () => {
  it('xuid manquant dans kpisByXuid → fallback sur kpis du main (pas de crash)', () => {
    const kpis = makeKPIs()
    const squad = {
      score: makeSquadScore(),
      players: [
        makePlayerCard('xuid-me', 'Me', 50, 'above'),
        makePlayerCard('xuid-missing', 'NoData', 40, 'below'),
      ],
      kpisByXuid: {
        'xuid-me': kpis,
        // xuid-missing absent volontairement
      },
      teamAvgKpis: makeKPIs(),
      activeXuid: 'xuid-me',
    }
    renderWithProviders(<SessionBriefing kpis={kpis} squad={squad} />)
    fireEvent.click(screen.getByText('NoData').closest('button')!)
    // Pas de crash : grille rendue avec les kpis du main par fallback
    expect(screen.getByText('8.70')).toBeInTheDocument()
  })
})
