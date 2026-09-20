/**
 * Tests ExplorerCombatProfile — section "profil de combat" (5 graphes).
 *
 * Les graphes montent ECharts via ChartCard (lazy import echarts-for-react) ;
 * on mocke echarts-for-react car jsdom n'a pas de canvas (cf. mémoire
 * reference_echarts_jsdom_test_mock). Le stub rend l'option en JSON, ce qui
 * permet aussi de vérifier le group-by des modes (G5 donut).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerTargetRecentMatch } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

import { ExplorerCombatProfile } from './ExplorerCombatProfile'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

// Stub i18n : retourne la clé (suffit pour vérifier le rendu structurel).
const t = ((key: string) => key) as (key: ExplorerManifestKey) => string

function match(over: Partial<ExplorerTargetRecentMatch> & { match_id: string }): ExplorerTargetRecentMatch {
  return {
    match_id: over.match_id,
    start_time: over.start_time ?? '2026-05-01T10:00:00Z',
    map_ui: over.map_ui ?? 'Aquarius',
    mode_ui: over.mode_ui ?? 'Slayer',
    outcome: over.outcome ?? 2,
    rank: over.rank,
    kills: over.kills ?? 12,
    deaths: over.deaths ?? 8,
    assists: over.assists ?? 4,
    kda: over.kda ?? 1.8,
    score: over.score ?? 1500,
    damage_dealt: over.damage_dealt ?? 2600,
    damage_taken: over.damage_taken ?? 2200,
    max_killing_spree: over.max_killing_spree ?? 3,
    perfect_kills: over.perfect_kills ?? 1,
  }
}

describe('ExplorerCombatProfile', () => {
  it('ne rend rien si aucun match et aucun statut live à signaler', () => {
    renderWithProviders(<ExplorerCombatProfile liveMatches={[]} localMatches={[]} locale="fr" t={t} />)
    expect(screen.queryByTestId('explorer-combat-profile')).toBeNull()
  })

  // Lot A3 (fin de la dégradation muette) : quand les 2 sources sont vides MAIS
  // que le statut live explique pourquoi (no_auth/failed/local_partial), un
  // rendu minimal titré + badge remplace l'absence silencieuse.
  it('rend un bloc titré + badge quand tout est vide mais le statut explique pourquoi', () => {
    renderWithProviders(
      <ExplorerCombatProfile liveMatches={[]} localMatches={[]} locale="fr" t={t} combatLiveStatus="no_auth" />,
    )
    expect(screen.getByTestId('explorer-combat-profile-status-only')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-live-status-badge-no_auth')).toBeInTheDocument()
  })

  // Le tab "En direct" vide (mais le local a des données) est expliqué par un
  // badge dans l'en-tête plutôt qu'un simple disabled muet.
  it('affiche le badge dans l\'en-tête quand live est vide mais local a des données', () => {
    const local = [match({ match_id: 'C1' })]
    renderWithProviders(
      <ExplorerCombatProfile liveMatches={[]} localMatches={local} locale="fr" t={t} combatLiveStatus="failed" />,
    )
    expect(screen.getByTestId('explorer-combat-profile')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-live-status-badge-failed')).toBeInTheDocument()
  })

  it('rend la section + 5 graphes quand des matchs sont fournis', async () => {
    const matches = [
      match({ match_id: 'm1', start_time: '2026-05-01T10:00:00Z', mode_ui: 'Slayer', rank: 1 }),
      match({ match_id: 'm2', start_time: '2026-05-02T10:00:00Z', mode_ui: 'CTF', rank: undefined }),
      match({ match_id: 'm3', start_time: '2026-05-03T10:00:00Z', mode_ui: 'Slayer', rank: 2 }),
    ]
    renderWithProviders(<ExplorerCombatProfile liveMatches={matches} localMatches={[]} locale="fr" t={t} />)

    expect(screen.getByTestId('explorer-combat-profile')).toBeTruthy()
    // 5 ChartCard (G1..G5) rendus de façon synchrone (wrapper).
    expect(screen.getAllByTestId('chart-card')).toHaveLength(5)
    // Le contenu ECharts est lazy (Suspense) → attendre les stubs.
    const stubs = await screen.findAllByTestId('echarts-stub')
    expect(stubs.length).toBe(5)
  })

  it('groupe les modes pour le donut (G5)', async () => {
    const matches = [
      match({ match_id: 'm1', mode_ui: 'Slayer' }),
      match({ match_id: 'm2', mode_ui: 'Slayer' }),
      match({ match_id: 'm3', mode_ui: 'CTF' }),
    ]
    renderWithProviders(<ExplorerCombatProfile liveMatches={matches} localMatches={[]} locale="fr" t={t} />)

    const stubs = await screen.findAllByTestId('echarts-stub')
    // Le donut sérialise une série pie dont les data portent les noms de modes.
    const donut = stubs.find((el) => el.textContent?.includes('"type":"pie"'))
    expect(donut).toBeTruthy()
    expect(donut?.textContent).toContain('Slayer')
    expect(donut?.textContent).toContain('CTF')
  })

  it('affiche le LIVE par défaut et bascule sur le LOCAL via le toggle', async () => {
    const live = [match({ match_id: 'L1', mode_ui: 'Slayer' })]
    const local = [match({ match_id: 'C1', mode_ui: 'Behemoth' })]
    renderWithProviders(
      <ExplorerCombatProfile liveMatches={live} localMatches={local} locale="fr" t={t} />,
    )

    // Défaut = live → le donut modes contient le mode live ("Slayer"), pas le local.
    let stubs = await screen.findAllByTestId('echarts-stub')
    let donut = stubs.find((el) => el.textContent?.includes('"type":"pie"'))
    expect(donut?.textContent).toContain('Slayer')
    expect(donut?.textContent).not.toContain('Behemoth')

    // Bascule sur local → le donut reflète le mode local ("Behemoth").
    fireEvent.click(screen.getByTestId('combat-source-local'))
    stubs = await screen.findAllByTestId('echarts-stub')
    donut = stubs.find((el) => el.textContent?.includes('"type":"pie"'))
    expect(donut?.textContent).toContain('Behemoth')
  })
})
