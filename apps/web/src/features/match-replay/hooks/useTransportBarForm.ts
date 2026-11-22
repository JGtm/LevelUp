/**
 * useTransportBarForm — CE QUE L'UTILISATEUR FAIT DE LA BARRE DE LECTURE, et la forme qui en découle
 * (`barForm`, règle pure). Le hook écoute et mesure rien d'autre : survol de la barre, focus du CLAVIER dans la
 * barre, touches de la page, et rend la forme avec les gestionnaires à poser sur la barre.
 *
 *  - LE SURVOL est celui de la barre et de ses descendants (un menu ouvert au-dessus d'elle en fait partie) ;
 *  - LE FOCUS compte seulement s'il est un focus clavier (`:focus-visible`) : un clic de souris qui laisse le focus
 *    sur un bouton ne doit pas garder la barre pleine à jamais ;
 *  - UN GESTE RÉCENT vaut `fineRetenuMs` : toute touche de la page (Espace, flèches, raccourcis) et la sortie du
 *    pointeur ouvrent ce délai, pendant lequel la barre reste pleine. L'écoute des touches n'existe que
 *    tant que la barre est en surimpression.
 */
import { useCallback, useEffect, useRef, useState, type FocusEvent } from 'react'

import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import { barForm, overlaysTransportBar, type BarForm } from '../model/transportBarLogic'
import type { ReplayFrame } from '../model/replayFrame'

/** Les gestionnaires à poser sur la racine de la barre. */
export interface TransportBarHandlers {
  onPointerEnter: () => void
  onPointerLeave: () => void
  onFocus: (e: FocusEvent<HTMLElement>) => void
  onBlur: (e: FocusEvent<HTMLElement>) => void
}

/** Cet élément a-t-il le focus du clavier ? Un moteur qui ne connaît pas `:focus-visible` répond non. */
function hasKeyboardFocus(el: Element): boolean {
  try {
    return el.matches(':focus-visible')
  } catch {
    return false
  }
}

export function useTransportBarForm(input: {
  frame: ReplayFrame
  thinPreferred: boolean
  playing: boolean
}): { form: BarForm; handlers: TransportBarHandlers } {
  const { frame, thinPreferred, playing } = input
  const overlay = overlaysTransportBar(frame, thinPreferred)
  const [hovered, setHovered] = useState(false)
  const [focused, setFocused] = useState(false)
  const [recentGesture, setRecentGesture] = useState(false)
  const timer = useRef<number | undefined>(undefined)

  const touch = useCallback(() => {
    setRecentGesture(true)
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => setRecentGesture(false), C.fineRetenuMs)
  }, [])

  useEffect(() => {
    if (!overlay) return
    window.addEventListener('keydown', touch)
    return () => window.removeEventListener('keydown', touch)
  }, [overlay, touch])
  useEffect(() => () => window.clearTimeout(timer.current), [])

  const handlers: TransportBarHandlers = {
    onPointerEnter: () => setHovered(true),
    onPointerLeave: () => {
      setHovered(false)
      touch()
    },
    onFocus: (e) => {
      if (hasKeyboardFocus(e.target)) setFocused(true)
    },
    onBlur: (e) => {
      if (!e.currentTarget.contains(e.relatedTarget)) setFocused(false)
    },
  }
  return { form: barForm({ frame, thinPreferred, playing, hovered, focused, recentGesture }), handlers }
}
