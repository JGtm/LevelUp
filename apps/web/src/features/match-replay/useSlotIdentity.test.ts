/**
 * useSlotIdentity.test.ts — CE QU'UNE VIE SANS PROPRIÉTAIRE VAUT POUR LE CALQUE.
 *
 * Le film porte des traces qui ne sont personne : caméras et spectateurs de fin de partie.
 * `buildPlayers` les écarte (`if (!track.xuid) continue`), leur slot n'entre donc dans aucune
 * table d'identité. Ce test épingle ce que `colorOfSlot` en fait — `null`, la convention de
 * `MarkerStyle` pour « ne rien dessiner » — parce que le repli sur l'encre neutre qui existait
 * avant semait des pions gris ne désignant personne (retour utilisateur du 2026-08-20).
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'
import type { XuidMeta } from '@/features/match-view/xuidMeta'

import { testReplayDoc } from './test/testDoc'
import { useSlotIdentity } from './useSlotIdentity'

/** La vie d'un joueur identifié. */
const VIE_JOUEUR = {
  slot: 512,
  team: -1,
  xuid: 'A',
  startFrame: 0,
  endFrame: 100,
  points: [{ t: 0, x: 0, y: 0 }],
}

/** La vie que PERSONNE ne possède : pas de xuid, un slot bien à elle. */
const VIE_ANONYME = {
  slot: 999,
  team: -1,
  startFrame: 0,
  endFrame: 100,
  points: [{ t: 0, x: 1, y: 1 }],
}

const SCOREBOARD = [{ xuid: 'A', gamertag: 'Alpha' }] as unknown as MatchScoreboardRow[]
const META: XuidMeta = new Map([['A', { gamertag: 'Alpha', ally: true }]])

function identite() {
  const doc = testReplayDoc({
    roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha' }],
    tracks: [VIE_JOUEUR, VIE_ANONYME],
  })
  return renderHook(() =>
    useSlotIdentity({
      doc,
      scoreboard: SCOREBOARD,
      xuidMeta: META,
      marks: undefined,
      teamColorOf: (ally) => (ally ? 'encre-alliee' : 'encre-ennemie'),
      neutral: 'encre-neutre',
    }),
  ).result.current
}

describe('useSlotIdentity — la vie sans propriétaire ne se dessine pas', () => {
  it('rend null pour un slot sans propriétaire, JAMAIS l encre neutre', () => {
    const { colorOfSlot } = identite()
    // Le repli gris était le bug : il peignait un pion pour une caméra.
    expect(colorOfSlot(999)).toBeNull()
    expect(colorOfSlot(999)).not.toBe('encre-neutre')
  })

  it('rend bien sa couleur d équipe au slot dont le propriétaire est connu', () => {
    const { colorOfSlot } = identite()
    expect(colorOfSlot(512)).toBe('encre-alliee')
  })

  it('la table brute ne porte que les slots possédés', () => {
    const { slotColors } = identite()
    expect(slotColors.has(512)).toBe(true)
    expect(slotColors.has(999)).toBe(false)
  })

  it('la marque et le nom manquent aussi, sans jamais être inventés', () => {
    const { markOfSlot, nameOfSlot, sideOfSlot } = identite()
    expect(markOfSlot(999)).toBeUndefined()
    expect(nameOfSlot(999)).toBeNull()
    expect(sideOfSlot(999)).toBeNull()
  })
})
