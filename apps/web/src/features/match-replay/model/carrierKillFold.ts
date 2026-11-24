/**
 * carrierKillFold.ts — LES ÉLIMINATIONS LIÉES À L'OBJECTIF NE SONT PLUS DES LIGNES À PART : ce sont des badges sur la
 * ligne du FRAG qui les a faites. Le fil dit déjà ce frag (tueur, arme, victime) ; la ligne d'objectif en était le
 * doublon, à quelques millisecondes près. C'est le cas de « a éliminé un porteur de drapeau » (badge `flagCarrierKill`)
 * et des sept autres éliminations que le décodeur nomme (`OBJECTIVE_KILL_STATS` : joueur qui rend le drapeau,
 * éliminations en rendant ou en portant le drapeau, défenseur et attaquant de base, VIP adverse, éliminations en
 * étant VIP), qui se lisent en infobulle de la ligne (`objectiveKills`).
 *
 * L'APPARIEMENT est contraint des deux côtés : le MÊME tueur (le xuid de l'action est celui du frag) et l'instant le
 * plus proche à `CARRIER_KILL_FRAG_TOL_MS` près ; un frag ne sert qu'UNE fois PAR STATISTIQUE (deux éliminations de
 * natures différentes peuvent se replier sur le même frag, jamais deux de la même nature). Une action sans frag
 * apparié GARDE sa ligne : ni xuid, ni frag de ce tueur à cet instant, ni frag encore libre — on ne perd jamais une
 * information. Aucun son : une élimination n'a pas de son d'objectif (`objectiveSound`, liste blanche).
 *
 * LE FIL REÇU N'EST PAS MODIFIÉ (`model.feed` reste entier : la frise et les effets de mort le lisent). Le frag
 * apparié est remplacé, dans la liste rendue, par une COPIE qui porte `flagCarrierKill` et `objectiveKills`.
 */
import type { ReplayFeedEntry } from './killFeedLogic'
import { OBJECTIVE_KILL_STATS, type ObjectiveFeedStat } from './objectiveFeedLogic'

/**
 * Tolérance de rattachement d'une action éliminatoire au frag du même tueur, en ms. Le film date l'action sur l'image
 * du frag : la tolérance absorbe l'arrondi d'une image et le décalage d'alignement des deux horloges, bien en deçà de
 * l'intervalle qui sépare deux frags d'un même joueur hors rafale. Entre les deux ordres de grandeur la distribution
 * mesurée est vide : le seuil est sur un plateau, pas sur une pente.
 */
export const CARRIER_KILL_FRAG_TOL_MS = 500

/** La statistique « porteur éliminé » : son badge est `flagCarrierKill`, celui des autres est `objectiveKills`. */
const CARRIER_KILLED_STAT = 'flag_carriers_killed'

/** Ce que `foldCarrierKills` rend : le fil avec ses badges, et les lignes d'événement restantes. */
export interface CarrierKillFold {
  feed: ReplayFeedEntry[]
  events: ReplayFeedEntry[]
}

/** La statistique et le xuid du tueur d'une action éliminatoire, ou `null` pour toute autre ligne ou sans xuid. */
function killActionOf(event: ReplayFeedEntry): { stat: ObjectiveFeedStat; killer: string } | null {
  const line = event.event
  if (!line || line.family !== 'objectives' || !('stat' in line) || !OBJECTIVE_KILL_STATS.has(line.stat)) return null
  return line.xuid ? { stat: line.stat, killer: line.xuid } : null
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
  taken: ReadonlySet<string>,
  stat: ObjectiveFeedStat,
  ms: number,
): number {
  let best = -1
  let bestGap = CARRIER_KILL_FRAG_TOL_MS + 1
  for (const i of candidates) {
    const gap = Math.abs(feed[i].replayMs - ms)
    if (!taken.has(`${i}|${stat}`) && gap < bestGap) {
      best = i
      bestGap = gap
    }
  }
  return best
}

/**
 * foldCarrierKills replie chaque action éliminatoire sur le frag de son tueur. Les actions sont prises dans l'ordre
 * du temps ; le frag le plus proche encore libre pour cette statistique l'emporte.
 */
export function foldCarrierKills(
  feed: readonly ReplayFeedEntry[],
  events: readonly ReplayFeedEntry[],
): CarrierKillFold {
  const byKiller = fragsByKiller(feed)
  const taken = new Set<string>()
  const stats = new Map<number, ObjectiveFeedStat[]>()
  const kept: ReplayFeedEntry[] = []
  for (const event of events) {
    const action = killActionOf(event)
    const frag = action === null ? -1 : nearestFreeFrag(feed, byKiller.get(action.killer) ?? [], taken, action.stat, event.replayMs)
    if (action === null || frag < 0) {
      kept.push(event)
      continue
    }
    taken.add(`${frag}|${action.stat}`)
    const list = stats.get(frag)
    if (list) list.push(action.stat)
    else stats.set(frag, [action.stat])
  }
  if (stats.size === 0) return { feed: [...feed], events: kept }
  return {
    feed: feed.map((entry, i) => {
      const folded = stats.get(i)
      if (!folded) return entry
      const others = folded.filter((s) => s !== CARRIER_KILLED_STAT)
      return {
        ...entry,
        ...(folded.includes(CARRIER_KILLED_STAT) ? { flagCarrierKill: true as const } : {}),
        ...(others.length > 0 ? { objectiveKills: others } : {}),
      }
    }),
    events: kept,
  }
}
