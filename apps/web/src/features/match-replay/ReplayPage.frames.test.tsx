/**
 * Tests — LES TROIS CADRES de la page du rejeu (Standard, Cinéma, Plein cadre) autour d'UNE toile.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et le vrai document de rejeu : seuls le
 * routeur et le serveur de réseau sont simulés. Ce que ces tests prouvent :
 *  - basculer de cadre ne remonte pas la toile : c'est le MÊME nœud DOM avant et après chaque bascule ;
 *  - chaque cadre rend son en-tête, sans sélecteur de cadre : les icônes « Cadre du lecteur » sont à droite de
 *    la barre de lecture, dans les trois cadres, et reflètent le cadre courant ;
 *  - l'en-tête du Standard a trois cellules (retour et titre, score, commandes) dans une page bornée et centrée ;
 *    au Cinéma et au Plein cadre il y est aussi (équipes dans la colonne), sauf si l'en-tête est trop étroit ;
 *  - les icônes de cadre envoient les gestes du modèle, et sont inertes pendant un export, sauf la sortie du Plein cadre ;
 *  - les états de chargement, d'erreur et d'absence ont leur place dans le Standard (sans toile, donc sans barre
 *    de lecture : aucune icône de cadre n'y mène ailleurs), avec le bouton « Générer le rejeu » ; l'avis
 *    « instant non calé » se rend dans la scène des trois cadres.
 */
