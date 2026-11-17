/**
 * Tests — LA COLONNE DU CADRE STANDARD montée dans la vraie page : la carte, puis une colonne de la
 * largeur de `COTES_COLONNE_STANDARD` (équipes en lignes, puis le fil), la fiche d'un joueur qui prend la
 * place du fil, et les deux autres cadres inchangés.
 *
 * Même montage que `ReplayPage.frames.test.tsx` : la vraie page, la vraie toile (contexte 2D enregistreur) et
 * le plus petit document du parc de fixtures de Go ; seuls le routeur et le réseau sont simulés. La
 * hauteur du fil et le défilement des équipes se résolvent dans le navigateur : ici on lit les cotes que
 * le rendu porte, jamais une mise en page.
 */
import { act, cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { testCapabilities } from '@/test/capabilities'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { REPLAY_TEXT } from './i18n/i18n'
import { COTES_LIGNE_FIL } from './model/cotesFeedLine'
import { COTES_COLONNE_STANDARD, FIL_STANDARD_HAUTEUR_MIN_PX } from './model/cotesStandard'
import type { ReplayFrame } from './model/replayFrame'
import { goFixtureEntries, loadGoFixture } from './test/goFixtures'
import { recordingContext } from './test/recordingContext'
import { scoreboardRow } from './test/scoreboardRow'
import { ReplayPage } from './ReplayPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    getRouteApi: () => ({
      useParams: () => ({ titleSlug: 'halo_infinite', playerSlug: 'jgtm', matchId: 'match-1' }),
      useSearch: () => ({}),
    }),
    useRouter: () => ({ history: { length: 2, back: vi.fn() }, navigate: vi.fn() }),
    Link: ({ children, ...rest }: { children?: ReactNode; 'aria-label'?: string }) => (
      <a href="#fiche" aria-label={rest['aria-label']}>
        {children}
      </a>
    ),
  }
})

const API = '/api/v1/players/jgtm/matches/match-1/replay'
const T = REPLAY_TEXT.fr

function smallDoc() {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return loadGoFixture(smallest).doc
}

beforeEach(() => {
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
  act(() => useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_generate_replays: false }) }))
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  act(() => useAppShellStore.setState({ capabilities: null }))
})

/** Le joueur de la page est le premier du roster : sa ligne est la ligne suivie. */
async function mountReady() {
  const doc = smallDoc()
  const me = (doc.roster ?? [])[0]
  const context = {
    header: { map_ui: 'Detachment', mode_ui: 'Assassin en équipe', playlist_label: 'Partie rapide', start_time_label: '10 nov. 2026, 21:40' },
    scoreboard: [scoreboardRow(me.xuid, 'jgtm', null, { is_me: true })],
    highlight_events: [],
    media: null,
    source_unavailable: false,
  }
  server.use(
    http.get(API, () => HttpResponse.json(doc)),
    http.get(`${API}/context`, () => HttpResponse.json(context)),
  )
  const view = renderWithProviders(<ReplayPage />)
  await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
  return view
}

function goTo(frame: ReplayFrame) {
  fireEvent.click(
    within(screen.getByRole('group', { name: T.frames.groupLabel })).getByRole('button', { name: T.frames.frameTip(frame) }),
  )
}

function columnOf(container: HTMLElement) {
  return container.querySelector<HTMLElement>('[data-replay-column="standard"]')
}

