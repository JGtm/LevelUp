/**
 * replayWindow — LA FENÊTRE DE GAMEPLAY : où commence et où finit le MATCH sur l'axe du film.
 *
 * LE FILM DÉBORDE LE MATCH DES DEUX CÔTÉS. Avant, il porte le countdown d'avant-match (les
 * joueurs sont déjà là, figés à leur apparition) ; après, il garde une queue de 5 à 6 secondes
 * (mesuré : 6,1 / 6,1 / 6,1 / 5,2 s sur les quatre témoins) — pas l'outro complète, juste ce
 * que l'enregistrement a eu le temps de prendre. Lu de bout en bout, le rejeu commençait donc
 * sur des statues et se terminait sur une scène qui n'appartient plus au match.
 *
 * DEUX AXES, ET C'EST TOUTE LA DIFFICULTÉ. Le film compte en IMAGES depuis son premier paquet
 * de position (`doc.originMs`, schéma >= 4, dit à quel instant du match cette image zéro
 * tombe) ; le match, lui, compte en millisecondes depuis son `start_time` (`header.t0_ms` =
 * coup d'envoi, `header.playable_duration_seconds` = durée de jeu). Passer de l'un à l'autre
 * est une soustraction, et les deux bornes en découlent :
 *
 *     début = max(0, t0_ms − originMs)                     (clamp, cf. plus bas)
 *     fin   = t0_ms + playable_duration_seconds × 1000 − originMs
 *
 * LA FIN RESTE JUSTE SANS `t0_ms`, et ce n'est pas une chance : le serveur calcule
 * `playable_duration_seconds` comme `duration_seconds − t0_ms/1000` quand le T0 est connu
 * (match_view_builders_header.go), donc l'ancrage se compense de lui-même — l'erreur
 * résiduelle est la troncature à la seconde. Un match sans T0 garde donc sa fin cadrée, et
 * démarre au premier paquet.
 *
 * LE CLAMP DU DÉBUT N'EST PAS DÉFENSIF : sur le témoin e94163af, le premier paquet de position
 * arrive 4,5 s APRÈS le coup d'envoi (originMs 39 772 > t0_ms 35 238). Le film n'a rien à
 * montrer avant sa propre image zéro.
 *
 * LA BORNE DE FIN N'UTILISE JAMAIS LE CALQUE DE SCORE, et c'est une mesure, pas une opinion :
 * le dernier point du score tombe 1,5 à 1,6 s avant la fin déclarée sur les trois témoins
 * gagnés au score — mais 133 s avant sur 64e8adfa, qui s'est terminé AU TEMPS. Une borne tirée
 * du score amputerait deux minutes de jeu sur ce match-là sans qu'aucun test ne rougisse.
 *
 * SANS DONNÉE, PAS DE CADRAGE (D-A3) : un artefact antérieur au schéma 4 (pas d'`originMs`) ou
 * un en-tête sans durée jouable rend `null`, et tout ce qui lit cette fenêtre retombe alors sur
 * le comportement d'avant ce lot — film entier, horloge du film. Une fenêtre incohérente (fin
 * avant début) rend `null` pour la même raison : mieux vaut le film entier qu'une lecture d'une
 * seule image.
 */
import type { MatchViewHeader } from '@/lib/api/types'

import { frameToMs, msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/** Ce que la fenêtre lit de l'en-tête du match : les deux bornes de l'axe du MATCH. */
export type ReplayWindowHeader = Pick<MatchViewHeader, 't0_ms' | 'playable_duration_seconds'>

/** La fenêtre de gameplay, dans les deux unités du rejeu (images pour la frise, ms pour l'horloge). */
export interface ReplayWindowBounds {
  /** Première image du gameplay : la borne basse de la frise ET le point de départ de la lecture. */
  startFrame: number
  /** Dernière image du gameplay : la lecture s'y arrête, la frise n'en sort pas. */
  endFrame: number
  /** Instant du coup d'envoi sur l'axe du film, en ms — l'origine de l'horloge AFFICHÉE. */
  startMs: number
  /** Instant de la fin déclarée sur l'axe du film, en ms, borné par la durée réelle du film. */
  endMs: number
}

/**
 * replayWindow rend la fenêtre de gameplay du document, ou `null` quand elle n'est pas
 * établie (cf. l'en-tête : artefact sans origine, en-tête sans durée jouable, bornes
 * incohérentes). Les images sont ARRONDIES à l'image la plus proche : à 100 ms d'intervalle,
 * tronquer coûterait jusqu'à une image de gameplay à chaque bout.
 */
export function replayWindow(
  doc: ReplayDocumentReady,
  header: ReplayWindowHeader | null | undefined,
): ReplayWindowBounds | null {
  const originMs = doc.originMs
  const playableSeconds = header?.playable_duration_seconds
  if (originMs == null || playableSeconds == null) return null
  const lastFrame = doc.frameCount - 1
  if (lastFrame <= 0) return null

  const t0Ms = header?.t0_ms ?? 0
  const startMs = Math.max(0, t0Ms - originMs)
  const filmEndMs = frameToMs(lastFrame, doc)
  const endMs = Math.min(t0Ms + playableSeconds * 1000 - originMs, filmEndMs)
  const startFrame = clampFrame(Math.round(msToFrames(startMs, doc)), lastFrame)
  const endFrame = clampFrame(Math.round(msToFrames(endMs, doc)), lastFrame)
  if (endFrame <= startFrame) return null
  return { startFrame, endFrame, startMs, endMs }
}

/**
 * displayClockMs recale un instant de l'axe du film sur l'HORLOGE DU GAMEPLAY (D-A2) : le
 * coup d'envoi se lit 0:00, et le countdown d'avant-match ne se compte plus. Plancher à zéro
 * — un événement d'avant le coup d'envoi (une mort en pré-match) s'affiche à 0:00 plutôt qu'en
 * négatif. Sans fenêtre, l'identité : l'axe interne du rejeu ne change pas, seul l'affichage.
 */
export function displayClockMs(ms: number, playWindow: ReplayWindowBounds | null): number {
  if (!playWindow) return ms
  return Math.max(0, ms - playWindow.startMs)
}

/** Borne une image dans le film : ni avant l'image zéro, ni après la dernière. */
function clampFrame(frame: number, lastFrame: number): number {
  return Math.min(Math.max(frame, 0), lastFrame)
}
