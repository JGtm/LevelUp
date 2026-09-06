/**
 * useReplayClock — L'HORLOGE AFFICHÉE du rejeu, et la publication de l'image courante aux
 * panneaux qui vivent hors du canvas.
 *
 * LA PUBLICATION VA DANS LE MAGASIN DE LECTURE (`model/playbackStore`, 2026-09-06) et non
 * plus dans un état React de la route : la position n'a qu'une cellule, celle que la boucle
 * de dessin écrit. Ce fichier n'en décide que le RYTHME.
 *
 * DIXIÈME EXTRACTION IMPOSÉE PAR LE SEUIL DE TAILLE (`max-lines` eslint, R5) : le
 * canvas était PILE à son plafond et le lot du cadrage (2026-08-26) y ajoutait la fenêtre de
 * gameplay. Les deux morceaux sortis ici n'appartiennent pas au DESSIN — ils disent OÙ ON EN
 * EST, l'un à l'écran, l'autre au reste de la page. La découpe suit donc la même frontière que
 * `useReplayPlayback` : le canvas peint, les hooks portent le temps.
 *
 * # L'HORLOGE EST CELLE DU MATCH, PAS CELLE DU FILM (D-A2)
 *
 * Le film commence avant le coup d'envoi et s'arrête quelques secondes après la fin déclarée
 * (cf. `replayWindow.ts`). Affichée brute, son horloge démarrait donc à un instant qui ne veut
 * rien dire pour qui regarde le match — « 0:14 » au coup d'envoi — et le total annonçait une
 * durée que le match n'a pas eue. Les deux se recalent ici, et ICI SEULEMENT : l'axe interne
 * (images, `frameToMs`) ne bouge pas d'un pas, c'est ce qui garde le dessin, le son et les
 * calques sur un seul et même référentiel.
 *
 * # POURQUOI LA PUBLICATION EST BRIDÉE
 *
 * Le canvas se redessine à la cadence de l'écran ; les fiches joueur, elles, sont du DOM. Les
 * re-rendre 60 fois par seconde coûterait tout le budget d'animation pour un contenu qui change
 * à peine. 150 ms reste bien en deçà de ce que l'œil perçoit comme un retard sur un compteur,
 * et divise le travail de React par dix.
 *
 * # LA DERNIÈRE IMAGE PASSE LE BRIDAGE, ET C'EST UNE CORRECTION DE BUG (2026-08-28)
 *
 * Le bridage vaut pour un flux : la publication sautée est rattrapée 150 ms plus tard. À la
 * BORNE DE FIN il n'y a pas de « plus tard » — la boucle de lecture peint cette image puis
 * s'arrête (`useReplayPlayback`). Bridée, elle n'était jamais publiée, et l'ÉCRAN DE FIN de
 * match, qui se dérive de `frame >= endFrame`, ne se rendait pas : à 60 fps le dernier pas
 * tombe à ~16 ms du précédent, donc la neuf fois sur dix. Le son, lui, part par `onEnded` —
 * un chemin distinct : d'où un rejeu qui sonnait la victoire sans jamais l'écrire.
 */
import { useCallback, useMemo, useRef, type RefObject } from 'react'

import { formatClock, frameToMs } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'
import { displayClockMs, type ReplayWindowBounds } from '../replayWindow'

/** Cadence de publication de l'image courante vers React, en millisecondes. */
const FRAME_PUBLISH_MS = 150

export interface ReplayClockOptions {
  doc: ReplayDocumentReady
  /** La fenêtre de gameplay ; `null` = horloge du film entier, comme avant le cadrage. */
  playWindow: ReplayWindowBounds | null
  /**
   * PUBLIE l'image courante, a cadence reduite, dans le magasin de lecture
   * (`model/playbackStore`) : c'est de la que les panneaux hors canvas la lisent. Le bridage
   * vit ICI et non dans le magasin — c'est celui qui peint qui connait le cout d'un rendu.
   */
  publish?: (frame: number) => void
}

export interface ReplayClock {
  /** L'horloge est écrite par la boucle de dessin (textContent), pas par React. */
  clockRef: RefObject<HTMLSpanElement | null>
  /** À appeler à la fin de chaque tracé, avec l'image qui vient d'être peinte. */
  tick: (frame: number) => void
}

export function useReplayClock({ doc, playWindow, publish }: ReplayClockOptions): ReplayClock {
  const clockRef = useRef<HTMLSpanElement>(null)
  const publishedAtRef = useRef(0)
  // LE TOTAL EST LA DURÉE DE JEU quand le match est cadré : la durée du film y ajouterait le
  // countdown et la queue, c'est-à-dire une dizaine de secondes qui n'ont pas été jouées.
  const totalLabel = useMemo(
    () =>
      formatClock(
        playWindow ? playWindow.endMs - playWindow.startMs : (doc.durationMs ?? frameToMs(doc.frameCount, doc)),
      ),
    [doc, playWindow],
  )
  const tick = useCallback(
    (frame: number) => {
      if (clockRef.current) {
        const nowMs = displayClockMs(frameToMs(frame, doc), playWindow)
        clockRef.current.textContent = `${formatClock(nowMs)} / ${totalLabel}`
      }
      if (!publish) return
      const now = performance.now()
      // La borne de fin est publiée SANS DÉLAI (cf. l'en-tête) : c'est la dernière image que la
      // boucle peindra, personne ne repassera derrière pour rattraper une publication sautée.
      const atEnd = playWindow != null && frame >= playWindow.endFrame
      if (!atEnd && now - publishedAtRef.current < FRAME_PUBLISH_MS) return
      publishedAtRef.current = now
      publish(Math.floor(frame))
    },
    [doc, playWindow, totalLabel, publish],
  )
  return { clockRef, tick }
}
