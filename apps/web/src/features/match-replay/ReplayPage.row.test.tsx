/**
 * Tests — LA RANGÉE DES ÉQUIPES du Cinéma et du Plein cadre, dans la vraie page.
 *
 * Monte la vraie page, la vraie toile (contexte 2D enregistreur) et un document à deux camps AVEC calque de
 * score ; la largeur du corps du cadre est posée par le test (jsdom ne met rien en page) et suivie par un
 * observateur de taille simulé. Ce que ces tests prouvent :
 *  - la rangée est posée quand la règle (`dispositionEquipes`) la retient : le score est AU CENTRE de la rangée
 *    (emplacement `rangee`), il n'est plus sur la toile, et le fil des éliminations passe SUR LE PLAN (la colonne
 *    est cachée tant qu'aucune fiche n'est ouverte, son fil complet reste monté) ;
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
  window.localStorage.clear()
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

/** Une élimination du joueur 0 sur le joueur 5 à `secondes` : le fil de la base, que la page recale sur le film. */
function elimination(secondes: number) {
  return { event_type: 'kill', actor_xuid: 'X0', victim_xuid: 'X5', victim_gamertag: 'Joueur5', event_time_ms: secondes * 1_000 }
}

/**
 * Un match à `nbCamps` camps de `parCamp` places (équipe 0, 1, ...), avec calque de score pour les deux premiers.
 * `highlights` : les événements de la base (les éliminations du fil).
 */
function match(parCamp: number, nbCamps = 2, highlights: object[] = []) {
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
    highlight_events: highlights,
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

  it('la colonne ne montre plus les équipes : le fil complet reste monté (caché), et la fiche du joueur prend la colonne quand elle est ouverte', async () => {
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

  it('le Standard : équipes en lignes dans sa colonne, score dans l’en-tête, jamais de rangée', async () => {
    const { container } = await mount(match(4), 4000)
    expect(rangee(container)).toBeNull()
    expect(colonneDesEquipes(container)).not.toBeNull()
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

const filDuPlan = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-plan-feed]')
const colonne = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-column]')
/** Trois éliminations au coup d'envoi : le rejeu s'ouvre en pause à l'image 0, elles sont déjà passées. */
const TROIS_ELIMINATIONS = [elimination(0), elimination(0), elimination(0)]

describe('ReplayPage : le fil des éliminations posé sur le plan', () => {
  it.each(['cinema', 'plein-cadre'] as const)('%s, rangée posée : les éliminations passées sont SUR LE PLAN, dans le bloc de la toile', async (frame) => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo(frame)
    const fil = filDuPlan(container)
    expect(fil).not.toBeNull()
    expect(fil?.querySelectorAll('li')).toHaveLength(3)
    expect(container.querySelector('[data-replay-grid] > section')?.contains(fil)).toBe(true)
    expect(fil?.contains(container.querySelector('canvas'))).toBe(false)
  })

  it('rangée posée : la colonne est cachée (le plan prend sa largeur) mais le fil complet y reste monté, même nœud', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    const fil = screen.getByTestId('replay-feed-slot')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    goTo('cinema')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    expect(colonne(container)?.classList.contains('hidden')).toBe(true)
    expect(colonne(container)?.contains(screen.getByTestId('replay-feed-slot'))).toBe(true)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
    expect(container.querySelector('[data-replay-grid]')?.className).toContain('xl:grid-cols-[minmax(0,1fr)]')
    expect(container.querySelector('[data-replay-grid]')?.className).not.toContain('30rem')
  })

  it("la colonne du Cinéma ne se montre que lorsqu'une fiche de joueur est ouverte, à la place d'aujourd'hui", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('cinema')
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
    fireEvent.click(within(rangee(container) as HTMLElement).getByRole('button', { name: 'Ouvrir la fiche de Joueur5' }))
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    expect(container.querySelector('[data-replay-grid]')?.className).toContain('30rem')
    expect(within(colonne(container) as HTMLElement).getByRole('region', { name: T.playerPanel.labelFmt('Joueur5') })).toBeInTheDocument()
    fireEvent.click(within(colonne(container) as HTMLElement).getByRole('button', { name: T.playerPanel.back }))
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
  })

  it('repli (équipes en colonne) : pas de fil sur le plan, la colonne garde les équipes et le fil complet', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4 - 1)
    goTo('cinema')
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    expect(colonne(container)?.contains(colonneDesEquipes(container))).toBe(true)
    expect(within(screen.getByTestId('replay-feed-slot')).getAllByRole('listitem').length).toBeGreaterThan(0)
  })

  it('largeur non mesurée : le repli aussi', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), 0)
    goTo('plein-cadre')
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
  })

  it('le Standard : jamais de fil sur le plan, sa colonne porte le fil et reste affichée', async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), 4000)
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    expect(within(screen.getByTestId('replay-feed-slot')).getAllByRole('listitem').length).toBeGreaterThan(0)
  })

  it("la fenêtre rétrécit puis s'élargit : le fil du plan apparaît et disparaît avec la rangée, sans remonter la toile ni le fil complet", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('cinema')
    const toile = container.querySelector('canvas')
    const fil = screen.getByTestId('replay-feed-slot')
    expect(filDuPlan(container)).not.toBeNull()
    redimensionner(L4 - 1)
    expect(filDuPlan(container)).toBeNull()
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('affichee')
    redimensionner(L4)
    expect(filDuPlan(container)).not.toBeNull()
    expect(container.querySelector('canvas')).toBe(toile)
    expect(screen.getByTestId('replay-feed-slot')).toBe(fil)
  })

  it("rien n'est remonté pendant la lecture : le fil du plan et la cellule du fil complet gardent leur nœud", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('cinema')
    const fil = filDuPlan(container)
    const cellule = screen.getByTestId('replay-feed-slot')
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(filDuPlan(container)).toBe(fil)
    expect(screen.getByTestId('replay-feed-slot')).toBe(cellule)
  })
})

