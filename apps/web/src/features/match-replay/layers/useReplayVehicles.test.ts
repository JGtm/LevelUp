/**
 * Tests — useReplayVehicles : LE COUPLE SPRITE + BORDURE est INDISSOCIABLE.
 *
 * DÉCISION UTILISATEUR : un véhicule ne se dessine JAMAIS à moitié. La bordure
 * (`{famille}_outline.png`, sprites redessinés du 2026-09-16) est cuite DANS la vignette,
 * sous le sprite teint ; tant qu'elle n'est pas là, `spriteOf` rend `null` et le calque ne
 * dessine rien plutôt que de montrer un sprite nu qui gagnerait sa bordure une image plus
 * tard. Le seul cas où le sprite se dessine seul est l'ÉCHEC de la bordure — un habillage
 * introuvable ne doit pas effacer le véhicule pour le reste de la lecture.
 *
 * CE QUE CE FICHIER VERROUILLE (aucun de ces trois points n'était couvert) :
 *  (a) bordure EN ATTENTE  -> `spriteOf` rend `null` ET NE MET RIEN EN CACHE (sans quoi la
 *      vignette sans bordure serait servie pour toujours une fois la bordure arrivée) ;
 *  (b) bordure CHARGÉE     -> `spriteOf` rend une vignette et `outlinedSpriteCanvas` a cuit
 *      la bordure sous le sprite teint ;
 *  (c) bordure EN ÉCHEC    -> `spriteOf` rend le sprite SANS bordure, un `console.warn`
 *      UNIQUE, et l'appel suivant ne relance AUCUNE requête (`outlineFailedRef`).
 *
 * COMMENT ON ATTEINT `spriteOf` : il n'est pas rendu par le hook (c'est une préoccupation du
 * calque), il est passé en style à `drawVehiclesLayer`. Le tracé est donc doublé pour le
 * capturer — même parti que les autres tests de calque, qui n'ouvrent jamais un vrai canvas.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayVehicleLabel, ReplayVehicleTrack } from '@/lib/api/types'

import { testReplayDoc } from '../test/testDoc'
import type { FxInk } from './fxInk'
import { outlinedSpriteCanvas } from './replayDraw'
import { useReplayVehicles } from './useReplayVehicles'
import { drawVehiclesLayer, type VehicleStyle } from './vehiclesPaint'

// Le TRACÉ est doublé : ce fichier n'observe pas des pixels, il observe le style que le hook
// compose — et `spriteOf` n'existe nulle part ailleurs.
vi.mock('./vehiclesPaint', () => ({ drawVehiclesLayer: vi.fn() }))

// `outlinedSpriteCanvas` garde son implémentation RÉELLE (la vignette cuite doit avoir une
// taille), elle est seulement observée : c'est elle qui prouve que la bordure est passée sous
// le sprite, et non un second tracé au moment du rendu.
vi.mock('./replayDraw', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./replayDraw')>()
  return { ...actual, outlinedSpriteCanvas: vi.fn(actual.outlinedSpriteCanvas) }
})

const VUE = { bounds: { minX: -20, minY: -20, maxX: 20, maxY: 20 }, width: 200, height: 200, pad: 0 }

const ENCRE_FX: FxInk = {
  tint: {
    kinetic: 'k', plasma_cool: 'pc', plasma_hot: 'ph',
    forerunner: 'f', electric: 'e', needle: 'n', blast: 'b', neutral: 'x',
  },
  core: 'c',
}

/** Une image factice : `withLoadedImage` en crée une par URL, le test les déclenche à la main. */
class FausseImage {
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  src = ''
  naturalWidth = 64
  naturalHeight = 32
  constructor() {
    creees.push(this)
  }
}
let creees: FausseImage[] = []

/** Le contexte 2D de jsdom n'existe pas : un faux suffit, on ne lit jamais de pixel. */
function fauxContexte2D() {
  return {
    translate: vi.fn(), rotate: vi.fn(), scale: vi.fn(), drawImage: vi.fn(),
    fillRect: vi.fn(), setTransform: vi.fn(), save: vi.fn(), restore: vi.fn(),
    globalCompositeOperation: 'source-over', fillStyle: '',
  }
}

function piste(family: string): ReplayVehicleTrack {
  return {
    slot: 1, gen: 1, family, chassis: '0x1', end: 'unknown', t0: 0, t1: 100, t1max: 100,
    spawn: { x: 0, y: 0 },
    samples: [{ t: 0, x: 0, y: 0, h: 0 }],
    rides: [],
  } as unknown as ReplayVehicleTrack
}

