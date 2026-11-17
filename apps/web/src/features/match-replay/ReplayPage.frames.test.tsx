/**
 * Tests — LES TROIS CADRES de la page du rejeu (Standard, Cinéma, Plein cadre) autour d'UNE toile.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et le vrai document de rejeu : seuls le
 * routeur et le serveur de réseau sont simulés. Ce que ces tests prouvent :
 *  - basculer de cadre ne remonte pas la toile : c'est le MÊME nœud DOM avant et après chaque bascule ;
 *  - chaque cadre rend son en-tête, et le groupe « Cadre du lecteur » reflète le cadre courant ;
 *  - les boutons de cadre envoient les gestes du modèle, et sont inertes pendant un export ;
 *  - les états de chargement, d'erreur et d'absence ont leur place dans les trois cadres, avec le bouton
 *    « Générer le rejeu » ; l'avis « instant non calé » se rend dans la scène.
 */
import { act, cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse, delay } from 'msw'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { testCapabilities } from '@/test/capabilities'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { exportFormatOf, exportLayoutFor } from './export/exportFormats'
import { requestExportLayout } from './export/exportLayoutStore'
import { REPLAY_TEXT } from './i18n/i18n'
import type { ReplayFrame } from './model/replayFrame'
import { goFixtureEntries, loadGoFixture } from './test/goFixtures'
import { recordingContext } from './test/recordingContext'
import { ReplayPage } from './ReplayPage'

const hoisted = vi.hoisted(() => ({ search: {} as Record<string, string> }))
const back = vi.hoisted(() => vi.fn())

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    getRouteApi: () => ({
      useParams: () => ({ titleSlug: 'halo_infinite', playerSlug: 'jgtm', matchId: 'match-1' }),
      useSearch: () => hoisted.search,
    }),
    useRouter: () => ({ history: { length: 2, back }, navigate: vi.fn() }),
    Link: ({ children, ...rest }: { children?: ReactNode; 'aria-label'?: string }) => (
      <a href="#fiche" aria-label={rest['aria-label']}>
        {children}
      </a>
    ),
  }
})

const API = '/api/v1/players/jgtm/matches/match-1/replay'
const T = REPLAY_TEXT.fr
const FRAMES = T.frames
const FRAME_ORDER: readonly ReplayFrame[] = ['standard', 'cinema', 'plein-cadre']

/** Le plus petit document de rejeu du manifeste de Go : il a des pistes, donc la page est « prête ». */
function smallDoc() {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return loadGoFixture(smallest).doc
}

function serveDoc(doc: object) {
  server.use(http.get(API, () => HttpResponse.json(doc)))
}

beforeEach(() => {
  hoisted.search = {}
  // Aucune boucle d'animation, aucun dessin réel : la toile se monte, c'est tout ce qu'on regarde.
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
  act(() => useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_generate_replays: false }) }))
})

afterEach(() => {
  act(() => requestExportLayout(null))
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  act(() => useAppShellStore.setState({ capabilities: null }))
})

function frameSwitch() {
  return within(screen.getByRole('group', { name: FRAMES.groupLabel }))
}

function goTo(frame: ReplayFrame) {
  fireEvent.click(frameSwitch().getByRole('button', { name: FRAMES.frame[frame] }))
}

function currentFrame(container: HTMLElement) {
  return container.querySelector('[data-replay-frame]')?.getAttribute('data-replay-frame')
}

/** Le contexte du match : de quoi nommer le match dans les en-têtes (mode, carte, playlist, date). */
const CONTEXT = {
  header: {
    map_ui: 'Detachment',
    mode_ui: 'Assassin en équipe',
    playlist_label: 'Partie rapide',
    start_time_label: '10 nov. 2026, 21:40',
  },
  scoreboard: [],
  highlight_events: [],
  media: null,
  source_unavailable: false,
}

async function mountReady() {
  serveDoc(smallDoc())
  server.use(http.get(`${API}/context`, () => HttpResponse.json(CONTEXT)))
  const view = renderWithProviders(<ReplayPage />)
  await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
  return view
}

describe('ReplayPage : la toile n’est jamais remontée par une bascule de cadre', () => {
  it('Standard, Cinéma, Plein cadre, retour : le même nœud de toile, le même nombre de toiles', async () => {
    const { container } = await mountReady()
    const canvas = container.querySelector('canvas')
    const canvasCount = container.querySelectorAll('canvas').length
    expect(currentFrame(container)).toBe('standard')

    const parcours: ReplayFrame[] = ['cinema', 'plein-cadre', 'standard', 'plein-cadre', 'cinema', 'standard']
    for (const frame of parcours) {
      goTo(frame)
      expect(currentFrame(container)).toBe(frame)
      expect(container.querySelector('canvas')).toBe(canvas)
      expect(container.querySelectorAll('canvas').length).toBe(canvasCount)
      expect(canvas?.isConnected).toBe(true)
    }
  })

  it('la colonne du fil garde aussi sa cellule d’un cadre à l’autre', async () => {
    const { container } = await mountReady()
    const feed = screen.getByTestId('replay-feed-slot')
    for (const frame of FRAME_ORDER) {
      goTo(frame)
      expect(screen.getByTestId('replay-feed-slot')).toBe(feed)
    }
    expect(container.contains(feed)).toBe(true)
  })
})

