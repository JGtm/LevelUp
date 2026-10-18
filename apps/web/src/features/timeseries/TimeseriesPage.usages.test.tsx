/**
 * TimeseriesPage.usages.test — le câblage du graphe « Contrôle des ressources, cumul par match » sur
 * l'onglet Usages : l'axe PÉRIODE (D12), donc une légende sans encoche de dominance. `ChartCard` est
 * rendu pour de bon (seul ECharts est doublé) : sa légende de pied est celle que la page lui passe.
 */
import { describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { TimeseriesPageResponse } from '@/lib/api/types'

import { TimeseriesUsagesTab } from './TimeseriesPage.usages'
import { MATCH_ROWS, soloEmprise, soloEmpriseSansFilm } from './usages/usages.fixtures'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

describe('Onglet Usages — « cumul par match » sur une période', () => {
  it('légende : ressources, victoire / défaite, 50 % — sans encoche de dominance', () => {
    const data = { total_matches: 4, match_rows: MATCH_ROWS, emprise: soloEmprise() } as unknown as TimeseriesPageResponse
    renderWithProviders(<TimeseriesUsagesTab data={data} locale="fr" t={(k) => k} />)
    const fil = screen.getByTestId('emprise-fil')
    const legend = within(fil).getByRole('list', { name: 'Contrôle des ressources, cumul par match' })
    expect(within(legend).getByText('Victoire, défaite')).toBeInTheDocument()
    expect(within(legend).queryByText('Drapeau de dominance')).not.toBeInTheDocument()
  })
})

describe('Onglet Usages — le film des matchs n’est pas lu', () => {
  const sansResume = (extra: Record<string, unknown>) =>
    ({ total_matches: 4, match_rows: MATCH_ROWS, emprise: { ...soloEmpriseSansFilm(), film_unavailable: undefined, ...extra } }) as unknown as TimeseriesPageResponse
  const SECTIONS = ['bilan', 'carte', 'mine', 'prendre', 'lives', 'equipment'] as const

  it('analyse en cours : chaque section tirée du film garde son intertitre et un emplacement dans son bloc', () => {
    renderWithProviders(<TimeseriesUsagesTab data={sansResume({ matches_film_pending: 3 })} locale="fr" t={(k) => k} />)
    // La barre des armes spéciales (feuille de match) reste : « Rendement des ressources » garde sa carte.
    for (const id of SECTIONS.filter((s) => s !== 'prendre')) {
      const bloc = within(screen.getByTestId(`usages-section-${id}`)).getByTestId('film-gap')
      expect(within(bloc).getByTestId('analysis-pending-slot').textContent).toContain('Analyse du film en attente')
    }
    // La portée (positions lues au film) garde son intertitre : le titre déclare la capability (Infinite).
    expect(within(screen.getByTestId('usages-section-range')).getByTestId('film-gap')).toBeInTheDocument()
    expect(within(screen.getByTestId('usages-section-bilan')).getByText('Ressources')).toBeInTheDocument()
    expect(within(screen.getByTestId('usages-section-lives')).getByText('Isolement')).toBeInTheDocument()
  })

  it('film non analysé : le pointillé est dans le bloc parent', () => {
    renderWithProviders(<TimeseriesUsagesTab data={sansResume({ matches_film_missing: 4 })} locale="fr" t={(k) => k} />)
    const bloc = within(screen.getByTestId('usages-section-bilan')).getByTestId('film-gap')
    expect(bloc.textContent).toContain('Film non analysé')
    const notice = bloc.querySelector('.border-dashed')
    expect(notice).not.toBeNull()
    expect(bloc.contains(notice)).toBe(true)
  })

  it('titre sans lecture du film : aucun état, et rien à montrer hors la barre de la feuille de match', () => {
    renderWithProviders(
      <TimeseriesUsagesTab data={sansResume({ film_unavailable: 'film_unsupported', matches_film_missing: 4 })} locale="fr" t={(k) => k} />,
    )
    expect(screen.queryByTestId('film-gap')).toBeNull()
  })

  it('tous les films lus mais rien à montrer : l’onglet le dit dans un bloc parent', () => {
    const vide = { total_matches: 0, match_rows: [], emprise: null } as unknown as TimeseriesPageResponse
    renderWithProviders(<TimeseriesUsagesTab data={vide} locale="fr" t={(k) => k} />)
    const bloc = screen.getByTestId('timeseries-usages-empty')
    expect(bloc.querySelector('.border-dashed')).not.toBeNull()
  })
})

describe('Onglet Usages — aucun match à objectif', () => {
  const formes = (extra: Record<string, unknown>) => ({ available: true, matches_measured: 4, matches_total: 4, matches: [], ...extra })
  const avec = (f: unknown) =>
    ({ total_matches: 4, match_rows: MATCH_ROWS, emprise: soloEmprise(), formes_retenues: f }) as unknown as TimeseriesPageResponse

  it('le bloc « Objectif » garde son intertitre et dit le constat dans un bloc parent', () => {
    renderWithProviders(<TimeseriesUsagesTab data={avec(formes({}))} locale="fr" t={(k) => k} />)
    const section = screen.getByTestId('usages-section-objectif')
    expect(within(section).getByRole('heading', { name: 'Objectif' })).toBeInTheDocument()
    expect(section).toHaveTextContent('Aucun match à objectif sur ce périmètre')
    expect(section.querySelector('.border-dashed')).not.toBeNull()
  })

  it('en anglais, le constat est traduit', () => {
    renderWithProviders(<TimeseriesUsagesTab data={avec(formes({}))} locale="en" t={(k) => k} />)
    expect(screen.getByTestId('usages-section-objectif')).toHaveTextContent('No objective match in this scope')
  })

  it('lecture de l’objectif en échec : le bloc garde son intertitre et le dit', () => {
    renderWithProviders(
      <TimeseriesUsagesTab data={avec(formes({ available: false, unavailable_reason: 'load_failed' }))} locale="fr" t={(k) => k} />,
    )
    const section = screen.getByTestId('usages-section-objectif')
    expect(within(section).getByRole('heading', { name: 'Objectif' })).toBeInTheDocument()
    expect(section).toHaveTextContent('Objectif indisponible pour ce périmètre')
    expect(section).not.toHaveTextContent('Aucun match à objectif')
  })

  it('titre sans objectif (capability absente) ou bloc absent : le bloc est retiré', () => {
    renderWithProviders(
      <TimeseriesUsagesTab data={avec(formes({ available: false, unavailable_reason: 'unsupported' }))} locale="fr" t={(k) => k} />,
    )
    expect(screen.queryByTestId('usages-section-objectif')).toBeNull()
  })

  it('sans autre contenu : l’onglet porte son état vide, pas un constat d’objectif seul', () => {
    const vide = { total_matches: 0, match_rows: [], emprise: null, formes_retenues: formes({}) } as unknown as TimeseriesPageResponse
    renderWithProviders(<TimeseriesUsagesTab data={vide} locale="fr" t={(k) => k} />)
    expect(screen.getByTestId('timeseries-usages-empty')).toBeInTheDocument()
    expect(screen.queryByTestId('usages-section-objectif')).toBeNull()
  })
})
