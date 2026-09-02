/**
 * seatLogic — LE SIÈGE : la fiche suit l'OCCUPANT, pas le joueur (retour user 2026-09-02).
 *
 * LE CAS QUI A IMPOSÉ CE MODÈLE (match témoin 1b2d9e08) : Winterhawk7676 quitte à 22:31:14,
 * 343 Razzle le remplace À LA MÊME SECONDE. Une fiche par JOUEUR donnait 7 fiches vivantes
 * plus une fiche fantôme « hors film » jusqu'à la fin — alors que la partie reste un 4v4.
 * LE SIÈGE est l'unité stable : un 4v4 a huit sièges, et la fiche d'un siège montre son
 * occupant À L'IMAGE LUE — Winterhawk avant le relais, Razzle après. Et ça chaîne : un
 * même siège peut changer d'occupant des dizaines de fois (A → bot → B → …).
 *
 * L'APPARIEMENT VIENT DE LA PARTICIPATION API, la source précise : un PARTANT
 * (`left_in_progress` + `last_leave_time`) s'apparie au REJOIGNANT de la MÊME ÉQUIPE
 * (`joined_in_progress` + `first_joined_time`) dont l'arrivée est la plus proche du départ
 * (fenêtre bornée — sur le témoin, l'écart est de 0 s). Les chaînes se construisent en
 * suivant les liens : le départ d'un remplaçant s'apparie au rejoignant suivant.
 *
 * SANS PARTICIPATION (matchs anciens, en-tête absent), chaque joueur garde SON siège —
 * exactement l'affichage d'avant ce lot, fiche « hors film » comprise : on n'invente pas
 * un relais que la base ne dit pas.
 *
 * L'INSTANT DU RELAIS se convertit en frame par la même doctrine que les médias et la
 * présence : frame((abs − start_time − originMs) / frameIntervalMs).
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

import type { PresenceHeader } from './presenceFeed'
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'

/** Fenêtre d'appariement départ↔arrivée. Large à dessein : le jeu peut mettre du temps à
 * combler un siège, et deux relais simultanés restent départagés par la proximité. */
export const SEAT_RELAY_WINDOW_MS = 120_000

/** Un occupant d'un siège : le joueur, et l'image à partir de laquelle il tient la fiche. */
export interface SeatOccupant {
  player: ReplayPlayer
  /** Image du relais ; 0 pour l'occupant d'origine. */
  fromFrame: number
}

/** Un siège : la suite de ses occupants, dans l'ordre du temps. */
export interface ReplaySeat {
  /** Clé stable de rendu : l'identité de l'occupant D'ORIGINE. */
  key: string
  side: string | null
  occupants: SeatOccupant[]
}

/** L'occupant d'un siège à une image : le dernier dont le relais est advenu. */
export function seatOccupantAt(seat: ReplaySeat, frame: number): ReplayPlayer {
  let cur = seat.occupants[0].player
  for (const o of seat.occupants) {
    if (o.fromFrame > frame) break
    cur = o.player
  }
  return cur
}

/**
 * buildSeats — les sièges, à partir des joueurs joints et de la participation API.
 *
 * L'ordre des sièges est celui des joueurs d'ORIGINE dans `players` (stable). Un
 * rejoignant apparié disparaît de la liste des sièges (il vit dans celui de son
 * prédécesseur) ; un rejoignant SANS partant apparié garde son propre siège (remplissage
 * d'une place restée vide — on n'a personne à qui le rattacher).
 */
