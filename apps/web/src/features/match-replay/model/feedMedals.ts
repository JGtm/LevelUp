/**
 * feedMedals.ts — LE RATTACHEMENT D'UNE MÉDAILLE AU KILL DU MÊME ACTEUR, dans le fil. Sans React.
 *
 * Une médaille se rattache au kill du même acteur le plus proche, à `MEDAL_ATTACH_MS` près, sur l'horloge gameplay
 * commune aux deux sources ; celle qui ne se rattache à aucun kill devient une ligne seule du fil. Extrait de
 * `killFeedLogic`, qui assemble le fil.
 */
import type { MedalEvent } from './medalEvent'

/**
 * Tolérance de rattachement d'une médaille au kill du même acteur, en ms. Mesuré sur
 * 000d5950 : 42/44 à ≤ 500 ms, et AUCUNE de plus ne se rattache avant 5 000 ms — le seuil
 * est dans un plateau, pas sur une pente.
 */
export const MEDAL_ATTACH_MS = 500

/** Ce que le rattachement lit d'un kill du fil, et la liste de médailles où il pose celles qu'il reçoit. */
export interface MedalCarrier {
  xuid: string
  tMs: number
  medals: MedalEvent[]
}

/**
 * attachMedals pose chaque médaille sur le kill du même acteur le plus proche (à `MEDAL_ATTACH_MS` près) et rend
 * celles qui n'en ont aucun, dans l'ordre reçu.
 */
export function attachMedals(kills: readonly MedalCarrier[], medals: readonly MedalEvent[]): MedalEvent[] {
  const alone: MedalEvent[] = []
  for (const m of medals) {
    let best: MedalCarrier | null = null
    let bestDt = MEDAL_ATTACH_MS + 1
    for (const k of kills) {
      if (k.xuid !== m.xuid) continue
      const dt = Math.abs(k.tMs - m.tMs)
      if (dt < bestDt) {
        best = k
        bestDt = dt
      }
    }
    if (best && bestDt <= MEDAL_ATTACH_MS) best.medals.push(m)
    else alone.push(m)
  }
  return alone
}
