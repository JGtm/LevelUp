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
 *
 * ET DEPUIS LE 2026-08-26, « le début » et « la fin » sont ceux du MATCH : la dernière série
 * vérifie que la fenêtre de gameplay (`replayWindow.ts`) déplace les deux bornes — départ,
 * arrêt, « Recommencer » et rembobinage — sans rien changer quand elle vaut `null`.
 *
 * DEPUIS LE 2026-09-02, « le début de la LECTURE » et « le début du MATCH » ne sont plus le même
 * point : la lecture se pose une seconde plus tôt (`leadInFrame`, décision D3), la frise et
 * l'horloge restent au coup d'envoi. Les cas ci-dessous distinguent donc les deux — `startFrame`
 * publié d'un côté, `frameRef` de l'autre. Un test qui les confondrait ne verrait pas la
 * régression que ce lot craint : un préambule qui contaminerait le cadrage.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type RefObject } from 'react'

import type { ReplayWindowBounds } from '../model/replayWindow'
import { testReplayDoc } from '../test/testDoc'
import { useReplayPlayback } from './useReplayPlayback'
import { AUTOPLAY_KEY } from '../settings/useReplaySettings'

/** Un document de 51 images (`endFrame` = 50) à la cadence par défaut. */
const DOC = testReplayDoc({ frameCount: 51 })

/** La file des rappels d'animation en attente, et le pas de temps qu'on leur sert. */
let pending: FrameRequestCallback[] = []

beforeEach(() => {
  // LA LECTURE AUTOMATIQUE EST UNE PRÉFÉRENCE PERSISTÉE depuis le 2026-08-29, et elle est
  // ÉTEINTE par défaut. Tout ce fichier — sauf la série qui teste le réglage lui-même — éprouve
  // la BOUCLE en marche : il l'allume donc explicitement, plutôt que de faire dépendre trente
  // tests d'un défaut de produit qui peut rebasculer.
  localStorage.clear()
  localStorage.setItem(AUTOPLAY_KEY, 'true')
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

function mount(
  frameRef: RefObject<number>,
  playWindow: ReplayWindowBounds | null = null,
  draw = vi.fn(),
  soundTick = vi.fn(),
  onEnded = vi.fn(),
  onTransportGesture = vi.fn(),
) {
  const view = renderHook(() =>
    useReplayPlayback({
      doc: DOC,
      playWindow,
      baseFps: 10,
      speed: 1,
      renderWidth: 480,
      frameRef,
      draw,
      soundTick,
      onEnded,
      onTransportGesture,
    }),
  )
  return { ...view, draw, soundTick, onEnded, onTransportGesture }
}

/**
 * Une fenêtre de gameplay dans le document de test : le match court de l'image 10 à la 40.
 *
 * LA CADENCE IMPLICITE DE CETTE FIXTURE EST DE 1 000 ms PAR IMAGE (10 000 ms pour l'image 10,
 * 40 000 pour la 40) : la seconde de préambule (`LEAD_IN_MS`) vaut donc UNE image, et
 * `leadInFrame` tombe sur la 9. C'est là que la lecture se pose ; la frise, elle, part
 * toujours de la 10 (D3, 2026-09-02).
 */
const FENETRE: ReplayWindowBounds = {
  startFrame: 10,
  leadInFrame: 9,
  endFrame: 40,
  startMs: 10_000,
  endMs: 40_000,
}

describe('useReplayPlayback — la lecture automatique (point 22 du 2026-08-29)', () => {
  it('sans préférence stockée : le rejeu s ouvre EN PAUSE (défaut du 2026-08-29)', () => {
    localStorage.removeItem(AUTOPLAY_KEY)
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result } = mount(frameRef)
    expect(result.current.playing).toBe(false)
    expect(pending).toHaveLength(0)
  })

  it('préférence allumée : le rejeu démarre tout seul', () => {
    localStorage.setItem(AUTOPLAY_KEY, 'true')
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    expect(mount(frameRef).result.current.playing).toBe(true)
  })

  it('préférence éteinte : le rejeu s ouvre EN PAUSE, et aucune image n est demandée', () => {
    localStorage.setItem(AUTOPLAY_KEY, 'false')
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result } = mount(frameRef)
    expect(result.current.playing).toBe(false)
    // La boucle ne tourne pas du tout : pas de rappel d'animation en attente. C'est ce qui
    // distingue « en pause » de « en lecture sur une image qui ne bouge pas ».
    expect(pending).toHaveLength(0)
  })

  it('en pause, le curseur se pose quand même AU COUP D ENVOI (moins son préambule), et la scène est peinte', () => {
    // SANS CE POSITIONNEMENT, un rejeu ouvert en pause resterait sur l'image zéro du FILM —
    // c'est-à-dire sur le countdown d'avant-match, joueurs figés. Le cadrage vaut lecture ou
    // pas ; seule la boucle dépend de la préférence.
    localStorage.setItem(AUTOPLAY_KEY, 'false')
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { draw } = mount(frameRef, FENETRE)
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    expect(draw).toHaveBeenCalled()
    expect(pending).toHaveLength(0)
  })

  it('en pause à l ouverture, « Lecture » démarre normalement', () => {
    localStorage.setItem(AUTOPLAY_KEY, 'false')
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result } = mount(frameRef)
    act(() => result.current.togglePlay())
    expect(result.current.playing).toBe(true)
  })
})

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