/**
 * Monte le hook sur UNE famille, attend le manifeste, puis charge le sprite SOURCE (pas la
 * bordure : c'est elle que chaque cas met en scène). Rend le `spriteOf` capté au tracé.
 *
 * Le nom de famille est unique par test : `withLoadedImage` a un cache PAR URL au niveau du
 * module, deux tests qui partageraient une URL partageraient son image.
 */
async function monter(family: string, opts: { outline?: string | null } = {}) {
  const redraw = vi.fn()
  const labels: Record<string, ReplayVehicleLabel> = {
    [family]: { img: `/sprite-${family}.png`, tinted: true },
  }
  const doc = testReplayDoc({ vehicles: [piste(family)], vehicleLabels: labels })

  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
    ok: true,
    json: () => Promise.resolve([
      { famille: family, scale_mm_per_px: 10, outline: opts.outline ?? `${family}_outline.png`, pad: 4 },
    ]),
  })))

  const vue = renderHook(() =>
    useReplayVehicles({
      doc, view: VUE, frameRef: { current: 0 }, enabled: true, locale: 'fr', showNames: false, showAim: false,
      colorOfSlot: () => '#123456', colorOfXuid: () => '#123456',
      nameOfSlot: () => null, nameOfXuid: () => null,
      offscreenLabelOf: () => '', offscreenGroupLabelOf: () => '',
      neutralInk: 'n', labelStroke: 's', markInk: { fill: 'm', outline: 'o' }, explosionInk: ENCRE_FX, reducedMotion: true,
      redraw,
    }),
  )

  // Le manifeste (fetch) puis le sprite SOURCE : sans l'un ou l'autre, `spriteOf` rend `null`
  // pour une raison qui n'est pas celle qu'on observe ici.
  await act(async () => {})
  const sourceImg = creees.find((im) => im.src === `/sprite-${family}.png`)
  expect(sourceImg, 'le sprite source doit avoir ete demande').toBeDefined()
  await act(async () => { sourceImg?.onload?.() })

  const spriteOf = () => {
    act(() => {
      vue.result.current.paint(fauxContexte2D() as unknown as CanvasRenderingContext2D, 0, 1)
    })
    const style = vi.mocked(drawVehiclesLayer).mock.calls.at(-1)?.[4] as VehicleStyle
    return style.spriteOf
  }
  return { spriteOf, redraw, urlBordure: `/static/vehicles-assets/halo_infinite/replay/${family}_outline.png` }
}

let getContextOriginal: typeof HTMLCanvasElement.prototype.getContext

beforeEach(() => {
  creees = []
  vi.stubGlobal('Image', FausseImage)
  getContextOriginal = HTMLCanvasElement.prototype.getContext
  HTMLCanvasElement.prototype.getContext = vi.fn(
    () => fauxContexte2D(),
  ) as unknown as typeof HTMLCanvasElement.prototype.getContext
  vi.mocked(outlinedSpriteCanvas).mockClear()
  vi.mocked(drawVehiclesLayer).mockClear()
})

