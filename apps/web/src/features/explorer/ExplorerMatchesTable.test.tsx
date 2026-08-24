/**
 * Tests ExplorerMatchesTable — pagination par defaultPageSize (mode Joueur).
 *
 * Mode compact (defaultPageSize=10) : 10 lignes/page + navigation par page,
 * SANS expander (retiré — redondant avec la pagination, cf. retour user).
 * Mode legacy (defaultPageSize undefined) : PAGE_SIZE=20 par page.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

// TanStack Router : <Link> exige un RouterProvider, absent en test unitaire. On le
// remplace par un <a> qui INTERPOLE les params dans le template de route — ce que le
// test veut vérifier (la route ciblée et ses params), pas le rendu du routeur.
// Patron : features/synthesis/SynthesisHighlightsSection.test.tsx.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  type LinkStubProps = {
    children?: React.ReactNode
    to: string
    params?: Record<string, string>
  } & React.AnchorHTMLAttributes<HTMLAnchorElement>
  return {
    ...actual,
    Link: ({ children, to, params, ...rest }: LinkStubProps) => {
      let href = to
      for (const [key, value] of Object.entries(params ?? {})) {
        href = href.replace(`$${key}`, value)
      }
      return (
        <a href={href} {...rest}>
          {children}
        </a>
      )
    },
  }
})

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerMatchRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { ExplorerMatchesTable } from './ExplorerMatchesTable'

/** Force les capabilities du titre courant (fail-open par défaut sinon —
 *  cf. useCapability). Miroir du pattern SquadNetLivesChart.test.tsx. */
function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      {
        slug: 'test_title',
        name: 'Test',
        status: 'active',
        capabilities: caps,
        is_default: true,
        effective_hp_to_kill: 225,
        provides_damage_taken: true,
        provides_team_mmr: true,
        provides_max_killing_spree: true,
        offensive_conversion_p80: 0.9,
        defensive_resistance_p80: 1.65,
      },
    ],
  })
}

/** État par défaut (déterministe — n'assume rien de la valeur hydratée depuis
 *  localStorage) : titre par défaut fail-open + préférence Waypoint ON. */
function resetTitleAndWaypointPref() {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  useSettingsDraftStore.setState((s) => ({
    localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: true },
  }))
}

beforeEach(resetTitleAndWaypointPref)
afterEach(resetTitleAndWaypointPref)

function makeRow(i: number, overrides: Partial<ExplorerMatchRow> = {}): ExplorerMatchRow {
  return {
    match_id: `match-${i}`,
    start_time: '2026-05-26T12:00:00Z',
    start_time_label: `2026-05-26 12:0${i}`,
    map_ui: `Map${i}`,
    mode_ui: 'Slayer',
    playlist_label: 'Quick Play',
    outcome_label: 'Victoire',
    outcome_code: 2,
    score_label: '50-30',
    is_with_friends: false,
    experience_type_label: 'PvP',
    kills: 10,
    deaths: 5,
    assists: 3,
    kda: 2.1,
    ...overrides,
  }
}

function makeRows(n: number): ExplorerMatchRow[] {
  return Array.from({ length: n }, (_, i) => makeRow(i + 1))
}

/** Ordre des lignes du tbody, identifiées par le nom de carte qu'elles contiennent. */
function bodyMapOrder(names: string[]): string[] {
  const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('ExplorerMatchesTable — pagination (defaultPageSize)', () => {
  it('compact mode (defaultPageSize=10) affiche 10 lignes/page, sans expander', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    // 10 rows visibles dans le tbody (page 1).
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    expect(tbody?.querySelectorAll('tr').length).toBe(10)
    // Expander retiré ; pagination présente (2 pages pour 15 lignes).
    expect(screen.queryByTestId('explorer-matches-table-expander')).not.toBeInTheDocument()
    expect(screen.getByText(/Page 1 \/ 2/)).toBeInTheDocument()
  })

  it('pagination : "Suivant" affiche les lignes restantes (page 2)', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    fireEvent.click(screen.getByText('Suivant'))
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    // Page 2 = 5 lignes restantes (15 - 10).
    expect(tbody?.querySelectorAll('tr').length).toBe(5)
  })

  it('legacy mode (defaultPageSize undefined) affiche 15 lignes sans expander', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable rows={rows} playerSlug="me" alwaysShowPagination />,
    )
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    // PAGE_SIZE=20 ≥ 15 → tout est visible.
    expect(tbody?.querySelectorAll('tr').length).toBe(15)
    // Pas de bouton expander.
    expect(screen.queryByTestId('explorer-matches-table-expander')).not.toBeInTheDocument()
  })

  it('compact mode avec ≤ defaultPageSize lignes : pas d\'expander', () => {
    const rows = makeRows(8)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    // Pas d'expander (rows.length=8 ≤ defaultPageSize=10).
    expect(screen.queryByTestId('explorer-matches-table-expander')).not.toBeInTheDocument()
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    expect(tbody?.querySelectorAll('tr').length).toBe(8)
  })

  it('sans extraColumns → aucune colonne « Δ rang » (réservée à la vue session)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(2)} playerSlug="me" />)
    expect(screen.getByTestId('explorer-matches-table')).toBeInTheDocument()
    expect(screen.queryByText('Δ rang')).not.toBeInTheDocument()
    expect(screen.queryByText('Δ rank')).not.toBeInTheDocument()
  })
})

