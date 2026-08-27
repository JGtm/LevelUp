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
 * L'ARRIVÉE EST AUSSI UN ÉVÉNEMENT (`onEnded`, lot C du 2026-08-27) : c'est d'ici que part le
 * son de fin de partie, parce que c'est le seul endroit qui sait distinguer une lecture qui
 * FRANCHIT la borne d'une frise qu'on a tirée jusqu'au bout. La condition tient en trois mots —
 * l'image d'AVANT le pas était en deçà de la borne.
 *
 * RELANCER RESTE À UN CLIC, et c'est la convention des lecteurs vidéo : « Lecture » sur un
 * rejeu terminé repart du début plutôt que de rester bloqué sur la dernière image, où la boucle
 * se rendormirait aussitôt. « Recommencer » ne change pas — il ramène au début à tout instant.
 *
 * # « LE DÉBUT » ET « LA FIN » SONT CEUX DU MATCH, PAS CEUX DU FILM
 *
 * Depuis le 2026-08-26, les deux bornes viennent de la FENÊTRE DE GAMEPLAY (`replayWindow.ts`) :
 * la lecture démarre au coup d'envoi et s'arrête à la fin déclarée, sans le countdown
 * d'avant-match ni la queue de 5-6 s que le film garde après. La mécanique ne change pas d'un
 * pas — seules les deux valeurs changent. Sans fenêtre (artefact ancien, en-tête sans durée
 * jouable), les bornes redeviennent celles du film : image zéro et dernière image.
 *
 * LE CURSEUR SE POSE AU DÉBUT QUAND LA FENÊTRE SE CONNAÎT, et pas avant : elle vient de la
 * Match View, qui arrive APRÈS le document du rejeu. Le repositionnement ne va donc que vers
 * l'AVANT (jamais un retour en arrière sous les doigts de qui a déjà déplacé la frise).
 *
 * # SE DÉPLACER SANS VISER : LES SAUTS, ET LE REMPLISSAGE DE LA FRISE
 *
 * Deux commandes s'ajoutent le 2026-08-28 avec la barre de lecture (planche 2a) : `seekBy`,
 * un saut de ±N SECONDES converti en images par la cadence du document, et `stepFrames`,
 * l'image par image. Les deux passent par le même `seekTo` borné à la fenêtre de gameplay —
 * une seule définition de « où le curseur a le droit d'aller ».
 *
 * `writeCursor` EST LE SEUL ENDROIT QUI DÉPLACE LE CURSEUR, et il en écrit DEUX choses : la
 * valeur du champ (ce que le navigateur dessine) et la variable CSS `--played` (ce que la
 * frise habillée remplit derrière lui). Les séparer les ferait diverger au premier chemin
 * oublié — c'est pourquoi la boucle, les sauts, le rembobinage, le glissé manuel et la pose
 * initiale l'appellent tous, sans exception.
 */
import { useCallback, useEffect, useRef, useState, type ChangeEvent, type RefObject } from 'react'

import { frameToMs } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayWindowBounds } from './replayWindow'

/** Ce dont la lecture a besoin (objet unique : la règle des 5 paramètres du dépôt). */
export interface ReplayPlaybackOptions {
  doc: ReplayDocumentReady
  /**
   * La fenêtre de gameplay du match (cf. `replayWindow.ts`). `null` = pas de cadrage établi :
   * la lecture reprend les bornes du film entier, exactement comme avant ce lot.
   */
  playWindow: ReplayWindowBounds | null
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
  /**
   * L'ARRIVÉE EN FIN DE MATCH : appelé quand la lecture FRANCHIT la borne de fin, jamais quand
   * elle y était déjà (cf. l'en-tête). Le son de fin de partie s'y branche.
   */
  onEnded: () => void
  /**
   * UN GESTE DE TRANSPORT VIENT D'AVOIR LIEU (« Lecture »/« Pause », « Recommencer »). Le son
   * s'en sert pour reprendre vie après un rechargement de page : un AudioContext ne naît que
   * dans un geste utilisateur, et ces deux boutons en sont (cf. `useReplaySound.wake`).
   */
  onTransportGesture: () => void
}

