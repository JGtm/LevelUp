/**
 * squadFragBreakdownChart.test.ts — « Répartition des frags » par joueur, barres
 * empilées PAR CLASSE (D8). Taxonomie dynamique (N classes, ordre canonique).
 */
import { describe, it, expect } from 'vitest'
import { buildFragBreakdownOption } from './squadFragBreakdownChart'
import { fragClassColor } from '@/lib/accessibility/scales'
import type { FragClassEntry } from '@/lib/api/types'

function cls(className: string, kills: number): FragClassEntry {
  return { class: className, kills, authoritative: false }
}

/** Libellé de classe stub (le vrai vient du manifeste `frags`). */
const classLabel = (c: string) => `L:${c}`
const ORDER = ['Me', 'F1']

type Serie = { name: string; type: string; stack: string; itemStyle: { color: string }; data: number[] }

describe('buildFragBreakdownOption (par classe)', () => {
  it('vide → option minimale (aucune série)', () => {
    const opt = buildFragBreakdownOption({}, { classLabel })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('aucune classe > 0 → option minimale (aucune série)', () => {
    const opt = buildFragBreakdownOption({ Me: [cls('shoulder', 0)] }, { playerOrder: ['Me'], classLabel })
    expect(opt.series).toBeUndefined()
  })

  it('union DYNAMIQUE des classes, ordre canonique, data alignée par joueur', () => {
    const rows = {
      Me: [cls('shoulder', 18), cls('melee', 6), cls('unattributed', 10)],
      F1: [cls('heavy', 5), cls('grenade', 4)],
    }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ORDER, classLabel })
    const series = opt.series as Serie[]
    // Union présente, ordonnée FRAG_CLASS_ORDER : shoulder, heavy, melee, grenade, unattributed.
    expect(series.map((s) => s.name)).toEqual([
      'L:shoulder',
      'L:heavy',
      'L:melee',
      'L:grenade',
      'L:unattributed',
    ])
    expect(series.every((s) => s.type === 'bar' && s.stack === 'frags')).toBe(true)
    expect(series[0].data).toEqual([18, 0]) // shoulder
    expect(series[1].data).toEqual([0, 5]) // heavy
    expect(series[2].data).toEqual([6, 0]) // melee
    expect(series[3].data).toEqual([0, 4]) // grenade
    expect(series[4].data).toEqual([10, 0]) // unattributed
  })

  it('classe H5 « Capacités spartanes » ventilée par joueur (D-P6-2)', () => {
    // Backend H5 (hasMechanics=true) produit désormais la classe spartan_ability +
    // le split Mêlée par joueur ; le chart, dynamique, la rend dans l'ordre canonique
    // (juste avant unattributed).
    const rows = {
      Me: [cls('melee', 5), cls('spartan_ability', 4), cls('unattributed', 2)],
      F1: [cls('shoulder', 3)],
    }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ORDER, classLabel })
    const series = opt.series as Serie[]
    expect(series.map((s) => s.name)).toEqual([
      'L:shoulder',
      'L:melee',
      'L:spartan_ability',
      'L:unattributed',
    ])
    const spartan = series.find((s) => s.name === 'L:spartan_ability')!
    expect(spartan.data).toEqual([4, 0])
    expect(spartan.itemStyle.color).toBe(fragClassColor('spartan_ability'))
  })

  it('couleurs PAR CLASSE via fragClassColor (hex fixes CVD-safe)', () => {
    const rows = { Me: [cls('shoulder', 3), cls('grenade', 2)] }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ['Me'], classLabel })
    const series = opt.series as Serie[]
    expect(series[0].itemStyle.color).toBe(fragClassColor('shoulder'))
    expect(series[1].itemStyle.color).toBe(fragClassColor('grenade'))
  })

  it('agrège plusieurs entrées d’une même classe pour un joueur', () => {
    const rows = { Me: [cls('shoulder', 4), cls('shoulder', 3)] }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ['Me'], classLabel })
    const series = opt.series as Serie[]
    expect(series).toHaveLength(1)
    expect(series[0].data).toEqual([7])
  })

  it('axe Y = joueurs dans l’ordre, inversé (main en haut)', () => {
    const rows = { Me: [cls('melee', 1)], F1: [cls('melee', 1)] }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ORDER, classLabel })
    const yAxis = opt.yAxis as { type: string; data: string[]; inverse: boolean }
    expect(yAxis.type).toBe('category')
    expect(yAxis.data).toEqual(['Me', 'F1'])
    expect(yAxis.inverse).toBe(true)
  })
})
