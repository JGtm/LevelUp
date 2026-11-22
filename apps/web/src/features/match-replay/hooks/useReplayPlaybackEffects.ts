/**
 * useReplayPlaybackEffects — LES EFFETS DE LA LECTURE (`useReplayPlayback`) : la pose initiale du
 * curseur, la boucle d'animation, l'ouverture à une image précise et le canal de saut de la page.
 *
 * Extraits de `useReplayPlayback` pour tenir la règle des 80 lignes par fonction : chaque effet garde
 * ses dépendances et son ordre d'exécution (le hook les appelle dans l'ordre où ils s'écrivaient), et
 * aucun ne dessine ni ne décide rien de plus qu'avant. Le temps de la lecture, ses commandes et leurs
 * contrats restent dans `useReplayPlayback.ts`.
 */
import { useEffect, useRef, type Dispatch, type RefObject, type SetStateAction } from 'react'

import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

interface InitialCursorInput {
  /** Posé vrai dès que `openAtFrame` a été appliqué : le cadrage ci-dessous ne l'écrase alors plus. */
  appliedOpenAtFrameRef: RefObject<boolean>
  leadInFrame: number
  frameRef: RefObject<number>
  draw: () => void
  writeCursor: (frame: number) => void
}

/**
 * LE COUP D'ENVOI (moins son préambule), DÈS QUE LA FENÊTRE SE CONNAÎT : elle arrive avec la Match View,
 * donc après le premier rendu. On ne pose le curseur que s'il est encore EN DEÇÀ — le repositionnement
 * ne recule jamais la lecture.
 *
 * `appliedOpenAtFrameRef` (lot M1b) : un lien tactique déjà positionné ne doit JAMAIS être repris par ce
 * cadrage, même sur un instant antérieur au coup d'envoi (la fenêtre de gameplay, asynchrone, peut
 * arriver APRÈS l'application du lien).
 */
export function useInitialCursor({ appliedOpenAtFrameRef, leadInFrame, frameRef, draw, writeCursor }: InitialCursorInput): void {
  useEffect(() => {
    if (appliedOpenAtFrameRef.current) return
    if (frameRef.current < leadInFrame) {
      frameRef.current = leadInFrame
      writeCursor(leadInFrame)
      draw()
      return
    }
    // LE REMPLISSAGE SE POSE MÊME SANS REPOSITIONNEMENT : sans cet appel, un montage où la
    // lecture est déjà au bon endroit (cas nominal sans fenêtre) laisserait `--played` vide,
    // et la frise s'afficherait creuse jusqu'au premier pas de la boucle.
    writeCursor(frameRef.current)
  }, [appliedOpenAtFrameRef, leadInFrame, frameRef, draw, writeCursor])
}

interface PlaybackLoopInput {
  playing: boolean
  setPlaying: Dispatch<SetStateAction<boolean>>
  doc: ReplayDocumentReady
  baseFps: number
  speed: number
  renderWidth: number
  endFrame: number
  frameRef: RefObject<number>
  writeCursor: (frame: number) => void
  draw: () => void
  soundTick: (ms: number) => void
  onEnded: () => void
  /** Annonce l'intervalle d'images que ce pas vient de parcourir (cf. `ReplayPlaybackOptions.onAdvance`). */
  onAdvance?: (from: number, to: number) => void
}

/**
 * La boucle de lecture (requestAnimationFrame) uniquement quand `playing`.
 *
 * LE SON BAT AVEC ELLE, et nulle part ailleurs : hors lecture (pause, onglet en arrière-plan, redessin au
 * changement de thème) il n'y a pas de battement, donc pas un son. C'est ce qui rend le silence d'un
 * lecteur à l'arrêt structurel, pas conditionnel.
 */
