/**
 * Tests Vitest pour la logique pure du MatchScoreboard :
 *  - getExtremes : min/max par colonne sur le lobby
 *  - cellState : best/worst/neutral en respectant `inverted` (deaths/dmg taken)
 *  - getMvpLvp : sélection MVP/LVP au niveau LOBBY (≥2 best / ≥2 worst, pas
 *    par équipe)
 *
 * Le rendu visuel n'est pas testé ici — focus algorithmique.
 */
import { describe, it, expect } from 'vitest'

import {
  cellState,
  getExtremes,
  getMvpLvp,
  type ColDef,
  type Extremes,
} from './MatchScoreboard.logic'
import type { MatchScoreboardRow } from '@/lib/api/types'

// Stub minimal pour MatchScoreboardRow : seuls les champs numériques utilisés
// par les helpers sont nécessaires.
function row(xuid: string, overrides: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return {
    xuid,
    gamertag: xuid,
    team_side: null,
    is_me: false,
    rank: null,
    score: null,
    kills: null,
    deaths: null,
    assists: null,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: 'win',
    ...overrides,
  }
}

describe('getExtremes', () => {
  it('retourne min/max sur les valeurs non-nulles', () => {
    const rows = [
      row('a', { kills: 10 }),
      row('b', { kills: 5 }),
      row('c', { kills: 20 }),
    ]
    expect(getExtremes(rows, 'kills')).toEqual({ min: 5, max: 20 })
  })

  it('retourne {null,null} avec moins de 2 valeurs (pas de comparaison utile)', () => {
    expect(getExtremes([row('a', { kills: 10 })], 'kills')).toEqual({ min: null, max: null })
    expect(getExtremes([], 'kills')).toEqual({ min: null, max: null })
  })

  it('ignore les nulls dans le calcul', () => {
    const rows = [
      row('a', { kills: 10 }),
      row('b', { kills: null }),
      row('c', { kills: 5 }),
    ]
    expect(getExtremes(rows, 'kills')).toEqual({ min: 5, max: 10 })
  })
})

describe('cellState', () => {
  const ex: Extremes = { min: 5, max: 20 }

  it('inverted=false : max=best, min=worst', () => {
    expect(cellState(20, ex, false)).toBe('best')
    expect(cellState(5, ex, false)).toBe('worst')
    expect(cellState(10, ex, false)).toBe('neutral')
  })

  it('inverted=true (deaths, damage_taken) : min=best, max=worst', () => {
    expect(cellState(5, ex, true)).toBe('best')
    expect(cellState(20, ex, true)).toBe('worst')
    expect(cellState(10, ex, true)).toBe('neutral')
  })

  it('null ou extremes uniformes → neutral', () => {
    expect(cellState(null, ex, false)).toBe('neutral')
    expect(cellState(10, { min: 10, max: 10 }, false)).toBe('neutral')
    expect(cellState(10, { min: null, max: null }, false)).toBe('neutral')
  })
})

