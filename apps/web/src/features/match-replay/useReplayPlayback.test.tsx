/**
 * Tests — useReplayPlayback (la lecture du rejeu et sa FIN).
 *
 * CE QU'ILS PROTÈGENT, et c'est la demande utilisateur du 2026-08-25 : arrivé au bout, le rejeu
 * RESTE sur son état final — curseur à la dernière image, dernière scène peinte, lecture en
 * pause. La boucle rebouclait à zéro : le match se terminait visuellement sur son coup d'envoi,
 * et le test « ne repart pas à zéro » échoue si l'on y revient.
 *
 * LA BOUCLE D'ANIMATION EST PILOTÉE À LA MAIN : `requestAnimationFrame` est remplacé par une
 * file qu'on vide pas à pas. Sans cela un test de fin de film dépendrait de la cadence réelle
 * du navigateur de test — c'est-à-dire de rien de reproductible.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type RefObject } from 'react'

import { testReplayDoc } from './test/testDoc'
import { useReplayPlayback } from './useReplayPlayback'

/** Un document de 51 images (`endFrame` = 50) à la cadence par défaut. */
const DOC = testReplayDoc({ frameCount: 51 })

/** La file des rappels d'animation en attente, et le pas de temps qu'on leur sert. */
let pending: FrameRequestCallback[] = []

beforeEach(() => {
  pending = []
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    pending.push(cb)
    return pending.length
  })
  vi.stubGlobal('cancelAnimationFrame', () => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/**
 * Fait avancer la boucle d'un pas : `ts` est l'horodatage servi au rappel.
 *
 * LE PREMIER PAS N'AVANCE JAMAIS, et les horodatages ci-dessous en tiennent compte : la boucle
 * amorce son horloge dessus (`last`) et calcule donc un écart nul. C'est le comportement de la
 * boucle réelle, pas un artifice de test — un rejeu démarre à l'image où il était.
 * Les horodatages servis sont NON NULS : `last` vaut zéro tant qu'il n'est pas amorcé, et
 * servir `ts = 0` ferait ré-amorcer l'horloge à chaque pas.
 */
function tick(ts: number) {
  const next = pending.shift()
  if (!next) throw new Error('aucune image demandée — la boucle ne tourne pas')
  act(() => {
    next(ts)
  })
}

function mount(frameRef: RefObject<number>, draw = vi.fn(), soundTick = vi.fn()) {
  const view = renderHook(() =>
    useReplayPlayback({
      doc: DOC,
      baseFps: 10,
      speed: 1,
      renderWidth: 480,
      frameRef,
      draw,
      soundTick,
    }),
  )
  return { ...view, draw, soundTick }
}

describe('useReplayPlayback — la lecture avance', () => {
  it('la boucle fait courir l’image et peint à chaque pas', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result, draw } = mount(frameRef)
    expect(result.current.playing).toBe(true)
    tick(1_000) // amorce l'horloge : écart nul
    tick(2_000) // 1 s à 10 images/s = 10 images
    expect(frameRef.current).toBeCloseTo(10, 5)
    expect(draw).toHaveBeenCalledTimes(2)
  })

  it('`endFrame` est la DERNIÈRE image du document', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result } = mount(frameRef)
    expect(result.current.endFrame).toBe(50)
  })
})

describe('useReplayPlayback — la fin du rejeu reste sur l’état final', () => {
  it('borne à la dernière image, la PEINT, puis met en pause — sans repartir à zéro', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 49
    const { result, draw, soundTick } = mount(frameRef)
    tick(1_000)
    draw.mockClear()
    soundTick.mockClear()
    tick(11_000) // très au-delà de la fin : 100 images demandées, 1 disponible
    // L'IMAGE FINALE, pas zéro — c'est le défaut que ce lot corrige.
    expect(frameRef.current).toBe(50)
    // La scène finale est peinte AVANT l'arrêt : sortir plus tôt la laisserait une image
    // en arrière.
    expect(draw).toHaveBeenCalledTimes(1)
    // Le curseur du son suit jusqu'au bout : sinon un son enjambé repartirait au clic suivant.
    expect(soundTick).toHaveBeenCalledTimes(1)
    expect(result.current.playing).toBe(false)
    // Et plus rien n'est demandé : la boucle s'est arrêtée, elle ne rejoue pas le film.
    expect(pending).toHaveLength(0)
  })

  it('« Lecture » sur un rejeu terminé repart du DÉBUT', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 50
    const { result } = mount(frameRef)
    // La boucle conclut « fin » dès son premier pas et se met en pause.
    tick(1_000)
    expect(result.current.playing).toBe(false)
    act(() => {
      result.current.togglePlay()
    })
    expect(frameRef.current).toBe(0)
    expect(result.current.playing).toBe(true)
  })

  it('« Lecture »/« Pause » en cours de film ne rembobine RIEN', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 20
    const { result } = mount(frameRef)
    act(() => {
      result.current.togglePlay()
    })
    expect(result.current.playing).toBe(false)
    expect(frameRef.current).toBe(20)
    act(() => {
      result.current.togglePlay()
    })
    expect(result.current.playing).toBe(true)
    expect(frameRef.current).toBe(20)
  })

  it('« Recommencer » ramène au début et relance, à tout instant', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 33
    const { result } = mount(frameRef)
    act(() => {
      result.current.restart()
    })
    expect(frameRef.current).toBe(0)
    expect(result.current.playing).toBe(true)
  })
})

describe('useReplayPlayback — la frise', () => {
  it('déplacer le curseur pose l’image ; en pause, elle est repeinte tout de suite', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result, draw } = mount(frameRef)
    act(() => {
      result.current.togglePlay() // pause
    })
    draw.mockClear()
    act(() => {
      result.current.onScrub({
        currentTarget: { value: '42' },
      } as unknown as React.ChangeEvent<HTMLInputElement>)
    })
    expect(frameRef.current).toBe(42)
    expect(draw).toHaveBeenCalledTimes(1)
  })
})