import { QueryClientProvider } from '@tanstack/react-query'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse, delay } from 'msw'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { testCapabilities } from '@/test/capabilities'
import { createTestQueryClient, renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { exportFormatOf, exportLayoutFor } from './export/exportFormats'
import { requestExportLayout } from './export/exportLayoutStore'
import { REPLAY_TEXT } from './i18n/i18n'
import { COTES_BANDEAU_SCORE } from './model/cotesBandeauScore'
import { COTES_ENTETE_SCORE } from './model/cotesEnteteScore'
import { BOITE_PAGE_STANDARD_MAX_PX, COTES_PAGE_STANDARD } from './model/cotesStandard'
import type { ReplayFrame } from './model/replayFrame'
import {
  goToFrame,
  MATCH_CONTEXT,
  mountReplayPage,
  REPLAY_API,
  serveDoc,
  smallDoc,
  stubReplayPage,
  unstubReplayPage,
} from './test/replayPageHarness'
import { goBack, ROUTE } from './test/replayPageRouter'
import { scoreboardRow } from './test/scoreboardRow'
import { ReplayPage } from './ReplayPage'

vi.mock('@tanstack/react-router', async (importOriginal) =>
  (await import('./test/replayPageRouter')).routerMock(await importOriginal<typeof import('@tanstack/react-router')>()),
)

const T = REPLAY_TEXT.fr
const FRAMES = T.frames
const FRAME_ORDER: readonly ReplayFrame[] = ['standard', 'cinema', 'plein-cadre']

beforeEach(() => {
  stubReplayPage()
})

afterEach(() => {
  act(() => requestExportLayout(null))
  unstubReplayPage()
})

/** Les icônes de cadre de la barre de lecture. */
function frameSwitch() {
  return within(screen.getByRole('group', { name: FRAMES.groupLabel }))
}

function frameIcon(frame: ReplayFrame) {
  return frameSwitch().getByRole('button', { name: FRAMES.frameTip(frame) })
}

function currentFrame(container: HTMLElement) {
  return container.querySelector('[data-replay-frame]')?.getAttribute('data-replay-frame')
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
  const context = { ...MATCH_CONTEXT, scoreboard: [scoreboardRow(me.xuid, 'jgtm', null, { is_me: true })] }
  return { doc: scored, context }
}

describe('ReplayPage : la toile n’est jamais remontée par une bascule de cadre', () => {
  it('Standard, Cinéma, Plein cadre, retour : le même nœud de toile, le même nombre de toiles', async () => {
    const { container } = await mountReplayPage()
    const canvas = container.querySelector('canvas')
    const canvasCount = container.querySelectorAll('canvas').length
    expect(currentFrame(container)).toBe('standard')

    const parcours: ReplayFrame[] = ['cinema', 'plein-cadre', 'standard', 'plein-cadre', 'cinema', 'standard']
    for (const frame of parcours) {
      goToFrame(frame)
      expect(currentFrame(container)).toBe(frame)
      expect(container.querySelector('canvas')).toBe(canvas)
      expect(container.querySelectorAll('canvas').length).toBe(canvasCount)
      expect(canvas?.isConnected).toBe(true)
    }
  })

  it('la lecture continue d’un cadre à l’autre : la barre d’une ligne est gardée entre Cinéma et Plein cadre, le zoom flottant est propre au Standard', async () => {
    await mountReplayPage()
    fireEvent.click(screen.getByRole('button', { name: T.play }))
    expect(screen.getByRole('button', { name: T.pause })).toBeInTheDocument()
    // Standard : la barre complète et sa croix de zoom flottante.
    expect(screen.getByRole('group', { name: T.zoomGroup })).toBeInTheDocument()
    goToFrame('cinema')
    // Cinéma : la barre d'une ligne, le zoom dans la barre ; la lecture n'a pas été interrompue.
    const pause = screen.getByRole('button', { name: T.pause })
    expect(screen.getAllByRole('button', { name: T.zoomIn })).toHaveLength(1)
    expect(document.querySelector('[data-replay-transport="compact"]')?.contains(pause)).toBe(true)
    goToFrame('plein-cadre')
    expect(screen.getByRole('button', { name: T.pause })).toBe(pause)
    goToFrame('standard')
    expect(screen.getByRole('button', { name: T.pause })).toBeInTheDocument()
    expect(document.querySelector('[data-replay-transport="compact"]')).toBeNull()
    expect(screen.getByRole('group', { name: T.zoomGroup })).toBeInTheDocument()
  })

  it('la colonne du fil garde aussi sa cellule d’un cadre à l’autre', async () => {
    const { container } = await mountReplayPage()
    const feed = screen.getByTestId('replay-feed-slot')
    for (const frame of FRAME_ORDER) {
      goToFrame(frame)
      expect(screen.getByTestId('replay-feed-slot')).toBe(feed)
    }
    expect(container.contains(feed)).toBe(true)
  })
})

describe('ReplayPage : l’en-tête de chaque cadre', () => {
  it('Standard : retour, titre du match et lien « Fiche », sans « Quitter » ni titre de page', async () => {
    await mountReplayPage()
    expect(screen.getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: new RegExp(T.title) })).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
    expect(screen.queryByRole('button', { name: new RegExp(FRAMES.exit) })).not.toBeInTheDocument()
  })

  it('Cinéma : retour, titre du match et lien « Fiche » ; plus le titre de page', async () => {
    await mountReplayPage()
    goToFrame('cinema')
    expect(screen.getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
    expect(screen.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
    expect(screen.getByText(/Partie rapide/)).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: new RegExp(T.title) })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: new RegExp(FRAMES.exit) })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: FRAMES.back }))
    expect(goBack).toHaveBeenCalledTimes(1)
  })

  it('Plein cadre : « Quitter » (Échap) et le titre, sans retour ni lien vers la fiche', async () => {
    await mountReplayPage()
    goToFrame('plein-cadre')
    expect(screen.getByRole('button', { name: new RegExp(FRAMES.exit) })).toHaveTextContent(FRAMES.escapeKey)
    expect(screen.queryByRole('button', { name: FRAMES.back })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: T.back })).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
  })
})

