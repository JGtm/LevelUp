/**
 * focusFilter.test.ts — ce que Seul et Masquer font à chaque genre d'élément qui n'est pas un pion.
 *
 * Le joueur A tient le slot 1 (images 0-50) puis le slot 3 (60-100) ; B tient le slot 2 (0-100). La règle
 * est celle de `ownershipShown` (testée à part) : ici, on prouve QUELLES parties chaque élément déclare.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { buildFocusFilter, type FocusFilter } from './focusFilter'
import { testReplayDoc } from '../test/testDoc'

const tracks: NonNullable<ReplayDocument['tracks']> = [
  { slot: 1, team: 0, xuid: 'A', startFrame: 0, endFrame: 50, points: [{ t: 0, x: 0, y: 0 }, { t: 50, x: 1, y: 0 }] },
  { slot: 3, team: 0, xuid: 'A', startFrame: 60, endFrame: 100, points: [{ t: 60, x: 0, y: 0 }, { t: 100, x: 1, y: 0 }] },
  { slot: 2, team: 1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 5, y: 5 }, { t: 100, x: 6, y: 5 }] },
]

function harness(over: Partial<ReplayDocument> = {}) {
  const doc = testReplayDoc({ frameIntervalMs: 100, tracks, ...over })
  const player = (xuid: string): ReplayPlayer => ({ xuid, lives: doc.tracks.filter((t) => t.xuid === xuid) })
  const focus = (mode: 'solo' | 'hide', xuid = 'A') => ({
    player: player(xuid), follow: false, solo: mode === 'solo', hide: mode === 'hide',
  })
  const filter = (mode: 'solo' | 'hide', xuid = 'A'): FocusFilter => buildFocusFilter(doc, focus(mode, xuid))!
  return { doc, filter }
}

describe('buildFocusFilter', () => {
  it('ni Seul ni Masquer : aucun filtre', () => {
    const { doc } = harness()
    expect(buildFocusFilter(doc, null)).toBeNull()
    const sansAction = { player: { xuid: 'A', lives: [] }, follow: true, solo: false, hide: false }
    expect(buildFocusFilter(doc, sansAction)).toBeNull()
  })
})

describe('armes au sol — lâcheur et ramasseur', () => {
  const arme = (over: object) => ({ dropper: -1, picker: -1, t0: 20, t1: 40, ...over })

  it('une arme lâchée par le joueur : Seul la garde, Masquer la retire', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 1 }))).toBe(true)
    expect(filter('hide').groundWeapon(arme({ dropper: 1 }))).toBe(false)
  })

  it('lâchée À LA MORT (une image après la fin de la vie) : elle reste au joueur qui vient de mourir', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 1, t0: 51 }))).toBe(true)
    expect(filter('hide').groundWeapon(arme({ dropper: 1, t0: 51 }))).toBe(false)
  })

  it('lâchée par un autre, reprise par le joueur : elle est aux deux', () => {
    const { filter } = harness()
    const a = arme({ dropper: 2, picker: 3, t1: 70 })
    expect(filter('solo').groundWeapon(a)).toBe(true)
    expect(filter('hide').groundWeapon(a)).toBe(false)
  })

  it('lâchée par un autre : Seul la retire, Masquer la garde', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 2 }))).toBe(false)
    expect(filter('hide').groundWeapon(arme({ dropper: 2 }))).toBe(true)
  })

  it('sans lâcheur ni ramasseur lu (arme de la carte) : elle n’est à personne, elle reste', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({}))).toBe(true)
    expect(filter('hide').groundWeapon(arme({}))).toBe(true)
  })

  it('lâchée par B (slot 2, toute la partie) : jamais à A, même tard dans le match', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 2, t0: 80 }))).toBe(false)
  })
})