describe('ExplorerMatchesTable — tri CLIENT par en-têtes (mode Matchs)', () => {
  it('sans `sortable` → en-têtes statiques (aucun bouton de tri, pas d’aria-sort)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(2)} playerSlug="me" />)
    const dateHeader = screen.getByRole('columnheader', { name: 'Date' })
    expect(within(dateHeader).queryByRole('button')).not.toBeInTheDocument()
    expect(dateHeader).not.toHaveAttribute('aria-sort')
  })

  it('avec `sortable` → Date triée descendante par défaut (aria-sort + ▼) ; colonnes non actives = aria-sort none', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(3)} playerSlug="me" sortable />)
    const dateBtn = screen.getByRole('button', { name: 'Trier par Date' })
    expect(dateBtn.closest('th')).toHaveAttribute('aria-sort', 'descending')
    expect(within(dateBtn).getByText('▼')).toBeInTheDocument()
    // Résultat est désormais triable (tri client) mais inactif → aria-sort none.
    const outcomeBtn = screen.getByRole('button', { name: 'Trier par Résultat' })
    expect(outcomeBtn.closest('th')).toHaveAttribute('aria-sort', 'none')
  })

  it('colonne numérique (Frags) trie NUMÉRIQUEMENT, asc/desc (pas lexicographique)', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Alpha', kills: 2 }),
      makeRow(2, { map_ui: 'Bravo', kills: 30 }),
      makeRow(3, { map_ui: 'Charlie', kills: 10 }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    const btn = screen.getByRole('button', { name: 'Trier par Frags' })
    fireEvent.click(btn) // desc (numérique) → 30, 10, 2
    expect(bodyMapOrder(names)).toEqual(['Bravo', 'Charlie', 'Alpha'])
    fireEvent.click(btn) // asc → 2, 10, 30 (lexicographique donnerait 10,2,30)
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
  })

  it('colonne texte (Carte) trie ALPHABÉTIQUEMENT, ascendant au 1er clic', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Charlie' }),
      makeRow(2, { map_ui: 'Alpha' }),
      makeRow(3, { map_ui: 'Bravo' }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    fireEvent.click(screen.getByRole('button', { name: 'Trier par Carte' }))
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('colonne dérivée (Résultat) trie sur outcome_code BRUT, pas le libellé traduit', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Alpha', outcome_code: 3 }), // Défaite
      makeRow(2, { map_ui: 'Bravo', outcome_code: 1 }), // Égalité
      makeRow(3, { map_ui: 'Charlie', outcome_code: 2 }), // Victoire
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    fireEvent.click(screen.getByRole('button', { name: 'Trier par Résultat' }))
    // desc sur le code : 3, 2, 1 → Alpha, Charlie, Bravo (un tri alpha du libellé
    // donnerait Défaite < Égalité < Victoire = Alpha, Bravo, Charlie).
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
  })

  it('valeurs nulles rangées EN BAS dans les deux sens (Frags null)', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Alpha', kills: 10 }),
      makeRow(2, { map_ui: 'Bravo', kills: null }),
      makeRow(3, { map_ui: 'Charlie', kills: 5 }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    const btn = screen.getByRole('button', { name: 'Trier par Frags' })
    fireEvent.click(btn) // desc → 10, 5, null(bas)
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
    fireEvent.click(btn) // asc → 5, 10, null(toujours bas)
    expect(bodyMapOrder(names)).toEqual(['Charlie', 'Alpha', 'Bravo'])
  })

  it('bascule la direction sur la colonne active (▼ desc → ▲ asc + aria-sort)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(3)} playerSlug="me" sortable />)
    const dateBtn = screen.getByRole('button', { name: 'Trier par Date' })
    expect(dateBtn.closest('th')).toHaveAttribute('aria-sort', 'descending')
    fireEvent.click(dateBtn)
    expect(dateBtn.closest('th')).toHaveAttribute('aria-sort', 'ascending')
    expect(within(dateBtn).getByText('▲')).toBeInTheDocument()
  })
})