/** Ce que la barre de lecture reçoit — l'état à afficher et les commandes. */
export interface ReplayPlayback {
  playing: boolean
  /**
   * LA PREMIÈRE IMAGE DU GAMEPLAY : la borne basse de la frise ET le point de départ de la
   * lecture, de « Recommencer » et du rembobinage de fin. Sans fenêtre : l'image zéro.
   */
  startFrame: number
  /**
   * LA DERNIÈRE IMAGE DU GAMEPLAY : la borne haute de la frise ET l'instant où la lecture
   * s'arrête. Une seule valeur pour les deux, sinon le curseur pourrait buter avant (ou
   * après) l'arrêt. Sans fenêtre : la dernière image du document.
   */
  endFrame: number
  /** Le curseur de la frise : piloté par la boucle, jamais contrôlé par React. */
  sliderRef: RefObject<HTMLInputElement | null>
  togglePlay: () => void
  restart: () => void
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
  /** Saut de ±N SECONDES sur l'axe réel (converti en images par `baseFps`). */
  seekBy: (seconds: number) => void
  /**
   * Image par image, bornée à la fenêtre de gameplay. ELLE MET LA LECTURE EN PAUSE : un pas
   * d'image sous la boucle d'animation serait écrasé en 16 ms, et avancer d'une image est un
   * geste d'arrêt sur image par nature.
   */
  stepFrames: (frames: number) => void
}

