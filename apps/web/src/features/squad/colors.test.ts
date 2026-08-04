/**
 * colors.test.ts — helper de couleurs squad partagées.
 */
import { describe, it, expect, vi } from 'vitest'

vi.mock('@/lib/accessibility', () => ({
  getSeriesColors: (n: number, tokens: string[]) =>
    Array.from({ length: n }, (_, i) => `hex(${tokens[i % tokens.length]})`),
}))

import {
  SQUAD_MAIN_PLAYER_TOKEN,
  SQUAD_TEAMMATE_COLOR_TOKENS,
  getSquadPlayerColors,
  getSquadTeammateColors,
} from './colors'

describe('SQUAD_MAIN_PLAYER_TOKEN / SQUAD_TEAMMATE_COLOR_TOKENS', () => {
  it('expose le token squad-player-1 pour le main', () => {
    expect(SQUAD_MAIN_PLAYER_TOKEN).toBe('squad-player-1')
  })

  it('expose 3 tokens teammates dans l\'ordre attendu (= ordre du combobox)', () => {
    expect(SQUAD_TEAMMATE_COLOR_TOKENS).toEqual(['squad-player-2', 'squad-player-3', 'squad-player-4'])
  })
})

describe('getSquadTeammateColors', () => {
  it('retourne 3 couleurs résolues par défaut', () => {
    expect(getSquadTeammateColors()).toEqual([
      'hex(squad-player-2)',
      'hex(squad-player-3)',
      'hex(squad-player-4)',
    ])
  })
})

describe('getSquadPlayerColors', () => {
  it('mappe main → squad-player-1 et teammates dans l\'ordre', () => {
    const map = getSquadPlayerColors('Me', ['F1', 'F2'])
    expect(map.Me).toBe('hex(squad-player-1)')
    expect(map.F1).toBe('hex(squad-player-2)')
    expect(map.F2).toBe('hex(squad-player-3)')
  })

  it('aucun teammate → seul le main', () => {
    expect(getSquadPlayerColors('Me', [])).toEqual({ Me: 'hex(squad-player-1)' })
  })

  it('main vide → ne pose pas de clé vide', () => {
    expect(getSquadPlayerColors('', ['F1'])).toEqual({ F1: 'hex(squad-player-2)' })
  })

  it('cycle si plus de teammates que de tokens (modulo)', () => {
    const map = getSquadPlayerColors('Me', ['F1', 'F2', 'F3', 'F4'])
    expect(map.F4).toBe('hex(squad-player-2)') // wrap-around
  })
})