describe('ExplorerMatchesTable — surlignage MVP/LVP par décile (DEC-DECILE)', () => {
  // NB : on assert `td.style.fontWeight` (600 best / 500 worst / '' neutre) plutôt
  // que backgroundColor : le CSSOM jsdom peut ignorer les valeurs `color-mix(...)`.
  // La teinte/couleur du token est vérifiée dans ExplorerMatchesTable.highlight.test.ts.
  // ≥ MIN_DECILE_SAMPLE (10) lignes pour que les déciles p10/p90 s'appliquent.

  // Frags = 100, 90, … 10 → p10=10, p90=90 : décile HAUT best, décile BAS worst.
  const decileKillsRows = () =>
    [100, 90, 80, 70, 60, 50, 40, 30, 20, 10].map((k, i) => makeRow(i + 1, { map_ui: `M${i}`, kills: k }))

  it('surligne le décile HAUT (600) et le décile BAS (500) des Frags sur tout le scope', () => {
    renderWithProviders(<ExplorerMatchesTable rows={decileKillsRows()} playerSlug="me" sortable />)
    expect((screen.getByText('100').closest('td') as HTMLElement).style.fontWeight).toBe('600')
    expect((screen.getByText('10').closest('td') as HTMLElement).style.fontWeight).toBe('500')
    // Milieu de distribution → neutre.
    expect((screen.getByText('50').closest('td') as HTMLElement).style.fontWeight).toBe('')
  })

  it('colonne quasi-uniforme (Morts toutes égales, p10 === p90) → aucune cellule surlignée', () => {
    const rows = Array.from({ length: 10 }, (_, i) => makeRow(i + 1, { map_ui: `M${i}`, deaths: 5 }))
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" />)
    for (const cell of screen.getAllByText('5')) {
      expect((cell.closest('td') as HTMLElement).style.fontWeight).toBe('')
    }
  })

  it('petit scope (< 10 matchs) → aucune cellule surlignée (décile non calculable)', () => {
    const rows = [
      makeRow(1, { map_ui: 'Alpha', kills: 33 }),
      makeRow(2, { map_ui: 'Bravo', kills: 17 }),
      makeRow(3, { map_ui: 'Charlie', kills: 25 }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" />)
    expect((screen.getByText('33').closest('td') as HTMLElement).style.fontWeight).toBe('')
    expect((screen.getByText('17').closest('td') as HTMLElement).style.fontWeight).toBe('')
  })

  it('surlignage indépendant du tri : le décile HAUT des Frags reste surligné après tri', () => {
    renderWithProviders(<ExplorerMatchesTable rows={decileKillsRows()} playerSlug="me" sortable />)
    fireEvent.click(screen.getByRole('button', { name: 'Trier par Frags' }))
    expect((screen.getByText('100').closest('td') as HTMLElement).style.fontWeight).toBe('600')
    expect((screen.getByText('10').closest('td') as HTMLElement).style.fontWeight).toBe('500')
  })

  it('surligne le décile MMR équipe (non inversé) ; la colonne Score n’est plus surlignée (DP-1 V6)', () => {
    // team_mmr = 100..900, 950 → p10=100, p90=900 ; non inversé (haut = meilleur).
    // Valeurs < 1000 → fmtMmr sans séparateur de milliers (getByText fiable).
    const rows = [950, 900, 800, 700, 600, 500, 400, 300, 200, 100].map((m, i) =>
      makeRow(i + 1, { map_ui: `M${i}`, team_mmr: m }),
    )
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    // MMR équipe : 950 (≥ p90) best (600), 100 (≤ p10) worst (500), 500 neutre.
    expect((screen.getByText('950').closest('td') as HTMLElement).style.fontWeight).toBe('600')
    expect((screen.getByText('100').closest('td') as HTMLElement).style.fontWeight).toBe('500')
    expect((screen.getByText('500').closest('td') as HTMLElement).style.fontWeight).toBe('')
    // Score (score_label « 50-30 ») retiré du set décile V6 → jamais surligné.
    for (const cell of screen.getAllByText('50-30')) {
      expect((cell.closest('td') as HTMLElement).style.fontWeight).toBe('')
    }
  })
})

describe('ExplorerMatchesTable — alignement par colonne (DEC-ALIGN)', () => {
  it('en-têtes : numériques à droite, texte à gauche (jamais centré)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(2)} playerSlug="me" />)
    // Numériques (Score, Perf) → th text-right (libellés d'en-tête non abrégés).
    expect(screen.getByText('Score').closest('th')).toHaveClass('text-right')
    expect(screen.getByText('Perf').closest('th')).toHaveClass('text-right')
    // Texte (Carte, Date) → th text-left.
    expect(screen.getByText('Carte').closest('th')).toHaveClass('text-left')
    expect(screen.getByText('Date').closest('th')).toHaveClass('text-left')
  })

  it('cellules : numériques à droite, texte à gauche', () => {
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow(1, { kills: 7, map_ui: 'Zulu' })]} playerSlug="me" />,
    )
    expect(screen.getByText('7').closest('td')).toHaveClass('text-right') // Frags (numérique)
    expect(screen.getByText('Zulu').closest('td')).toHaveClass('text-left') // Carte (texte)
  })
})

