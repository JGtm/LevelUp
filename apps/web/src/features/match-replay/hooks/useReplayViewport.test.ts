/**
 * Tests — useReplayViewport (la place que l'écran laisse au terrain).
 *
 * CE QU'ILS PROTÈGENT, dans l'ordre d'importance :
 *
 * 1. QUE LE TERRAIN GRANDIT. C'était la demande explicite du 2026-09-02 : sur un grand écran,
 *    une carte servie à 480 px gaspille de la résolution qu'on a déjà téléchargée (les fonds
 *    de carte pèsent jusqu'à 1,4 Mio). Un test échoue si l'offre replafonne à l'ancienne
 *    constante.
 * 2. QUE L'EXPORT NE SUIT PAS. `exportScaleFor` vise une hauteur de sortie stable : une toile
 *    plus grande donne un facteur plus petit, donc la MÊME vidéo. Sans quoi la taille de la
 *    fenêtre de qui exporte déciderait du poids du fichier.
 * 3. LE PLANCHER, sous lequel on laisse la page défiler plutôt que rendre une carte illisible.
 * 4. LA QUANTIFICATION ET LE DÉLAI, qui ne sont pas du confort : sans eux, un glissement de bord
 *    de fenêtre recuirait les quatre calques statiques à chaque image.
 * 5. LA CONVERGENCE. Ajuster la hauteur change la hauteur du conteneur, donc réveille le
 *    `ResizeObserver` : si la seconde mesure ne retombait pas sur la première, la boucle ne
 *    s'arrêterait jamais.
 * 6. LES DEUX SOURCES D'ÉVÉNEMENTS. Rétrécir une fenêtre en HAUTEUR seulement ne change pas la
 *    largeur du conteneur : sans l'écoute de `resize`, rien ne se passerait.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'

import {
  CANVAS_HEIGHT_CEILING,
  CANVAS_HEIGHT_DEFAULT,
  CANVAS_HEIGHT_MIN,
  EXPORT_SUPERSAMPLE,
  EXPORT_TARGET_HEIGHT,
  exportScaleFor,
} from './useReplayView'
import {
  fitCanvasHeight,
  useReplayViewport,
  VIEWPORT_BOTTOM_MARGIN,
  VIEWPORT_HEIGHT_STEP,
  VIEWPORT_SETTLE_MS,
} from './useReplayViewport'

describe('fitCanvasHeight — quantifier, puis borner', () => {
  it('laisse le terrain DÉPASSER l ancienne constante quand l écran le permet', () => {
    expect(fitCanvasHeight(10_000)).toBe(CANVAS_HEIGHT_CEILING)
    expect(CANVAS_HEIGHT_CEILING).toBeGreaterThan(CANVAS_HEIGHT_DEFAULT)
  })

  it('ne descend JAMAIS sous le plancher, si petit que soit l écran', () => {
    expect(fitCanvasHeight(0)).toBe(CANVAS_HEIGHT_MIN)
    expect(fitCanvasHeight(-500)).toBe(CANVAS_HEIGHT_MIN)
  })

  it('quantifie par pas, vers le bas', () => {
    const free = 433
    expect(free % VIEWPORT_HEIGHT_STEP).not.toBe(0)
    const got = fitCanvasHeight(free)
    expect(got % VIEWPORT_HEIGHT_STEP).toBe(0)
    expect(got).toBeLessThanOrEqual(free)
    expect(free - got).toBeLessThan(VIEWPORT_HEIGHT_STEP)
  })

  it('une variation plus petite que le pas ne produit aucune valeur nouvelle', () => {
    const base = 432
    for (let d = 0; d < VIEWPORT_HEIGHT_STEP; d += 1) {
      expect(fitCanvasHeight(base + d)).toBe(fitCanvasHeight(base))
    }
  })
})

describe('exportScaleFor — la vidéo ne change pas de format avec la fenêtre', () => {
  it('à l ancienne hauteur, c est exactement le comportement d avant', () => {
    expect(exportScaleFor(CANVAS_HEIGHT_DEFAULT)).toBe(EXPORT_SUPERSAMPLE)
  })

  it('sort la MÊME hauteur de vidéo quelle que soit la toile', () => {
    for (const h of [CANVAS_HEIGHT_DEFAULT, 560, 640, CANVAS_HEIGHT_CEILING]) {
      expect(h * exportScaleFor(h)).toBeCloseTo(EXPORT_TARGET_HEIGHT, 6)
    }
  })

  it('ne suréchantillonne jamais plus qu avant, même sur une petite toile', () => {
    expect(exportScaleFor(CANVAS_HEIGHT_MIN)).toBe(EXPORT_SUPERSAMPLE)
    expect(exportScaleFor(0)).toBe(EXPORT_SUPERSAMPLE)
  })
})

// ────────────────────────────────────────────────────────────────────────────
// Le hook lui-même, sur un conteneur simulé.
// ────────────────────────────────────────────────────────────────────────────

/** Ce que le conteneur porte en plus du terrain : marges, frise, barre de lecture. */
const CHROME = 200
const WIDTH = 900
const TOP = 150

let painted = CANVAS_HEIGHT_DEFAULT
let host: HTMLDivElement
let canvasEl: HTMLCanvasElement
let observed: (() => void) | null = null

/**
 * Le conteneur reflète la hauteur PEINTE, comme le vrai DOM : quand le terrain rétrécit, le
 * conteneur rétrécit d'autant. Sans cela, la mesure suivante croirait que le chrome a grandi —
 * et le test de convergence ne prouverait rien.
 */