describe('useReplayPlayback — la fenêtre de gameplay borne la lecture', () => {
  it('expose les DEUX bornes du match, pas celles du film — et le préambule NE les déplace pas', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result } = mount(frameRef, FENETRE)
    // LA BORNE PUBLIÉE RESTE LE COUP D'ENVOI (D3) : c'est elle que la frise prend pour minimum
    // (`useReplayTimeline.minFrame`). La lecture, elle, se pose une image plus tôt.
    expect(result.current.startFrame).toBe(10)
    expect(result.current.endFrame).toBe(40)
  })

  it('la PREMIÈRE lecture démarre au PRÉAMBULE (une seconde avant le coup d’envoi), et la scène y est peinte', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { draw } = mount(frameRef, FENETRE)
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    expect(draw).toHaveBeenCalled()
  })

  it('ne RECULE jamais la lecture déjà engagée au-delà du coup d’envoi', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 25
    mount(frameRef, FENETRE)
    expect(frameRef.current).toBe(25)
  })

  it('s’arrête à la FIN DÉCLARÉE, pas à la dernière image du film', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 39
    const { result } = mount(frameRef, FENETRE)
    tick(1_000)
    tick(11_000) // 100 images demandées : très au-delà des deux bornes
    expect(frameRef.current).toBe(40)
    expect(result.current.playing).toBe(false)
    expect(pending).toHaveLength(0)
  })

  it('« Recommencer » ramène au préambule du coup d’envoi, pas à l’image zéro du film', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 33
    const { result } = mount(frameRef, FENETRE)
    act(() => {
      result.current.restart()
    })
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    expect(result.current.playing).toBe(true)
  })

  it('« Lecture » sur un rejeu terminé repart du préambule du coup d’envoi', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 40
    const { result } = mount(frameRef, FENETRE)
    tick(1_000)
    expect(result.current.playing).toBe(false)
    act(() => {
      result.current.togglePlay()
    })
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    expect(result.current.playing).toBe(true)
  })

  it('SANS fenêtre, les bornes restent celles du film — rien ne change', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { result } = mount(frameRef)
    expect(result.current.startFrame).toBe(0)
    expect(result.current.endFrame).toBe(50)
    act(() => {
      result.current.restart()
    })
    expect(frameRef.current).toBe(0)
  })
})

/**
 * L'ARRIVÉE EN FIN DE MATCH (lot C, 2026-08-27) — c'est d'ici que part le son de fin de partie.
 *
 * CE QUE CES CAS TIENNENT, et ce sont deux erreurs qui ne se voient pas à la relecture : une
 * annonce qui partirait DEUX fois (fanfare doublée), et une annonce qui partirait sur une frise
 * tirée jusqu'au bout alors que la décision D-C1 la réserve à la LECTURE. La distinction est
 * dans un seul mot du code — l'image d'AVANT le pas était-elle en deçà de la borne.
 */
describe('useReplayPlayback — l’arrivée en fin de match s’annonce', () => {
  it('la lecture qui FRANCHIT la borne annonce, une fois et une seule', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 39
    const { onEnded } = mount(frameRef, FENETRE)
    tick(1_000) // amorce : rien n'a encore été parcouru
    expect(onEnded).not.toHaveBeenCalled()
    tick(11_000) // la lecture passe 39 -> 40
    expect(onEnded).toHaveBeenCalledTimes(1)
    // La boucle s'est arrêtée : plus une image demandée, donc plus une annonce possible.
    expect(pending).toHaveLength(0)
  })

  it('une frise tirée JUSQU’AU BOUT n’annonce rien (D-C1)', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 40 // le curseur est DÉJÀ sur la borne : le pas ne parcourt rien
    const { result, onEnded } = mount(frameRef, FENETRE)
    tick(1_000)
    expect(result.current.playing).toBe(false)
    expect(onEnded).not.toHaveBeenCalled()
  })

  it('« Recommencer » réarme : la prochaine arrivée s’annonce de nouveau', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 39
    const { result, onEnded } = mount(frameRef, FENETRE)
    tick(1_000)
    tick(11_000)
    expect(onEnded).toHaveBeenCalledTimes(1)
    act(() => {
      result.current.restart()
    })
    tick(20_000) // amorce de la nouvelle lecture, au coup d'envoi
    tick(24_000) // 4 s à 10 images/s : la borne est de nouveau franchie
    expect(frameRef.current).toBe(40)
    expect(onEnded).toHaveBeenCalledTimes(2)
  })

  it('sans fenêtre, c’est la fin du FILM qui s’annonce — même règle', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 49
    const { onEnded } = mount(frameRef)
    tick(1_000)
    tick(11_000)
    expect(frameRef.current).toBe(50)
    expect(onEnded).toHaveBeenCalledTimes(1)
  })
})

