/**
 * Tests — LES VARIABLES DE POSITION DU CURSEUR, et OÙ elles se posent (2026-09-06).
 *
 * UN FICHIER À PART, ET PAS UNE SÉRIE DE PLUS dans `useReplayPlayback.test.tsx` : celui-ci
 * éprouve la MÉCANIQUE DE LECTURE (la boucle, les bornes, les sauts, la fin du film) et pilote
 * `requestAnimationFrame` à la main pour cela. Ce qui suit n'éprouve pas la lecture mais son
 * EFFET DE BORD DANS LE DOM — quel élément reçoit `--played` et `--played-r`, et avec quelle
 * valeur. Deux responsabilités, deux fichiers ; l'autre frôlait par ailleurs le plafond de
 * taille du dépôt (`max-lines`), et la règle est d'extraire plutôt que de relever le seuil.
 *
 * # CE QUE CES CAS TIENNENT, ET COMMENT ILS ÉCHOUERAIENT
 *
 *  1. LE RATIO EST BORNÉ À [0, 1]. Il sert de MULTIPLICATEUR dans une géométrie de piste
 *     (`trackLeftVar` : `calc(8px + (100% - 16px) * var(--played-r))`). Un ratio négatif — et le
 *     préambule d'avant coup d'envoi en produit un à chaque ouverture — poserait le trait de
 *     lecture À GAUCHE de la frise, hors de son conteneur ; au-delà de 1, à droite. Le bornage
 *     n'est pas une politesse d'affichage, c'est ce qui garde l'objet dans sa boîte.
 *  2. LES DEUX VARIABLES SE POSENT SUR LA RACINE DE LA FRISE, jamais sur la rangée du champ.
 *     Les pistes vivent AU-DESSUS du curseur : posées un cran trop bas, elles ne les verraient
 *     pas (une propriété personnalisée n'hérite que vers le BAS) et le trait de lecture
 *     resterait figé à l'origine pendant que le curseur avance — sans une erreur, sans un log.
 *  3. LE REPLI HORS FRISE reste fonctionnel : un champ sans racine porteuse (champ détaché,
 *     refonte future qui déplacerait l'attribut) pose sur son parent plutôt que nulle part.
 *
 * LA LECTURE AUTOMATIQUE EST ÉTEINTE ICI : aucun de ces cas n'a besoin de la boucle, et un
 * rejeu qui démarre tout seul demanderait des images d'animation dont ce fichier n'a que faire.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createRef, type RefObject } from 'react'

import type { ReplayWindowBounds } from '../model/replayWindow'
import { testReplayDoc } from '../test/testDoc'
import { AUTOPLAY_KEY } from '../settings/useReplaySettings'
import {
  CURSOR_HOST_ATTR,
  CURSOR_PLAYED_VAR,
  CURSOR_RATIO_VAR,
  useReplayPlayback,
} from './useReplayPlayback'

/** Un document de 51 images (`endFrame` = 50) à la cadence par défaut. */
const DOC = testReplayDoc({ frameCount: 51 })

/**
 * La même fenêtre de gameplay que la suite de la lecture : le match court de l'image 10 à la 40,
 * une image par seconde, préambule à la 9. Trente images de course, donc des ratios ronds.
 */
const FENETRE: ReplayWindowBounds = {
  startFrame: 10,
  leadInFrame: 9,
  endFrame: 40,
  startMs: 10_000,
  endMs: 40_000,
}

beforeEach(() => {
  localStorage.clear()
  localStorage.setItem(AUTOPLAY_KEY, 'false')
})

function mount(frame: number) {
  const frameRef = createRef<number>() as RefObject<number>
  frameRef.current = frame
  return renderHook(() =>
    useReplayPlayback({
      doc: DOC,
      playWindow: FENETRE,
      baseFps: 10,
      speed: 1,
      renderWidth: 480,
      frameRef,
      draw: vi.fn(),
      soundTick: vi.fn(),
      onEnded: vi.fn(),
      onTransportGesture: vi.fn(),
    }),
  )
}

/** Le champ DANS sa frise : une racine porteuse, une rangée intermédiaire, le champ au fond. */
function attachDansLaFrise(ref: RefObject<HTMLInputElement | null>) {
  const racine = document.createElement('div')
  racine.setAttribute(CURSOR_HOST_ATTR, '')
  const rangee = document.createElement('div')
  const el = document.createElement('input')
  el.type = 'range'
  rangee.appendChild(el)
  racine.appendChild(rangee)
  ref.current = el
  return { racine, rangee, el }
}

function ratio(el: HTMLElement): string {
  return el.style.getPropertyValue(CURSOR_RATIO_VAR)
}

describe('useReplayPlayback — la position du curseur se pose sur la RACINE de la frise', () => {
  it('écrit les deux variables sur la racine, et rien sur la rangée ni sur le champ', () => {
    const { result } = mount(10)
    const { racine, rangee, el } = attachDansLaFrise(result.current.sliderRef)
    act(() => {
      result.current.seekBy(1.5) // 15 images : la moitié des 30 de la fenêtre
    })
    expect(ratio(racine)).toBe('0.5')
    expect(racine.style.getPropertyValue(CURSOR_PLAYED_VAR)).toBe('50%')
    // La rangée intermédiaire ne reçoit rien : c'est bien la RACINE qui est visée, pas « le
    // parent ». Le champ, lui, les reçoit par héritage — ce que le navigateur assure, pas nous.
    expect(ratio(rangee)).toBe('')
    expect(ratio(el)).toBe('')
  })

  it('sans racine porteuse, la pose retombe sur le parent du champ', () => {
    const { result } = mount(10)
    const parent = document.createElement('div')
    const el = document.createElement('input')
    el.type = 'range'
    parent.appendChild(el)
    result.current.sliderRef.current = el
    act(() => {
      result.current.seekBy(1.5)
    })
    expect(ratio(parent)).toBe('0.5')
  })
})

describe('useReplayPlayback — le ratio nu reste dans [0, 1]', () => {
  it('le préambule d’avant coup d’envoi donne ZÉRO, jamais un ratio négatif', () => {
    const { result } = mount(25)
    const { racine } = attachDansLaFrise(result.current.sliderRef)
    act(() => {
      result.current.restart() // ramène à `leadInFrame`, EN DEÇÀ de la frise
    })
    expect(ratio(racine)).toBe('0')
  })

  it('la fin de la fenêtre donne UN, jamais au-delà', () => {
    const { result } = mount(10)
    const { racine } = attachDansLaFrise(result.current.sliderRef)
    act(() => {
      result.current.onScrub({
        currentTarget: { value: '40' },
      } as unknown as React.ChangeEvent<HTMLInputElement>)
    })
    expect(ratio(racine)).toBe('1')
  })

  it('entre les deux, c’est la part parcourue de la FENÊTRE — pas du film', () => {
    const { result } = mount(10)
    const { racine } = attachDansLaFrise(result.current.sliderRef)
    act(() => {
      result.current.seekBy(1) // 10 images sur les 30 de la fenêtre
    })
    expect(ratio(racine)).toBe(String(10 / 30))
  })
})
