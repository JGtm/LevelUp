/**
 * Tests — LES TROIS CADRES de la page du rejeu (Standard, Cinéma, Plein cadre) autour d'UNE toile.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et le vrai document de rejeu : seuls le
 * routeur et le serveur de réseau sont simulés. Ce que ces tests prouvent :
 *  - basculer de cadre ne remonte pas la toile : c'est le MÊME nœud DOM avant et après chaque bascule ;
 *  - chaque cadre rend son en-tête, et le groupe « Cadre du lecteur » reflète le cadre courant ;
 *  - l'en-tête du Standard a trois cellules (retour et titre, score, commandes) dans une page bornée et centrée ;
 *    le score n'est dans la scène qu'en Cinéma et en Plein cadre ;
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
import { COTES_BANDEAU_SCORE } from './model/cotesBandeauScore'
import { BOITE_PAGE_STANDARD_MAX_PX, COTES_PAGE_STANDARD } from './model/cotesStandard'
import type { ReplayFrame } from './model/replayFrame'
import { goFixtureEntries, loadGoFixture } from './test/goFixtures'
import { recordingContext } from './test/recordingContext'
import { scoreboardRow } from './test/scoreboardRow'
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

async function mountReady(doc: object = smallDoc(), context: object = CONTEXT) {
  serveDoc(doc)
  server.use(http.get(`${API}/context`, () => HttpResponse.json(context)))
  const view = renderWithProviders(<ReplayPage />)
  await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
  return view
}

/**
 * Un document à deux camps AVEC calque de score, et le contexte qui désigne son premier joueur comme le joueur
 * de la page : le bandeau de score a de quoi se lire (aucun document du parc de fixtures n'en porte un).
 */
function scoredMatch() {
  const doc = smallDoc()
  const roster = doc.roster ?? []
  const me = roster[0]
  const teams = [...new Set(roster.map((r) => r.team))].filter((team): team is number => team !== undefined)
  const serie = (teamId: number, v: number) => {
    const points = [{ t: 0, v: 0 }, { t: 10, v }]
    return { teamId, rounds: [{ round: 0, points }], total: points }
  }
  const scored = { ...doc, scoreTimeline: { teams: [serie(teams[0], 7), serie(teams[1], 3)], players: [] } }
  const context = { ...CONTEXT, scoreboard: [scoreboardRow(me.xuid, 'jgtm', null, { is_me: true })] }
  return { doc: scored, context }
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
  it('Standard : retour, titre du match et lien « Fiche », sans « Quitter » ni titre de page', async () => {
    await mountReady()
    expect(screen.getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: new RegExp(T.title) })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
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

/** Le bandeau de score de la page, où qu'il soit posé (un seul par page, jamais deux). */
function scoreBanner(container: HTMLElement) {
  return container.querySelector<HTMLElement>('[data-replay-score-banner]')
}

function headerCell(container: HTMLElement, cell: 'gauche' | 'centre' | 'droite') {
  const node = container.querySelector<HTMLElement>(`[data-replay-header="standard"] [data-replay-header-cell="${cell}"]`)
  if (!node) throw new Error(`cellule « ${cell} » absente de l'en-tête du Standard`)
  return node
}

describe('ReplayPage : l’en-tête du Standard en trois cellules', () => {
  it('à gauche le retour et le titre du match, au centre le score, à droite les cadres et la fiche', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReady(doc, context)
    const gauche = within(headerCell(container, 'gauche'))
    expect(gauche.getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(gauche.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
    expect(gauche.getByText(/Partie rapide/)).toBeInTheDocument()
    const centre = within(headerCell(container, 'centre'))
    expect(centre.getByRole('group', { name: T.scoreBannerLabel })).toBeInTheDocument()
    const droite = within(headerCell(container, 'droite'))
    expect(droite.getByRole('group', { name: FRAMES.groupLabel })).toBeInTheDocument()
    expect(droite.getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
  })

  it('le score est dans l’en-tête et ne coiffe plus le terrain : il n’est pas dans la scène', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReady(doc, context)
    const banner = scoreBanner(container)
    expect(banner).not.toBeNull()
    expect(banner).toHaveAttribute('data-replay-score-banner', 'entete')
    expect(headerCell(container, 'centre').contains(banner)).toBe(true)
    expect(container.querySelectorAll('[data-replay-score-banner]')).toHaveLength(1)
    expect(container.querySelector('section')?.contains(banner)).toBe(false)
  })

  it('Cinéma et Plein cadre : le score reste dans la scène, sur la toile', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReady(doc, context)
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      goTo(frame)
      const banner = scoreBanner(container)
      expect(banner).toHaveAttribute('data-replay-score-banner', 'scene')
      expect(container.querySelector('section')?.contains(banner)).toBe(true)
      expect(container.querySelector('[data-replay-header="standard"]')).toBeNull()
    }
    goTo('standard')
    expect(scoreBanner(container)).toHaveAttribute('data-replay-score-banner', 'entete')
  })

  it('score absent (chacun pour soi, document sans calque de score) : la cellule centrale ne rend rien, les deux autres gardent leur place', async () => {
    const { container } = await mountReady()
    expect(scoreBanner(container)).toBeNull()
    expect(headerCell(container, 'centre')).toBeEmptyDOMElement()
    expect(within(headerCell(container, 'gauche')).getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(within(headerCell(container, 'droite')).getByRole('group', { name: FRAMES.groupLabel })).toBeInTheDocument()
  })

  it('la grille de l’en-tête réserve la cellule du score : 440 px au centre, deux cellules souples autour', async () => {
    const { container } = await mountReady()
    const grid = headerCell(container, 'centre').parentElement
    expect(grid?.style.gridTemplateColumns).toBe(`minmax(0,1fr) ${COTES_BANDEAU_SCORE.entete.largeurPx}px minmax(0,1fr)`)
    expect(grid?.style.height).toBe(`${COTES_PAGE_STANDARD.enteteHauteurPx}px`)
  })
})

describe('ReplayPage : la page bornée et centrée du Standard', () => {
  it('l’en-tête et le corps du Standard portent la même largeur maximale, marges comprises', async () => {
    const { container } = await mountReady()
    const box = `${BOITE_PAGE_STANDARD_MAX_PX}px`
    expect(container.querySelector<HTMLElement>('[data-replay-body="standard"]')?.style.maxWidth).toBe(box)
    expect(container.querySelector<HTMLElement>('[data-replay-header="standard"]')?.style.maxWidth).toBe(box)
    expect(BOITE_PAGE_STANDARD_MAX_PX).toBe(COTES_PAGE_STANDARD.contenuMaxPx + 2 * COTES_PAGE_STANDARD.margeLateralePx)
  })

  it('Cinéma et Plein cadre prennent la largeur du corps : aucune largeur maximale posée', async () => {
    const { container } = await mountReady()
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      goTo(frame)
      expect(container.querySelector<HTMLElement>(`[data-replay-body="${frame}"]`)?.style.maxWidth).toBe('')
    }
  })
})

describe('ReplayPage : le score change d’emplacement sans remonter la toile', () => {
  it('Standard (en-tête), Cinéma (scène), Plein cadre (scène), retour : la même toile', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReady(doc, context)
    const canvas = container.querySelector('canvas')
    for (const frame of ['cinema', 'plein-cadre', 'standard', 'cinema', 'standard'] as const) {
      goTo(frame)
      expect(scoreBanner(container)).toHaveAttribute('data-replay-score-banner', frame === 'standard' ? 'entete' : 'scene')
      expect(container.querySelector('canvas')).toBe(canvas)
    }
  })
})
