/**
 * _killDistanceChart.test.ts — les sections par arme, le bâton min→max et son losange.
 *
 * Ce que ces tests verrouillent : le REGROUPEMENT par arme (une ligne d'en-tête « Arme ×N »
 * puis une ligne par joueur), l'ORDRE (armes par frags mesurés toutes équipes décroissants,
 * joueurs de même dans chaque section, le nom départageant), la troncature du gamertag sur
 * l'axe, l'inversion de l'axe Y au montage, la technique du bâton flottant (socle transparent
 * = min, barre visible = max − min), l'encre par JOUEUR, et le fait qu'une ligne d'en-tête ne
 * porte AUCUNE barre.
 */
import { describe, expect, it } from 'vitest'

import type { MatchKillDistanceWeapon } from '@/lib/api/types'

import {
  buildKillDistanceOption,
  killDistanceHeight,
  killDistanceRows,
  type KillDistancePlayerInput,
} from './_killDistanceChart'

function weapon(
  key: string,
  label: string,
  kills: number,
  min: number,
  avg: number,
  max: number,
): MatchKillDistanceWeapon {
  return {
    weapon_key: key,
    label,
    label_en: label ? `${label} EN` : '',
    measured_kills: kills,
    avg_distance_m: avg,
    min_distance_m: min,
    max_distance_m: max,
  }
}

const PLAYERS: KillDistancePlayerInput[] = [
  {
    xuid: 'xuid(1)',
    gamertag: 'Alice',
    color: '#111111',
    weapons: [weapon('hinf_br75', 'BR75', 3, 3.1, 12.4, 21.6)],
  },
  {
    xuid: 'xuid(2)',
    gamertag: 'UnGamertagVraimentTresLong',
    color: '#222222',
    weapons: [
      weapon('hinf_br75', 'BR75', 5, 2, 6, 10),
      weapon('hinf_repulsor', '', 1, 5, 5, 5),
    ],
  },
]

const TC = {
  axisLine: '#000',
  splitLine: '#333',
  axisLabel: '#000',
  tooltipBg: '#000',
  tooltipBorder: '#000',
  text: '#fff',
} as never

function option(rows = killDistanceRows(PLAYERS, 'fr')) {
  return buildKillDistanceOption({
    rows,
    tc: TC,
    colorOf: (player) => (player === 'Alice' ? '#111111' : '#222222'),
    avgColor: '#999999',
    fmtDistance: (m) => `${m} m`,
    labels: { kills: 'Frags mesurés', min: 'Plus proche', avg: 'Moyenne', max: 'Plus loin' },
  }) as {
    yAxis: { data: string[] }
    series: {
      type: string
      data: unknown[]
      itemStyle?: { color?: string }
      markArea?: { data: unknown[] }
    }[]
  }
}

describe('killDistanceRows — les sections par arme', () => {
  it('groupe par arme, ordonne par frags mesurés toutes équipes, joueurs sous leur arme', () => {
    const rows = killDistanceRows(PLAYERS, 'fr')
    expect(rows.map((r) => `${r.kind}:${r.axisLabel}`)).toEqual([
      'weapon:BR75 ×8',
      'player:UnGamertagVra…',
      'player:Alice',
      'weapon:hinf_repulsor ×1',
      'player:UnGamertagVra…',
    ])
  })

  it('résout le libellé par locale et replie sur weapon_key', () => {
    const rows = killDistanceRows(PLAYERS, 'en')
    expect(rows.filter((r) => r.kind === 'weapon').map((r) => r.axisLabel)).toEqual([
      'BR75 EN ×8',
      'hinf_repulsor ×1',
    ])
  })

  it('la ligne de joueur porte le gamertag COMPLET (infobulle) et ses trois distances', () => {
    const ligne = killDistanceRows(PLAYERS, 'fr')[2]
    expect(ligne).toMatchObject({
      player: 'Alice',
      weapon: 'BR75',
      kills: 3,
      min: 3.1,
      avg: 12.4,
      max: 21.6,
    })
  })

  it('la hauteur suit le nombre de lignes, avec un plancher', () => {
    expect(killDistanceHeight(0)).toBe(140)
    expect(killDistanceHeight(10)).toBe(276)
  })
})

describe('buildKillDistanceOption — un seul graphe, sections par arme', () => {
  it("la première ligne du modèle est EN HAUT : l'axe Y inverse la liste", () => {
    expect(option().yAxis.data).toEqual([
      'UnGamertagVra…',
      'hinf_repulsor ×1',
      'Alice',
      'UnGamertagVra…',
      'BR75 ×8',
    ])
  })

  it('socle transparent = min, barre visible = max − min, moyenne en scatter', () => {
    const [socle, plage, moyenne] = option().series
    expect(socle.itemStyle?.color).toBe('transparent')
    // Ordre inversé : repulsor (joueur), en-tête, Alice, long gamertag, en-tête.
    expect(socle.data).toEqual([5, null, 3.1, 2, null])
    expect((plage.data as { value: number | null }[]).map((d) => d.value)).toEqual([
      0,
      null,
      21.6 - 3.1,
      8,
      null,
    ])
    expect(moyenne.type).toBe('scatter')
    expect(moyenne.data).toEqual([
      [5, 0],
      [12.4, 2],
      [6, 3],
    ])
  })

  it("l'encre du bâton vient du JOUEUR, pas de l'arme", () => {
    const plage = option().series[1]
    const couleurs = (plage.data as { itemStyle?: { color: string } }[]).map(
      (d) => d.itemStyle?.color,
    )
    expect(couleurs).toEqual(['#222222', undefined, '#111111', '#222222', undefined])
  })

  it("chaque ligne d'en-tête porte une bande de séparation", () => {
    const bandes = option().series[1].markArea?.data
    expect(bandes).toEqual([
      [{ yAxis: 'hinf_repulsor ×1' }, { yAxis: 'hinf_repulsor ×1' }],
      [{ yAxis: 'BR75 ×8' }, { yAxis: 'BR75 ×8' }],
    ])
  })

  it('un seul frag mesuré : le bâton dégénère en point, le losange reste le témoin', () => {
    const [, plage, moyenne] = option().series
    expect((plage.data as { value: number | null }[])[0].value).toBe(0)
    expect(moyenne.data[0]).toEqual([5, 0])
  })
})
