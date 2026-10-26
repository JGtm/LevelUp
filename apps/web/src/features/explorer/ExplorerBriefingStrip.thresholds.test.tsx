/**
 * Tests ExplorerBriefingStrip — seuils du briefing : échantillon faible (D6) et cartes
 * en état vide (D3, D5). Fichier à part de ExplorerBriefingStrip.test.tsx (plafond de
 * lignes) avec ses propres fixtures, minimales : le stub i18n expose ici les valeurs
 * interpolées pour contrôler le seuil et les comptes affichés.
 */
import { beforeEach, describe, expect, it } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerBriefing } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

import { ExplorerBriefingStrip } from './ExplorerBriefingStrip'

// Stub i18n : renvoie la clé, suivie des valeurs interpolées quand il y en a.
const t = ((key: string, values?: Record<string, string | number>) =>
  values ? `${key}|${JSON.stringify(values)}` : key) as (
  key: ExplorerManifestKey,
  values?: Record<string, string | number>,
) => string

const baseScope: NonNullable<ExplorerBriefing['scope']> = {
  matches: 120,
  wins: 70,
  losses: 45,
  ties: 5,
  dnf: 0,
  win_rate: 0.58,
  kda: 1.4,
  avg_perf: 62,
}

describe('ExplorerBriefingStrip — échantillon faible : seuls les blocs qui comparent disparaissent (D6)', () => {
  beforeEach(() => {
    // État fail-open : capability 'ranked' active → bloc Classement rendu.
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  // 8 matchs : socle + Classement + Arme + Séries + Moments forts, sans baseline ni « Par… ».
  const lowSampleBriefing: ExplorerBriefing = {
    scope: {
      ...baseScope,
      matches: 8,
      wins: 5,
      losses: 3,
      ties: 0,
      win_rate: 0.625,
      total_duration_seconds: 7200,
      peak_ranks: [{ rating_type: 'csr', tier_label: 'Onyx' }],
    },
    period_start: '2025-03-03T10:00:00Z',
    period_end: '2025-03-12T10:00:00Z',
    low_sample: true,
    ranked: {
      kinds: [
        { kind: 'CSR', matches: 8, tier_start_label: 'Or III', tier_end_label: 'Platine I', delta_per_match: 1.2 },
      ],
    },
    weapons: {
      entries: [{ label: 'Fusil de combat', kills: 40, class: 'shoulder' }],
      measured_kills: 40,
      scope_kills: 80,
    },
    streaks: { best_win_streak: 4, worst_loss_streak: 2 },
    dominance: { dominations: 2 },
  }

  it('rend Classement, Arme favorite, Moments forts, Durée, Séries et Pic rang ; aucune carte « Par… »', () => {
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={lowSampleBriefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.ranked_title')
    expect(text).toContain('explorer.briefing.weapons_title')
    expect(text).toContain('explorer.briefing.highlights_title')
    expect(text).toContain('explorer.briefing.duration_total_label')
    expect(text).toContain('explorer.briefing.streaks_title')
    expect(text).toContain('explorer.briefing.peak_rank_label')
    expect(text).not.toContain('explorer.briefing.dim_map')
    expect(text).not.toContain('explorer.briefing.dim_mode')
    expect(text).not.toContain('explorer.briefing.dim_playlist')
    expect(text).not.toContain('explorer.briefing.context_split_title')
    expect(text).not.toContain('explorer.briefing.vs_baseline')
  })

  it('garde la mention « Échantillon faible (N matchs) » comme explication unique', () => {
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={lowSampleBriefing} t={t} />)
    expect(container.textContent ?? '').toContain('explorer.briefing.low_sample|{"n":8}')
  })
})

describe('ExplorerBriefingStrip — cartes en état vide (D3, D5)', () => {
  const briefingWith = (over: Partial<ExplorerBriefing>): ExplorerBriefing => ({
    scope: baseScope,
    min_group_matches: 5,
    ...over,
  })

  it('dimension insufficient : titre « Par carte » conservé + texte d’état vide avec le seuil', () => {
    const briefing = briefingWith({ dimensions: [{ dimension: 'map', entries: [], insufficient: true }] })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.dim_map')
    expect(text).toContain('explorer.briefing.insufficient_title')
    expect(text).toContain('explorer.briefing.dim_insufficient_map|{"n":5}')
    // L'(i) de la carte reste là : même en-tête qu'une carte pleine.
    const header = Array.from(container.querySelectorAll('.border-b')).find((h) =>
      h.textContent?.includes('explorer.briefing.dim_map'),
    )
    expect(header).toBeDefined()
    expect(header?.querySelector('button')).not.toBeNull()
    // Aucune ligne de comparaison, aucune action dans l'état vide.
    expect(container.querySelectorAll('li')).toHaveLength(0)
  })

  it('chaque dimension porte son propre libellé d’état vide (mode, sélection)', () => {
    const briefing = briefingWith({
      dimensions: [
        { dimension: 'mode', entries: [], insufficient: true },
        { dimension: 'playlist', entries: [], insufficient: true },
      ],
    })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.dim_insufficient_mode|{"n":5}')
    expect(text).toContain('explorer.briefing.dim_insufficient_playlist|{"n":5}')
  })

  it('contexte insufficient : texte avec les deux comptes et le seuil', () => {
    const briefing = briefingWith({
      context_split: {
        insufficient: true,
        solo: { matches: 3, win_rate: 0, kda: 0 },
        squad: { matches: 63, win_rate: 0, kda: 0 },
      },
    })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.context_split_title')
    expect(text).toContain('explorer.briefing.insufficient_title')
    expect(text).toContain('explorer.briefing.context_insufficient|{"solo":3,"squad":63,"n":5}')
    expect(text).not.toContain('explorer.filters.context_solo')
  })
})
