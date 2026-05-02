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
  it('expose le token compare-a pour le main', () => {
    expect(SQUAD_MAIN_PLAYER_TOKEN).toBe('compare-a')
  })

  it('expose 3 tokens teammates dans l\'ordre attendu (= ordre du combobox)', () => {
    expect(SQUAD_TEAMMATE_COLOR_TOKENS).toEqual(['narrative-dominant', 'perf-tier-3', 'divergent-pos'])
  })
})

describe('getSquadTeammateColors', () => {
  it('retourne 3 couleurs résolues par défaut', () => {
    expect(getSquadTeammateColors()).toEqual([
      'hex(narrative-dominant)',
      'hex(perf-tier-3)',
      'hex(divergent-pos)',
    ])
  })
})

describe('getSquadPlayerColors', () => {
  it('mappe main → compare-a et teammates dans l\'ordre', () => {
    const map = getSquadPlayerColors('Me', ['F1', 'F2'])
    expect(map.Me).toBe('hex(compare-a)')
    expect(map.F1).toBe('hex(narrative-dominant)')
    expect(map.F2).toBe('hex(perf-tier-3)')
  })

  it('aucun teammate → seul le main', () => {
    expect(getSquadPlayerColors('Me', [])).toEqual({ Me: 'hex(compare-a)' })
  })

  it('main vide → ne pose pas de clé vide', () => {
    expect(getSquadPlayerColors('', ['F1'])).toEqual({ F1: 'hex(narrative-dominant)' })
  })

  it('cycle si plus de teammates que de tokens (modulo)', () => {
    const map = getSquadPlayerColors('Me', ['F1', 'F2', 'F3', 'F4'])
    expect(map.F4).toBe('hex(narrative-dominant)') // wrap-around
  })
})
