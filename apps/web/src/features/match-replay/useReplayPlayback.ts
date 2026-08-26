/**
 * useReplayPlayback — LA LECTURE du rejeu : l'état lu / en pause, la boucle d'animation, le
 * curseur de la frise, et ce qui se passe quand le film ARRIVE AU BOUT.
 *
 * EXTRAIT DE `ReplayCanvas.tsx` LE 2026-08-25 (huitième extraction imposée par le cliquet de
 * taille, `placementFamily.guard.test.ts`) : le lot y ajoutait la politique de fin de rejeu, et
 * la règle du dépôt est d'extraire plutôt que de relever le plafond. La découpe tombe sur une
 * frontière nette — ce fichier porte le TEMPS de la lecture, le canvas porte le DESSIN. Il ne
 * dessine rien lui-même : il appelle le tracé que l'appelant lui passe.
 *
 * # LE REJEU S'ARRÊTE SUR SON ÉTAT FINAL
 *
 * Demande utilisateur du 2026-08-25 : « quand le rejeu arrive au bout, rester sur l'état
 * final ». La boucle REBOUCLAIT à zéro (`if (next >= doc.frameCount - 1) next = 0`) — la
 * dernière image du match n'était donc jamais celle qu'on avait sous les yeux : elle était
 * remplacée, dans la même image d'animation, par la toute première. Un match se terminait
 * visuellement sur son coup d'envoi.
 *
 * Elle s'arrête maintenant SUR la dernière image : le curseur y reste, la scène finale reste
 * peinte, et la lecture passe en pause. Trois choses tiennent ensemble et aucune n'est
 * décorative — le curseur à `endFrame` (sans quoi la frise dirait un autre instant que le
 * terrain), le dernier `draw()` AVANT l'arrêt (sans quoi la scène resterait à l'avant-dernière
 * image), et le `soundTick` de cette même image (le curseur du son suit toujours la lecture,
 * cf. useReplaySound — le laisser en arrière ferait repartir un son enjambé au prochain clic).
 *
 * RELANCER RESTE À UN CLIC, et c'est la convention des lecteurs vidéo : « Lecture » sur un
 * rejeu terminé repart de zéro plutôt que de rester bloqué sur la dernière image, où la boucle
 * se rendormirait aussitôt. « Recommencer » ne change pas — il ramène au début à tout instant.
 */
import { useEffect, useRef, useState, type ChangeEvent, type RefObject } from 'react'

import { frameToMs } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/** Ce dont la lecture a besoin (objet unique : la règle des 5 paramètres du dépôt). */
export interface ReplayPlaybackOptions {
  doc: ReplayDocumentReady
  /** Cadence « 1× » du document, en images par seconde (cf. useReplayTiming). */
  baseFps: number
  /** Multiplicateur de vitesse choisi dans la barre de lecture. */
  speed: number
  /** Largeur de dessin : 0 = le canvas n'est pas encore mesuré, rien ne tourne. */
  renderWidth: number
  /** L'image courante, PARTAGÉE avec le dessin et les calques survolables. */
  frameRef: RefObject<number>
  /** Le tracé d'une image, tel que le canvas le publie. */
  draw: () => void
  /** Le battement du son : l'instant courant du rejeu, en ms (cf. useReplaySound.tick). */
  soundTick: (ms: number) => void
}

/** Ce que la barre de lecture reçoit — l'état à afficher et les commandes. */
export interface ReplayPlayback {
  playing: boolean
  /**
   * LA DERNIÈRE IMAGE du document : la borne de la frise ET l'instant où la lecture s'arrête.
   * Une seule valeur pour les deux, sinon le curseur pourrait buter avant (ou après) l'arrêt.
   */
  endFrame: number
  /** Le curseur de la frise : piloté par la boucle, jamais contrôlé par React. */
  sliderRef: RefObject<HTMLInputElement | null>
  togglePlay: () => void
  restart: () => void
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
}

export function useReplayPlayback(o: ReplayPlaybackOptions): ReplayPlayback {
  const { doc, baseFps, speed, renderWidth, frameRef, draw, soundTick } = o
  const sliderRef = useRef<HTMLInputElement>(null)
  const [playing, setPlaying] = useState(true)
  const endFrame = Math.max(doc.frameCount - 1, 0)

  // Boucle de lecture (requestAnimationFrame) uniquement quand `playing`.
  //
  // LE SON BAT AVEC ELLE, et nulle part ailleurs : hors lecture (pause, onglet en
  // arrière-plan, redessin au changement de thème) il n'y a pas de battement, donc pas un
  // son. C'est ce qui rend le silence d'un lecteur à l'arrêt structurel, pas conditionnel.
  useEffect(() => {
    if (!playing || renderWidth === 0) return
    const fps = baseFps * speed
    let raf = 0
    let last = 0
    const step = (ts: number) => {
      if (last === 0) last = ts
      const dtSec = (ts - last) / 1000
      last = ts
      let next = frameRef.current + dtSec * fps
      // LA FIN DU FILM : on borne à la dernière image, on la PEINT, puis on s'arrête (cf.
      // l'en-tête). L'ordre compte — sortir avant le tracé laisserait la scène une image en
      // arrière, et sortir avant le curseur laisserait la frise mentir.
      const ended = next >= endFrame
      if (ended) next = endFrame
      frameRef.current = next
      if (sliderRef.current) sliderRef.current.value = String(Math.round(next))
      soundTick(frameToMs(next, doc))
      draw()
      if (ended) {
        setPlaying(false)
        return
      }
      raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [playing, baseFps, speed, doc, renderWidth, draw, soundTick, endFrame, frameRef])

  const onScrub = (e: ChangeEvent<HTMLInputElement>) => {
    frameRef.current = Number(e.currentTarget.value)
    if (!playing) draw()
  }

  const restart = () => {
    frameRef.current = 0
    if (sliderRef.current) sliderRef.current.value = '0'
    setPlaying(true)
  }

  const togglePlay = () => {
    // REPARTIR DU DÉBUT SUR UN REJEU TERMINÉ (cf. l'en-tête) : sans ce rembobinage, la boucle
    // relancée à `endFrame` conclurait « fin » à son premier pas et se rendormirait — le bouton
    // « Lecture » n'aurait aucun effet visible.
    if (!playing && frameRef.current >= endFrame) {
      frameRef.current = 0
      if (sliderRef.current) sliderRef.current.value = '0'
    }
    setPlaying((p) => !p)
  }

  return { playing, endFrame, sliderRef, togglePlay, restart, onScrub }
}