describe('ReplayPage : les icônes « Cadre du lecteur » de la barre de lecture', () => {
  it('reflète le cadre courant et envoie les bons gestes', async () => {
    const { container } = await mountReplayPage()
    const pressed = () =>
      FRAME_ORDER.filter(
        (f) => frameIcon(f).getAttribute('aria-pressed') === 'true',
      )
    expect(pressed()).toEqual(['standard'])
    goToFrame('cinema')
    expect(pressed()).toEqual(['cinema'])
    goToFrame('plein-cadre')
    expect(pressed()).toEqual(['plein-cadre'])
    // Plein cadre ouvert depuis le Cinéma : « Standard » est deux gestes, et mène bien au Standard.
    goToFrame('standard')
    expect(pressed()).toEqual(['standard'])
    expect(currentFrame(container)).toBe('standard')
  })

  it('« Quitter » rend le cadre d’où l’on vient', async () => {
    const { container } = await mountReplayPage()
    goToFrame('cinema')
    goToFrame('plein-cadre')
    fireEvent.click(screen.getByRole('button', { name: new RegExp(FRAMES.exit) }))
    expect(currentFrame(container)).toBe('cinema')
    goToFrame('standard')
    goToFrame('plein-cadre')
    fireEvent.click(screen.getByRole('button', { name: new RegExp(FRAMES.exit) }))
    expect(currentFrame(container)).toBe('standard')
  })

  it('les icônes sont dans la barre de lecture, jamais dans un en-tête, dans les trois cadres', async () => {
    const { container } = await mountReplayPage()
    for (const frame of FRAME_ORDER) {
      goToFrame(frame)
      const groupe = screen.getByRole('group', { name: FRAMES.groupLabel })
      expect(screen.getAllByRole('group', { name: FRAMES.groupLabel })).toHaveLength(1)
      expect(container.querySelector('[data-replay-header]')?.contains(groupe) ?? false).toBe(false)
      // La rangée de la barre porte à la fois « Recommencer » (transport) et le groupe, à droite.
      const rangee = screen.getByRole('button', { name: /^Recommencer/ }).parentElement?.parentElement
      expect(rangee?.contains(groupe)).toBe(true)
    }
  })

  it('Plein cadre : l’icône courante sort, même pendant un export ; les deux autres restent inertes', async () => {
    const { container } = await mountReplayPage()
    goToFrame('cinema')
    goToFrame('plein-cadre')
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    expect(frameIcon('standard')).toBeDisabled()
    expect(frameIcon('cinema')).toBeDisabled()
    expect(frameIcon('plein-cadre')).toBeEnabled()
    fireEvent.click(frameIcon('cinema'))
    expect(currentFrame(container)).toBe('plein-cadre')
    fireEvent.click(frameIcon('plein-cadre'))
    expect(currentFrame(container)).toBe('cinema')
  })

  it('les trois icônes sont inertes pendant un export, et se rouvrent après', async () => {
    const { container } = await mountReplayPage()
    goToFrame('cinema')
    act(() => requestExportLayout(exportLayoutFor(exportFormatOf('720p'))))
    for (const frame of FRAME_ORDER) {
      const button = frameIcon(frame)
      expect(button).toBeDisabled()
      fireEvent.click(button)
    }
    expect(currentFrame(container)).toBe('cinema')
    act(() => requestExportLayout(null))
    goToFrame('standard')
    expect(currentFrame(container)).toBe('standard')
  })
})