describe('getMvpLvp — lobby-wide', () => {
  const cols: ColDef[] = [
    { key: 'kills', label: 'Frags', inverted: false },
    { key: 'assists', label: 'Assists', inverted: false },
    { key: 'damage_dealt', label: 'DD', inverted: false },
    { key: 'deaths', label: 'Morts', inverted: true },
  ]

  function buildExtremes(rows: MatchScoreboardRow[]): Record<string, Extremes> {
    return Object.fromEntries(cols.map((c) => [String(c.key), getExtremes(rows, c.key)]))
  }

  it('attribue MVP au joueur avec ≥2 best (lobby entier, pas par équipe)', () => {
    // Mid (xuid=m) domine 3 colonnes : kills, assists, damage_dealt
    // Reste : worst pour 'm' = 0 sur les colonnes monotones (deaths inverted)
    const rows = [
      row('m', { kills: 20, assists: 10, damage_dealt: 5000, deaths: 5, team_side: 't0' }),
      row('a', { kills: 10, assists: 5, damage_dealt: 2000, deaths: 8, team_side: 't0' }),
      row('b', { kills: 5, assists: 2, damage_dealt: 1000, deaths: 12, team_side: 't1' }),
      row('c', { kills: 7, assists: 3, damage_dealt: 1500, deaths: 10, team_side: 't1' }),
    ]
    const extremes = buildExtremes(rows)
    const result = getMvpLvp(rows, cols, extremes)

    expect(result.mvp).toBe('m') // 3 best (kills, assists, damage_dealt)
    expect(result.lvp).toBe('b') // 3 worst (kills, assists, damage_dealt) + best deaths inverted=12 (worst)
  })

  it('seuil <2 best = pas de MVP (évite les faux positifs sur petit lobby)', () => {
    // Aucun joueur n'a 2+ best.
    const rows = [
      row('a', { kills: 10, assists: 5, damage_dealt: 1000 }),
      row('b', { kills: 8, assists: 10, damage_dealt: 800 }),
      row('c', { kills: 5, assists: 3, damage_dealt: 1500 }),
    ]
    const extremes = buildExtremes(rows)
    const result = getMvpLvp(rows, cols, extremes)

    // a=1 best (kills), b=1 best (assists), c=1 best (damage_dealt) → personne
    expect(result.mvp).toBeNull()
  })

  it('un seul MVP et un seul LVP même avec 2 équipes', () => {
    // Deux équipes, joueur exceptionnel sur t0 et joueur catastrophique sur t1.
    const rows = [
      row('star', { kills: 30, assists: 20, damage_dealt: 8000, deaths: 3, team_side: 't0' }),
      row('mid_a', { kills: 12, assists: 8, damage_dealt: 3000, deaths: 7, team_side: 't0' }),
      row('mid_b', { kills: 11, assists: 7, damage_dealt: 2900, deaths: 6, team_side: 't1' }),
      row('flop', { kills: 2, assists: 0, damage_dealt: 200, deaths: 18, team_side: 't1' }),
    ]
    const extremes = buildExtremes(rows)
    const result = getMvpLvp(rows, cols, extremes)

    expect(result.mvp).toBe('star')
    expect(result.lvp).toBe('flop')
  })

  it('retourne null/null pour < 2 joueurs', () => {
    expect(getMvpLvp([row('a', { kills: 10 })], cols, {})).toEqual({ mvp: null, lvp: null })
    expect(getMvpLvp([], cols, {})).toEqual({ mvp: null, lvp: null })
  })

  it('exclut assassinat/charge spartane/coup au sol du départage MVP/LVP', () => {
    // Miroir front du test Go ComputeMVPLVP_ExcludesMechanicKills : seules les
    // colonnes Frags + Passes sont actives pour isoler l'effet des frags nets.
    const twoCol: ColDef[] = [
      { key: 'kills', label: 'Frags', inverted: false },
      { key: 'assists', label: 'Passes', inverted: false },
    ]
    const ex = (r: MatchScoreboardRow[]) =>
      Object.fromEntries(twoCol.map((c) => [String(c.key), getExtremes(r, c.key)]))

    // Contrôle (frags bruts) : a domine Frags(20) et Passes(10) → MVP=a, LVP=b.
    const raw = [row('a', { kills: 20, assists: 10 }), row('b', { kills: 8, assists: 5 })]
    expect(getMvpLvp(raw, twoCol, ex(raw))).toEqual({ mvp: 'a', lvp: 'b' })

    // Exclusion : a a 18 frags mécaniques (6+6+6) → frags nets 2 < b(8). La
    // cellule Frags bascule vers b ; a/b se partagent 1 best + 1 worst chacun →
    // aucun n'atteint le seuil de 2 → plus de MVP/LVP net. C'est cette bascule
    // qui prouve que les mécaniques sont bien retranchées.
    const adj = [
      row('a', {
        kills: 20,
        assists: 10,
        assassination_kills: 6,
        shoulder_bash_kills: 6,
        ground_pound_kills: 6,
      }),
      row('b', { kills: 8, assists: 5 }),
    ]
    expect(getMvpLvp(adj, twoCol, ex(adj))).toEqual({ mvp: null, lvp: null })
  })

  it('respecte les colonnes inverted (deaths) : best = moins de morts', () => {
    const rows = [
      row('survivor', { kills: 5, assists: 2, deaths: 1, damage_dealt: 1000 }),
      row('feeder1', { kills: 15, assists: 10, deaths: 20, damage_dealt: 4000 }),
      row('feeder2', { kills: 14, assists: 9, deaths: 18, damage_dealt: 3800 }),
    ]
    const extremes = buildExtremes(rows)
    const result = getMvpLvp(rows, cols, extremes)

    // survivor : best deaths (1) MAIS worst sur kills/assists/damage_dealt = 3 worst → LVP
    // feeder1  : best kills + best assists + best damage_dealt + worst deaths = 3 best → MVP
    expect(result.mvp).toBe('feeder1')
    expect(result.lvp).toBe('survivor')
  })
})
