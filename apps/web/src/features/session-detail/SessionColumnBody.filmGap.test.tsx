/**
 * SessionColumnBody.filmGap.test.tsx — « FRAGS ET USAGES » QUAND LE FILM DES MATCHS N'EST PAS LU.
 *
 * Retour du 2026-10-17 : une famille de cartes tirée du film (Ressources, Rendement, Isolement, Équipement,
 * Portée) disparaissait avec son intertitre quand les films de la soirée n'étaient pas encore analysés, et
 * rien ne disait pourquoi. Elle garde désormais son intertitre et UNE carte d'état dans un bloc parent :
 * « Analyse du film en cours » (indicateur) ou « Film non analysé » (pointillé). Un titre qui ne lit pas
 * le film (capability absente) n'a toujours rien.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { soloEmpriseSansFilm } from '@/features/timeseries/usages/usages.fixtures'
import type { SoloEmpriseBlock } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SessionColumnBody } from './SessionColumnBody'
import { buildSessionEmpriseModels, drawnCards, sessionColumnPresence, sessionFilmGaps } from './sessionEmprise.logic'
import type { SessionColumnBlocks } from './sessionEmprise.logic'
import { session2209, sessionSolo } from './sessionEmprise.fixtures'

vi.mock('echarts-for-react', () => ({ default: () => null }))
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useParams: () => ({ playerSlug: 'moi' }), useSearch: () => ({}), useNavigate: () => vi.fn() }
})

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const byKey = (o: { key?: string | null }) => o.key ?? ''

/** Un bloc d'Emprise dont aucun match n'a son résumé d'usage, et que le titre sait lire. */
function sansResume(extra: Partial<SoloEmpriseBlock>): SoloEmpriseBlock {
  return { ...soloEmpriseSansFilm(), film_unavailable: undefined, ...extra } as SoloEmpriseBlock
}

/** La colonne : la feuille de match seule (frags par classe, outils) et un film non lu. */
function colonneSansFilm(extra: Partial<SoloEmpriseBlock>): SessionColumnBlocks {
  return { ...sessionSolo(), emprise: sansResume(extra), lives: null, formes: null, rangeProfiles: null }
}

const monter = (blocks: SessionColumnBlocks, compact = false) =>
  renderWithProviders(<SessionColumnBody blocks={blocks} playerSlug="moi" compact={compact} />)

function intertitres(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('[data-session-subgroup]')).map((h) => h.textContent ?? '')
}

describe('sessionFilmGaps — une famille sans aucune carte dessinée porte l’état', () => {
  it('film non lu : une carte d’ancrage par famille tirée du film (la barre des armes spéciales, lue sur la feuille de match, reste)', () => {
    const col = colonneSansFilm({ matches_film_pending: 2 })
    const drawn = drawnCards(col, buildSessionEmpriseModels(col, byKey))
    expect(sessionFilmGaps(col, drawn)).toEqual({ control: 'pending', lives: 'pending', equipment: 'pending' })
    const presence = sessionColumnPresence(col)
    expect(presence.control && presence.production && presence.lives && presence.equipment).toBe(true)
    expect(presence.fil || presence.grid || presence.mine || presence.yield).toBe(false)
  })

  it('une famille qui dessine quelque chose ne porte pas d’état', () => {
    const base = session2209()
    const col: SessionColumnBlocks = { ...base, emprise: { ...base.emprise!, matches_film_missing: 1 } as SoloEmpriseBlock }
    const gaps = sessionFilmGaps(col, drawnCards(col, buildSessionEmpriseModels(col, byKey)))
    expect(Object.keys(gaps)).toEqual(['equipment'])
    expect(gaps.equipment).toBe('missing')
  })

  it('tous les films lus : aucun état ; titre sans lecture du film : aucun état', () => {
    const lu = colonneSansFilm({})
    expect(sessionFilmGaps(lu, drawnCards(lu, buildSessionEmpriseModels(lu, byKey)))).toEqual({})
    const h5 = colonneSansFilm({ film_unavailable: 'film_unsupported', matches_film_missing: 4 })
    expect(sessionFilmGaps(h5, drawnCards(h5, buildSessionEmpriseModels(h5, byKey)))).toEqual({})
  })
})

describe('SessionColumnBody — film non lu', () => {
  it('analyse en cours : chaque famille garde son intertitre et un emplacement « en cours » dans un bloc', () => {
    const { container } = monter(colonneSansFilm({ matches_film_pending: 2 }))
    expect(intertitres(container)).toEqual(['Ressources', 'Rendement des ressources', 'Isolement', 'Équipement'])
    const blocs = screen.getAllByTestId('film-gap')
    expect(blocs).toHaveLength(3)
    for (const bloc of blocs) {
      expect(within(bloc).getByTestId('analysis-pending-slot').textContent).toContain('Analyse du film en cours')
    }
    expect(screen.getByText('Frags et usages')).toBeInTheDocument()
  })

  it('film non analysé : le pointillé est DANS le bloc parent, jamais nu sous l’intertitre', () => {
    monter(colonneSansFilm({ matches_film_missing: 3 }))
    for (const bloc of screen.getAllByTestId('film-gap')) {
      expect(bloc.textContent).toContain('Film non analysé')
      const notice = bloc.querySelector('.border-dashed')
      expect(notice).not.toBeNull()
      expect(notice).not.toBe(bloc)
      expect(bloc.contains(notice)).toBe(true)
    }
  })

  it('la portée garde sa carte titrée, en colonne divisée comme en pleine page', () => {
    const { container } = monter(colonneSansFilm({ matches_film_pending: 1 }), true)
    const portee = container.querySelector('[data-testid="session-portee-film-gap"]')
    expect(portee).not.toBeNull()
    expect(portee!.textContent).toContain('Analyse du film en cours')
    expect(screen.getAllByText('Portée des engagements').length).toBeGreaterThan(0)
  })

  it('Halo 5 (film_unsupported) : aucun état, les familles du film restent absentes', () => {
    const { container } = monter({ ...sessionSolo(), emprise: soloEmpriseSansFilm(), lives: null, formes: null })
    expect(screen.queryByTestId('film-gap')).toBeNull()
    expect(intertitres(container)).not.toContain('Ressources')
  })

  it('anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    monter(colonneSansFilm({ matches_film_missing: 3 }))
    expect(screen.getAllByText('Film not analysed').length).toBeGreaterThan(0)
  })
})
