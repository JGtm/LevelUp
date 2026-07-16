/**
 * Tests ExplorerBriefingStrip — masquage des deltas « vs habituel » (item 1, P-1).
 *
 * Deux états couverts : plein historique (scope == baseline → aucun delta rendu,
 * ni socle ni ligne de dimension) et scope filtré (deltas présents, V1 inchangé).
 * Le stub i18n renvoie la clé ; les deltas sont posés NON NULS même en plein
 * historique pour prouver que le masquage dépend du flag, pas de la valeur.
 */
import { describe, expect, it } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerBriefing, ExplorerBriefingContextSplit } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

import { ExplorerBriefingStrip } from './ExplorerBriefingStrip'

// Stub i18n : renvoie la clé (suffit pour un contrôle structurel du rendu).
const t = ((key: string) => key) as (
  key: ExplorerManifestKey,
  values?: Record<string, string | number>,
) => string

function makeBriefing(scopeMatches: number, baselineMatches: number): ExplorerBriefing {
  return {
    scope: {
      matches: scopeMatches,
      wins: 18,
      losses: 10,
      ties: 2,
      dnf: 0,
      win_rate: 0.6,
      kda: 1.5,
      avg_perf: 65,
    },
    baseline: {
      matches: baselineMatches,
      win_rate: 0.5,
      kda: 1.0,
      avg_perf: 55,
      delta_win_rate: 0.3, // → "+30 pts" (socle Bilan)
      delta_kda: 0.5,
      delta_perf: 10,
    },
    period_start: '2025-03-03T10:00:00Z',
    period_end: '2025-03-12T10:00:00Z',
    outcome_sequence: [],
    low_sample: false,
    dimensions: [
      {
        dimension: 'map',
        entries: [
          { label: 'MapA', matches: 12, win_rate: 0.7, delta_win_rate: 0.2 }, // → "+20 pts"
        ],
      },
    ],
  }
}

describe('ExplorerBriefingStrip — deltas vs habituel', () => {
  it('scope filtré : deltas socle ET ligne de dimension présents (V1)', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(30, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.vs_baseline')
    expect(text).toContain('+30 pts') // delta socle Bilan
    expect(text).toContain('+20 pts') // delta ligne de dimension
  })

  it('plein historique : aucun delta rendu, socle et dimensions conservés', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.vs_baseline')
    expect(text).not.toContain('+30 pts')
    expect(text).not.toContain('+20 pts')
    // Le reste demeure : libellé de dimension + label d'entrée.
    expect(text).toContain('MapA')
  })
})

const contextSplit: ExplorerBriefingContextSplit = {
  solo: { matches: 25, win_rate: 0.68, kda: 1.8 },
  squad: { matches: 15, win_rate: 0.4, kda: 1.1 },
}

describe('ExplorerBriefingStrip — carte contexte solo/escouade (item 6)', () => {
  it('rend la carte quand context_split est présent', () => {
    const briefing = { ...makeBriefing(120, 120), context_split: contextSplit }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.context_split_title')
    expect(text).toContain('explorer.filters.context_solo')
    expect(text).toContain('explorer.filters.context_squad')
  })

  it('omet la carte quand context_split est absent', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.context_split_title')
  })
})