export function usePlaybackLoop(i: PlaybackLoopInput): void {
  const { playing, setPlaying, doc, baseFps, speed, renderWidth, endFrame, frameRef, writeCursor, draw, soundTick, onEnded, onAdvance } = i
  useEffect(() => {
    if (!playing || renderWidth === 0) return
    const fps = baseFps * speed
    let raf = 0
    let last = 0
    const step = (ts: number) => {
      if (last === 0) last = ts
      const dtSec = (ts - last) / 1000
      last = ts
      const from = frameRef.current
      let next = from + dtSec * fps
      // LA FIN DU FILM : on borne à la dernière image, on la PEINT, puis on s'arrête (cf.
      // l'en-tête de `useReplayPlayback`). L'ordre compte — sortir avant le tracé laisserait la scène
      // une image en arrière, et sortir avant le curseur laisserait la frise mentir.
      const ended = next >= endFrame
      if (ended) next = endFrame
      frameRef.current = next
      writeCursor(next)
      soundTick(frameToMs(next, doc))
      onAdvance?.(from, next)
      draw()
      if (ended) {
        // FRANCHIR LA BORNE, PAS Y ÊTRE. `from < endFrame` est ce qui distingue une ARRIVÉE
        // d'un simple constat : une frise tirée jusqu'au bout pose déjà le curseur sur la
        // borne, et le pas suivant conclurait « fin » sans que la lecture ait rien parcouru.
        // C'est la décision D-C1 (« pas de son sur un scrub qui atteint la fin ») ; l'unicité,
        // elle, est structurelle — la boucle s'arrête ici, et repartir passe par un rembobinage
        // ou une position en deçà de la borne.
        if (from < endFrame) onEnded()
        setPlaying(false)
        return
      }
      raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [playing, setPlaying, baseFps, speed, doc, renderWidth, draw, soundTick, onEnded, onAdvance, endFrame, frameRef, writeCursor])
}

/**
 * LE `seekTo` DU DERNIER RENDU, tenu en référence : `seekTo` se recrée à chaque rendu (fermeture sur
 * `leadInFrame` / `endFrame`), et les deux effets ci-dessous doivent l'appeler sans se réinscrire à
 * chaque fois. À appeler AVANT eux : la référence est à jour quand ils s'exécutent.
 */
export function useLatestSeek(seekTo: (frame: number) => void): RefObject<(frame: number) => void> {
  const seekToRef = useRef(seekTo)
  useEffect(() => {
    seekToRef.current = seekTo
  })
  return seekToRef
}

/**
 * OUVRIR AU LIEN TACTIQUE (lot M1b, 2026-09-08) : appliqué UNE SEULE FOIS — `openAtFrame` peut arriver
 * dès le premier rendu (frame déjà convertie par la route) ou rester `null` tant que le document ne l'a
 * pas résolu, mais une fois posé il ne doit jamais être répété (un utilisateur qui a depuis déplacé la
 * frise ne doit pas y être ramené). Passe par le MÊME `seekTo` que la frise et les repères
 * d'entrée/sortie — pas un chemin de pose séparé.
 */
export function useOpenAtFrame(
  openAtFrame: number | null | undefined,
  appliedOpenAtFrameRef: RefObject<boolean>,
  seekToRef: RefObject<(frame: number) => void>,
): void {
  useEffect(() => {
    if (openAtFrame == null || appliedOpenAtFrameRef.current) return
    appliedOpenAtFrameRef.current = true
    seekToRef.current(openAtFrame)
  }, [openAtFrame, appliedOpenAtFrameRef, seekToRef])
}

/**
 * LE CANAL DE SAUT (`ReplayPlaybackOptions.onSeekRequest`) : l'exécuteur se désinscrit au démontage. Une
 * demande met la lecture en PAUSE puis la pose sur l'image, par le même `seekTo` borné que tout saut.
 */
export function useSeekRequests(
  onSeekRequest: ((executer: (frame: number) => void) => () => void) | undefined,
  setPlaying: Dispatch<SetStateAction<boolean>>,
  seekToRef: RefObject<(frame: number) => void>,
): void {
  useEffect(() => {
    if (!onSeekRequest) return
    return onSeekRequest((frame) => {
      setPlaying(false)
      seekToRef.current(frame)
    })
  }, [onSeekRequest, setPlaying, seekToRef])
}
