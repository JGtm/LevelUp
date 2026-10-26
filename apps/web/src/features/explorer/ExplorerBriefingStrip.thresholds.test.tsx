/**
 * Tests ExplorerBriefingStrip — seuils du briefing : échantillon faible (D6, avertissement sur
 * la tuile Matchs sans rien retirer), cartes en état vide (D3, D5) et grille du socle. Fichier
 * à part de ExplorerBriefingStrip.test.tsx (plafond de lignes) avec ses propres fixtures,
 * minimales : le stub i18n expose ici les valeurs interpolées pour contrôler le seuil et les
 * comptes affichés.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ExplorerBriefing } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

import { SOCLE_GRID_CLASSES } from './ExplorerBriefing.logic'
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

describe('ExplorerBriefingStrip — échantillon faible : rien n’est retiré, un avertissement sur la tuile Matchs (D6)', () => {
  beforeEach(() => {
    // État fail-open : capability 'ranked' active → bloc Classement rendu.
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  // 8 matchs : socle + baseline + « Par… » + « Par contexte » + Classement + Arme + Séries + Moments forts.
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
    baseline: {
      matches: 120,
      win_rate: 0.5,
      kda: 1.0,
      avg_perf: 55,
      delta_win_rate: 0.125,
      delta_kda: 0.4,
      delta_perf: 7,
    },
    period_start: '2025-03-03T10:00:00Z',
    period_end: '2025-03-12T10:00:00Z',
    low_sample: true,
    min_group_matches: 5,
    dimensions: [
      { dimension: 'map', entries: [{ label: 'MapA', matches: 6, win_rate: 0.8, delta_win_rate: 0.3 }] },
    ],
    context_split: {
      solo: { matches: 5, win_rate: 0.6, kda: 1.5 },
      squad: { matches: 3, win_rate: 0.7, kda: 1.6 },
    },
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

  it('rend tous les blocs : socle, écarts « vs habituel », cartes « Par… », « Par contexte », Classement, Arme, Moments forts', () => {
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={lowSampleBriefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.ranked_title')
    expect(text).toContain('explorer.briefing.weapons_title')
    expect(text).toContain('explorer.briefing.highlights_title')
    expect(text).toContain('explorer.briefing.duration_total_label')
    expect(text).toContain('explorer.briefing.streaks_title')
    expect(text).toContain('explorer.briefing.peak_rank_label')
    expect(text).toContain('explorer.briefing.dim_map')
    expect(text).toContain('MapA')
    expect(text).toContain('explorer.briefing.context_split_title')
    expect(text).toContain('explorer.briefing.vs_baseline')
  })

  it('pose l’avertissement sur la tuile Matchs, avec son texte en libellé accessible, sans ligne à part', () => {
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={lowSampleBriefing} t={t} />)
    const warning = container.querySelector('[data-testid="briefing-low-sample"]')
    expect(warning).not.toBeNull()
    expect(warning?.getAttribute('role')).toBe('img')
    expect(warning?.getAttribute('aria-label')).toBe('explorer.briefing.low_sample|{"n":8}')
    // Dans la tuile Matchs : première enfant du socle.
    const socle = container.querySelector('[data-testid="briefing-socle"]')
    expect(socle?.children[0]?.contains(warning)).toBe(true)
    // Plus de ligne de texte sous le socle : le texte n'existe que dans le libellé de l'icône.
    expect(container.textContent ?? '').not.toContain('explorer.briefing.low_sample')
    expect(container.querySelector('p.text-2xs')).toBeNull()
  })

  it('montre le texte dans l’infobulle au survol de l’icône', () => {
    renderWithProviders(<ExplorerBriefingStrip briefing={lowSampleBriefing} t={t} />)
    fireEvent.mouseEnter(screen.getByTestId('briefing-low-sample'))
    expect(screen.getByRole('tooltip').textContent).toBe('explorer.briefing.low_sample|{"n":8}')
  })

  it('sans échantillon faible : aucune icône d’avertissement', () => {
    const briefing: ExplorerBriefing = { ...lowSampleBriefing, low_sample: false }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    expect(container.querySelector('[data-testid="briefing-low-sample"]')).toBeNull()
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

describe('ExplorerBriefingStrip — grille du socle : colonnes égales par container queries', () => {
  const base5: ExplorerBriefing = { scope: { ...baseScope, total_duration_seconds: 3600 } }

  it('8 tuiles : grille 2 -> 3 -> 4 -> 8 colonnes sous un conteneur @container', () => {
    const briefing: ExplorerBriefing = {
      scope: { ...baseScope, peak_ranks: [{ rating_type: 'csr', tier_label: 'Onyx' }], peak_team_mmr: 1523 },
      streaks: { best_win_streak: 4, worst_loss_streak: 2 },
    }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const socle = container.querySelector('[data-testid="briefing-socle"]')
    expect(socle?.children).toHaveLength(8)
    expect(socle?.className).toContain('grid gap-2')
    expect(socle?.className).toContain(SOCLE_GRID_CLASSES[8])
    expect(socle?.parentElement?.className).toContain('@container')
  })

  it('5 tuiles : grille 2 -> 3 -> 5 colonnes', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={base5} t={t} />,
    )
    const socle = container.querySelector('[data-testid="briefing-socle"]')
    expect(socle?.children).toHaveLength(5)
    expect(socle?.className).toContain(SOCLE_GRID_CLASSES[5])
    expect(socle?.className).not.toMatch(/\b(grow|basis-)/)
  })
})