export function buildSeats(
  players: readonly ReplayPlayer[],
  header: PresenceHeader | null | undefined,
  doc: ReplayDocumentReady,
): ReplaySeat[] {
  const successor = pairSuccessions(players, header, doc)
  const replaced = new Set<string>()
  for (const s of successor.values()) replaced.add(s.player.xuid)
  const seats: ReplaySeat[] = []
  for (const p of players) {
    if (replaced.has(p.xuid)) continue // il vit dans le siège de son prédécesseur
    const occupants: SeatOccupant[] = [{ player: p, fromFrame: 0 }]
    // La chaîne : chaque occupant peut avoir son propre successeur.
    for (let cur = p; ; ) {
      const next = successor.get(cur.xuid)
      if (!next) break
      occupants.push(next)
      cur = next.player
    }
    seats.push({ key: p.xuid, side: seatSide(occupants), occupants })
  }
  return seats
}

/** Le camp du siège : celui de son premier occupant à en porter un (même équipe partout
 * par construction de l'appariement). */
function seatSide(occupants: SeatOccupant[]): string | null {
  for (const o of occupants) {
    const side = o.player.board?.team_side
    if (side != null) return side
  }
  return null
}

/** partant.xuid -> son successeur (joueur + image du relais). */
function pairSuccessions(
  players: readonly ReplayPlayer[],
  header: PresenceHeader | null | undefined,
  doc: ReplayDocumentReady,
): Map<string, SeatOccupant> {
  const out = new Map<string, SeatOccupant>()
  const matchStartMs = parseInstant(header?.start_time)
  if (matchStartMs === null) return out
  const leavers = players.filter(
    (p) => p.board?.left_in_progress && parseInstant(p.board.last_leave_time) !== null,
  )
  const joiners = players
    .filter((p) => p.board?.joined_in_progress && parseInstant(p.board.first_joined_time) !== null)
    .sort(
      (a, b) => parseInstant(a.board?.first_joined_time)! - parseInstant(b.board?.first_joined_time)!,
    )
  const taken = new Set<string>()
  for (const j of joiners) {
    const joinMs = parseInstant(j.board?.first_joined_time)!
    let best: ReplayPlayer | null = null
    let bestGap = SEAT_RELAY_WINDOW_MS + 1
    for (const l of leavers) {
      if (taken.has(l.xuid) || l.xuid === j.xuid) continue
      if (l.board?.team_side !== j.board?.team_side) continue
      const gap = Math.abs(joinMs - parseInstant(l.board?.last_leave_time)!)
      if (gap < bestGap) {
        best = l
        bestGap = gap
      }
    }
    if (!best) continue
    taken.add(best.xuid)
    out.set(best.xuid, { player: j, fromFrame: frameOfAbs(joinMs, matchStartMs, doc) })
  }
  return out
}

/** L'image d'un instant absolu — la conversion des médias et de la présence, bornée à 0. */
function frameOfAbs(absMs: number, matchStartMs: number, doc: ReplayDocumentReady): number {
  const origin = Number.isFinite(doc.originMs) ? (doc.originMs as number) : 0
  const interval = doc.frameIntervalMs || 100
  return Math.max(0, Math.round((absMs - matchStartMs - origin) / interval))
}

function parseInstant(iso: string | null | undefined): number | null {
  if (!iso) return null
  const ms = Date.parse(iso)
  return Number.isFinite(ms) ? ms : null
}

/** Les sièges rangés par camp — le pendant de `groupByTeam`, sur l'unité SIÈGE. */
export interface ReplaySeatGroup {
  side: string | null
  seats: ReplaySeat[]
}

export function groupSeatsByTeam(seats: readonly ReplaySeat[]): ReplaySeatGroup[] {
  const groups = new Map<string, ReplaySeatGroup>()
  for (const s of seats) {
    const key = s.side ?? ''
    let g = groups.get(key)
    if (!g) {
      g = { side: s.side, seats: [] }
      groups.set(key, g)
    }
    g.seats.push(s)
  }
  return [...groups.values()].sort((a, b) => (a.side ?? '￿').localeCompare(b.side ?? '￿'))
}

// MatchScoreboardRow n'est référencé qu'au travers de ReplayPlayer.board — l'import type
// garde la signature lisible pour le lecteur du fichier.
export type { MatchScoreboardRow as SeatScoreboardRow }