export function useReplayPlayback(o: ReplayPlaybackOptions): ReplayPlayback {
  const { doc, playWindow, baseFps, speed, renderWidth, frameRef, draw } = o
  const { soundTick, onEnded, onTransportGesture } = o
  const sliderRef = useRef<HTMLInputElement>(null)
  const [playing, setPlaying] = useState(true)
  const lastFrame = Math.max(doc.frameCount - 1, 0)
  const startFrame = playWindow?.startFrame ?? 0
  const endFrame = playWindow?.endFrame ?? lastFrame

  /**
   * writeCursor POSE LE CURSEUR : la valeur du champ, et le REMPLISSAGE de la frise habillée.
   *
   * `--played` est la part parcourue, en pourcentage de la fenêtre. Le dégradé de la piste la
   * consomme depuis les classes du champ lui-même (`ReplayTimelineTracks.tsx`, variantes
   * `[&::-webkit-slider-runnable-track]` / `[&::-moz-range-track]` — même technique que le
   * volume dans `ReplaySoundControls.tsx`) : aucune feuille de style à tenir à jour, et aucun
   * rendu React pour un remplissage qui suit la lecture. Elle s'écrit ICI et nulle part
   * ailleurs — un chemin qui déplacerait le curseur sans elle laisserait le remplissage figé
   * sur la position précédente.
   */
  const writeCursor = useCallback(
    (frame: number) => {
      const el = sliderRef.current
      if (!el) return
      el.value = String(Math.round(frame))
      const span = endFrame - startFrame
      const pct = span > 0 ? ((frame - startFrame) / span) * 100 : 0
      // Borné : la frise ne se remplit ni en deçà de son début ni au-delà de sa fin, même si
      // un appelant lui sert une image hors fenêtre.
      el.style.setProperty('--played', `${Math.min(100, Math.max(0, pct))}%`)
    },
    [startFrame, endFrame],
  )

  // LE COUP D'ENVOI, DÈS QUE LA FENÊTRE SE CONNAÎT (cf. l'en-tête) : elle arrive avec la Match
  // View, donc après le premier rendu. On ne pose le curseur que s'il est encore EN DEÇÀ du
  // début — le repositionnement ne recule jamais la lecture.
  useEffect(() => {
    if (frameRef.current < startFrame) {
      frameRef.current = startFrame
      writeCursor(startFrame)
      draw()
      return
    }
    // LE REMPLISSAGE SE POSE MÊME SANS REPOSITIONNEMENT : sans cet appel, un montage où la
    // lecture est déjà au bon endroit (cas nominal sans fenêtre) laisserait `--played` vide,
    // et la frise s'afficherait creuse jusqu'au premier pas de la boucle.
    writeCursor(frameRef.current)
  }, [startFrame, frameRef, draw, writeCursor])

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
      const from = frameRef.current
      let next = from + dtSec * fps
      // LA FIN DU FILM : on borne à la dernière image, on la PEINT, puis on s'arrête (cf.
      // l'en-tête). L'ordre compte — sortir avant le tracé laisserait la scène une image en
      // arrière, et sortir avant le curseur laisserait la frise mentir.
      const ended = next >= endFrame
      if (ended) next = endFrame
      frameRef.current = next
      writeCursor(next)
      soundTick(frameToMs(next, doc))
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
  }, [playing, baseFps, speed, doc, renderWidth, draw, soundTick, onEnded, endFrame, frameRef, writeCursor])

  const onScrub = (e: ChangeEvent<HTMLInputElement>) => {
    frameRef.current = Number(e.currentTarget.value)
    // LE REMPLISSAGE SUIT LE GLISSÉ : le champ porte déjà sa valeur (c'est lui qui l'émet),
    // mais `--played` ne se met à jour pour personne — sans cet appel, la piste resterait
    // remplie jusqu'à la position d'AVANT le geste.
    writeCursor(frameRef.current)
    if (!playing) draw()
  }

  const rewind = () => {
    frameRef.current = startFrame
    writeCursor(startFrame)
  }

  /**
   * seekTo — LE SEUL CHEMIN DE DÉPLACEMENT DIRECT, et il porte le bornage : la lecture ne
   * sort pas de la fenêtre de gameplay, quel que soit le geste qui l'y envoie. Il peint
   * (`draw`) et fait battre le son (`soundTick`) comme un pas de boucle : un curseur déplacé
   * sans repeindre montrerait la scène de l'instant précédent.
   */
  const seekTo = (frame: number) => {
    const next = Math.min(endFrame, Math.max(startFrame, frame))
    frameRef.current = next
    writeCursor(next)
    soundTick(frameToMs(next, doc))
    draw()
  }

  const seekBy = (seconds: number) => seekTo(frameRef.current + seconds * baseFps)

  const stepFrames = (frames: number) => {
    // EN PAUSE D'ABORD (cf. l'en-tête) : sous la boucle, le pas d'image serait écrasé à la
    // frame suivante et le geste n'aurait aucun effet visible.
    setPlaying(false)
    seekTo(Math.round(frameRef.current) + frames)
  }

  // LES DEUX COMMANDES DE TRANSPORT PRÉVIENNENT LE SON, en PREMIER : ce sont des gestes
  // utilisateur, la seule fenêtre où un AudioContext démarre en marche. C'est ce qui rend son
  // son à un rejeu rechargé dont la préférence était restée à « activé » (cf. useReplaySound).
  const restart = () => {
    onTransportGesture()
    rewind()
    setPlaying(true)
  }

  const togglePlay = () => {
    onTransportGesture()
    // REPARTIR DU DÉBUT SUR UN REJEU TERMINÉ (cf. l'en-tête) : sans ce rembobinage, la boucle
    // relancée à `endFrame` conclurait « fin » à son premier pas et se rendormirait — le bouton
    // « Lecture » n'aurait aucun effet visible.
    if (!playing && frameRef.current >= endFrame) rewind()
    setPlaying((p) => !p)
  }

  return { playing, startFrame, endFrame, sliderRef, togglePlay, restart, onScrub, seekBy, stepFrames }
}