describe('ReplayPage : la colonne du Standard', () => {
  it('la grille est la carte puis une colonne de la largeur des cotes', async () => {
    const { container } = await mountReady()
    const grid = container.querySelector<HTMLElement>('[data-replay-grid="standard"]')
    expect(grid).not.toBeNull()
    expect(grid?.style.getPropertyValue('--replay-colonne-standard')).toBe(`${COTES_COLONNE_STANDARD.largeurPx}px`)
    // La variable de la classe est celle que la grille pose : la colonne a bien la largeur de la cote.
    expect(grid?.className).toContain('var(--replay-colonne-standard)')
    expect(grid?.className).not.toContain('30rem')
    const [carte, aside] = Array.from(grid?.children ?? [])
    expect(carte.querySelector('canvas')).not.toBeNull()
    expect(aside.tagName).toBe('ASIDE')
    expect(aside.contains(columnOf(container))).toBe(true)
  })

  it('la colonne porte la marge, l’écart et un filet neutre à gauche', async () => {
    const { container } = await mountReady()
    const colonne = columnOf(container)
    expect(colonne?.style.padding).toBe(`${COTES_COLONNE_STANDARD.margeVerticalePx}px ${COTES_COLONNE_STANDARD.margeLateralePx}px`)
    expect(colonne?.style.gap).toBe(`${COTES_COLONNE_STANDARD.ecartBlocsPx}px`)
    expect(colonne?.className).toContain('xl:border-l')
    expect(colonne?.className).toContain('xl:border-border')
  })

  it('de haut en bas : les équipes, puis le fil', async () => {
    const { container } = await mountReady()
    const [equipes, fil] = Array.from(columnOf(container)?.children ?? [])
    // L'enveloppe des équipes (`contents` : sa boîte est celle de son contenu) porte le bloc des équipes.
    expect(equipes).toHaveAttribute('data-testid', 'replay-above-slot')
    expect(equipes.className).toContain('contents')
    expect(equipes.firstElementChild).toHaveAttribute('data-replay-roster-column')
    expect(fil.contains(screen.getByTestId('replay-feed-slot'))).toBe(true)
    // Plus aucune colonne de fiches en plafond de 30 rem dans le Standard.
    expect(columnOf(container)?.querySelector('[data-replay-player-card]')).not.toBeNull()
    expect(container.querySelector('[data-replay-body="standard"] .xl\\:max-h-\\[30rem\\]')).toBeNull()
  })

  it('le fil garde au moins cinq lignes : sa cellule a un plancher de cinq lignes du fil, titre compris', async () => {
    const { container } = await mountReady()
    const cellule = screen.getByTestId('replay-feed-slot').parentElement as HTMLElement
    expect(parseFloat(cellule.style.minHeight)).toBe(FIL_STANDARD_HAUTEUR_MIN_PX)
    expect(FIL_STANDARD_HAUTEUR_MIN_PX).toBeGreaterThanOrEqual(COTES_COLONNE_STANDARD.filLignesMin * COTES_LIGNE_FIL.hauteurPx)
    // Les équipes sont la part qui cède : rétrécissables, jamais un bloc qui garde sa taille.
    expect(columnOf(container)?.querySelector('[data-replay-roster-column]')?.className).not.toContain('shrink-0')
  })

  it('le joueur suivi est déplié dans la colonne, et lui seul', async () => {
    const { container } = await mountReady()
    expect(screen.getAllByText(T.roster.followed)).toHaveLength(1)
    expect(columnOf(container)?.querySelectorAll('[data-replay-roster-detail]')).toHaveLength(1)
  })

  it('un clic sur une ligne remplace TOUTE la colonne par la fiche : équipes et fil restent montés, masqués ; « Équipes » la referme', async () => {
    const { container } = await mountReady()
    const ligne = columnOf(container)?.querySelector<HTMLElement>('[data-replay-player-card]') as HTMLElement
    expect(ligne.getAttribute('aria-pressed')).toBe('false')
    const feedSlot = screen.getByTestId('replay-feed-slot')
    const equipes = screen.getByTestId('replay-above-slot')
    const roster = container.querySelector('[data-replay-roster-column]')
    expect(equipes.hasAttribute('aria-hidden')).toBe(false)
    fireEvent.click(ligne)
    expect(ligne.getAttribute('aria-pressed')).toBe('true')
    // Les équipes ne restent plus au-dessus : masquées, comme le fil, mais toujours montées.
    expect(equipes.getAttribute('aria-hidden')).toBe('true')
    expect(equipes.className).toContain('invisible')
    expect(feedSlot.getAttribute('aria-hidden')).toBe('true')
    // La fiche est dans la colonne, et c'est elle qui la remplit : retour « Équipes » (et non « Retour au fil »).
    const fiche = columnOf(container)?.querySelector<HTMLElement>('[data-replay-fiche-column]') as HTMLElement
    expect(fiche).not.toBeNull()
    expect(fiche.parentElement?.className).toContain('absolute inset-0')
    expect(feedSlot.parentElement?.contains(fiche)).toBe(true)
    expect(columnOf(container)?.className).toContain('relative')
    expect(screen.queryByRole('button', { name: T.playerPanel.back })).toBeNull()
    const retour = within(fiche).getByRole('button', { name: T.playerPanel.backToTeams })
    fireEvent.click(retour)
    expect(container.querySelector('[data-replay-fiche-column]')).toBeNull()
    expect(screen.getByTestId('replay-feed-slot')).toBe(feedSlot)
    expect(screen.getByTestId('replay-above-slot')).toBe(equipes)
    expect(container.querySelector('[data-replay-roster-column]')).toBe(roster)
    expect(feedSlot.hasAttribute('aria-hidden')).toBe(false)
    expect(equipes.hasAttribute('aria-hidden')).toBe(false)
    expect(ligne.getAttribute('aria-pressed')).toBe('false')
  })

  it('Échap rend la colonne, une couche à la fois, sans quitter le cadre', async () => {
    const { container } = await mountReady()
    fireEvent.click(columnOf(container)?.querySelector<HTMLElement>('[data-replay-player-card]') as HTMLElement)
    expect(screen.getByRole('button', { name: T.playerPanel.backToTeams })).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('button', { name: T.playerPanel.backToTeams })).toBeNull()
    // La fiche fermée, la colonne est celle du Standard : le cadre n'a pas été quitté par la même frappe.
    expect(columnOf(container)).not.toBeNull()
  })

  it('« Fil complet » rend la colonne, et le fil retrouve la même cellule', async () => {
    const { container } = await mountReady()
    const feedSlot = screen.getByTestId('replay-feed-slot')
    fireEvent.click(columnOf(container)?.querySelector<HTMLElement>('[data-replay-player-card]') as HTMLElement)
    fireEvent.click(screen.getByRole('button', { name: T.playerPanel.fullFeed }))
    expect(container.querySelector('[data-replay-fiche-column]')).toBeNull()
    expect(screen.getByTestId('replay-feed-slot')).toBe(feedSlot)
  })

  it('la toile ne se remonte pas à l’ouverture de la fiche', async () => {
    const { container } = await mountReady()
    const canvas = container.querySelector('canvas')
    fireEvent.click(columnOf(container)?.querySelector<HTMLElement>('[data-replay-player-card]') as HTMLElement)
    expect(container.querySelector('canvas')).toBe(canvas)
    expect(container.querySelectorAll('canvas')).toHaveLength(1)
  })
})