function mountHost() {
  painted = CANVAS_HEIGHT_DEFAULT
  host = document.createElement('div')
  canvasEl = document.createElement('canvas')
  Object.defineProperty(host, 'offsetHeight', {
    configurable: true,
    get: () => CHROME + painted,
  })
  host.getBoundingClientRect = () =>
    ({ top: TOP, left: 0, width: WIDTH, height: CHROME + painted }) as DOMRect
  canvasEl.getBoundingClientRect = () =>
    ({ top: TOP, left: 0, width: WIDTH, height: painted }) as DOMRect
  host.appendChild(canvasEl)
  document.body.appendChild(host)
}

beforeEach(() => {
  vi.useFakeTimers()
  observed = null
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(cb: () => void) {
        observed = cb
      }
      observe() {}
      disconnect() {
        observed = null
      }
    },
  )
  mountHost()
})

afterEach(() => {
  host.remove()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

/** Monte le hook et tient la hauteur peinte à jour, comme le ferait le rendu. */
function renderViewport() {
  const ref = { current: host as HTMLElement | null }
  const canvasRef = { current: canvasEl as HTMLCanvasElement | null }
  const view = renderHook(() => useReplayViewport(ref, canvasRef))
  painted = view.result.current.freeHeight
  return view
}

describe('useReplayViewport', () => {
  it('mesure la largeur du conteneur', () => {
    window.innerHeight = 1200
    const { result } = renderViewport()
    expect(result.current.width).toBe(WIDTH)
  })

  // LA DEMANDE DU 2026-09-02, TENUE PAR UN TEST : sur un grand écran, le terrain doit dépasser
  // l'ancienne constante — sinon on gaspille une résolution déjà téléchargée.
  it('sur un grand écran, il offre PLUS que l ancienne hauteur fixe', () => {
    window.innerHeight = 1200
    const { result } = renderViewport()
    expect(result.current.freeHeight).toBeGreaterThan(CANVAS_HEIGHT_DEFAULT)
    expect(result.current.freeHeight).toBe(CANVAS_HEIGHT_CEILING)
  })

  it('sur un écran contraint, le terrain rétrécit à ce qui reste', () => {
    window.innerHeight = 800
    const { result } = renderViewport()
    const free = 800 - TOP - CHROME - VIEWPORT_BOTTOM_MARGIN
    expect(result.current.freeHeight).toBe(fitCanvasHeight(free))
    expect(result.current.freeHeight).toBeLessThan(CANVAS_HEIGHT_DEFAULT)
    expect(result.current.freeHeight).toBeGreaterThanOrEqual(CANVAS_HEIGHT_MIN)
  })

  it('sur un écran très court, il s arrête au plancher et laisse la page défiler', () => {
    window.innerHeight = 600
    const { result } = renderViewport()
    expect(result.current.freeHeight).toBe(CANVAS_HEIGHT_MIN)
  })

  // LA SOURCE QUE LE ResizeObserver NE COUVRE PAS : la fenêtre rétrécie en hauteur seule.
  it('suit un rétrécissement VERTICAL de la fenêtre, que l observateur ne voit pas', () => {
    window.innerHeight = 1200
    const { result } = renderViewport()
    expect(result.current.freeHeight).toBe(CANVAS_HEIGHT_CEILING)

    act(() => {
      window.innerHeight = 800
      window.dispatchEvent(new Event('resize'))
      vi.advanceTimersByTime(VIEWPORT_SETTLE_MS)
    })
    painted = result.current.freeHeight
    expect(result.current.freeHeight).toBe(
      fitCanvasHeight(800 - TOP - CHROME - VIEWPORT_BOTTOM_MARGIN),
    )
  })

  it('ne remesure RIEN avant la fin du délai — un glissement ne cuit qu une fois', () => {
    window.innerHeight = 1200
    const { result } = renderViewport()

    act(() => {
      for (let i = 0; i < 10; i += 1) {
        window.innerHeight = 1200 - i * 40
        window.dispatchEvent(new Event('resize'))
        vi.advanceTimersByTime(VIEWPORT_SETTLE_MS - 10)
      }
    })
    expect(result.current.freeHeight).toBe(CANVAS_HEIGHT_CEILING)

    act(() => {
      vi.advanceTimersByTime(VIEWPORT_SETTLE_MS)
    })
    expect(result.current.freeHeight).toBeLessThan(CANVAS_HEIGHT_CEILING)
  })

  // LA BOUCLE QUI DOIT S ARRÊTER : ajuster la hauteur change la taille du conteneur, donc
  // réveille l'observateur. La deuxième mesure doit retomber sur la première.
  it('converge — la mesure déclenchée par son propre ajustement ne bouge plus', () => {
    window.innerHeight = 800
    const { result } = renderViewport()
    const settled = result.current.freeHeight
    expect(settled).toBeLessThan(CANVAS_HEIGHT_CEILING)

    act(() => {
      observed?.()
      vi.advanceTimersByTime(VIEWPORT_SETTLE_MS)
    })
    expect(result.current.freeHeight).toBe(settled)
  })

  it('se débranche au démontage — plus aucune mesure après', () => {
    window.innerHeight = 1200
    const { result, unmount } = renderViewport()
    const before = result.current.freeHeight
    unmount()
    expect(observed).toBeNull()

    act(() => {
      window.innerHeight = 600
      window.dispatchEvent(new Event('resize'))
      vi.advanceTimersByTime(VIEWPORT_SETTLE_MS * 4)
    })
    expect(result.current.freeHeight).toBe(before)
  })
})
