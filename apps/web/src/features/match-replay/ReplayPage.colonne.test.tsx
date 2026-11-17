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
    expect(equipes).toHaveAttribute('data-replay-roster-column')
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

  it('un clic sur une ligne ouvre la fiche à la place du fil, marque la ligne ; « Retour au fil » la referme', async () => {
    const { container } = await mountReady()
    const ligne = columnOf(container)?.querySelector<HTMLElement>('[data-replay-player-card]') as HTMLElement
    expect(ligne.getAttribute('aria-pressed')).toBe('false')
    const feedSlot = screen.getByTestId('replay-feed-slot')
    fireEvent.click(ligne)
    expect(ligne.getAttribute('aria-pressed')).toBe('true')
    expect(feedSlot.getAttribute('aria-hidden')).toBe('true')
    // Les équipes restent au-dessus : la fiche ne prend que la cellule du fil, dans la même colonne.
    expect(columnOf(container)?.contains(ligne)).toBe(true)
    const retour = screen.getByRole('button', { name: T.playerPanel.back })
    expect(feedSlot.parentElement?.contains(retour)).toBe(true)
    fireEvent.click(retour)
    expect(screen.queryByRole('button', { name: T.playerPanel.back })).toBeNull()
    expect(screen.getByTestId('replay-feed-slot').hasAttribute('aria-hidden')).toBe(false)
    expect(ligne.getAttribute('aria-pressed')).toBe('false')
  })

  it('Échap ferme la fiche ouverte dans la colonne', async () => {
    const { container } = await mountReady()
    fireEvent.click(columnOf(container)?.querySelector<HTMLElement>('[data-replay-player-card]') as HTMLElement)
    expect(screen.getByRole('button', { name: T.playerPanel.back })).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('button', { name: T.playerPanel.back })).toBeNull()
  })
})

describe('ReplayPage : Cinéma et Plein cadre gardent leur colonne', () => {
  it('colonne souple de 30 rem, bloc des fiches plafonné, pas de colonne du Standard', async () => {
    const { container } = await mountReady()
    for (const frame of ['cinema', 'plein-cadre'] as const) {
      goTo(frame)
      const grid = container.querySelector<HTMLElement>(`[data-replay-grid="${frame}"]`)
      expect(grid?.className).toContain('minmax(0,30rem)')
      expect(grid?.style.getPropertyValue('--replay-colonne-standard')).toBe('')
      expect(columnOf(container)).toBeNull()
      expect(container.querySelector('[data-replay-roster-column]')).toBeNull()
      expect(container.querySelector('aside .xl\\:max-h-\\[30rem\\]')).not.toBeNull()
    }
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