describe('ReplayPage : la pleine largeur de la coquille suit le Cinéma', () => {
  const NEXT_API = '/api/v1/players/jgtm/matches/match-2/replay'
  const pleineLargeur = () => useAppShellStore.getState().fullWidthPage

  afterEach(() => act(() => useAppShellStore.setState({ fullWidthPage: false })))

  it('posée au Cinéma seulement : retirée au Standard et au Plein cadre, jamais posée avant', async () => {
    await mountReplayPage()
    expect(pleineLargeur()).toBe(false)
    goToFrame('cinema')
    expect(pleineLargeur()).toBe(true)
    goToFrame('plein-cadre')
    expect(pleineLargeur()).toBe(false)
    goToFrame('cinema')
    expect(pleineLargeur()).toBe(true)
    goToFrame('standard')
    expect(pleineLargeur()).toBe(false)
  })

  it('retirée au démontage de la page', async () => {
    const { unmount } = await mountReplayPage()
    goToFrame('cinema')
    expect(pleineLargeur()).toBe(true)
    unmount()
    expect(pleineLargeur()).toBe(false)
  })

  it('retirée au changement de match (retour au Standard)', async () => {
    serveDoc(smallDoc())
    server.use(
      http.get(`${REPLAY_API}/context`, () => HttpResponse.json(MATCH_CONTEXT)),
      http.get(NEXT_API, () => HttpResponse.json(smallDoc())),
      http.get(`${NEXT_API}/context`, () => HttpResponse.json(MATCH_CONTEXT)),
    )
    const client = createTestQueryClient()
    const tree = () => (
      <QueryClientProvider client={client}>
        <ReplayPage />
      </QueryClientProvider>
    )
    const view = render(tree())
    await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
    goToFrame('cinema')
    expect(pleineLargeur()).toBe(true)
    ROUTE.matchId = 'match-2'
    view.rerender(tree())
    expect(currentFrame(view.container)).toBe('standard')
    expect(pleineLargeur()).toBe(false)
  })
})

describe('ReplayPage : un cadre sans toile est toujours le Standard', () => {
  const NEXT_API = '/api/v1/players/jgtm/matches/match-2/replay'

  /** La page, montée sous un client de requêtes STABLE : changer de match ne doit pas, par ailleurs, vider le cache. */
  function mountStable() {
    const client = createTestQueryClient()
    // Un élément NEUF à chaque rendu : le même élément ne referait pas rendre la page.
    const tree = () => (
      <QueryClientProvider client={client}>
        <ReplayPage />
      </QueryClientProvider>
    )
    const view = render(tree())
    return { view, switchMatch: (id: string) => { ROUTE.matchId = id; view.rerender(tree()) } }
  }

  it('changement de match en Cinéma : la page revient au Standard, d’abord sans rejeu, puis avec le nouveau', async () => {
    serveDoc(smallDoc())
    server.use(
      http.get(`${REPLAY_API}/context`, () => HttpResponse.json(MATCH_CONTEXT)),
      http.get(NEXT_API, () => HttpResponse.json(smallDoc())),
      http.get(`${NEXT_API}/context`, () => HttpResponse.json(MATCH_CONTEXT)),
    )
    const { view, switchMatch } = mountStable()
    await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
    goToFrame('cinema')
    expect(currentFrame(view.container)).toBe('cinema')

    switchMatch('match-2')
    expect(currentFrame(view.container)).toBe('standard')
    expect(view.container.querySelector('[data-replay-header="standard"]')).not.toBeNull()
    await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
    expect(currentFrame(view.container)).toBe('standard')
    expect(frameIcon('standard').getAttribute('aria-pressed')).toBe('true')
  })

  it('rejeu absent après un Cinéma : l’en-tête et le corps du Standard, avec la phrase d’état vide', async () => {
    serveDoc(smallDoc())
    server.use(
      http.get(`${REPLAY_API}/context`, () => HttpResponse.json(MATCH_CONTEXT)),
      http.get(NEXT_API, () => HttpResponse.json({ code: 'not_found', message: 'aucun rejeu', retryable: false }, { status: 404 })),
      http.get(`${NEXT_API}/generation`, () => HttpResponse.json({ state: 'not_generatable' })),
    )
    const { view, switchMatch } = mountStable()
    await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
    goToFrame('plein-cadre')
    expect(currentFrame(view.container)).toBe('plein-cadre')

    switchMatch('match-2')
    expect(await screen.findByText(T.empty)).toBeInTheDocument()
    expect(currentFrame(view.container)).toBe('standard')
    expect(view.container.querySelector('[data-replay-header="standard"]')).not.toBeNull()
    expect(screen.queryByRole('button', { name: new RegExp(FRAMES.exit) })).not.toBeInTheDocument()
    expect(view.container.querySelector('canvas')).toBeNull()
  })
})

