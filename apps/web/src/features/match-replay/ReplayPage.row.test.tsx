/**
 * Tests — LA RANGÉE DES ÉQUIPES du Cinéma et du Plein cadre, dans la vraie page.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et un document à deux camps AVEC calque de
 * score ; la largeur du corps du cadre est posée par le test (jsdom ne met rien en page) et suivie par un
 * observateur de taille simulé. Ce que ces tests prouvent :
 *  - la rangée est posée quand la règle (`dispositionEquipes`) la retient : le score est AU CENTRE de la rangée
 *    (emplacement `rangee`), il n'est plus sur la toile, et la colonne ne garde que le fil ;
 *  - sinon (fenêtre trop étroite, largeur pas mesurée, chacun pour soi, Standard) les équipes sont dans la colonne
 *    et le score reste où il était ;
 *  - passer de la rangée au repli (la fenêtre rétrécit) et basculer de cadre ne remontent NI la toile NI la
 *    cellule du fil.
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
import { largeurRequiseRangee, largeurRequiseRangeeGrandeEquipe } from './model/rowLayout'
import type { ReplayFrame } from './model/replayFrame'
import { recordingContext } from './test/recordingContext'
import { scoreboardRow } from './test/scoreboardRow'
import { testReplayDoc } from './test/testDoc'
import { ReplayPage } from './ReplayPage'

const hoisted = vi.hoisted(() => ({ search: {} as Record<string, string> }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    getRouteApi: () => ({
      useParams: () => ({ titleSlug: 'halo_infinite', playerSlug: 'jgtm', matchId: 'match-1' }),
      useSearch: () => hoisted.search,
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
const FRAMES = T.frames

/** La largeur du corps du cadre que le test pose ; 0 = non mesurée. */
let largeurCorps = 0
/** Les observateurs de taille que la page a posés sur un corps de cadre. */
const observateurs: { cible: Element; rappel: () => void }[] = []

class ObservateurDeTaille {
  private readonly rappel: () => void
  constructor(rappel: () => void) {
    this.rappel = rappel
  }
  observe(cible: Element) {
    observateurs.push({ cible, rappel: this.rappel })
  }
  unobserve() {}
  disconnect() {
    for (let i = observateurs.length - 1; i >= 0; i--) if (observateurs[i].rappel === this.rappel) observateurs.splice(i, 1)
  }
}

/** La fenêtre change de taille : le corps du cadre prend cette largeur et ses observateurs sont avertis. */
function redimensionner(largeur: number) {
  largeurCorps = largeur
  act(() => {
    for (const { cible, rappel } of [...observateurs]) if (cible.hasAttribute('data-replay-body')) rappel()
  })
}

beforeEach(() => {
  hoisted.search = {}
  largeurCorps = 0
  observateurs.length = 0
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.stubGlobal('ResizeObserver', ObservateurDeTaille)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
  const original = HTMLElement.prototype.getBoundingClientRect
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    if (!this.hasAttribute('data-replay-body')) return original.call(this)
    return { width: largeurCorps, height: 0, top: 0, left: 0, right: largeurCorps, bottom: 0, x: 0, y: 0, toJSON: () => ({}) }
  })
  act(() => useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_generate_replays: false }) }))
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  act(() => useAppShellStore.setState({ capabilities: null }))
})

