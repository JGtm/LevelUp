/**
 * lifeEndProof.ts — LA PREUVE QU'UNE FIN DE VIE SANS FRAG EST UNE MORT.
 *
 * Une fin de vie qu'aucun frag ne revendique est soit une mort (chute, suicide, source de dégât sans
 * pictogramme), soit une fin de manche : le jeu supprime alors le corps encore commandé et replace les
 * joueurs, sans mort. Le film sait les distinguer, le client ne le déduit pas d'une fenêtre de temps :
 *
 *  - S1, le LÂCHER DE COMMANDE : le joueur lâche la commande de son corps (`endReason === 'released'`,
 *    instant `releaseFrame`). Un corps supprimé encore commandé (`'deleted'`) est une fin de manche.
 *  - S2, le COMPTE À REBOURS : le compte à rebours de réapparition du joueur (`respawnCountdowns[]`)
 *    commence à la fin de la vie. Une fin de manche replace les joueurs sans compte à rebours.
 *
 * Une vie est prouvée par l'un OU l'autre : les deux canaux se confirment à la frame près quand ils sont
 * tous deux lus, et chacun couvre des films que l'autre ne lit pas.
 *
 * Fonction pure, sans React. Un document qui ne publie ni `endReason` ni `respawnCountdowns` ne prouve rien
 * (`null`) : la mort neutre reste alors sans croix.
 */
import { frameToMs, trackWindow } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'

/**
 * Tolérance du lâcher de commande autour de la fin de piste, en ms : de -2 à +6 images de 33 ms, soit
 * l'écart maximal lu sur les morts certaines (vies revendiquées par un frag) de 99 films, aucun au-delà.
 * Écrite en ms pour ne pas dépendre de la cadence du document.
 */
export const UNIT_RELEASE_TOL_MS = { before: 66, after: 200 } as const

/**
 * Tolérance du début du compte à rebours de réapparition autour de la fin de piste, en ms : de -2 à +6
 * images de 33 ms sur les morts certaines des mêmes films ; au-delà de 10 images, le compte est celui d'une
 * autre mort du joueur. Deux canaux, deux mesures : la constante est distincte de celle du lâcher même si
 * les valeurs coïncident.
 */
export const RESPAWN_COUNTDOWN_TOL_MS = { before: 66, after: 200 } as const

/** Le signal qui prouve la mort : `release` (S1), `countdown` (S2), ou `null` (rien ne la prouve). */
export type LifeEndProof = 'release' | 'countdown' | null

/** La valeur de `endReason` que publie le décodeur quand le joueur lâche la commande de son corps. */
const END_REASON_RELEASED = 'released'

/**
 * Les débuts de compte à rebours d'un document, par xuid.
 *
 * `starts` : les frames de début (`from`) de chaque joueur résolu `filmIndex` -> roster -> xuid.
 * `unresolved` : les comptes que l'on ne peut attribuer à personne (index de film porté par plusieurs
 * entrées du roster, entrée sans xuid ou bot, index absent du roster) ; ils ne servent pas.
 */
export interface CountdownStarts {
  starts: Map<string, number[]>
  unresolved: number
}

/**
 * countdownStartsByXuid construit, une fois par document, les débuts de compte à rebours de chaque joueur.
 * Un document sans `respawnCountdowns` rend une table vide.
 */
export function countdownStartsByXuid(
  doc: Pick<ReplayDocumentReady, 'roster' | 'respawnCountdowns'>,
): CountdownStarts {
  const starts = new Map<string, number[]>()
  let unresolved = 0
  if (doc.respawnCountdowns.length === 0) return { starts, unresolved }

  const entriesByIndex = new Map<number, typeof doc.roster>()
  for (const r of doc.roster) {
    const list = entriesByIndex.get(r.filmIndex)
    if (list) list.push(r)
    else entriesByIndex.set(r.filmIndex, [r])
  }
  for (const c of doc.respawnCountdowns) {
    const entries = entriesByIndex.get(c.filmIndex)
    const owner = entries?.length === 1 ? entries[0] : undefined
    if (!owner || owner.bot || !owner.xuid) {
      unresolved++
      continue
    }
    const list = starts.get(owner.xuid)
    if (list) list.push(c.from)
    else starts.set(owner.xuid, [c.from])
  }
  return { starts, unresolved }
}

/** Vrai quand `deltaMs` (instant du signal moins fin de piste) tombe dans la tolérance. */
function withinTolerance(deltaMs: number, tol: { before: number; after: number }): boolean {
  return deltaMs >= -tol.before && deltaMs <= tol.after
}

/**
 * lifeEndProof dit si la fin de la piste est prouvée comme une mort, et par quel signal. S1 d'abord (il lit
 * la fin de la vie elle-même), puis S2. `starts` vient de `countdownStartsByXuid` ; un joueur absent de la
 * table (sans index de film, ou sans compte) n'a que S1.
 *
 * La fin de piste est celle que le reste du rejeu emploie (`trackWindow(track).end`).
 */
export function lifeEndProof(
  track: ReplayTrackReady,
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
  starts: ReadonlyMap<string, readonly number[]>,
): LifeEndProof {
  const endMs = frameToMs(trackWindow(track).end, doc)

  if (track.endReason === END_REASON_RELEASED && track.releaseFrame !== undefined) {
    if (withinTolerance(frameToMs(track.releaseFrame, doc) - endMs, UNIT_RELEASE_TOL_MS)) return 'release'
  }

  const own = track.xuid ? starts.get(track.xuid) : undefined
  if (own?.some((from) => withinTolerance(frameToMs(from, doc) - endMs, RESPAWN_COUNTDOWN_TOL_MS))) {
    return 'countdown'
  }
  return null
}