describe('ReplayPage : les états hors rejeu dans le Standard', () => {
  const frame: ReplayFrame = 'standard'

  it('chargement : la phrase d’attente, sous l’en-tête du cadre', async () => {
    server.use(http.get(REPLAY_API, async () => delay('infinite')))
    const { container } = renderWithProviders(<ReplayPage />)
    expect(currentFrame(container)).toBe(frame)
    expect(await screen.findByText(T.loading)).toBeInTheDocument()
    expect(container.querySelector('canvas')).toBeNull()
  })

  it('erreur : l’avis d’erreur et son bouton « réessayer »', async () => {
    server.use(http.get(REPLAY_API, () => HttpResponse.json({ code: 'boom', message: 'x', retryable: true }, { status: 500 })))
    const { container } = renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(T.loadErrorTitle)).toBeInTheDocument()
    expect(currentFrame(container)).toBe(frame)
    expect(screen.getByText(T.loadErrorTitle)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: T.loadErrorRetry })).toBeInTheDocument()
  })

  it('absence de rejeu : la phrase d’état vide et le bouton « Générer le rejeu »', async () => {
    act(() => useAppShellStore.setState({ capabilities: testCapabilities({ can_generate_replays: true }) }))
    server.use(
      http.get(REPLAY_API, () => HttpResponse.json({ code: 'not_found', message: 'aucun rejeu', retryable: false }, { status: 404 })),
      http.get(`${REPLAY_API}/generation`, () => HttpResponse.json({ state: 'generatable' })),
    )
    const { container } = renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(T.empty)).toBeInTheDocument()
    expect(currentFrame(container)).toBe(frame)
    expect(screen.getByText(T.empty)).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Générer le rejeu' })).toBeInTheDocument()
  })

})

