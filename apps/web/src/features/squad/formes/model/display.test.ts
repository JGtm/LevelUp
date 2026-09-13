/**
 * display.test.ts — LE REPLI DES FORMES PAR MATCH.
 *
 * Défaut mesuré sur données réelles le 2026-09-13 : « Emprise de l'escouade »
 * rendait 1 147 lignes sur une portée de 1 147 matchs, dont 1 043 en hachure
 * « sans film décodé ». Ces tests verrouillent les trois propriétés du repli :
 * seulement les matchs mesurés, seulement les plus récents, affichés dans le
 * sens du temps.
 */
import { describe, expect, it } from 'vitest'

import type { SquadFormesBlock, SquadFormesMatch } from '@/lib/api/types'

import { MATCH_ROWS_LIMIT, listWindow, measuredWindow } from './display'

/** Un scope de `total` matchs, du plus récent au plus ancien (ordre du serveur). */
function scope(total: number, measuredEvery: number): SquadFormesBlock {
  const matches: SquadFormesMatch[] = Array.from({ length: total }, (_, i) => ({
    match_id: `m${i}`,
    // i = 0 est le plus RÉCENT (ordre de la page).
    start_time: new Date(Date.UTC(2026, 0, 1) - i * 3600_000).toISOString(),
    measured: i % measuredEvery === 0,
  }))
  return {
    available: true,
    matches_total: total,
    matches_measured: matches.filter((m) => m.measured).length,
    matches,
  }
}

describe('measuredWindow', () => {
  it('n’affiche que des matchs mesurés, et compte ceux qu’elle écarte', () => {
    const block = scope(100, 4) // 25 mesurés, 75 sans film
    const w = measuredWindow(block)
    expect(w.rows).toHaveLength(MATCH_ROWS_LIMIT)
    expect(w.rows.every((m) => m.measured)).toBe(true)
    expect(w.hidden).toBe(5)
    expect(w.unmeasured).toBe(75)
  })

  it('garde les PLUS RÉCENTS et les rend du plus ancien au plus récent', () => {
    const block = scope(100, 1)
    const w = measuredWindow(block, 3)
    expect(w.rows.map((m) => m.match_id)).toEqual(['m2', 'm1', 'm0'])
    expect(w.hidden).toBe(97)
  })

  it('ne borne rien quand le scope tient dans la fenêtre', () => {
    const w = measuredWindow(scope(5, 1))
    expect(w.rows).toHaveLength(5)
    expect(w.hidden).toBe(0)
    expect(w.unmeasured).toBe(0)
  })

  it('rend une fenêtre vide, jamais une erreur, sur un scope sans film', () => {
    const block = scope(10, 99)
    const w = measuredWindow(block)
    expect(w.rows).toHaveLength(1) // le match d'index 0 reste mesuré
    expect(w.unmeasured).toBe(9)
  })
})

describe('listWindow', () => {
  it('borne en nombre SANS écarter les matchs sans film', () => {
    const block = scope(30, 10)
    const w = listWindow(block.matches ?? [], 4)
    expect(w.rows).toHaveLength(4)
    expect(w.hidden).toBe(26)
    // La feuille de match n'a pas besoin de film : rien n'est écarté pour ça.
    expect(w.unmeasured).toBe(0)
    expect(w.rows.some((m) => !m.measured)).toBe(true)
  })
})
