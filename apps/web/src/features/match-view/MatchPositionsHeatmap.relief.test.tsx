/**
 * MatchPositionsHeatmap — le fond Relief, ses traits et la coupe par étage d'« Occupation du terrain » (item D.4
 * du plan du fond Relief) : le bloc a SON choix de fond (une pilule), l'interrupteur des traits et la commande d'étage ;
 * les positions hors de l'étage choisi sont MASQUÉES, et une ligne d'état dit combien (« n sur total ») ; une carte sans
 * maillage n'a pas de commande d'étage ; un relief illisible laisse le plan, journalisé.
 *
 * Le tracé (`peindreLesPositions`), les fichiers du relief et le calculateur sont remplacés par des doubles : ce qui se
 * teste est ce que le bloc DÉCIDE et ce qu'il donne au tracé.
 */
import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { MatchPlayerPosition } from '@/lib/api/types'
import { parseReliefData, type ReliefFrame } from '@/lib/replay/relief'
import { rawRelief, sceneCalibration } from '@/lib/replay/relief/reliefScene.testutil'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { peindreLesPositions } from './_positionsPaint'
import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'

const mocks = vi.hoisted(() => ({
  background: vi.fn(),
  files: vi.fn(),
  render: vi.fn(),
}))

vi.mock('@/lib/replay/queries', () => ({
  useReplayMapBackground: () => mocks.background(),
  useReplayMapImage: () => ({ image: { width: 40, height: 10 }, variante: 'plan' }),
}))
vi.mock('./queries', () => ({ useMatchView: () => ({ data: { header: { map_id: 'map-streets' } } }) }))
vi.mock('@tanstack/react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@tanstack/react-router')>()),
  Link: ({ children }: { children: React.ReactNode }) => <a href="#tactique">{children}</a>,
}))
vi.mock('./_positionsPaint', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./_positionsPaint')>()),
  peindreLesPositions: vi.fn(),
}))
vi.mock('@/lib/replay/reliefQueries', () => ({ useReliefFiles: mocks.files }))
vi.mock('@/lib/replay/useReliefRender', () => ({ useReliefRender: mocks.render }))
vi.mock('@/lib/replay/useReliefInks', () => ({ useReliefInks: () => ({ base: '#6b7078', outline: '#ffffff' }) }))
vi.mock('@/lib/replay/relief/reliefBackground', () => ({
  ReliefCanvases: class {
    reliefImage = () => toileDuRelief
    maskedImage = () => toileDuRelief
  },
}))

const toileDuRelief = document.createElement('canvas')
toileDuRelief.width = 40
toileDuRelief.height = 10

const CAL = { ...sceneCalibration(40, 10), convention: 'test' }

/** Trois étages (0, 2,2, 4,4 ; les trois premiers mètres), ou aucun maillage. */
function donnees(sansMaillage = false) {
  return parseReliefData(
    rawRelief({
      width: 40,
      height: 10,
      levels: sansMaillage ? [] : [0, 2.2, 4.4],
      rects: sansMaillage
        ? []
        : [
            { i0: 0, j0: 0, i1: 9, j1: 9, z: 0 },
            { i0: 10, j0: 0, i1: 19, j1: 9, z: 2.2 },
            { i0: 20, j0: 0, i1: 29, j1: 9, z: 4.4 },
          ],
      meshShare: sansMaillage ? undefined : 0.46,
      degradations: sansMaillage ? ['sans-maillage'] : [],
    }),
  )
}

function frame(): ReliefFrame {
  const empty = { points: new Float32Array(0), offsets: new Uint32Array(1) }
  return { width: 40, height: 10, cut: null, image: new Uint8ClampedArray(4), alpha: null, planAlpha: null, lines: { steps: empty, contours: empty, walls: empty } }
}

// Un joueur par étage, et un hors maillage (x = 3,5 : au-delà des trois blocs) à la hauteur du premier étage.
const POSITIONS: MatchPlayerPosition[] = [
  { timeMs: 0, x: 0.5, y: 0.5, z: 0, team: 0 },
  { timeMs: 0, x: 1.5, y: 0.5, z: 2.2, team: 0 },
  { timeMs: 0, x: 2.5, y: 0.5, z: 4.4, team: 1 },
  { timeMs: 0, x: 3.5, y: 0.5, z: 2.2, team: 1 },
]