describe('ReplayPage : Cinéma et Plein cadre gardent leur colonne', () => {
  it('colonne souple de 30 rem, sans la largeur de colonne du Standard ; sans rangée posée, les équipes y sont en lignes', async () => {
    const { container } = await mountReady()
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      goTo(frame)
      const grid = container.querySelector<HTMLElement>(`[data-replay-grid="${frame}"]`)
      expect(grid?.className).toContain('minmax(0,30rem)')
      expect(grid?.style.getPropertyValue('--replay-colonne-standard')).toBe('')
      expect(columnOf(container)).toBeNull()
      expect(container.querySelector('[data-replay-roster-column]')).not.toBeNull()
      expect(container.querySelector('aside .xl\\:max-h-\\[30rem\\]')).not.toBeNull()
    }
  })

  it('Cinéma : la fiche s’ouvre en fenêtre, la colonne garde ses équipes et son fil, sans sélecteur ni pied', async () => {
    const { container } = await mountReady()
    goTo('cinema')
    const feed = screen.getByTestId('replay-feed-slot')
    const roster = container.querySelector('[data-replay-roster-column]')
    fireEvent.click(container.querySelector<HTMLElement>('[data-replay-roster-column] [data-replay-player-card]') as HTMLElement)
    expect(screen.getByRole('button', { name: T.playerPanel.close })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: T.playerPanel.back })).toBeNull()
    expect(container.querySelector('[data-replay-fiche-column]')).toBeNull()
    expect(container.querySelector('[data-replay-player-picker]')).toBeNull()
    expect(container.querySelector('[data-replay-fiche-extract]')).toBeNull()
    expect(screen.queryByTestId('replay-above-slot')).toBeNull()
    expect(screen.getByTestId('replay-feed-slot')).toBe(feed)
    expect(feed).not.toHaveAttribute('aria-hidden')
    expect(container.querySelector('[data-replay-roster-column]')).toBe(roster)
  })

  it('la cellule du fil survit au passage du Standard au Cinéma et retour', async () => {
    await mountReady()
    const feed = screen.getByTestId('replay-feed-slot')
    for (const frame of ['cinema', 'standard', 'plein-cadre', 'standard'] as const) {
      goTo(frame)
      expect(screen.getByTestId('replay-feed-slot')).toBe(feed)
    }
  })
})
