/**
 * hillHoldLogic — LA PROGRESSION DE GARDE DE LA COLLINE, au frame lu.
 *
 * EN KOTH CE QUI MARQUE, C'EST DE TENIR LA COLLINE. La prendre est une capture courte (environ
 * une seconde, instantanée en classé — la jauge de la colline, calque des zones) ; ensuite chaque
 * tic de garde rapproche du point. Ce module dit où en est chaque camp vers le point suivant : la
 * barre ne redescend pas quand la colline est perdue, elle reprend où elle en était quand le camp
 * la reprend, et elle repart de zéro au point (la colline tourne).
 *
 * ELLE EST LUE, PAS RECONSTRUITE — et c'est tout ce qui compte ici. La série vient de
 * `scoreTimeline.holdTicks`, que l'artefact publie depuis le compteur du jeu lui-même
 * (`comp 23 A` du statborg, qui reproduit `StrongholdScoringTicks` de l'API exactement joueur
 * par joueur). Ce module ne fait donc que TROIS choses, et aucune n'est une estimation :
 *
 *   la remise à zéro   la colline TOURNE à chaque point ; les instants de point sont les
 *                      paliers des courbes d'équipe. La barre repart de zéro à chacun, APRÈS
 *                      une fenêtre d'une seconde (`HOLD_FULL_WINDOW_MS`) à partir de l'image du
 *                      point : pendant elle, le camp qui marque (celui dont la courbe monte à
 *                      cet instant) est PLEIN et l'autre garde sa valeur d'avant le point.
 *                      Un tic qui tombe dans la fenêtre ne la change pas (la fenêtre l'emporte) ;
 *                      un nouveau point ouvre sa propre fenêtre et remplace la précédente.
 *   le différentiel    `ticks(frame) − ticks(dernier point)`, puisque la série est cumulative
 *                      sur tout le match.
 *   le dénominateur    `scoreTimeline.holdTicksPerPoint` — un COMPTE et non un réglage, que
 *                      l'artefact MESURE DANS LE FILM, match par match : au point, la garde
 *                      prise par le camp qui marque est le seuil (35 en arène et en Doubles,
 *                      40 en classé). La table de la variante (`regulation.toml`) n'est qu'un
 *                      repli pour un match sans point, compté dans `coverage.fallbacks`.
 *
 * UNE VERSION ANTÉRIEURE DE CE MODULE INTÉGRAIT LES INTERVALLES DE PROPRIÉTÉ avec un seuil de
 * 43 s. C'était une reconstruction, et elle était fausse d'environ 20 % — l'écart n'a pas été
 * corrigé en ajustant la constante (ce qui aurait été caler une mesure sur un rendu) mais en
 * changeant de SOURCE. L'historique des trois lots vit dans
 * `.ai/archive/V7.5/PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md`.
 *
 * LES CAMPS SE LISENT DANS LA MÊME CONVENTION QUE LE BANDEAU. `holdTicks[].teamId` et
 * `teams[].teamId` portent tous deux l'index d'équipe du REGISTRE ; ce module reçoit les
 * identifiants que le bandeau a déjà résolus — les camps du film, le côté allié vu du point de
 * vue (`readScoreBanner`) — et se tait s'il ne les trouve pas.
 *
 * Module PUR : ni React, ni DOM, ni couleur.
 */
import { msToFrames } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

/**
 * HOLD_FULL_WINDOW_MS — combien de temps la barre du camp qui vient de marquer reste PLEINE à
 * partir de l'image du point, en temps réel (converti en images par `msToFrames`, comme
 * `ZONE_GAUGE_HOLD_MS`). Le dernier tic de garde tombe sur l'image même du point : sans cette
 * fenêtre, la barre se vide avant d'avoir jamais été vue pleine.
 */
export const HOLD_FULL_WINDOW_MS = 1_000

/** Le document, vu par ce module : le calque de score et la cadence du film. */
export type HillHoldDocument = Pick<ReplayDocumentReady, 'scoreTimeline' | 'frameIntervalMs'>

/** La lecture : une fraction 0..1 par camp, ou `null` quand rien ne doit être dessiné. */
export interface HillHoldReading {
  ally: number
  enemy: number
}

