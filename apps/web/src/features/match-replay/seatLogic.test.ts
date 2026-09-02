/**
 * seatLogic.test.ts — le siège suit l'occupant.
 *
 * Le scénario de référence est LE MATCH TÉMOIN 1b2d9e08 (retour user 2026-09-02) :
 * Winterhawk quitte à T, Razzle arrive à T, même équipe — une seule fiche, deux occupants.
 * S'y ajoutent : la CHAÎNE (A → bot → B sur le même siège), le repli sans participation
 * (chacun son siège), et le rejoignant sans partant (nouveau siège, pas de rattachement
 * inventé).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayPlayer } from './rosterLogic'
import { buildSeats, seatOccupantAt } from './seatLogic'

const DOC = { frameIntervalMs: 100, frameCount: 6_000, originMs: 0 } as ReplayDocumentReady
const HEADER = { start_time: '2026-09-01T22:26:39Z' }

function joueur(xuid: string, side: string, board: Record<string, unknown> = {}): ReplayPlayer {
  return {
    xuid,
    filmName: xuid,
    lives: [],
    board: { xuid, gamertag: xuid, team_side: side, ...board } as ReplayPlayer['board'],
  }
}

describe('buildSeats — le scénario témoin', () => {
  it('le partant cède sa fiche au remplaçant de la même équipe, à l’image du relais', () => {
    // Winterhawk quitte à +275 s (22:31:14) ; Razzle arrive à la même seconde, équipe t1.
    const winterhawk = joueur('W', 't1', {
      left_in_progress: true,
      last_leave_time: '2026-09-01T22:31:14Z',
    })
    const razzle = joueur('bot:343 Razzle [bot]', 't1', {
      joined_in_progress: true,
      first_joined_time: '2026-09-01T22:31:14Z',
    })
    const autre = joueur('A', 't0')
    const seats = buildSeats([winterhawk, razzle, autre], HEADER, DOC)
    expect(seats).toHaveLength(2) // le remplaçant vit dans le siège du partant
    const siege = seats.find((s) => s.key === 'W')!
    expect(siege.occupants.map((o) => o.player.xuid)).toEqual(['W', 'bot:343 Razzle [bot]'])
    // 275 s à 100 ms la frame = image 2750.
    expect(siege.occupants[1].fromFrame).toBe(2750)
    expect(seatOccupantAt(siege, 2749).xuid).toBe('W')
    expect(seatOccupantAt(siege, 2750).xuid).toBe('bot:343 Razzle [bot]')
  })

  it('la CHAÎNE tient : A part, bot le remplace, bot part, B le remplace — un seul siège', () => {
    const a = joueur('A', 't0', { left_in_progress: true, last_leave_time: '2026-09-01T22:28:00Z' })
    const bot = joueur('bot:X [bot]', 't0', {
      joined_in_progress: true,
      first_joined_time: '2026-09-01T22:28:00Z',
      left_in_progress: true,
      last_leave_time: '2026-09-01T22:30:00Z',
    })
    const b = joueur('B', 't0', {
      joined_in_progress: true,
      first_joined_time: '2026-09-01T22:30:05Z',
    })
    const seats = buildSeats([a, bot, b], HEADER, DOC)
    expect(seats).toHaveLength(1)
    expect(seats[0].occupants.map((o) => o.player.xuid)).toEqual(['A', 'bot:X [bot]', 'B'])
  })

  it('équipes différentes = jamais appariés ; rejoignant sans partant = son propre siège', () => {
    const partantT0 = joueur('A', 't0', {
      left_in_progress: true,
      last_leave_time: '2026-09-01T22:28:00Z',
    })
    const arrivantT1 = joueur('J', 't1', {
      joined_in_progress: true,
      first_joined_time: '2026-09-01T22:28:00Z',
    })
    const seats = buildSeats([partantT0, arrivantT1], HEADER, DOC)
    expect(seats).toHaveLength(2)
    expect(seats.map((s) => s.occupants.length)).toEqual([1, 1])
  })

  it('sans en-tête (pas de repère absolu), chacun garde son siège — l’affichage d’avant', () => {
    const partant = joueur('A', 't0', {
      left_in_progress: true,
      last_leave_time: '2026-09-01T22:28:00Z',
    })
    const arrivant = joueur('J', 't0', {
      joined_in_progress: true,
      first_joined_time: '2026-09-01T22:28:00Z',
    })
    expect(buildSeats([partant, arrivant], null, DOC)).toHaveLength(2)
  })
})
