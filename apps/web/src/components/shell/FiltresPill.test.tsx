/**
 * Tests FiltresPill — détection zombie en temps réel et compatibilité des filtres en cascade.
 *
 * Couvre :
 *  - Aucun zombie : sélection 100% compatible → pas de banner d'incompatibilité.
 *  - Zombie experience type : type sélectionné absent des options disponibles.
 *  - Zombie playlist : playlist sélectionnée incompatible avec le filtre actif.
 *  - Zombie mode : mode sélectionné incompatible avec la playlist.
 *  - Zombie map : carte sélectionnée incompatible avec le mode.
 *  - Plusieurs zombies simultanés → compteur correct.
 *  - Sélection mixte : certaines valeurs OK, d'autres zombie.
 *  - Click sur un zombie le désélectionne.
 *  - Pill "Filtres" s'affiche en rouge si zombie(s) présents.
 *
 * Tests d'intégration (FilterOmnibar + useFiltersPreview) :
 *  - Zombie affiché AVANT "Analyser" via useFiltersPreview (temps réel).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useSoloFilterStore as useGlobalFilterStore } from '@/stores/soloFilterStore'
import { DEFAULT_GAP_MINUTES } from '@/stores/createFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import type { FilterContextResolved } from '@/lib/api/types'

import { FiltresPill, FilterOmnibar } from './FilterOmnibar'
import type { FiltresPillProps } from './FilterOmnibar'

// ─── Helpers ─────────────────────────────────────────────────────────────────

function makeAvailable(overrides: Partial<FiltresPillProps['available']> = {}): FiltresPillProps['available'] {
  return {
    experience_types: [
      { label: 'PVP non classé', value: 'PVP non classé', count: 1 },
      { label: 'PVP classé', value: 'PVP classé', count: 1 },
    ],
    playlists: [
      { label: 'Quick Play', value: 'Quick Play', count: 1 },
      { label: 'Ranked Arena', value: 'Ranked Arena', count: 1 },
    ],
    modes: [
      { label: 'Slayer', value: 'Slayer', count: 1 },
      { label: 'CTF', value: 'CTF', count: 1 },
    ],
    maps: [
      { label: 'Aquarius', value: 'Aquarius', count: 1 },
      { label: 'Streets', value: 'Streets', count: 1 },
    ],
    ...overrides,
  }
}

function renderPill(props: Partial<FiltresPillProps> = {}) {
  const defaults: FiltresPillProps = {
    open: true,
    onToggle: vi.fn(),
    onClose: vi.fn(),
    available: makeAvailable(),
    cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
    cascadeCount: 0,
    onSetCascade: vi.fn(),
  }
  return renderWithProviders(<FiltresPill {...defaults} {...props} />)
}

// Contexte effectif conforme au contrat (champs requis, nullable explicites).
const DEFAULT_EFFECTIVE: FilterContextResolved['effective'] = {
  filter_mode: 'period',
  period: { start_date: null, end_date: null },
  sessions: {
    gap_minutes: DEFAULT_GAP_MINUTES,
    picked_sessions: [],
    picked_session_label: null,
    picked_solo_session_label: null,
    picked_squad_session_label: null,
  },
  cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
}

function buildResolved(availOverrides: Partial<FilterContextResolved['available_options']> = {}): FilterContextResolved {
  return {
    effective: DEFAULT_EFFECTIVE,
    available_options: {
      experience_types: [{ label: 'PVP non classé', value: 'PVP non classé', count: 1 }],
      playlists: [{ label: 'Quick Play', value: 'Quick Play', count: 1 }],
      modes: [{ label: 'Slayer', value: 'Slayer', count: 1 }],
      maps: [{ label: 'Aquarius', value: 'Aquarius', count: 1 }],
      ...availOverrides,
    },
    session_options: { all_sessions: [], solo_labels: [], squad_labels: [] },
    counts: { total_matches_before_filters: 5, total_matches_after_filters: 5 },
    period_presets: [],
  }
}

// ─── Tests FiltresPill — zombie detection ─────────────────────────────────────

describe('FiltresPill — zombie detection', () => {
  // Locale pinnée : le libellé « Filtres » et le tooltip d'incompatibilité sont
  // résolus via le manifest i18n (GH-4) — on fige 'fr' pour rendre les assertions
  // FR explicites.
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  // ── 1. Aucun zombie ──────────────────────────────────────────────────────

  it('pas de banner incompatibilité quand toutes les sélections sont dans les options disponibles', () => {
    renderPill({
      cascade: {
        experience_types: ['PVP non classé'],
        playlists: ['Quick Play'],
        modes: ['Slayer'],
        maps: ['Aquarius'],
      },
      cascadeCount: 4,
    })
    expect(screen.queryByText(/incompatible/i)).not.toBeInTheDocument()
  })

  it('pill Filtres affichée sans classe destructive si pas de zombie', () => {
    const { container } = renderPill({
      cascade: { experience_types: ['PVP non classé'], playlists: [], modes: [], maps: [] },
      cascadeCount: 1,
    })
    const btn = container.querySelector('button[aria-haspopup="dialog"]')
    expect(btn?.className).not.toMatch(/destructive/)
  })

  // ── 2. Zombie experience type ────────────────────────────────────────────

  it('zombie experience : type sélectionné absent des options → banner + strikethrough', () => {
    renderPill({
      available: makeAvailable({ experience_types: [{ label: 'PVP non classé', value: 'PVP non classé', count: 1 }] }),
      cascade: { experience_types: ['PVP classé'], playlists: [], modes: [], maps: [] },
      cascadeCount: 1,
    })
    expect(screen.getByText(/1 filtre incompatible/i)).toBeInTheDocument()
    const zombieEntry = screen.getByTitle(/incompatible avec les filtres actifs/i)
    expect(zombieEntry).toBeInTheDocument()
  })

  it('zombie experience : les options disponibles restent affichées normalement', () => {
    renderPill({
      available: makeAvailable({ experience_types: [{ label: 'PVP non classé', value: 'PVP non classé', count: 1 }] }),
      cascade: { experience_types: ['PVP classé'], playlists: [], modes: [], maps: [] },
      cascadeCount: 1,
    })
    const pvpCheckbox = screen.getByRole('checkbox', { name: /PVP non classé/i })
    expect(pvpCheckbox).toBeInTheDocument()
    expect(pvpCheckbox).not.toBeChecked()
  })

  // ── 3. Zombie playlist ───────────────────────────────────────────────────

  it('zombie playlist : playlist sélectionnée non disponible → banner', () => {
    renderPill({
      available: makeAvailable({ playlists: [{ label: 'Quick Play', value: 'Quick Play', count: 1 }] }),
      cascade: { experience_types: [], playlists: ['Ranked Arena'], modes: [], maps: [] },
      cascadeCount: 1,
    })
    expect(screen.getByText(/1 filtre incompatible/i)).toBeInTheDocument()
    // "Ranked Arena" apparaît barré (titre zombie)
    const zombieLabels = screen.getAllByTitle(/incompatible avec les filtres actifs/i)
    const zombieTexts = zombieLabels.map((el) => el.textContent)
    expect(zombieTexts.join(' ')).toContain('Ranked Arena')
  })

  // ── 4. Zombie mode ───────────────────────────────────────────────────────

  it('zombie mode : mode sélectionné non disponible → banner', () => {
    renderPill({
      available: makeAvailable({ modes: [{ label: 'Slayer', value: 'Slayer', count: 1 }] }),
      cascade: { experience_types: [], playlists: [], modes: ['SWAT'], maps: [] },
      cascadeCount: 1,
    })
    expect(screen.getByText(/1 filtre incompatible/i)).toBeInTheDocument()
    const zombieLabels = screen.getAllByTitle(/incompatible avec les filtres actifs/i)
    expect(zombieLabels.map((el) => el.textContent).join(' ')).toContain('SWAT')
  })

  // ── 5. Zombie map ────────────────────────────────────────────────────────

  it('zombie map : carte sélectionnée non disponible → banner', () => {
    renderPill({
      available: makeAvailable({ maps: [{ label: 'Aquarius', value: 'Aquarius', count: 1 }] }),
      cascade: { experience_types: [], playlists: [], modes: [], maps: ['Bazaar'] },
      cascadeCount: 1,
    })
    expect(screen.getByText(/1 filtre incompatible/i)).toBeInTheDocument()
    const zombieLabels = screen.getAllByTitle(/incompatible avec les filtres actifs/i)
    expect(zombieLabels.map((el) => el.textContent).join(' ')).toContain('Bazaar')
  })

  // ── 6. Plusieurs zombies simultanés ──────────────────────────────────────

  it('plusieurs zombies → banner affiche le bon compteur', () => {
    renderPill({
      available: makeAvailable({
        playlists: [{ label: 'Quick Play', value: 'Quick Play', count: 1 }],
        modes: [{ label: 'Slayer', value: 'Slayer', count: 1 }],
      }),
      cascade: {
        experience_types: [],
        playlists: ['Ranked Arena'],    // zombie
        modes: ['SWAT'],               // zombie
        maps: [],
      },
      cascadeCount: 2,
    })
    expect(screen.getByText(/2 filtres incompatibles/i)).toBeInTheDocument()
  })

  it('plusieurs zombies sur 3 dimensions → compteur = 3', () => {
    renderPill({
      available: makeAvailable({
        experience_types: [{ label: 'PVP non classé', value: 'PVP non classé', count: 1 }],
        playlists: [{ label: 'Quick Play', value: 'Quick Play', count: 1 }],
        modes: [],
      }),
      cascade: {
        experience_types: ['PVP classé'],  // zombie
        playlists: ['Ranked Arena'],        // zombie
        modes: ['Slayer'],                  // zombie (modes vides)
        maps: [],
      },
      cascadeCount: 3,
    })
    expect(screen.getByText(/3 filtres incompatibles/i)).toBeInTheDocument()
  })

  // ── 7. Sélection mixte : certaines OK, d'autres zombie ──────────────────

  it('sélection mixte : options compatibles affichées normalement, zombies barrés', () => {
    renderPill({
      available: makeAvailable({
        modes: [{ label: 'Slayer', value: 'Slayer', count: 1 }],
      }),
      cascade: {
        experience_types: [],
        playlists: [],
        modes: ['Slayer', 'SWAT'], // Slayer OK, SWAT zombie
        maps: [],
      },
      cascadeCount: 2,
    })
    // Slayer est coché normalement
    const slayerCheckbox = screen.getByRole('checkbox', { name: /^Slayer/i })
    expect(slayerCheckbox).toBeChecked()
    // SWAT est zombie (barré avec titre)
    expect(screen.getByText(/1 filtre incompatible/i)).toBeInTheDocument()
    const zombieLabels = screen.getAllByTitle(/incompatible avec les filtres actifs/i)
    expect(zombieLabels.map((el) => el.textContent).join(' ')).toContain('SWAT')
  })

  // ── 8. Click sur un zombie le désélectionne ──────────────────────────────

  it('click sur zombie appelle onSetCascade sans la valeur zombie', () => {
    const onSetCascade = vi.fn()
    renderPill({
      available: makeAvailable({ modes: [{ label: 'Slayer', value: 'Slayer', count: 1 }] }),
      cascade: { experience_types: [], playlists: [], modes: ['SWAT'], maps: [] },
      cascadeCount: 1,
      onSetCascade,
    })
    // Le label zombie contient une checkbox checked
    const zombieEntry = screen.getByTitle(/incompatible avec les filtres actifs/i)
    const zombieCheckbox = zombieEntry.querySelector('input[type="checkbox"]')
    expect(zombieCheckbox).toBeInTheDocument()
    fireEvent.click(zombieCheckbox!)
    expect(onSetCascade).toHaveBeenCalledWith(
      expect.objectContaining({ modes: [] }),
    )
  })

  // ── 9. Pill Filtres rouge si zombie ──────────────────────────────────────

  it('pill Filtres affiche ⚠ quand incompatibilités présentes (pill ouverte = popover visible)', () => {
    renderPill({
      open: false, // pill fermée pour voir le bouton trigger
      available: makeAvailable({ modes: [] }),
      cascade: { experience_types: [], playlists: [], modes: ['Slayer'], maps: [] },
      cascadeCount: 1,
    })
    // Le bouton trigger doit afficher ⚠
    const btn = screen.getByRole('button', { name: /^filtres/i })
    expect(btn).toHaveTextContent('⚠')
  })
})

// ─── Tests intégration : live preview via useFiltersPreview ──────────────────

describe('FilterOmnibar — zombie en temps réel (useFiltersPreview)', () => {
  beforeEach(() => {
    useGlobalFilterStore.getState().resetFilters()
    useGlobalFilterStore.getState().setResolvedContext(
      buildResolved({
        // Au départ : Slayer disponible dans les modes
        modes: [{ label: 'Slayer', value: 'Slayer', count: 1 }],
      }),
    )
    // Pas de player → useFiltersPreview désactivé → fallback sur resolvedContext
    useAppShellStore.setState({ currentPlayer: null })
  })

  it('sans player slug, les options disponibles proviennent du resolvedContext commité', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /^filtres/i }))
    expect(screen.getByRole('checkbox', { name: /^Slayer/i })).toBeInTheDocument()
  })

  it('zombie affiché immédiatement quand resolvedContext ne contient plus le mode sélectionné', async () => {
    // Scénario : l'utilisateur a "Slayer" dans sa cascade stockée, mais le
    // resolvedContext retourné (après changement de session par ex.) ne contient
    // plus Slayer dans les modes disponibles.
    useGlobalFilterStore.getState().setCascade({
      experience_types: [],
      playlists: [],
      modes: ['SWAT'],  // SWAT n'est pas dans le resolvedContext (Slayer seulement)
      maps: [],
    })
    renderWithProviders(<FilterOmnibar />)
    // La pill Filtres doit afficher ⚠ sans qu'on ait cliqué sur Analyser
    await waitFor(() => {
      const filtresPill = screen.getByRole('button', { name: /^filtres/i })
      expect(filtresPill).toHaveTextContent('⚠')
    })
  })

  it('pas de zombie quand la valeur sélectionnée est dans les options disponibles', () => {
    useGlobalFilterStore.getState().setCascade({
      experience_types: [],
      playlists: [],
      modes: ['Slayer'],  // Slayer est dans le resolvedContext
      maps: [],
    })
    renderWithProviders(<FilterOmnibar />)
    const filtresPill = screen.getByRole('button', { name: /^filtres/i })
    expect(filtresPill).not.toHaveTextContent('⚠')
  })
})