describe.each(FRAME_ORDER)('ReplayPage : l’avis « instant non calé » dans le cadre %s', (frame) => {
  it('l’avis se rend dans la scène, avec la toile', async () => {
    ROUTE.search = { t: '5000', clock: 'match' }
    serveDoc({ ...smallDoc(), originMs: null })
    const { container } = renderWithProviders(<ReplayPage />)
    await waitFor(() => expect(container.querySelector('canvas')).not.toBeNull())
    goToFrame(frame)
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
  it('à gauche le retour et le titre du match, au centre le score, à droite la fiche : aucun sélecteur de cadre', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReplayPage(doc, context)
    const gauche = within(headerCell(container, 'gauche'))
    expect(gauche.getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(gauche.getByRole('heading', { level: 1, name: /Detachment/ })).toBeInTheDocument()
    expect(gauche.getByText(/Partie rapide/)).toBeInTheDocument()
    const centre = within(headerCell(container, 'centre'))
    expect(centre.getByRole('group', { name: T.scoreBannerLabel })).toBeInTheDocument()
    const droite = within(headerCell(container, 'droite'))
    expect(droite.queryByRole('group', { name: FRAMES.groupLabel })).not.toBeInTheDocument()
    expect(droite.queryByRole('button', { name: /^(Standard|Cinéma|Plein cadre)/ })).not.toBeInTheDocument()
    expect(droite.getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
  })

  it('le score est dans l’en-tête et ne coiffe plus le terrain : il n’est pas dans la scène', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReplayPage(doc, context)
    const banner = scoreBanner(container)
    expect(banner).not.toBeNull()
    expect(banner).toHaveAttribute('data-replay-score-banner', 'entete')
    expect(headerCell(container, 'centre').contains(banner)).toBe(true)
    expect(container.querySelectorAll('[data-replay-score-banner]')).toHaveLength(1)
    expect(container.querySelector('section')?.contains(banner)).toBe(false)
  })

  it('Cinéma et Plein cadre, équipes dans la colonne : le score est dans la cellule centrale de l’en-tête, pas sur la toile', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReplayPage(doc, context)
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      goToFrame(frame)
      const banner = scoreBanner(container)
      expect(banner).toHaveAttribute('data-replay-score-banner', 'entete')
      expect(container.querySelector('section')?.contains(banner)).toBe(false)
      const entete = container.querySelector<HTMLElement>(`[data-replay-header="${frame}"]`)
      expect(entete?.querySelector('[data-replay-header-cell="centre"]')?.contains(banner)).toBe(true)
      expect(entete?.style.gridTemplateColumns).toBe(`minmax(0,1fr) ${COTES_BANDEAU_SCORE.entete.largeurPx}px minmax(0,1fr)`)
      expect(entete?.style.height).toBe(`${COTES_ENTETE_SCORE.hauteurPx}px`)
      expect(container.querySelector('[data-replay-header="standard"]')).toBeNull()
    }
    goToFrame('standard')
    expect(scoreBanner(container)).toHaveAttribute('data-replay-score-banner', 'entete')
  })

  it('score absent (chacun pour soi, document sans calque de score) : la cellule centrale ne rend rien, les deux autres gardent leur place', async () => {
    const { container } = await mountReplayPage()
    expect(scoreBanner(container)).toBeNull()
    expect(headerCell(container, 'centre')).toBeEmptyDOMElement()
    expect(within(headerCell(container, 'gauche')).getByRole('button', { name: FRAMES.back })).toBeInTheDocument()
    expect(within(headerCell(container, 'droite')).getByRole('link', { name: T.back })).toHaveTextContent(FRAMES.sheet)
  })

  it('la grille de l’en-tête réserve la cellule du score : 440 px au centre, deux cellules souples autour', async () => {
    const { container } = await mountReplayPage()
    const grid = headerCell(container, 'centre').parentElement
    expect(grid?.style.gridTemplateColumns).toBe(`minmax(0,1fr) ${COTES_BANDEAU_SCORE.entete.largeurPx}px minmax(0,1fr)`)
    expect(grid?.style.height).toBe(`${COTES_PAGE_STANDARD.enteteHauteurPx}px`)
  })
})

describe('ReplayPage : la page bornée et centrée du Standard', () => {
  it('l’en-tête et le corps du Standard portent la même largeur maximale, marges comprises', async () => {
    const { container } = await mountReplayPage()
    const box = `${BOITE_PAGE_STANDARD_MAX_PX}px`
    expect(container.querySelector<HTMLElement>('[data-replay-body="standard"]')?.style.maxWidth).toBe(box)
    expect(container.querySelector<HTMLElement>('[data-replay-header="standard"]')?.style.maxWidth).toBe(box)
    expect(BOITE_PAGE_STANDARD_MAX_PX).toBe(COTES_PAGE_STANDARD.contenuMaxPx + 2 * COTES_PAGE_STANDARD.margeLateralePx)
  })

  it('Cinéma et Plein cadre prennent la largeur du corps : aucune largeur maximale posée', async () => {
    const { container } = await mountReplayPage()
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      goToFrame(frame)
      expect(container.querySelector<HTMLElement>(`[data-replay-body="${frame}"]`)?.style.maxWidth).toBe('')
    }
  })
})

describe('ReplayPage : le score change d’en-tête sans remonter la toile', () => {
  it('Standard, Cinéma, Plein cadre, retour : le score reste dans un en-tête et la toile est la même', async () => {
    const { doc, context } = scoredMatch()
    const { container } = await mountReplayPage(doc, context)
    const canvas = container.querySelector('canvas')
    for (const frame of ['cinema', 'plein-cadre', 'standard', 'cinema', 'standard'] as const) {
      goToFrame(frame)
      expect(scoreBanner(container)).toHaveAttribute('data-replay-score-banner', 'entete')
      expect(container.querySelector('canvas')).toBe(canvas)
    }
  })
})
