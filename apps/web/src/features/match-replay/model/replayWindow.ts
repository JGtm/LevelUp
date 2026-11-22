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
 * de position (l'ORIGINE que porte l'horloge de la page — schéma >= 4 — dit à quel instant du
 * match cette image zéro tombe) ; le match, lui, compte en ms depuis son `start_time` (`header.t0_ms` =
 * coup d'envoi, `header.gameplay_duration_seconds` = durée de jeu). Passer de l'un à l'autre
 * est une soustraction, et les deux bornes en découlent :
 *
 *     début = max(0, t0_ms − originMs)                     (clamp, cf. plus bas)
 *     fin   = t0_ms + gameplay_duration_seconds × 1000 − originMs
 *
 * UN SEUL COUP D'ENVOI, celui du match. `header.t0_ms` est résolu une fois par le serveur
 * (ADR 0050) : le départ de l'horloge du film quand le match a un rejeu, sinon l'estimation
 * des `first_joined_time` de l'API, sinon inconnu. Le même sert la fenêtre, l'horloge affichée,
 * le recalage des événements et le découpage de la Dominance : le rejeu ne choisit plus entre
 * deux coups d'envoi, et ne relit pas non plus celui du document.
 *
 * LA FIN SE COMPENSE D'ELLE-MÊME : le serveur calcule `gameplay_duration_seconds` comme la
 * durée du match moins `t0_ms` (tronquée à la seconde), donc `t0_ms + durée de jeu` retombe sur
 * la fin du match — l'erreur résiduelle est la troncature, moins d'une seconde. Un match sans T0
 * garde donc sa fin cadrée, et démarre au premier paquet.
 *
 * LE CLAMP DU DÉBUT N'EST PAS DÉFENSIF : sur le témoin e94163af, le premier paquet de position
 * arrive 4,5 s APRÈS le coup d'envoi (originMs 39 772 > t0_ms 35 238). Le film n'a rien à
 * montrer avant sa propre image zéro.
 *
 * LE PRÉAMBULE (`leadInFrame`) EST UN CONFORT DE LECTURE, PAS UN CADRAGE (D3, user 02/09) :
 * la lecture se pose UNE seconde avant le coup d'envoi, le temps de poser les yeux sur la
 * scène avant que l'action parte. Rien d'autre ne bouge — l'horloge affichée, la frise et les
 * bornes d'export restent sur `startMs` / `startFrame`, et l'horloge lit donc 0:00 pendant ce
 * préambule (plancher de `displayClockMs`).
 *
 * LA BORNE DE FIN N'UTILISE JAMAIS LE CALQUE DE SCORE, et c'est une mesure, pas une opinion :
 * le dernier point du score tombe 1,5 à 1,6 s avant la fin déclarée sur les trois témoins
 * gagnés au score — mais 133 s avant sur 64e8adfa, qui s'est terminé AU TEMPS. Une borne tirée
 * du score amputerait deux minutes de jeu sur ce match-là sans qu'aucun test ne rougisse.
 *
 * SANS DONNÉE, PAS DE CADRAGE (D-A3) : un artefact dont l'horloge n'est pas établie
 * (`model/replayClock` — pas d'origine publiée, pas d'échelle temporelle, moins de deux
 * images) ou un en-tête sans durée de jeu rend `null`, et tout ce qui lit cette fenêtre
 * retombe alors sur le comportement d'avant ce lot — film entier, horloge du film. Une
 * fenêtre incohérente (fin avant début) rend `null` pour la même raison : mieux vaut le film
 * entier qu'une lecture d'une seule image.
 *
 * L'ORIGINE NE SE LIT PLUS ICI (2026-09-05, P0-5) : elle vient de `replayClock`, le verdict
 * unique de la page — le même que celui du fil, des médias, de la présence et des sièges.
 */
import type { MatchViewHeader } from '@/lib/api/types'

import { replayClock } from './replayClock'
import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Ce que la fenêtre lit de l'en-tête du match : les deux bornes de l'axe du MATCH. */
export type ReplayWindowHeader = Pick<MatchViewHeader, 't0_ms' | 'gameplay_duration_seconds'>

/**
 * PRÉAMBULE DE LECTURE, en millisecondes AVANT le coup d'envoi (D3, user 2026-09-02).
 *
 * Une seconde : de quoi voir la scène s'installer avant que l'action parte, sans jamais
 * redonner à voir le countdown d'avant-match. Ce n'est PAS un déplacement du cadrage — la
 * fenêtre de gameplay, l'horloge et l'export ne connaissent que `startMs` / `startFrame`.
 */
export const LEAD_IN_MS = 1000

/** La fenêtre de gameplay, dans les deux unités du rejeu (images pour la frise, ms pour l'horloge). */
export interface ReplayWindowBounds {
  /**
   * Première image du gameplay : la borne basse de la FRISE, l'origine de l'horloge affichée
   * et le début de la plage d'export. Ce n'est plus l'endroit où la lecture SE POSE — celui-là
   * est `leadInFrame`, une seconde plus tôt.
   */
  startFrame: number
  /**
   * OÙ LA LECTURE SE POSE, une seconde en deçà du coup d'envoi (`LEAD_IN_MS`) — jamais avant
   * l'image zéro du film. C'est la SEULE borne que le préambule déplace : la frise part
   * toujours de `startFrame`, et l'horloge affiche 0:00 tant que la lecture est en deçà.
   */
  leadInFrame: number
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
  const clock = replayClock(doc, header)
  const gameplaySeconds = header?.gameplay_duration_seconds
  if (!clock || gameplaySeconds == null) return null
  // L'horloge garantit au moins deux images : la dernière existe donc.
  const lastFrame = clock.frameCount - 1

  // LE COUP D'ENVOI DU MATCH sert les deux bornes : c'est lui que le serveur a soustrait pour
  // produire `gameplay_duration_seconds` (cf. l'en-tête, § de la compensation).
  const t0Ms = clock.t0Ms
  const startMs = Math.max(0, clock.filmMsOfMatchMs(t0Ms))
  const filmEndMs = frameToMs(lastFrame, doc)
  const endMs = Math.min(clock.filmMsOfMatchMs(t0Ms + gameplaySeconds * 1000), filmEndMs)
  const startFrame = clampFrame(clock.frameOfFilmMs(startMs), lastFrame)
  const endFrame = clampFrame(clock.frameOfFilmMs(endMs), lastFrame)
  if (endFrame <= startFrame) return null
  const leadInFrame = clampFrame(clock.frameOfFilmMs(startMs - LEAD_IN_MS), lastFrame)
  return { startFrame, leadInFrame, endFrame, startMs, endMs }
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
