/**
 * Tests ExplorerBriefingStrip — masquage des deltas « vs habituel » (item 1, P-1).
 *
 * Deux états couverts : plein historique (scope == baseline → aucun delta rendu,
 * ni socle ni ligne de dimension) et scope filtré (deltas présents, V1 inchangé).
 * Le stub i18n renvoie la clé ; les deltas sont posés NON NULS même en plein
 * historique pour prouver que le masquage dépend du flag, pas de la valeur.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
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

describe('ExplorerBriefingStrip — carte « Par contexte » (DP-4)', () => {
  it('rend la carte dans la grille « Par… », aux côtés des dimensions', () => {
    const briefing = { ...makeBriefing(120, 120), context_split: contextSplit }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.context_split_title')
    expect(text).toContain('explorer.filters.context_solo')
    expect(text).toContain('explorer.filters.context_squad')
    // Carte contexte = 4e cellule de la MÊME grille que les dimensions (DEC-3).
    const grid = container.querySelector('[class*="xl:grid-cols-4"]')
    expect(grid?.textContent).toContain('explorer.briefing.context_split_title')
    expect(grid?.textContent).toContain('explorer.briefing.dim_map')
  })

  it('omet la carte quand context_split est absent', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.context_split_title')
  })
})

describe('ExplorerBriefingStrip — tuile « Séries » (DP-3)', () => {
  it('rend la valeur bicolore V/D quand les deux segments sont non nuls', () => {
    const briefing = { ...makeBriefing(120, 120), streaks: { best_win_streak: 7, worst_loss_streak: 4 } }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.streaks_title')
    expect(text).toContain('explorer.briefing.streak_wins')
    expect(text).toContain('explorer.briefing.streak_losses')
  })

  it('omet le segment à zéro (scope 100 % victoires → pas de pire série)', () => {
    const briefing = { ...makeBriefing(120, 120), streaks: { best_win_streak: 5, worst_loss_streak: 0 } }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.streak_wins')
    expect(text).not.toContain('explorer.briefing.streak_losses')
  })

  it('omet la tuile quand les deux segments sont à zéro', () => {
    const briefing = { ...makeBriefing(120, 120), streaks: { best_win_streak: 0, worst_loss_streak: 0 } }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.streaks_title')
  })
})

// rankedSingle : un seul type de rating (CSR) avec paliers résolus + pt/match.
const rankedSingle = {
  kinds: [
    { kind: 'CSR', matches: 20, tier_start_label: 'Or III', tier_end_label: 'Platine I', delta_per_match: -1.4 },
  ],
}
// rankedMulti : deux types (CSR majoritaire + LUSR secondaire → 2e ligne compacte).
const rankedMulti = {
  kinds: [
    { kind: 'CSR', matches: 20, tier_start_label: 'Or III', tier_end_label: 'Platine I', delta_per_match: -1.4 },
    { kind: 'LUSR', matches: 10, tier_start_label: 'Or I', tier_end_label: 'Or IV', delta_per_match: 0.8 },
  ],
}

describe('ExplorerBriefingStrip — tuile « Classement » (DP-2)', () => {
  afterEach(() => {
    // Restaure l'état fail-open (capability 'ranked' active par défaut).
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  it('rend la tuile (palier de fin + type + depuis + pt/match) quand ranked + capability', () => {
    const briefing = { ...makeBriefing(120, 120), ranked: rankedSingle }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.ranked_title')
    expect(text).toContain('Platine I') // palier de FIN (valeur de la tuile)
    expect(text).toContain('CSR') // type majoritaire
    expect(text).toContain('explorer.briefing.ranked_since')
    expect(text).toContain('explorer.briefing.ranked_per_match')
  })

  it('affiche une 2e ligne de sous-texte pour un second type (multi-type)', () => {
    const briefing = { ...makeBriefing(120, 120), ranked: rankedMulti }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('CSR')
    expect(text).toContain('LUSR')
    expect(text).toContain('Or I → Or IV') // progression compacte du 2e type
  })

  it('omet la tuile quand la capability « ranked » est absente du titre', () => {
    useAppShellStore.setState({
      currentTitleSlug: 'partial',
      availableTitles: [
        { slug: 'partial', name: 'Partial', status: 'active', capabilities: ['matchmaking'], is_default: true, effective_hp_to_kill: 225 },
      ],
    })
    const briefing = { ...makeBriefing(120, 120), ranked: rankedSingle }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.ranked_title')
  })
})

describe('ExplorerBriefingStrip — bande « Moments forts » (DP-5)', () => {
  it('rend une bande NUE (libellé + pastilles, sans en-tête de carte)', () => {
    const briefing = {
      ...makeBriefing(120, 120),
      dominance: { dominations: 3, remontadas: 1 },
    }
    const { container, getByText } = renderWithProviders(
      <ExplorerBriefingStrip briefing={briefing} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.highlights_title')
    expect(text).toContain('×3')
    expect(text).toContain('×1')
    // Bande NUE : le libellé n'est PAS dans un en-tête de carte bordé (border-b),
    // contrairement aux cartes-sections « Par… ».
    const label = getByText('explorer.briefing.highlights_title')
    expect(label.closest('.border-b')).toBeNull()
  })

  it('omet la bande quand tous les compteurs sont à zéro/absents', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.highlights_title')
  })
})

// aria-label réel de l'icône (i) (common.tooltip.more_info_aria) — FR|EN.
const TIP_ARIA = /Plus d'informations|More information/i

// Briefing COMPLET : 3 dimensions + contexte + classement + séries + moments forts,
// pour exercer les 10 tooltips (5 tuiles hors Matchs + 4 cartes « Par… » + 1 bande).
function makeFullBriefing(): ExplorerBriefing {
  return {
    ...makeBriefing(120, 120),
    dimensions: [
      { dimension: 'map', entries: [{ label: 'MapA', matches: 12, win_rate: 0.7, delta_win_rate: 0.2 }] },
      { dimension: 'mode', entries: [{ label: 'Slayer', matches: 20, win_rate: 0.55, delta_win_rate: 0.05 }] },
      { dimension: 'playlist', entries: [{ label: 'Arène', matches: 30, win_rate: 0.5, delta_win_rate: 0 }] },
    ],
    context_split: contextSplit,
    ranked: rankedSingle,
    streaks: { best_win_streak: 7, worst_loss_streak: 4 },
    dominance: { dominations: 3, remontadas: 1 },
  }
}

describe('ExplorerBriefingStrip — tooltips de légende (i) (DP-9)', () => {
  beforeEach(() => {
    // État fail-open : capability 'ranked' active (aucun titre restreint) → tuile
    // Classement rendue avec son tooltip.
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  it('pose une icône (i) sur 5 tuiles (hors Matchs) + 4 cartes « Par… » + la bande', () => {
    const { getAllByRole } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeFullBriefing()} t={t} />,
    )
    // Taux de victoire, FDA, Perf, Classement, Séries (5) + 3 dimensions + « Par
    // contexte » (4) + bande Moments forts (1) = 10. La tuile Matchs n'a PAS de (i).
    expect(getAllByRole('button', { name: TIP_ARIA })).toHaveLength(10)
  })

  it('pose MOINS d’icônes quand des blocs sont omis (ni classement/séries/contexte/bande)', () => {
    // makeBriefing de base : scope + baseline + 1 dimension, aucun bloc conditionnel.
    const { getAllByRole } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    // Taux de victoire + FDA + Perf (3 tuiles) + 1 carte dimension = 4.
    expect(getAllByRole('button', { name: TIP_ARIA })).toHaveLength(4)
  })

  it('ouvre le contenu du tooltip de la carte dimension au clic (tip_dimensions)', () => {
    const { getByText } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    // En-tête « Par carte » : le libellé et l'icône (i) partagent le même span.
    const dimTitle = getByText('explorer.briefing.dim_map')
    fireEvent.click(within(dimTitle).getByRole('button', { name: TIP_ARIA }))
    expect(screen.getByRole('tooltip').textContent).toContain('explorer.briefing.tip_dimensions')
  })
})