describe('ExplorerMatchesTable — colonne « Ouvrir sur Halo Waypoint » (I19)', () => {
  const WAYPOINT_LABEL = 'Ouvrir sur Halo Waypoint'

  it('visible par défaut (capability fail-open + préférence locale ON)', () => {
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow(1)]} playerSlug="Chocoboflor" />,
    )
    const link = screen.getByRole('link', { name: WAYPOINT_LABEL })
    expect(link).toHaveAttribute(
      'href',
      'https://www.halowaypoint.com/halo-infinite/players/Chocoboflor/matches/match-1',
    )
  })

  it('masquée quand le titre courant ne déclare pas la capability waypoint_match_url', () => {
    setTitleCaps(['team_mmr'])
    renderWithProviders(<ExplorerMatchesTable rows={[makeRow(1)]} playerSlug="me" />)
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })

  it('masquée quand la préférence locale showWaypointColumn est désactivée', () => {
    useSettingsDraftStore.setState((s) => ({
      localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: false },
    }))
    renderWithProviders(<ExplorerMatchesTable rows={[makeRow(1)]} playerSlug="me" />)
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })

  it('la prop columnVisibility du consommateur garde priorité (masquage explicite)', () => {
    renderWithProviders(
      <ExplorerMatchesTable
        rows={[makeRow(1)]}
        playerSlug="me"
        columnVisibility={{ waypoint: false }}
      />,
    )
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })
})

describe('ExplorerMatchesTable — colonne « Rejeu »', () => {
  const REPLAY_LABEL = 'Ouvrir le rejeu 2D du match'

  it('rend un lien interne vers la page de rejeu quand has_replay est vrai', () => {
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow(1, { has_replay: true })]} playerSlug="Chocoboflor" />,
    )
    const link = screen.getByRole('link', { name: REPLAY_LABEL })
    // Lien INTERNE (route de l'app), pas une URL externe.
    expect(link.getAttribute('href')).toContain('/matches/match-1/replay')
  })

  it('ne rend RIEN quand has_replay est faux ou absent', () => {
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow(1, { has_replay: false }), makeRow(2)]} playerSlug="me" />,
    )
    expect(screen.queryByRole('link', { name: REPLAY_LABEL })).not.toBeInTheDocument()
  })

  it('un seul lien par ligne portant un artefact', () => {
    renderWithProviders(
      <ExplorerMatchesTable
        rows={[makeRow(1, { has_replay: true }), makeRow(2), makeRow(3, { has_replay: true })]}
        playerSlug="me"
      />,
    )
    expect(screen.getAllByRole('link', { name: REPLAY_LABEL })).toHaveLength(2)
  })
})

