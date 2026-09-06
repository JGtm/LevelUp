/**
 * hillHoldLogic — LA PROGRESSION DE GARDE DE LA COLLINE, au frame lu.
 *
 * EN KOTH IL N'Y A PAS DE CAPTURE. La colline se prend instantanément dès qu'un joueur y entre
 * sans adversaire dedans ; ce qui marque, c'est de la TENIR. Ce module dit où en est chaque camp
 * vers le point suivant.
 *
 * ELLE EST LUE, PAS RECONSTRUITE — et c'est tout ce qui compte ici. La série vient de
 * `scoreTimeline.holdTicks`, que l'artefact publie depuis le compteur du jeu lui-même
 * (`comp 23 A` du statborg, qui reproduit `StrongholdScoringTicks` de l'API exactement joueur
 * par joueur). Ce module ne fait donc que TROIS choses, et aucune n'est une estimation :
 *
 *   la remise à zéro   la colline TOURNE à chaque point ; les instants de point sont les
 *                      paliers des courbes d'équipe. La barre repart de zéro à chacun.
 *   le différentiel    `ticks(frame) − ticks(dernier point)`, puisque la série est cumulative
 *                      sur tout le match.
 *   le dénominateur    `scoreTimeline.holdTicksPerPoint` — 35, un COMPTE et non un réglage :
 *                      le camp qui marque rend exactement 35 sur 15 périodes sur 16 (4 films,
 *                      4 cartes), quand le camp qui ne marque pas rend 1 à 25 et jamais 35.
 *
 * UNE VERSION ANTÉRIEURE DE CE MODULE INTÉGRAIT LES INTERVALLES DE PROPRIÉTÉ avec un seuil de
 * 43 s. C'était une reconstruction, et elle était fausse d'environ 20 % — l'écart n'a pas été
 * corrigé en ajustant la constante (ce qui aurait été caler une mesure sur un rendu) mais en
 * changeant de SOURCE. L'historique des trois lots vit dans
 * `.ai/V7.5/PLAN_KOTH_GARDE_VIVANTE_2026-08-30.md`.
 *
 * LES CAMPS SE LISENT DANS LA MÊME CONVENTION QUE LE BANDEAU. `holdTicks[].teamId` et
 * `teams[].teamId` portent tous deux l'index d'équipe du REGISTRE ; ce module reçoit les
 * identifiants que le bandeau a déjà résolus par le tableau de bord, et se tait s'il ne les
 * trouve pas.
 *
 * Module PUR : ni React, ni DOM, ni couleur.
 */
import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

/** Le document, vu par ce module : deux champs, pas un de plus. */
export interface HillHoldDocument {
  scoreTimeline?: ReplayScoreTimelineReady
}

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
 * absente ne ment pas, une jauge à zéro si) : artefact sans série de garde (mode sans colline —
 * le producteur ne la construit que sur une variante déclarée —, ou artefact antérieur au
 * champ), variante sans dénominateur mesuré (le KOTH CLASSÉ est dans ce cas), ou camps que le
 * calque ne situe pas.
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

  const since = lastResetFrame(timeline, frame)
  return {
    ally: clamp01(gained(ally, since, frame) / perPoint),
    enemy: clamp01(gained(enemy, since, frame) / perPoint),
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

/**
 * lastResetFrame rend l'image du DERNIER point marqué à `frame` ou avant, tous camps confondus —
 * l'instant où les deux jauges sont reparties de zéro. `-1` avant le premier point.
 *
 * UN PALIER, PAS UNE ÉMISSION. Les séries sont cumulatives et ré-émettent leur valeur ; seule
 * une valeur PLUS HAUTE que la précédente du même camp est un point marqué.
 */
function lastResetFrame(timeline: ReplayScoreTimelineReady, frame: number): number {
  let last = -1
  for (const team of timeline.teams) {
    let prev = 0
    for (const tick of team.total) {
      if (tick.v <= prev) continue
      prev = tick.v
      if (tick.t <= frame && tick.t > last) last = tick.t
    }
  }
  return last
}

function clamp01(v: number): number {
  if (!Number.isFinite(v) || v <= 0) return 0
  return v > 1 ? 1 : v
}