describe('ReplayPage : l’en-tête de chaque cadre', () => {
  it('Standard : le titre de la page et le lien vers la fiche du match, sans « Quitter »', async () => {
    await mountReady()
    expect(screen.getByRole('heading', { name: new RegExp(T.title) })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: T.back })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: new RegExp(FRAMES.exit) })).not.toBeInTheDocument()
  })

  it('Cinéma : retour, titre du match et lien « Fiche » ; plus le titre de page', async () => {
    await mountReady()
    goTo('cinema')
    expect(screen.getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
    expect(screen.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
    expect(screen.getByText(/Partie rapide/)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: new RegExp(T.title) })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: new RegExp(FRAMES.exit) })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: FRAMES.back }))
    expect(back).toHaveBeenCalledTimes(1)
  })

  it('Plein cadre : « Quitter » (Échap) et le titre, sans retour ni lien vers la fiche', async () => {
    await mountReady()
    goTo('plein-cadre')
    expect(screen.getByRole('button', { name: new RegExp(FRAMES.exit) })).toHaveTextContent(FRAMES.escapeKey)
    expect(screen.queryByRole('button', { name: FRAMES.back })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: T.back })).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
  })
})

describe('ReplayPage : le groupe « Cadre du lecteur »', () => {
  it('reflète le cadre courant et envoie les bons gestes', async () => {
    const { container } = await mountReady()
    const pressed = () =>
      FRAME_ORDER.filter(
        (f) => frameSwitch().getByRole('button', { name: FRAMES.frame[f] }).getAttribute('aria-pressed') === 'true',
      )
    expect(pressed()).toEqual(['standard'])
    goTo('cinema')
    expect(pressed()).toEqual(['cinema'])
    goTo('plein-cadre')
    expect(pressed()).toEqual(['plein-cadre'])
    // Plein cadre ouvert depuis le Cinéma : « Standard » est deux gestes, et mène bien au Standard.
    goTo('standard')
    expect(pressed()).toEqual(['standard'])
    expect(currentFrame(container)).toBe('standard')
  })

  it('« Quitter » rend le cadre d’où l’on vient', async () => {
    const { container } = await mountReady()
    goTo('cinema')
    goTo('plein-cadre')
    fireEvent.click(screen.getByRole('button', { name: new RegExp(FRAMES.exit) }))
    expect(currentFrame(container)).toBe('cinema')
    goTo('standard')
    goTo('plein-cadre')
    fireEvent.click(screen.getByRole('button', { name: new RegExp(FRAMES.exit) }))
    expect(currentFrame(container)).toBe('standard')
  })

  it('les trois boutons sont inertes pendant un export, et se rouvrent après', async () => {
    const { container } = await mountReady()
    goTo('cinema')
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    for (const frame of FRAME_ORDER) {
      const button = frameSwitch().getByRole('button', { name: FRAMES.frame[frame] })
      expect(button).toBeDisabled()
      fireEvent.click(button)
    }
    expect(currentFrame(container)).toBe('cinema')
    act(() => requestExportLayout(null))
    goTo('standard')
    expect(currentFrame(container)).toBe('standard')
  })
})

describe.each(FRAME_ORDER)('ReplayPage : les états hors rejeu dans le cadre %s', (frame) => {
  it('chargement : la phrase d’attente, sous l’en-tête du cadre', async () => {
    server.use(http.get(API, async () => delay('infinite')))
    const { container } = renderWithProviders(<ReplayPage />)
    goTo(frame)
    expect(currentFrame(container)).toBe(frame)
    expect(await screen.findByText(T.loading)).toBeInTheDocument()
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('erreur : l’avis d’erreur et son bouton « réessayer »', async () => {
    server.use(http.get(API, () => HttpResponse.json({ code: 'boom', message: 'x', retryable: true }, { status: 500 })))
    const { container } = renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(T.loadErrorTitle)).toBeInTheDocument()
    goTo(frame)
    expect(currentFrame(container)).toBe(frame)
    expect(screen.getByText(T.loadErrorTitle)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: T.loadErrorRetry })).toBeInTheDocument()
  })

  it('absence de rejeu : la phrase d’état vide et le bouton « Générer le rejeu »', async () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities({ can_generate_replays: true }) }))
    server.use(
      http.get(API, () => HttpResponse.json({ code: 'not_found', message: 'aucun rejeu', retryable: false }, { status: 404 })),
      http.get(`${API}/generation`, () => HttpResponse.json({ state: 'generatable' })),
    )
    const { container } = renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(T.empty)).toBeInTheDocument()
    goTo(frame)
    expect(currentFrame(container)).toBe(frame)
    expect(screen.getByText(T.empty)).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Générer le rejeu' })).toBeInTheDocument()
  })

  it('instant non calé : l’avis se rend dans la scène, avec la toile', async () => {
    hoisted.search = { t: '5000', clock: 'match' }
    serveDoc({ ...smallDoc(), originMs: null })
    const { container } = renderWithProviders(<ReplayPage />)
    await waitFor(() => expect(container.querySelector('canvas')).not.toBeNull())
    goTo(frame)
    const notice = screen.getByText(T.openAtNoOriginTitle)
    expect(container.querySelector('section')?.contains(notice)).toBe(true)
    expect(container.querySelector('section')?.contains(container.querySelector('canvas'))).toBe(true)
  })
})
