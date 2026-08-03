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

describe('ExplorerBriefingStrip — carte « Par contexte » (DP-3)', () => {
  it('rend la carte comme cellule SIBLING de la grille « Par… », aux côtés des dimensions', () => {
    const briefing = { ...makeBriefing(120, 120), context_split: contextSplit }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.context_split_title')
    expect(text).toContain('explorer.filters.context_solo')
    expect(text).toContain('explorer.filters.context_squad')
    // DP-3 : « Par contexte » est un ENFANT DIRECT de la grille « Par… » (sibling des
    // dimensions), plus empilé dans une sous-colonne flex.
    const grids = container.querySelectorAll('[class*="grid-template-columns"]')
    const modulesGrid = grids[grids.length - 1]
    const ctxCell = Array.from(modulesGrid?.children ?? []).find((c) =>
      c.textContent?.includes('explorer.briefing.context_split_title'),
    )
    expect(ctxCell).toBeDefined()
    expect(modulesGrid?.textContent).toContain('explorer.briefing.dim_map')
  })

  it('omet la carte quand context_split est absent', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.context_split_title')
  })
})

describe('ExplorerBriefingStrip — compteurs de dimension (DP-10)', () => {
  it('affiche le compteur en nombre seul ; « X matchs » reste en title (aria)', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    // DimensionRow (MapA, 12 matchs) : le span compteur affiche « 12 » ; le libellé
    // dim_matches (« X matchs ») est porté par l'attribut title (survol/aria), pas rendu
    // en texte visible.
    const counter = Array.from(container.querySelectorAll('span')).find(
      (s) => s.getAttribute('title') === 'explorer.briefing.dim_matches',
    )
    expect(counter).toBeDefined()
    expect(counter?.textContent).toBe('12')
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
// rankedLusrMulti : DEUX chaînes du MÊME type LUSR (playlist_group distincts) →
// deux lignes AVEC label de chaîne (lusrChainLabel), jamais de flèche inter-chaînes.
const rankedLusrMulti = {
  kinds: [
    { kind: 'LUSR', playlist_group: 'arena_slayer', matches: 15, tier_start_label: 'Or I', tier_end_label: 'Or IV', delta_per_match: 0.5 },
    { kind: 'LUSR', playlist_group: 'btb', matches: 10, tier_start_label: 'Argent II', tier_end_label: 'Or I', delta_per_match: 0.3 },
  ],
}

describe('ExplorerBriefingStrip — Classement (RankedBlock, DEC-LAYOUT/DEC-RANK-FE)', () => {
  beforeEach(() => {
    // État fail-open : capability 'ranked' active → RankedBlock rendu.
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })
  afterEach(() => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  it('rend le bloc Classement (progression début → fin + pt/match) quand ranked + capability', () => {
    const briefing = { ...makeBriefing(120, 120), ranked: rankedSingle }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.ranked_title')
    expect(text).toContain('CSR')
    expect(text).toContain('Or III → Platine I') // progression début → fin (une chaîne)
    expect(text).toContain('explorer.briefing.ranked_per_match')
  })

  it('rend une ligne PAR CHAÎNE (LUSR multi) avec label de chaîne, jamais de flèche inter-chaînes', () => {
    const briefing = { ...makeBriefing(120, 120), ranked: rankedLusrMulti }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('Or I → Or IV') // chaîne arena_slayer
    expect(text).toContain('Argent II → Or I') // chaîne btb
    expect(text).toContain('Grande Équipe') // label de chaîne btb (≥ 2 chaînes du type)
    expect(text).not.toContain('Or IV → Argent II') // jamais de flèche inter-chaînes
  })

  it('rend le Classement comme cellule SIBLING de la grille « Par… », PAS dans le socle', () => {
    const briefing = { ...makeBriefing(120, 120), ranked: rankedSingle }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    // Socle = flex-wrap (V6/DP-4) ; « Par… » = grille adaptative (grid-template-columns).
    const socle = container.querySelector('[class*="flex-wrap"]')
    expect(socle?.textContent).not.toContain('explorer.briefing.ranked_title')
    const grids = container.querySelectorAll('[class*="grid-template-columns"]')
    const modulesGrid = grids[grids.length - 1]
    // DP-3 : le Classement est un ENFANT DIRECT de la grille (sibling des dimensions).
    const rankedCell = Array.from(modulesGrid?.children ?? []).find((c) =>
      c.textContent?.includes('explorer.briefing.ranked_title'),
    )
    expect(rankedCell).toBeDefined()
    expect(modulesGrid?.textContent).toContain('explorer.briefing.dim_map')
  })

  it('rend « Par contexte » ET Classement comme cellules SÉPARÉES (siblings, plus empilées, DP-3)', () => {
    const briefing = { ...makeBriefing(120, 120), context_split: contextSplit, ranked: rankedSingle }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const grids = container.querySelectorAll('[class*="grid-template-columns"]')
    const modulesGrid = grids[grids.length - 1]
    const cells = Array.from(modulesGrid?.children ?? [])
    const ctxCell = cells.find((c) => c.textContent?.includes('explorer.briefing.context_split_title'))
    const rankedCell = cells.find((c) => c.textContent?.includes('explorer.briefing.ranked_title'))
    expect(ctxCell).toBeDefined()
    expect(rankedCell).toBeDefined()
    // Cellules DISTINCTES (siblings) : plus d'empilement dans un wrapper flex-col (V4).
    expect(ctxCell).not.toBe(rankedCell)
  })

  it('omet le bloc Classement quand la capability « ranked » est absente du titre', () => {
    useAppShellStore.setState({
      currentTitleSlug: 'partial',
      availableTitles: [
        { slug: 'partial', name: 'Partial', status: 'active', capabilities: ['matchmaking'], is_default: true, effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
      ],
    })
    const briefing = { ...makeBriefing(120, 120), ranked: rankedSingle }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    expect(container.textContent ?? '').not.toContain('explorer.briefing.ranked_title')
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
// pour exercer les 11 tooltips (4 tuiles hors Matchs + Séries marquantes + 4 cartes
// « Par… » + 1 bande ; le Pic FDA autonome a été retiré en V5, DP-1).
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

  it('pose une icône (i) sur 4 tuiles (hors Matchs) + Séries marquantes + 4 cartes « Par… » + la bande', () => {
    const { getAllByRole } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeFullBriefing()} t={t} />,
    )
    // Socle (i) : Taux de victoire, FDA, Perf, Durée totale (4) + Séries marquantes
    // (1 ; makeFullBriefing n'a ni Pic rang ni Pic MMR). Modules : 3 dimensions +
    // « Par contexte » + Classement (RankedBlock) + bande Moments forts (6). Total =
    // 11 (le Pic FDA autonome a été retiré, DP-1). La tuile Matchs n'a PAS de (i).
    expect(getAllByRole('button', { name: TIP_ARIA })).toHaveLength(11)
  })

  it('pose MOINS d’icônes quand des blocs sont omis (ni classement/séries/contexte/bande)', () => {
    // makeBriefing de base : scope + baseline + 1 dimension, aucun bloc conditionnel.
    const { getAllByRole } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    // Taux de victoire + FDA + Perf + Durée totale (4 tuiles) + 1 carte dimension = 5
    // (aucune conditionnelle : ni série, ni pic rang, ni classement ; plus de Pic FDA).
    expect(getAllByRole('button', { name: TIP_ARIA })).toHaveLength(5)
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

// makeBriefing enrichi de pics/durée pour exercer le socle refondu V4.
function withScope(over: Partial<NonNullable<ExplorerBriefing['scope']>>): ExplorerBriefing {
  const b = makeBriefing(120, 120)
  return { ...b, scope: { ...b.scope!, ...over } }
}

describe('ExplorerBriefingStrip — socle refondu V4 (DEC-TILES)', () => {
  it('rend le Taux de victoire hero : valeur neutre + ruban OutcomeBar + V-D-N (sans sparkline)', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(30, 120)} t={t} />,
    )
    const text = container.textContent ?? ''
    expect(text).toContain('60%') // valeur WR neutre (formatPercentInt(0.6))
    // Ruban OutcomeBar présent (barre h-1.5) + victoires (18) / défaites (10) flanquant.
    expect(container.querySelector('[class*="h-1.5"]')).not.toBeNull()
    expect(text).toContain('18')
    expect(text).toContain('10')
  })

  it('colore la valeur (moyenne) de la tuile Perf (getPerfColor → CSS var)', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    // La moyenne (mid) du triptyque Perf porte le style couleur ; le span RACINE du
    // triptyque a le même textContent quand min/max sont absents → cibler celui qui
    // porte un style inline couleur.
    const perfSpan = Array.from(container.querySelectorAll('span')).find(
      (s) => s.textContent === '65' && (s.getAttribute('style') ?? '').includes('color'),
    )
    expect(perfSpan).toBeDefined()
    expect(perfSpan?.getAttribute('style') ?? '').toContain('var(--')
  })

  it('rend Durée totale (« h min ») hors low_sample ; Pic FDA fusionné (plus de tuile autonome)', () => {
    const briefing = withScope({ total_duration_seconds: 152400, peak_kda: 4.2 })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.duration_total_label')
    expect(text).toContain('42 h 20') // formatDurationHM(152400)
    // Tuile Pic FDA autonome supprimée (DP-1) ; peak_kda réutilisé comme MAX du
    // triptyque FDA.
    expect(text).not.toContain('explorer.briefing.peak_fda_label')
    expect(text).toContain('4.20') // peak_kda.toFixed(2) = borne haute du triptyque FDA
  })

  it('omet Durée totale en low_sample (jamais de tuile Pic FDA)', () => {
    const briefing = {
      ...withScope({ total_duration_seconds: 152400, peak_kda: 4.2 }),
      low_sample: true,
    }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).not.toContain('explorer.briefing.duration_total_label')
    expect(text).not.toContain('explorer.briefing.peak_fda_label')
  })
})

describe('ExplorerBriefingStrip — Pic rang (DEC-PEAKRANK)', () => {
  it('affiche une ligne (1er système) quand un seul palier', () => {
    const briefing = withScope({ peak_ranks: [{ rating_type: 'lusr', tier_label: 'Diamant IV' }] })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.peak_rank_label')
    expect(text).toContain('Diamant IV')
    expect(text).toContain('lusr')
  })

  it('affiche 2 lignes (LUSR + CSR) quand deux systèmes', () => {
    const briefing = withScope({
      peak_ranks: [
        { rating_type: 'lusr', tier_label: 'Diamant IV' },
        { rating_type: 'csr', tier_label: 'Onyx' },
      ],
    })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('Diamant IV')
    expect(text).toContain('Onyx')
    expect(text).toContain('csr')
  })

  it('omet la tuile Pic rang quand peak_ranks est vide', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    expect(container.textContent ?? '').not.toContain('explorer.briefing.peak_rank_label')
  })
})

describe('ExplorerBriefingStrip — cascade des conditionnelles (DP-2, cap 8)', () => {
  it('rend les 3 conditionnelles quand présentes (Séries marquantes + Pic rang + Pic MMR)', () => {
    const briefing = {
      ...withScope({
        peak_ranks: [{ rating_type: 'lusr', tier_label: 'Diamant IV' }],
        peak_team_mmr: 1523,
      }),
      streaks: { best_win_streak: 7, worst_loss_streak: 4 },
    }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.streaks_title') // priorité 1
    expect(text).toContain('explorer.briefing.peak_rank_label') // priorité 2
    expect(text).toContain('explorer.briefing.peak_mmr_label') // priorité 3 — visible (DP-2)
  })

  it('plafonne le socle à 8 tuiles (5 base + 3 conditionnelles), sans trou', () => {
    const briefing = {
      ...withScope({
        peak_ranks: [{ rating_type: 'lusr', tier_label: 'Diamant IV' }],
        peak_team_mmr: 1523,
      }),
      streaks: { best_win_streak: 7, worst_loss_streak: 4 },
    }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    // Socle = flex-wrap (V6/DP-4) ; 5 tuiles de base + 3 conditionnelles = 8 flex-items.
    const socle = container.querySelector('[class*="flex-wrap"]')
    expect(socle?.children.length).toBe(8)
  })

  it('rend Pic MMR quand seules 2 conditionnelles sont présentes', () => {
    const briefing = {
      ...withScope({ peak_team_mmr: 1523 }),
      streaks: { best_win_streak: 7, worst_loss_streak: 4 },
    }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('explorer.briefing.streaks_title')
    expect(text).toContain('explorer.briefing.peak_mmr_label')
  })
})

describe('ExplorerBriefingStrip — triptyques, accents & centrage (V5 : DP-1/DP-6/DP-8)', () => {
  it('FDA : rend min · moyenne (colorée) · max ; bornes lisibles (DP-2) ; Pic FDA absente', () => {
    const briefing = withScope({ kda: 1.5, min_kda: 0.4, peak_kda: 4.2 })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('0.40') // min
    expect(text).toContain('1.50') // moyenne (mid)
    expect(text).toContain('4.20') // max = peak_kda
    expect(text).not.toContain('explorer.briefing.peak_fda_label')
    // La moyenne (mid) est colorée (style inline).
    const midSpan = Array.from(container.querySelectorAll('span')).find((s) => s.textContent === '1.50')
    expect(midSpan?.getAttribute('style') ?? '').toContain('color')
    // DP-2 (V6) : les bornes min/max sont lisibles — text-foreground + text-xs, plus muted/2xs.
    const minSpan = Array.from(container.querySelectorAll('span')).find((s) => s.textContent === '0.40')
    const maxSpan = Array.from(container.querySelectorAll('span')).find((s) => s.textContent === '4.20')
    expect(minSpan?.className).toContain('text-foreground')
    expect(minSpan?.className).toContain('text-xs')
    expect(minSpan?.className).not.toContain('text-muted-foreground')
    expect(maxSpan?.className).toContain('text-foreground')
  })

  it('Perf : rend min · moyenne (colorée) · max', () => {
    const briefing = withScope({ avg_perf: 65, min_perf: 40, max_perf: 90 })
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const text = container.textContent ?? ''
    expect(text).toContain('40') // min
    expect(text).toContain('65') // moyenne
    expect(text).toContain('90') // max
    const midSpan = Array.from(container.querySelectorAll('span')).find((s) => s.textContent === '65')
    expect(midSpan?.getAttribute('style') ?? '').toContain('var(--')
  })

  it('omet les bornes nulles : moyenne seule, sans « — » parasite', () => {
    // makeBriefing de base : ni min/max FDA ni min/max Perf → seules les moyennes.
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />)
    expect((container.textContent ?? '')).toContain('1.50') // moyenne FDA seule
    // Les triptyques (span inline-flex justify-center) ne rendent aucun tiret « — »
    // tant que la moyenne existe.
    const triptychs = container.querySelectorAll('span.inline-flex.items-baseline.justify-center')
    expect(triptychs.length).toBe(2) // FDA + Perf
    for (const tr of Array.from(triptychs)) {
      expect(tr.textContent).not.toContain('—')
    }
  })

  it('pose un accent (barre 3px) sur les 8 tuiles du socle (DP-6)', () => {
    const briefing = {
      ...withScope({
        peak_ranks: [{ rating_type: 'lusr', tier_label: 'Diamant IV' }],
        peak_team_mmr: 1523,
      }),
      streaks: { best_win_streak: 7, worst_loss_streak: 4 },
    }
    const { container } = renderWithProviders(<ExplorerBriefingStrip briefing={briefing} t={t} />)
    const socle = container.querySelector('[class*="flex-wrap"]')
    const accentBars = Array.from(socle?.querySelectorAll('div') ?? []).filter((d) =>
      d.className.includes('h-[3px]'),
    )
    expect(accentBars.length).toBe(8)
  })

  it('centre la valeur de première ligne des tuiles (text-center, DP-8)', () => {
    const { container } = renderWithProviders(
      <ExplorerBriefingStrip briefing={makeBriefing(120, 120)} t={t} />,
    )
    const centered = Array.from(container.querySelectorAll('div')).filter(
      (d) => d.className.includes('text-center') && d.className.includes('text-xl'),
    )
    // 5 tuiles de base (Matchs/WR/FDA/Perf/Durée), chacune avec sa div valeur centrée.
    expect(centered.length).toBe(5)
  })
})