function carte({ relief = true, sansMaillage = false, error = null as unknown } = {}) {
  mocks.background.mockReturnValue({
    data: { calibration: CAL, planImage: 'plan.webp', ...(relief ? { reliefImage: 'r.webp', reliefData: 'r.json' } : {}) },
  })
  mocks.files.mockReturnValue({ data: relief && !error ? donnees(sansMaillage) : null, blob: relief && !error ? new Blob() : null, loading: false, error })
}

function rendre(positions: MatchPlayerPosition[] = POSITIONS) {
  return renderWithProviders(<MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={positions} locale="fr" />)
}

const etage = (nom: string) => fireEvent.click(within(screen.getByTestId('match-positions-etage')).getByRole('button', { name: nom }))
const statut = () => screen.queryByTestId('match-positions-hidden')
const dernierTrace = () => vi.mocked(peindreLesPositions).mock.calls.at(-1)?.[3]

beforeEach(() => {
  window.localStorage.clear()
  mocks.files.mockReset()
  mocks.render.mockReset().mockReturnValue({ status: 'ready', info: { width: 40, height: 10, hasMesh: true }, frame: frame(), error: null })
  carte()
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
  vi.spyOn(HTMLCanvasElement.prototype, 'clientWidth', 'get').mockReturnValue(200)
  vi.spyOn(HTMLCanvasElement.prototype, 'clientHeight', 'get').mockReturnValue(100)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ clearRect: vi.fn() } as never)
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.mocked(peindreLesPositions).mockClear()
})