/**
 * LES GESTES DE TRANSPORT PRÉVIENNENT LE SON (correctif du 2026-08-27) — c'est par là qu'un
 * rejeu rechargé, dont la préférence était restée à « activé », retrouve son lecteur : un
 * AudioContext ne naît que dans un geste utilisateur, et ces deux boutons en sont.
 */
describe('useReplayPlayback — les commandes de transport préviennent le son', () => {
  it('« Lecture »/« Pause » est un geste : le son en est averti', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 20
    const { result, onTransportGesture } = mount(frameRef)
    act(() => {
      result.current.togglePlay()
    })
    expect(onTransportGesture).toHaveBeenCalledTimes(1)
  })

  it('« Recommencer » aussi, et le rembobinage se fait quand même', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 33
    const { result, onTransportGesture } = mount(frameRef, FENETRE)
    act(() => {
      result.current.restart()
    })
    expect(onTransportGesture).toHaveBeenCalledTimes(1)
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    expect(result.current.playing).toBe(true)
  })

  it('la boucle d’animation, elle, ne prévient personne : ce n’est pas un geste', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const { onTransportGesture } = mount(frameRef)
    tick(1_000)
    tick(2_000)
    expect(onTransportGesture).not.toHaveBeenCalled()
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

/**
 * LES SAUTS (planche 2a, 2026-08-28) — `seekBy` en SECONDES, `stepFrames` en images.
 *
 * CE QUE CES CAS TIENNENT : le bornage à la fenêtre de gameplay (un saut de 10 s près d'un
 * bout ne doit pas sortir du match), la conversion secondes -> images par `baseFps` (10 ici),
 * et la mise en PAUSE du pas d'image — sans elle, la boucle écraserait le pas au rendu suivant
 * et le bouton n'aurait aucun effet visible.
 */
describe('useReplayPlayback — les sauts', () => {
  it('`seekBy` convertit les secondes en images par la cadence du document', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 20
    const { result, draw, soundTick } = mount(frameRef, FENETRE)
    draw.mockClear()
    soundTick.mockClear()
    act(() => {
      result.current.seekBy(1) // 1 s à 10 images/s
    })
    expect(frameRef.current).toBe(30)
    // Un saut REPEINT et fait battre le son : sinon la scène montrerait l'instant d'avant.
    expect(draw).toHaveBeenCalledTimes(1)
    expect(soundTick).toHaveBeenCalledTimes(1)
  })

  it('`seekBy` est borné aux DEUX bouts : le préambule en bas, la fin déclarée en haut', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 20
    const { result } = mount(frameRef, FENETRE)
    act(() => {
      result.current.seekBy(-10) // 100 images en arrière : bien avant le coup d'envoi
    })
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    act(() => {
      result.current.seekBy(10) // 100 images en avant : bien après la fin déclarée
    })
    expect(frameRef.current).toBe(40)
  })

  it('`stepFrames` met la lecture EN PAUSE et avance d’une image', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 20
    const { result } = mount(frameRef, FENETRE)
    expect(result.current.playing).toBe(true)
    act(() => {
      result.current.stepFrames(1)
    })
    expect(frameRef.current).toBe(21)
    expect(result.current.playing).toBe(false)
  })

  it('`stepFrames` ne sort pas de la fenêtre, à l’une ou l’autre borne', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 40
    const { result } = mount(frameRef, FENETRE)
    act(() => {
      result.current.stepFrames(1)
    })
    expect(frameRef.current).toBe(40)
    act(() => {
      result.current.stepFrames(-40)
    })
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
  })
})

/**
 * LE REMPLISSAGE DE LA FRISE (`--played`) — la part parcourue, écrite par `writeCursor`.
 *
 * POURQUOI UN TEST PAR CHEMIN : la variable ne se met à jour pour personne toute seule. Chaque
 * chemin qui déplace le curseur doit passer par `writeCursor`, et c'est exactement ce qu'un
 * oubli ferait perdre — une frise remplie jusqu'à la position d'avant le geste.
 */
