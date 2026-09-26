/**
 * usageLobbyTrackModel.test.ts — buildLobbyTrack, la piste du lobby découpée par joueur.
 * Extrait de `usageLogic.test.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import { USAGE_TEXT } from './usageI18n'
import { buildLobbyTrack } from './usageLobbyTrackModel'

const t = USAGE_TEXT.fr

describe('buildLobbyTrack — piste du lobby découpée par joueur', () => {
  const squadPlayers = [
    { xuid: '1', gamertag: 'Madina97294' },
    { xuid: '2', gamertag: 'Chocoboflor' },
  ]

  it('découpe moi + coéquipiers + reste d équipe + eux (hachuré)', () => {
    const track = buildLobbyTrack({
      meLabel: 'JGtm',
      shares: { player_total: 9, team_total: 20, lobby_total: 43 },
      squadPlayers,
      squadShares: [
        { xuid: '1', total: 6 },
        { xuid: '2', total: 3 },
      ],
      t,
      locale: 'fr',
    })
    expect(track).not.toBeNull()
    expect(track!.map((s) => [s.kind, s.count])).toEqual([
      ['me', 9],
      ['squad', 6],
      ['squad', 3],
      ['team-rest', 2],
      ['enemy', 23],
    ])
    // Le segment adverse n a NI nom d équipe NI couleur : son libellé est anonyme.
    expect(track!.at(-1)!.label).toBe(t.segEnemy)
  })

  it('refuse de découper sans frontière nous/eux (team_total absent)', () => {
    const track = buildLobbyTrack({
      meLabel: 'JGtm',
      shares: { player_total: 9, lobby_total: 43 },
      squadPlayers: [],
      squadShares: null,
      t,
      locale: 'fr',
    })
    expect(track).toBeNull()
  })

  it('borne les résidus à zéro (scopes joueur/équipe légèrement disjoints)', () => {
    const track = buildLobbyTrack({
      meLabel: 'JGtm',
      shares: { player_total: 25, team_total: 20, lobby_total: 22 },
      squadPlayers: [],
      squadShares: null,
      t,
      locale: 'fr',
    })
    expect(track!.every((s) => s.count >= 0)).toBe(true)
  })
})