describe('MatchPositionsHeatmap — le choix de fond du bloc', () => {
  it('propose Plan, Rendu, Relief, avec le plan retenu par défaut', () => {
    rendre()
    const pilule = screen.getByTestId('match-positions-fond-pilule')
    expect(within(pilule).getAllByRole('button').map((b) => b.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(within(pilule).getByRole('button', { name: 'Plan' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('choisir le Relief l’écrit dans la préférence du rejeu et donne la toile du relief au tracé', () => {
    rendre()
    expect(dernierTrace()?.image).not.toBe(toileDuRelief)
    fireEvent.click(within(screen.getByTestId('match-positions-fond-pilule')).getByRole('button', { name: 'Relief' }))
    expect(window.localStorage.getItem('replay-background-style')).toBe('relief')
    expect(dernierTrace()?.image).toBe(toileDuRelief)
  })

  it('le plan des murs se trace sur le relief (qui ne les porte pas), pas sur le plan', () => {
    window.localStorage.setItem('replay-background-style', 'relief')
    carte()
    mocks.background.mockReturnValue({
      data: { calibration: CAL, planImage: 'plan.webp', reliefImage: 'r.webp', reliefData: 'r.json', plan: { edges: [[0, 0, 1, 1]], faces: 1, source: 'navmesh', vertices: 4 } },
    })
    rendre()
    expect(dernierTrace()?.plan?.edges).toEqual([[0, 0, 1, 1]])
  })

  it('carte sans relief : Plan et Rendu seulement, ni traits ni étage', () => {
    carte({ relief: false })
    rendre()
    const pilule = screen.getByTestId('match-positions-fond-pilule')
    expect(within(pilule).getAllByRole('button').map((b) => b.textContent)).toEqual(['Plan', 'Rendu'])
    expect(screen.queryByTestId('match-positions-traits')).toBeNull()
    expect(screen.queryByTestId('match-positions-etage')).toBeNull()
  })
})

describe('MatchPositionsHeatmap — les traits du relief', () => {
  it('éteints sur le Plan, allumés d’office sur le Relief : le tracé les reçoit alors', () => {
    rendre()
    expect(screen.getByTestId('match-positions-traits')).toHaveAttribute('aria-pressed', 'false')
    expect(dernierTrace()?.reliefLines ?? null).toBeNull()
    fireEvent.click(within(screen.getByTestId('match-positions-fond-pilule')).getByRole('button', { name: 'Relief' }))
    expect(screen.getByTestId('match-positions-traits')).toHaveAttribute('aria-pressed', 'true')
    expect(dernierTrace()?.reliefLines?.calibration.widthPx).toBe(40)
  })
})

describe('MatchPositionsHeatmap — la coupe par étage masque les positions hors de l’étage', () => {
  it('« Tout » : aucune position masquée, aucune ligne d’état', () => {
    rendre()
    expect(within(screen.getByTestId('match-positions-etage')).getAllByRole('button').map((b) => b.textContent)).toEqual(['Tout', 'Bas', 'Étage 1', 'Étage 2'])
    expect(statut()).toBeNull()
  })

  it('l’étage Bas ne garde que la position du bas : « 3 sur 4 » masquées', () => {
    rendre()
    etage('Bas')
    expect(statut()).toHaveTextContent('Positions hors de l’étage choisi : masquées (3 sur 4).')
  })

  it('l’étage 1 garde la position de l’étage et celle sans sol sous elle (jugée sur sa hauteur) : « 2 sur 4 »', () => {
    rendre()
    etage('Étage 1')
    expect(statut()).toHaveTextContent('masquées (2 sur 4)')
  })

  it('« Tout » lève la coupe et la ligne d’état disparaît', () => {
    rendre()
    etage('Bas')
    etage('Tout')
    expect(statut()).toBeNull()
  })

  it('le compte se fait sur l’équipe affichée : « Équipe B » masque 2 sur 2 à l’étage Bas', () => {
    rendre()
    fireEvent.click(screen.getByRole('button', { name: 'Équipe B' }))
    etage('Bas')
    expect(statut()).toHaveTextContent('masquées (2 sur 2)')
  })

  it('un étage sans aucune position ne retire pas le bloc de l’écran : ses commandes restent, la chaleur est vide', () => {
    rendre()
    fireEvent.click(screen.getByRole('button', { name: 'Équipe B' }))
    expect(dernierTrace()?.lissee).not.toBeNull()
    etage('Bas')
    expect(screen.getByTestId('match-positions-frame')).toBeInTheDocument()
    expect(screen.getByTestId('match-positions-etage')).toBeInTheDocument()
    expect(dernierTrace()?.lissee).toBeNull()
  })

  it('les positions masquées ne comptent plus dans la chaleur peinte', () => {
    rendre()
    const somme = () => dernierTrace()?.lissee?.cells.reduce((acc, c) => acc + c.value, 0) ?? 0
    const toutes = somme()
    etage('Bas')
    expect(somme()).toBeGreaterThan(0)
    expect(somme()).toBeLessThan(toutes)
  })

  it('se dit en anglais', () => {
    renderWithProviders(<MatchPositionsHeatmap playerSlug="JGtm" matchId="m1" positions={POSITIONS} locale="en" />)
    fireEvent.click(within(screen.getByTestId('match-positions-etage')).getByRole('button', { name: 'Floor 1' }))
    expect(statut()).toHaveTextContent('Positions outside the chosen floor: hidden (2 of 4).')
  })
})

describe('MatchPositionsHeatmap — carte sans maillage de navigation', () => {
  beforeEach(() => carte({ sansMaillage: true }))

  it('pas de commande d’étage, pas de ligne d’état ; le relief et ses traits restent proposés', () => {
    rendre()
    expect(screen.queryByTestId('match-positions-etage')).toBeNull()
    expect(statut()).toBeNull()
    expect(within(screen.getByTestId('match-positions-fond-pilule')).getByRole('button', { name: 'Relief' })).toBeInTheDocument()
    expect(screen.getByTestId('match-positions-traits')).toBeInTheDocument()
  })
})

describe('MatchPositionsHeatmap — repli sur le plan, journalisé', () => {
  it('un relief illisible : plus de « Relief » proposé, le plan reste retenu, le journal le dit', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    window.localStorage.setItem('replay-background-style', 'relief')
    carte({ error: new Error('image des hauteurs illisible') })
    rendre()
    const pilule = screen.getByTestId('match-positions-fond-pilule')
    expect(within(pilule).queryByRole('button', { name: 'Relief' })).toBeNull()
    expect(within(pilule).getByRole('button', { name: 'Plan' })).toHaveAttribute('aria-pressed', 'true')
    expect(warn).toHaveBeenCalledWith(expect.stringContaining('[relief-view]'))
  })
})