/**
 * readHillHold rend la progression de garde des deux camps au frame lu, ou `null` quand la
 * jauge ne doit PAS se dessiner.
 *
 * LES CAS `null` SONT LA MOITIÉ DU CONTRAT, et ils suivent la doctrine du bandeau (une jauge
 * absente ne ment pas, une jauge à zéro si) : artefact sans série de garde (mode sans colline,
 * ou artefact antérieur au champ), match sans dénominateur (aucun point lisible et variante
 * absente de la table, ou score à la seconde où chaque tic est un point — le producteur ne publie
 * alors ni la série ni le seuil), ou camps que le calque ne situe pas.
 */
export function readHillHold(
  doc: HillHoldDocument,
  allyTeamId: number,
  enemyTeamId: number,
  frame: number,
): HillHoldReading | null {
  const timeline = doc.scoreTimeline
  if (!timeline) return null
  const perPoint = timeline.holdTicksPerPoint
  if (!perPoint || perPoint <= 0) return null
  const ally = holdSeriesOf(timeline, allyTeamId)
  const enemy = holdSeriesOf(timeline, enemyTeamId)
  if (!ally || !enemy) return null

  const reset = lastReset(timeline, frame)
  const windowFrames = msToFrames(HOLD_FULL_WINDOW_MS, doc)
  if (reset.frame >= 0 && frame - reset.frame < windowFrames) {
    const before = lastReset(timeline, reset.frame - 1).frame
    return {
      ally: reset.scorers.has(allyTeamId) ? 1 : clamp01(gained(ally, before, reset.frame - 1) / perPoint),
      enemy: reset.scorers.has(enemyTeamId) ? 1 : clamp01(gained(enemy, before, reset.frame - 1) / perPoint),
    }
  }
  return {
    ally: clamp01(gained(ally, reset.frame, frame) / perPoint),
    enemy: clamp01(gained(enemy, reset.frame, frame) / perPoint),
  }
}

/** Les tics d'un camp, ou `null` si l'artefact ne le situe pas (voir `readHillHold`). */
function holdSeriesOf(
  timeline: ReplayScoreTimelineReady,
  teamId: number,
): ReadonlyArray<{ t: number; v: number }> | null {
  const hold = timeline.holdTicks?.find((h) => h.teamId === teamId)
  return hold?.ticks ?? null
}

/**
 * gained rend les tics pris dans `]since, frame]` — la série étant cumulative, c'est la
 * différence de ses valeurs aux deux bornes.
 *
 * `since < 0` (avant le premier point) fait courir l'accumulation depuis le début de l'axe.
 */
function gained(ticks: ReadonlyArray<{ t: number; v: number }>, since: number, frame: number): number {
  return valueAt(ticks, frame) - valueAt(ticks, since)
}

/** valueAt rend la dernière valeur en vigueur à `frame` — une série en escalier. */
function valueAt(ticks: ReadonlyArray<{ t: number; v: number }>, frame: number): number {
  let v = 0
  for (const tick of ticks) {
    if (tick.t > frame) break
    v = tick.v
  }
  return v
}

/** Le dernier point à `frame` ou avant : son image (`-1` avant le premier) et les camps qui marquent. */
interface HillReset {
  frame: number
  scorers: ReadonlySet<number>
}

/**
 * lastReset rend le DERNIER point marqué à `frame` ou avant, tous camps confondus — l'instant où
 * les deux jauges repartent de zéro — avec les camps dont la courbe monte à cette image.
 *
 * UN PALIER, PAS UNE ÉMISSION. Les séries sont cumulatives et ré-émettent leur valeur ; seule
 * une valeur PLUS HAUTE que la précédente du même camp est un point marqué.
 */
function lastReset(timeline: ReplayScoreTimelineReady, frame: number): HillReset {
  let last = -1
  let scorers = new Set<number>()
  for (const team of timeline.teams) {
    let prev = 0
    for (const tick of team.total) {
      if (tick.v <= prev) continue
      prev = tick.v
      if (tick.t > frame || tick.t < last) continue
      if (tick.t > last) scorers = new Set<number>()
      last = tick.t
      if (team.teamId !== undefined) scorers.add(team.teamId)
    }
  }
  return { frame: last, scorers }
}

function clamp01(v: number): number {
  if (!Number.isFinite(v) || v <= 0) return 0
  return v > 1 ? 1 : v
}
