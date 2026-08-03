/**
 * Tests ExplorerMatchesTable — colonne « Rang » en image (V73-L2 2.4a) et colonne
 * « Score personnel » (V73-L2 2.5).
 *
 * Ces deux colonnes sont définies dans `baseColumns`, donc héritées par les CINQ
 * surfaces qui montent ce tableau (Explorer mode Matchs, Explorer mode Joueur
 * allié/ennemi, vue Session, onglet Progression, « Matchs marquants » Carrière) —
 * ce fichier est le seul endroit où leur contrat de rendu est vérifié.
 *
 * Le point sensible du badge : l'URL vient du BACKEND (adaptateur d'assets du
 * titre). Quand elle manque — titre sans badge, palier inconnu, placement en
 * cours — la colonne doit retomber sur le texte localisé, jamais afficher une
 * image cassée.
 *
 * Locale par défaut = fr → libellés français.
 */
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerMatchRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { ExplorerMatchesTable } from './ExplorerMatchesTable'

function resetTitleAndWaypointPref() {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  useSettingsDraftStore.setState((s) => ({
    localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: true },
  }))
}

beforeEach(resetTitleAndWaypointPref)
afterEach(resetTitleAndWaypointPref)

function makeRow(overrides: Partial<ExplorerMatchRow> = {}): ExplorerMatchRow {
  return {
    match_id: 'm1',
    start_time: '2026-05-26T12:00:00Z',
    start_time_label: '2026-05-26 12:00',
    map_ui: 'Aquarius',
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

/** Ordre des lignes du tbody, identifiées par le nom de carte qu'elles portent. */
function bodyMapOrder(names: string[]): string[] {
  const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('Colonne Rang — badge image avec repli texte', () => {
  it('rend le badge servi par le backend, avec le palier comme alternative textuelle', () => {
    renderWithProviders(
      <ExplorerMatchesTable
        rows={[
          makeRow({
            skill_tier_label: 'Diamant IV',
            skill_rank_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Diamond4.png',
          }),
        ]}
        playerSlug="me"
      />,
    )
    const img = screen.getByAltText('Diamant IV')
    expect(img).toHaveAttribute('src', '/static/ranks/halo_infinite/120px-HINF-CSR_Diamond4.png')
    // Le libellé n'est plus écrit en clair dans la cellule : l'image le remplace.
    expect(screen.queryByText('Diamant IV')).toBeNull()
  })

  it('sans URL de badge (titre sans image), garde le texte localisé du palier', () => {
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow({ skill_tier_label: 'Diamant IV' })]} playerSlug="me" />,
    )
    expect(screen.getByText('Diamant IV')).toBeInTheDocument()
    expect(screen.queryByAltText('Diamant IV')).toBeNull()
  })

  it('placement en cours : « X/Y » prime sur le badge', () => {
    renderWithProviders(
      <ExplorerMatchesTable
        rows={[
          makeRow({
            skill_tier_label: 'Diamant IV',
            skill_rank_image_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Diamond4.png',
            placement_done: 3,
            placement_total: 10,
          }),
        ]}
        playerSlug="me"
      />,
    )
    expect(screen.getByText('3/10')).toBeInTheDocument()
    expect(screen.queryByAltText('Diamant IV')).toBeNull()
  })
})

describe('Colonne Score personnel', () => {
  it('affiche le score personnel, distinct du score d’équipe', () => {
    renderWithProviders(
      <ExplorerMatchesTable
        rows={[makeRow({ personal_score: 2400, score_label: '50-30' })]}
        playerSlug="me"
      />,
    )
    // Score personnel formaté par la locale : le séparateur de milliers dépend
    // de l'ICU de l'environnement, on ne fige que les chiffres.
    expect(screen.getByText(/^2.?400$/)).toBeInTheDocument()
    // Le score d'ÉQUIPE reste rendu dans sa propre colonne.
    expect(screen.getByText('50-30')).toBeInTheDocument()
  })

  it('score absent → « - » (jamais un 0 trompeur)', () => {
    const { container } = renderWithProviders(
      <ExplorerMatchesTable
        rows={[makeRow({ personal_score: null, score_label: '50-30' })]}
        playerSlug="me"
      />,
    )
    // Comparaison de cellules ENTIÈRES : aucune ne vaut « 0 » (les autres
    // colonnes de la ligne valent 10 / 5 / 3 / 2.1 / 50-30).
    const cells = Array.from(container.querySelectorAll('tbody td')).map((td) => td.textContent)
    expect(cells).toContain('-')
    expect(cells).not.toContain('0')
  })

  it('en-tête FR + libellé complet exposé à l’assistance vocale', () => {
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow({ personal_score: 2400 })]} playerSlug="me" sortable />,
    )
    // L'en-tête est rendu sur 2 lignes (« Score » / « personnel ») : c'est
    // l'aria-label du bouton de tri qui porte le libellé complet et lisible.
    expect(screen.getByLabelText('Trier par Score personnel')).toBeInTheDocument()
  })

  it('parité EN : le libellé anglais est servi et rendu en entier', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(
      <ExplorerMatchesTable rows={[makeRow({ personal_score: 2400 })]} playerSlug="me" sortable />,
    )
    expect(screen.getByLabelText('Sort by Personal score')).toBeInTheDocument()
    // « Personal score » est le libellé d'en-tête le plus long du tableau : il est
    // coupé sur 2 lignes, mais jamais tronqué (aucun mot perdu).
    const header = screen.getByLabelText('Sort by Personal score')
    expect(header.textContent?.replace(/\s+/g, '')).toContain('Personalscore')
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('trie sur la valeur numérique', () => {
    renderWithProviders(
      <ExplorerMatchesTable
        rows={[
          makeRow({ match_id: 'a', map_ui: 'Aquarius', personal_score: 900 }),
          makeRow({ match_id: 'b', map_ui: 'Bazaar', personal_score: 2400 }),
          makeRow({ match_id: 'c', map_ui: 'Catalyst', personal_score: 1500 }),
        ]}
        playerSlug="me"
        sortable
      />,
    )
    const names = ['Aquarius', 'Bazaar', 'Catalyst']
    const header = screen.getByLabelText('Trier par Score personnel')

    fireEvent.click(header) // 1er clic : décroissant (NUMERIC_SORT)
    expect(bodyMapOrder(names)).toEqual(['Bazaar', 'Catalyst', 'Aquarius'])

    fireEvent.click(header) // 2e clic : croissant
    expect(bodyMapOrder(names)).toEqual(['Aquarius', 'Catalyst', 'Bazaar'])
  })
})
