/**
 * carrierKillFold.ts — « A ÉLIMINÉ UN PORTEUR DE DRAPEAU » N'EST PLUS UNE LIGNE À PART : c'est un
 * badge sur la ligne du FRAG qui l'a fait. Le fil dit déjà ce frag (tueur, arme, victime) ; la ligne
 * d'objectif en était le doublon, à quelques millisecondes près.
 *
 * L'APPARIEMENT est contraint des deux côtés : le MÊME tueur (le xuid de l'action est celui du
 * frag) et l'instant le plus proche à `CARRIER_KILL_FRAG_TOL_MS` près ; un frag ne sert qu'UNE fois.
 * Une action sans frag apparié GARDE sa ligne : ni xuid, ni frag de ce tueur à cet instant, ni frag
 * encore libre — on ne perd jamais une information.
 *
 * LE FIL REÇU N'EST PAS MODIFIÉ (`model.feed` reste entier : la frise et les effets de mort le
 * lisent). Le frag apparié est remplacé, dans la liste rendue, par une COPIE qui porte
 * `flagCarrierKill`.
 */
import type { ReplayFeedEntry } from './killFeedLogic'

/**
 * Tolérance de rattachement d'une action « porteur éliminé » au frag du même tueur, en ms. Le film
 * date l'action sur l'image du frag : la tolérance absorbe l'arrondi d'une image et le décalage
 * d'alignement des deux horloges, bien en deçà de l'intervalle qui sépare deux frags d'un même
 * joueur hors rafale. Entre les deux ordres de grandeur la distribution mesurée est vide : le seuil
 * est sur un plateau, pas sur une pente.
 */
export const CARRIER_KILL_FRAG_TOL_MS = 500

/** La statistique du décodeur que ce module replie (cf. `objectiveFeedLogic`). */
const CARRIER_KILLED_STAT = 'flag_carriers_killed'

/** Ce que `foldCarrierKills` rend : le fil avec ses badges, et les lignes d'événement restantes. */
export interface CarrierKillFold {
  feed: ReplayFeedEntry[]
  events: ReplayFeedEntry[]
}

/** Le xuid du tueur d'une action « porteur éliminé », ou `null` pour toute autre ligne ou sans xuid. */
function carrierKillerOf(event: ReplayFeedEntry): string | null {
  const line = event.event
  if (!line || line.family !== 'objectives' || !('stat' in line) || line.stat !== CARRIER_KILLED_STAT) return null
  return line.xuid || null
}

/** Les index des frags du fil, par xuid de tueur. */
function fragsByKiller(feed: readonly ReplayFeedEntry[]): Map<string, number[]> {
  const byKiller = new Map<string, number[]>()
  feed.forEach((entry, i) => {
    if (!entry.kill) return
    const list = byKiller.get(entry.kill.xuid)
    if (list) list.push(i)
    else byKiller.set(entry.kill.xuid, [i])
  })
  return byKiller
}

/** Le frag libre le plus proche de `ms` parmi `candidates`, dans la tolérance, ou `-1`. */
function nearestFreeFrag(
  feed: readonly ReplayFeedEntry[],
  candidates: readonly number[],
  taken: ReadonlySet<number>,
  ms: number,
): number {
  let best = -1
  let bestGap = CARRIER_KILL_FRAG_TOL_MS + 1
  for (const i of candidates) {
    const gap = Math.abs(feed[i].replayMs - ms)
    if (!taken.has(i) && gap < bestGap) {
      best = i
      bestGap = gap
    }
  }
  return best
}

/**
 * foldCarrierKills replie chaque action « porteur éliminé » sur le frag de son tueur. Les actions
 * sont prises dans l'ordre du temps ; le frag le plus proche encore libre l'emporte.
 */
export function foldCarrierKills(
  feed: readonly ReplayFeedEntry[],
  events: readonly ReplayFeedEntry[],
): CarrierKillFold {
  const byKiller = fragsByKiller(feed)
  const taken = new Set<number>()
  const kept: ReplayFeedEntry[] = []
  for (const event of events) {
    const killer = carrierKillerOf(event)
    const frag = killer === null ? -1 : nearestFreeFrag(feed, byKiller.get(killer) ?? [], taken, event.replayMs)
    if (frag < 0) kept.push(event)
    else taken.add(frag)
  }
  if (taken.size === 0) return { feed: [...feed], events: kept }
  return { feed: feed.map((entry, i) => (taken.has(i) ? { ...entry, flagCarrierKill: true } : entry)), events: kept }
}