afterEach(() => {
  HTMLCanvasElement.prototype.getContext = getContextOriginal
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('useReplayVehicles — sprite et bordure sont indissociables', () => {
  it("(a) bordure EN ATTENTE : spriteOf rend null, et rien n'est mis en cache", async () => {
    const { spriteOf, urlBordure } = await monter('warthog_a')
    const rendre = spriteOf()

    expect(rendre('warthog_a', '#ff0000')).toBeNull()
    // La bordure a bien ete DEMANDEE (c'est le premier trace qui la demande), pas cuite.
    expect(creees.some((im) => im.src === urlBordure)).toBe(true)
    expect(outlinedSpriteCanvas).not.toHaveBeenCalled()

    // RIEN EN CACHE : un second appel, bordure toujours en attente, rend encore `null` —
    // si la vignette sans bordure avait ete memorisee, elle serait servie ici.
    expect(rendre('warthog_a', '#ff0000')).toBeNull()
    expect(outlinedSpriteCanvas).not.toHaveBeenCalled()
  })

  it('(b) bordure CHARGÉE : spriteOf rend une vignette où la bordure est cuite sous le sprite', async () => {
    const { spriteOf, urlBordure } = await monter('warthog_b')
    const rendre = spriteOf()
    expect(rendre('warthog_b', '#ff0000')).toBeNull()

    const bordure = creees.find((im) => im.src === urlBordure)
    expect(bordure).toBeDefined()
    await act(async () => { bordure?.onload?.() })

    const vignette = rendre('warthog_b', '#ff0000')
    expect(vignette).not.toBeNull()
    expect(outlinedSpriteCanvas).toHaveBeenCalledTimes(1)
    // Premier argument = l'image de BORDURE (elle passe SOUS), second = le sprite teint.
    expect(vi.mocked(outlinedSpriteCanvas).mock.calls[0][0]).toBe(bordure)
    expect((vignette as HTMLCanvasElement).width).toBe(64)
  })

  it("(c) bordure EN ÉCHEC : sprite sans bordure, un seul avertissement, aucune nouvelle requête", async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { spriteOf, urlBordure } = await monter('warthog_c')
    const rendre = spriteOf()
    expect(rendre('warthog_c', '#ff0000')).toBeNull()

    const bordure = creees.find((im) => im.src === urlBordure)
    expect(bordure).toBeDefined()
    await act(async () => { bordure?.onerror?.() })
    expect(warn).toHaveBeenCalledTimes(1)

    const demandesAvant = creees.length
    const vignette = rendre('warthog_c', '#ff0000')
    expect(vignette).not.toBeNull()
    expect(outlinedSpriteCanvas).not.toHaveBeenCalled()

    // DEUXIÈME appel : ni nouvelle image demandée, ni second avertissement.
    expect(rendre('warthog_c', '#ff0000')).toBe(vignette)
    expect(creees.length).toBe(demandesAvant)
    expect(warn).toHaveBeenCalledTimes(1)
  })
})

/**
 * « DISPONIBLE » ET LE DÉCOR DE CARTE (retours du rejeu 2026-09-23, lot L1.3 ; revue RR-L1-02 ;
 * depuis le lot M7 du 2026-09-24, le décor est DÉCLARÉ par le serveur — `vehicleScenery.hidden` —,
 * posé par la carte hors de sa zone jouable). Un document qui ne porte que du décor de carte
 * n'a rien que le calque dessinerait : la bascule ne doit pas s'afficher. Sans ce cas, remplacer
 * `vehicleIsHidden` par `vehicleIsDecor` dans le hook ne faisait tomber aucun test.
 */
function decor(slot: number, family: string): ReplayVehicleTrack {
  return {
    slot, gen: 1, family, chassis: '0x1', end: 'film_end', t0: 0, t1: 6457, t1max: 6457,
    spawn: { x: 1, y: -130, z: 81 },
    samples: [{ t: 0, x: 1, y: -130 }],
    rides: [],
  } as unknown as ReplayVehicleTrack
}

function disponible(vehicles: ReplayVehicleTrack[], decorSlots: number[] = []): boolean {
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve([]) })))
  const hidden = decorSlots.map((slot) => ({ slot, gen: 1, reason: 'off_play_area' }))
  const vehicleScenery = {
    zone: 'map', floor: 'played', candidates: hidden.length, inPlayArea: 0, zoneUnknown: 0, hidden,
  }
  const doc = testReplayDoc({ vehicles, vehicleScenery })
  const vue = renderHook(() =>
    useReplayVehicles({
      doc, view: VUE, frameRef: { current: 0 }, enabled: true, locale: 'fr', showNames: false, showAim: false,
      colorOfSlot: () => '#123456', colorOfXuid: () => '#123456',
      nameOfSlot: () => null, nameOfXuid: () => null,
      offscreenLabelOf: () => '', offscreenGroupLabelOf: () => '',
      neutralInk: 'n', labelStroke: 's', markInk: { fill: 'm', outline: 'o' }, explosionInk: ENCRE_FX, reducedMotion: true,
      redraw: vi.fn(),
    }),
  )
  return vue.result.current.available
}

describe('useReplayVehicles — le décor de carte ne rend pas le calque disponible', () => {
  it('un document qui ne porte que du décor de carte (Starboard) : calque indisponible', () => {
    expect(disponible([decor(771, 'scorpion'), decor(772, 'wasp'), decor(774, 'warthog')], [771, 772, 774])).toBe(false)
  })

  it('témoin : le même décor plus un véhicule simulé — calque disponible', () => {
    expect(disponible([decor(771, 'scorpion'), piste('warthog')], [771])).toBe(true)
  })

  it('M7 : les mêmes véhicules posés SANS verdict du serveur (hors zone non établi) : calque disponible', () => {
    expect(disponible([decor(771, 'scorpion'), decor(772, 'wasp')])).toBe(true)
  })
})
