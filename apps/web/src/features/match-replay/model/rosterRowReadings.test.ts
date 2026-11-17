/**
 * Tests — les lectures propres à la ligne d'un joueur (`rosterRowReadings`) et la règle de la place en
 * attente (`seatWaitingReading`), purs : le compte à rebours dit sa source, la cellule de grenades ne
 * devine jamais le type qui partira, les munitions ne disent « pleines » qu'en infobulle.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import type { NowGrenade, NowWeapon } from './playerNowReadings'
import { seatWaitingReading } from './playerCardReadings'
import { ammoParts, respawnText, rowGrenade } from './rosterRowReadings'

const T = REPLAY_TEXT.fr
const DOC = { frameIntervalMs: 100 }

function grenade(rank: number, count: number, stowed = false): NowGrenade {
  return { rank, name: `type ${rank}`, count, stowed, icon: null }
}

describe('respawnText', () => {
  it('écrit le retour lu : la vie suivante part 40 images plus tard, à 100 ms l’image', () => {
    expect(respawnText({ respawnFrame: 180 }, DOC, 140, T)).toBe(`${T.respawnIn} 4.0 s`)
  })

  it('sans vie suivante : « Hors film · ne revient plus », jamais un délai deviné', () => {
    expect(respawnText({ respawnFrame: -1 }, DOC, 140, T)).toBe(`${T.goneLabel} · ${T.goneValue}`)
  })

  it('un retour déjà passé ne rend jamais un délai négatif', () => {
    expect(respawnText({ respawnFrame: 100 }, DOC, 140, T)).toBe(`${T.respawnIn} 0.0 s`)
  })
})

describe('rowGrenade', () => {
  it('aucune grenade portée : cellule vide', () => {
    expect(rowGrenade([])).toBeNull()
  })

  it('un type désigné : sa vignette et son stock', () => {
    const r = rowGrenade([grenade(0, 2), grenade(1, 3, true)])
    expect(r?.count).toBe(2)
    expect(r?.selected?.rank).toBe(0)
  })

  it('un seul type porté : désigné par déduction', () => {
    expect(rowGrenade([grenade(1, 2)])).toMatchObject({ count: 2, selected: { rank: 1 } })
  })

  it('plusieurs types sans sélecteur lu : le total porté, sans type deviné', () => {
    expect(rowGrenade([grenade(0, 1), grenade(1, 2)])).toEqual({ count: 3, selected: null })
  })
})

describe('ammoParts', () => {
  const arme = (ammo: NowWeapon['ammo']): Pick<NowWeapon, 'ammo'> => ({ ammo })

  it('un chargeur : le compte et sa réserve', () => {
    expect(ammoParts(arme({ kind: 'count', mag: 36, res: 108 }))).toEqual({ value: '36', reserve: '/108' })
  })

  it('une arme à charge : le pour-cent restant', () => {
    expect(ammoParts(arme({ kind: 'charge', pct: 87 }))).toEqual({ value: '87 %', reserve: '' })
  })

  it('« pleines » et rien de lu ne s’écrivent pas', () => {
    expect(ammoParts(arme({ kind: 'full' }))).toBeNull()
    expect(ammoParts(arme(null))).toBeNull()
  })
})

describe('seatWaitingReading', () => {
  it('sans occupant : la place est libre et ne montre aucun nom', () => {
    expect(seatWaitingReading(null, T)).toEqual({ name: null, label: T.seatVacant, hint: T.seatVacantHint })
  })
})