describe("ReplayPage : l'option « Fil des éliminations » du panneau Vue", () => {
  const option = () => screen.queryByRole('switch', { name: T.frames.planFeed })
  const ouvrirLaVue = () => fireEvent.click(screen.getByRole('button', { name: /^Vue / }))

  it("Cinéma, rangée posée : l'option est proposée, allumée ; l'éteindre retire le fil du plan, le rallumer le rend", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('cinema')
    expect(filDuPlan(container)).not.toBeNull()
    ouvrirLaVue()
    expect(option()).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(option() as HTMLElement)
    expect(filDuPlan(container)).toBeNull()
    fireEvent.click(option() as HTMLElement)
    expect(filDuPlan(container)).not.toBeNull()
  })

  it('éteinte, elle le reste au remontage de la page', async () => {
    const premier = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('plein-cadre')
    ouvrirLaVue()
    fireEvent.click(option() as HTMLElement)
    expect(filDuPlan(premier.container)).toBeNull()
    premier.unmount()
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('plein-cadre')
    expect(filDuPlan(container)).toBeNull()
    ouvrirLaVue()
    expect(option()).toHaveAttribute('aria-checked', 'false')
  })

  it("Standard : l'option n'est pas montrée (aucun effet à l'écran)", async () => {
    await mount(match(4, 2, TROIS_ELIMINATIONS), 4000)
    ouvrirLaVue()
    expect(screen.getByRole('region', { name: T.viewButton })).toBeInTheDocument()
    expect(option()).toBeNull()
  })

  it("repli (équipes en colonne) au Cinéma : l'option n'est pas montrée", async () => {
    await mount(match(4, 2, TROIS_ELIMINATIONS), L4 - 1)
    goTo('cinema')
    ouvrirLaVue()
    expect(screen.getByRole('region', { name: T.viewButton })).toBeInTheDocument()
    expect(option()).toBeNull()
  })

  it("option éteinte au Cinéma, rangée posée : la colonne reste cachée (elle ne sert qu'à une fiche ouverte)", async () => {
    const { container } = await mount(match(4, 2, TROIS_ELIMINATIONS), L4)
    goTo('cinema')
    ouvrirLaVue()
    fireEvent.click(option() as HTMLElement)
    expect(colonne(container)?.getAttribute('data-replay-column')).toBe('cachee')
  })
})