/** Un match à `nbCamps` camps de `parCamp` places (équipe 0, 1, ...), avec calque de score pour les deux premiers. */
function match(parCamp: number, nbCamps = 2) {
  const n = parCamp * nbCamps
  const equipe = (i: number) => (nbCamps === 1 ? -1 : Math.floor(i / parCamp))
  const serie = (teamId: number, v: number) => {
    const points = [{ t: 0, v: 0 }, { t: 10, v }]
    return { teamId, rounds: [{ round: 0, points }], total: points }
  }
  const doc = testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: Array.from({ length: n }, (_, i) => ({
      xuid: `X${i}`, filmIndex: i, seat: i % parCamp, name: `Joueur${i}`, team: equipe(i),
    })),
    tracks: Array.from({ length: n }, (_, i) => ({
      slot: 512 + i, team: equipe(i), xuid: `X${i}`, startFrame: 0, endFrame: 290, points: [{ t: 0, x: i, y: i, sh: 1, hp: 1 }],
    })),
    ...(nbCamps === 2 ? { scoreTimeline: { teams: [serie(0, 7), serie(1, 3)], players: [] } } : {}),
  })
  const scoreboard = Array.from({ length: n }, (_, i) =>
    scoreboardRow(`X${i}`, `Joueur${i}`, equipe(i) === 0 ? 'a' : 'b', { is_me: i === 0 }),
  )
  const context = {
    header: { map_ui: 'Detachment', mode_ui: 'Assassin en équipe', playlist_label: 'Partie rapide', start_time_label: '10 nov. 2026' },
    scoreboard,
    highlight_events: [],
    media: null,
    source_unavailable: false,
  }
  return { doc, context }
}

async function mount({ doc, context }: ReturnType<typeof match>, largeur: number) {
  largeurCorps = largeur
  server.use(http.get(API, () => HttpResponse.json(doc)))
  server.use(http.get(`${API}/context`, () => HttpResponse.json(context)))
  const view = renderWithProviders(<ReplayPage />)
  await waitFor(() => expect(view.container.querySelector('canvas')).not.toBeNull())
  return view
}

function goTo(frame: ReplayFrame) {
  const switcher = within(screen.getByRole('group', { name: FRAMES.groupLabel }))
  fireEvent.click(switcher.getByRole('button', { name: FRAMES.frameTip(frame) }))
}

const rangee = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-teams-row]')
const colonneDesEquipes = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-roster-column]')
const bandeaux = (c: HTMLElement) => [...c.querySelectorAll<HTMLElement>('[data-replay-score-banner]')]
const L4 = largeurRequiseRangee(4)
const LGE = largeurRequiseRangeeGrandeEquipe()

