/**
 * replaySoundCursor.ts — LA LECTURE DE LA PISTE : quand un événement part, et quand il ne
 * part pas. Rien de ce qui SONNE ne vit ici (le manifeste et la piste sont dans
 * replaySound.ts) — seulement la comptabilité du curseur.
 *
 * EXTRAIT DE `replaySound.ts` LE 2026-08-18 (lot R2-G) pour payer l'ajout des explosions de
 * fin de vol : le fichier d'origine annonçait trois sujets (« ce qui sonne, quand, et
 * comment le curseur ne rejoue rien deux fois ») et arrivait au plafond de 500 lignes. Le
 * troisième est le seul qui ne touche jamais au manifeste : il ne connaît de la piste que
 * son instant et son ordre.
 *
 * LA RÈGLE QUE CE FICHIER TIENT, ET ELLE EST LA SEULE : un événement part UNE fois, et un
 * DÉPLACEMENT (scrub, rebouclage, reprise après pause) ne rejoue pas ce qu'il a enjambé.
 *
 * Pas de React, pas de Web Audio : logique pure, testée (replaySound.test.ts).
 */
import type { ReplaySoundEvent } from './replaySound'

/**
 * Saut au-delà duquel une avance n'est PAS une lecture continue mais un déplacement
 * (scrub, retour au début, reprise après pause longue) : le curseur se RECALE sans rien
 * jouer — rejouer d'un coup tous les sons enjambés ferait un mur de bruit. À 4x, un pas
 * d'animation avance de ~70 ms de rejeu : 1 s de marge ne peut pas confondre les deux.
 */
export const SOUND_RESYNC_JUMP_MS = 1_000

/**
 * Vitesse de lecture au-delà de laquelle le son se TAIT.
 *
 * POURQUOI. Un son tient ~1 s de temps réel quelle que soit la vitesse : à 4×, cette
 * seconde couvre 4 s de match, et les éliminations d'un même échange (2 à 4 s d'écart) se
 * recouvrent en permanence. Ce qu'on entendrait alors n'est plus le rythme du match mais
 * celui du lecteur. À 2×, elles restent distinctes — c'est la borne.
 */
export const SOUND_MAX_SPEED = 2

/** soundPlaysAtSpeed — le son a-t-il un sens à cette vitesse de lecture ? */
export function soundPlaysAtSpeed(multiplier: number): boolean {
  return multiplier <= SOUND_MAX_SPEED
}

/** Le curseur de lecture sonore : dernier instant servi, index du prochain événement. */
export interface SoundCursor {
  ms: number
  idx: number
}

/** resyncSoundCursor pose le curseur À l'instant donné : rien avant lui ne jouera. */
export function resyncSoundCursor(timeline: ReplaySoundEvent[], ms: number): SoundCursor {
  // Premier index strictement postérieur à `ms` (recherche binaire, timeline triée).
  let lo = 0
  let hi = timeline.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (timeline[mid].ms <= ms) lo = mid + 1
    else hi = mid
  }
  return { ms, idx: lo }
}

/**
 * advanceSoundCursor avance le curseur à `nowMs` et rend les événements à jouer.
 *
 * Lecture continue (avance courte) : tout événement dans (cursor.ms, nowMs] part UNE
 * fois. Recul ou saut long : recalage silencieux — le son accompagne la lecture, il ne
 * raconte pas ce qu'on a enjambé.
 */
export function advanceSoundCursor(
  timeline: ReplaySoundEvent[],
  cursor: SoundCursor,
  nowMs: number,
): { cursor: SoundCursor; fire: ReplaySoundEvent[] } {
  if (nowMs < cursor.ms || nowMs - cursor.ms > SOUND_RESYNC_JUMP_MS) {
    return { cursor: resyncSoundCursor(timeline, nowMs), fire: [] }
  }
  const fire: ReplaySoundEvent[] = []
  let idx = cursor.idx
  while (idx < timeline.length && timeline[idx].ms <= nowMs) {
    fire.push(timeline[idx])
    idx++
  }
  return { cursor: { ms: nowMs, idx }, fire }
}