// V72-32, corrigé V72-34 : badge « En placement » sur Perf/ΔPerf/Note quand la
// note manque parce que la phase CORRESPONDANTE est en placement. Deux signaux
// ÉTANCHES : Perf/ΔPerf lisent perf_placement_* (chaîne de performance), la Note
// lit placement_* (classement LUSR/CSR). Sans le signal correspondant → "-"
// inchangé, pas de faux état fabriqué.
//
// Depuis le fix badge (JGtm) : Perf/ΔPerf restent du TEXTE "En placement"
// (calibration de chaîne perf) ; la Note (colonne rating_type) ET la cellule
// de Rang (colonne skill_tier_label, même tableau) rendent désormais le badge
// IMAGE unranked_N (alt localisé "En placement"/"In placement") — les deux
// lisent le même placement_done/placement_total, d'où 2 images identiques.
describe('ExplorerMatchesTable — badge « En placement » (Perf/ΔPerf/Note, V72-34)', () => {
  it('placement des DEUX phases → texte sur Perf + ΔPerf (2), badge image sur Note + Rang (2)', () => {
    const row = makeRow(1, {
      perf_score: null,
      perf_tier: undefined,
      delta_perf: null,
      rating_type: null,
      placement_done: 3,
      placement_total: 10,
      perf_placement_done: 3,
      perf_placement_total: 10,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    const textBadges = screen.getAllByText('En placement')
    expect(textBadges).toHaveLength(2) // Perf + ΔPerf
    for (const badge of textBadges) {
      expect(badge).toHaveAttribute('title', expect.stringContaining('7'))
    }
    const imgBadges = screen.getAllByAltText('En placement')
    expect(imgBadges).toHaveLength(2) // Note + Rang, même signal placement_*
    for (const badge of imgBadges) {
      expect(badge).toHaveAttribute('title', expect.stringContaining('7'))
      expect(badge).toHaveAttribute('src', expect.stringContaining('unranked_3.png'))
    }
  })

  // LE cas de la correction (JGtm, 3 matchs BTB du 24/07) : Note LUSR établie mais
  // chaîne de performance encore en calibration (8/10) → badge sur Perf/ΔPerf
  // UNIQUEMENT, la colonne Note affiche son rating normalement.
  it('JGtm : LUSR établi + chaîne perf sous le seuil → badge sur Perf/ΔPerf seulement, Note intacte', () => {
    const row = makeRow(1, {
      perf_score: null,
      perf_tier: undefined,
      delta_perf: null,
      rating_type: 'LUSR',
      skill_tier_label: 'Gold II',
      placement_done: null,
      placement_total: null,
      perf_placement_done: 8,
      perf_placement_total: 10,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    const badges = screen.getAllByText('En placement')
    expect(badges).toHaveLength(2) // Perf + ΔPerf, PAS la Note
    for (const badge of badges) {
      expect(badge).toHaveAttribute('title', expect.stringContaining('2')) // 10 - 8
    }
    expect(screen.getByText('LUSR')).toBeInTheDocument()
  })

  // Symétrique : placement de classement seul (aucune perf en calibration) ne doit
  // PAS colorer Perf/ΔPerf — c'était le faux positif du signal partagé V72-32.
  it('placement de classement SEUL → badge image sur Note + Rang, Perf/ΔPerf intacts ("-")', () => {
    const row = makeRow(1, {
      perf_score: null,
      perf_tier: undefined,
      delta_perf: null,
      rating_type: null,
      placement_done: 3,
      placement_total: 10,
      perf_placement_done: null,
      perf_placement_total: null,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    expect(screen.queryByText('En placement')).not.toBeInTheDocument() // Perf/ΔPerf pas concernés
    expect(screen.getAllByAltText('En placement')).toHaveLength(2) // Note + Rang
  })

  it('cas structurel (perf_score nul SANS aucun signal) → "-" inchangé, pas de badge', () => {
    const row = makeRow(1, {
      perf_score: null,
      perf_tier: undefined,
      delta_perf: null,
      rating_type: null,
      placement_done: null,
      placement_total: null,
      perf_placement_done: null,
      perf_placement_total: null,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    expect(screen.queryByText('En placement')).not.toBeInTheDocument()
  })

  it('note présente (hors placement) → valeur affichée normalement, pas de badge', () => {
    const row = makeRow(1, {
      perf_score: 73,
      perf_tier: 2,
      delta_perf: 23,
      rating_type: 'LUSR',
      placement_done: null,
      placement_total: null,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    expect(screen.queryByText('En placement')).not.toBeInTheDocument()
    expect(screen.getByText('73')).toBeInTheDocument()
    expect(screen.getByText('LUSR')).toBeInTheDocument()
  })
})

describe('ExplorerMatchesTable — pastille « Prolongation » (colonne Dominance)', () => {
  it('is_overtime : pastille rendue avec le tooltip du dépassement', () => {
    const row = makeRow(1, { is_overtime: true, overtime_seconds: 43 })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    const pill = screen.getByTestId('explorer-overtime-pill')
    expect(pill).toHaveTextContent('Prolongation')
    expect(pill).toHaveAttribute('title', 'Prolongation : +0:43')
  })

  it('sans is_overtime et sans dominance : cellule "-" inchangée', () => {
    const row = makeRow(1, { dominance_flag: 0 })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    expect(screen.queryByTestId('explorer-overtime-pill')).not.toBeInTheDocument()
  })

  it('dominance ET prolongation coexistent sur la même ligne', () => {
    const row = makeRow(1, { dominance_flag: 3, is_overtime: true, overtime_seconds: 60 })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    expect(screen.getByTestId('explorer-overtime-pill')).toBeInTheDocument()
    expect(screen.getByText('Remontada')).toBeInTheDocument()
  })
})