describe('useReplayPlayback — le remplissage de la frise suit le curseur', () => {
  /** Le champ n'existe pas dans un test de hook : on l'attache à la ref rendue par le hook. */
  function attachSlider(ref: RefObject<HTMLInputElement | null>): HTMLInputElement {
    const el = document.createElement('input')
    el.type = 'range'
    ref.current = el
    return el
  }

  function played(el: HTMLInputElement): string {
    return el.style.getPropertyValue('--played')
  }

  it('un saut écrit la part parcourue de la FENÊTRE, pas du film', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 10
    const { result } = mount(frameRef, FENETRE)
    const el = attachSlider(result.current.sliderRef)
    act(() => {
      result.current.seekBy(1.5) // 15 images : la moitié des 30 de la fenêtre
    })
    expect(el.value).toBe('25')
    expect(played(el)).toBe('50%')
  })

  it('« Recommencer » ramène le remplissage à zéro', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 25
    const { result } = mount(frameRef, FENETRE)
    const el = attachSlider(result.current.sliderRef)
    act(() => {
      result.current.restart()
    })
    expect(el.value).toBe('9')
    // Le préambule est EN DEÇÀ de la frise : la part parcourue y serait négative, le bornage
    // de `writeCursor` la ramène à zéro. C'est ce qui rend la seconde de préambule invisible
    // sur la barre plutôt que fautive.
    expect(played(el)).toBe('0%')
  })

  it('un glissé manuel le met à jour lui aussi', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 10
    const { result } = mount(frameRef, FENETRE)
    const el = attachSlider(result.current.sliderRef)
    act(() => {
      result.current.onScrub({
        currentTarget: { value: '40' },
      } as unknown as React.ChangeEvent<HTMLInputElement>)
    })
    expect(played(el)).toBe('100%')
  })

  /**
   * LA POSE INITIALE — le cas que les autres ne couvrent pas, et c'est le scénario RÉEL : la
   * fenêtre de gameplay vient de la Match View, qui arrive APRÈS le document du rejeu. Le champ
   * existe donc déjà quand elle se connaît, et c'est l'effet de pose (et non un geste) qui doit
   * écrire les deux choses. Sans son appel à `writeCursor`, la frise s'afficherait creuse
   * jusqu'au premier pas de la boucle.
   */
  it('la fenêtre qui ARRIVE pose le curseur ET son remplissage', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 0
    const draw = vi.fn()
    const view = renderHook(
      ({ playWindow }: { playWindow: ReplayWindowBounds | null }) =>
        useReplayPlayback({
          doc: DOC,
          playWindow,
          baseFps: 10,
          speed: 1,
          renderWidth: 480,
          frameRef,
          draw,
          soundTick: vi.fn(),
          onEnded: vi.fn(),
          onTransportGesture: vi.fn(),
        }),
      { initialProps: { playWindow: null as ReplayWindowBounds | null } },
    )
    const el = attachSlider(view.result.current.sliderRef)
    view.rerender({ playWindow: FENETRE })
    // Le curseur est au PRÉAMBULE du coup d'envoi, et le remplissage part de zéro AVEC lui.
    expect(frameRef.current).toBe(FENETRE.leadInFrame)
    expect(el.value).toBe('9')
    expect(played(el)).toBe('0%')
  })

  it('une lecture DÉJÀ engagée garde sa position, et son remplissage est posé quand même', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 25 // au-delà du coup d'envoi : le repositionnement ne recule jamais
    const view = renderHook(
      ({ playWindow }: { playWindow: ReplayWindowBounds | null }) =>
        useReplayPlayback({
          doc: DOC,
          playWindow,
          baseFps: 10,
          speed: 1,
          renderWidth: 480,
          frameRef,
          draw: vi.fn(),
          soundTick: vi.fn(),
          onEnded: vi.fn(),
          onTransportGesture: vi.fn(),
        }),
      { initialProps: { playWindow: null as ReplayWindowBounds | null } },
    )
    const el = attachSlider(view.result.current.sliderRef)
    view.rerender({ playWindow: FENETRE })
    expect(frameRef.current).toBe(25)
    expect(played(el)).toBe('50%')
  })

  it('la boucle de lecture l’écrit à chaque pas', () => {
    const frameRef = createRef<number>() as RefObject<number>
    frameRef.current = 10
    const { result } = mount(frameRef, FENETRE)
    const el = attachSlider(result.current.sliderRef)
    tick(1_000) // amorce
    tick(3_000) // 2 s à 10 images/s : 20 images, soit les deux tiers de la fenêtre
    expect(el.value).toBe('30')
    expect(played(el)).toBe(`${((30 - 10) / 30) * 100}%`)
  })
})