describe('ReplayPage : la rangée des équipes, 4 contre 4', () => {
  it.each(['cinema', 'plein-cadre'] as const)('%s : la rangée est posée, le score est à son centre et plus sur la toile', async (frame) => {
    const { container } = await mount(match(4), L4)
    goTo(frame)
    const ligne = rangee(container)
    expect(ligne).not.toBeNull()
    expect(ligne?.getAttribute('data-replay-teams-row')).toBe('fiches')
    expect(ligne?.querySelectorAll('[data-replay-player-card]')).toHaveLength(8)
    const [banniere, ...autres] = bandeaux(container)
    expect(autres).toHaveLength(0)
    expect(banniere.getAttribute('data-replay-score-banner')).toBe('rangee')
    expect(ligne?.querySelector('[data-replay-teams-row-score]')?.contains(banniere)).toBe(true)
    expect(container.querySelector('section')?.contains(banniere)).toBe(false)
    expect(container.querySelector('[data-replay-body] > [data-replay-teams-row]')).not.toBeNull()
  })

  it('la colonne ne montre plus les équipes : le fil seul, et la fiche du joueur à sa place quand elle est ouverte', async () => {
    const { container } = await mount(match(4), L4)
    goTo('cinema')
    expect(colonneDesEquipes(container)).toBeNull()
    expect(screen.getByTestId('replay-feed-slot')).toBeInTheDocument()
    fireEvent.click(within(rangee(container) as HTMLElement).getByRole('button', { name: 'Ouvrir la fiche de Joueur5' }))
    expect(within(rangee(container) as HTMLElement).getByRole('button', { name: 'Ouvrir la fiche de Joueur5' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByTestId('replay-feed-slot')).toHaveAttribute('aria-hidden', 'true')
    expect(screen.getByRole('region', { name: T.playerPanel.labelFmt('Joueur5') })).toBeInTheDocument()
  })

  it('le joueur de la page est le joueur suivi : sa fiche a la bordure de suivi', async () => {
    const { container } = await mount(match(4), L4)
    goTo('cinema')
    const fiches = within(rangee(container) as HTMLElement)
    expect(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur0' }).style.borderColor).toBe('var(--ring)')
    expect(fiches.getByRole('button', { name: 'Ouvrir la fiche de Joueur1' }).style.borderColor).toBe('var(--border)')
  })

  it('trop étroit (un pixel de moins que la demande) : le repli, équipes en lignes dans la colonne, score sur la toile', async () => {
    const { container } = await mount(match(4), L4 - 1)
    goTo('cinema')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    const [banniere, ...autres] = bandeaux(container)
    expect(autres).toHaveLength(0)
    expect(banniere.getAttribute('data-replay-score-banner')).toBe('scene')
    expect(container.querySelector('section')?.contains(banniere)).toBe(true)
  })

  it('largeur du corps non mesurée : le repli', async () => {
    const { container } = await mount(match(4), 0)
    goTo('cinema')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
  })

  it('le Standard ne change pas : colonne des fiches d’avant, score dans l’en-tête, jamais de rangée', async () => {
    const { container } = await mount(match(4), 4000)
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).toBeNull()
    expect(container.querySelectorAll('[data-replay-player-card]')).toHaveLength(8)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['entete'])
  })

  it('chacun pour soi (un seul camp) : le repli, même sur un corps très large', async () => {
    const { container } = await mount(match(8, 1), 4000)
    goTo('cinema')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
  })
})

describe('ReplayPage : la rangée des équipes en Grande équipe', () => {
  it('12 contre 12 : six emplacements de deux fiches par camp, le score au centre', async () => {
    const { container } = await mount(match(12), LGE)
    goTo('cinema')
    const ligne = rangee(container) as HTMLElement
    expect(ligne.getAttribute('data-replay-teams-row')).toBe('grande-equipe')
    expect(ligne.querySelectorAll('section')).toHaveLength(2)
    for (const camp of ligne.querySelectorAll('section')) expect(camp.children).toHaveLength(6)
    expect(ligne.querySelectorAll('[data-replay-player-card]')).toHaveLength(24)
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['rangee'])
  })

  it('trop étroit pour la Grande équipe : le repli, équipes en paires dans la colonne, score sur la toile', async () => {
    const { container } = await mount(match(12), LGE - 1)
    goTo('plein-cadre')
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['scene'])
  })
})

describe('ReplayPage : la rangée ne remonte ni la toile ni le fil', () => {
  it('Standard, Cinéma, Plein cadre, retour : la même toile et la même cellule du fil, rangée posée', async () => {
    const { container } = await mount(match(4), L4)
    const toile = container.querySelector('canvas')
    const fil = screen.getByTestId('replay-feed-slot')
    for (const frame of ['cinema', 'plein-cadre', 'standard', 'plein-cadre', 'cinema', 'standard'] as const) {
      goTo(frame)
      expect(container.querySelector('canvas')).toBe(toile)
      expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
    }
  })

  it('la fenêtre rétrécit puis s’élargit au Cinéma : rangée, repli, rangée, sans remonter la toile ni le fil', async () => {
    const { container } = await mount(match(4), L4)
    goTo('cinema')
    const toile = container.querySelector('canvas')
    const fil = screen.getByTestId('replay-feed-slot')
    expect(rangee(container)).not.toBeNull()

    redimensionner(L4 - 1)
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['scene'])
    expect(container.querySelector('canvas')).toBe(toile)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)

    redimensionner(L4)
    expect(rangee(container)).not.toBeNull()
    expect(colonneDesEquipes(container)).toBeNull()
    expect(bandeaux(container).map((b) => b.getAttribute('data-replay-score-banner'))).toEqual(['rangee'])
    expect(container.querySelector('canvas')).toBe(toile)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
  })
})
