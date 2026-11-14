/**
 * squadRangeRolesRibbon.test.ts — le ruban « Rôles de portée » : une ligne par joueur, une case
 * par match joué à la couleur de son rôle (trois jetons ordonnés), case vide sans rôle,
 * infobulle (rôle, écart signé, frags mesurés, petit échantillon), hauteur selon le nombre de
 * joueurs, rendu réel des teintes.
 */
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

import { renderOptionToSvg } from '@/components/charts/svgRender.testutil'
import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'

import type { LigneRuban } from '../squadRangeRoles.logic'
import {
  buildRangeRolesRibbonOption,
  hauteurRuban,
  HAUTEUR_LIGNE,
  JETON_ROLE,
  type RangeRolesRibbonOpts,
} from './squadRangeRolesRibbon'

const OPTS: RangeRolesRibbonOpts = {
  categories: ['#1 · A', '#2 · B', '#3 · C'],
  couleurs: { JGtm: '#111111', Kaya: '#222222' },
  libelles: {
    xAxis: 'Matchs',
    roles: { front: 'Front', polyvalent: 'Poly', sniper: 'Sniper' },
    tooltipRole: (r) => `rôle : ${r}`,
    tooltipNoRole: 'pas encore de rôle',
    tooltipDelta: (d) => `écart ${d} m`,
    tooltipMeasured: (n) => `${n} frags`,
    tooltipUnsure: (n) => `${n} frags, données insuffisantes`,
  },
  fmtDeltaM: (v) => (v > 0 ? `+${v}` : String(v)),
}

const LIGNES: LigneRuban[] = [
  {
    xuid: 'x1',
    gamertag: 'JGtm',
    cellules: [
      { ordre: 0, matchId: 'm0', role: null, plein: true, ecartM: 1, mesures: 10 },
      { ordre: 1, matchId: 'm1', role: 'front', plein: true, ecartM: -3, mesures: 12 },
      { ordre: 2, matchId: 'm2', role: 'sniper', plein: true, ecartM: 6, mesures: 9 },
    ],
  },
  {
    xuid: 'x2',
    gamertag: 'Kaya',
    cellules: [
      { ordre: 0, matchId: 'm0', role: null, plein: false, ecartM: 4, mesures: 3 },
      { ordre: 1, matchId: 'm1', role: 'polyvalent', plein: true, ecartM: 0.5, mesures: 8 },
    ],
  },
]

type Serie = { type: string; name: string; data: { value: [number, number, number]; tip: string }[]; itemStyle: { color: string } }
type Opt = {
  series: Serie[]
  yAxis: { data: string[]; inverse: boolean; axisLabel: { color: (v: string) => string } }
  xAxis: { data: string[]; name: string }
  legend: { data: { name: string; itemStyle: { color: string } }[] }
  tooltip: { formatter: (p: unknown) => string }
}
const build = (l: LigneRuban[] = LIGNES) => buildRangeRolesRibbonOption(l, OPTS) as unknown as Opt

describe('buildRangeRolesRibbonOption', () => {
  beforeAll(() => applyPalette(defaultPalette, 'default'))
  afterAll(() => _resetActivePalette())

  it('sans ligne : option vide', () => {
    expect(buildRangeRolesRibbonOption([], OPTS)).toEqual({ backgroundColor: 'transparent' })
  })

  it('une ligne par joueur (premier en haut), nom à sa couleur ; une colonne par match', () => {
    const o = build()
    expect(o.yAxis.data).toEqual(['JGtm', 'Kaya'])
    expect(o.yAxis.inverse).toBe(true)
    expect(o.yAxis.axisLabel.color('JGtm')).toBe('#111111')
    expect(o.yAxis.axisLabel.color('Kaya')).toBe('#222222')
    expect(o.xAxis.data).toEqual(OPTS.categories)
    expect(o.xAxis.name).toBe('Matchs')
  })

  it('trois séries de rôle aux jetons chaud, neutre, froid ; les cases vides en transparent', () => {
    const o = build()
    expect(o.series.map((s) => s.name)).toEqual(['Front', 'Poly', 'Sniper', ''])
    expect(o.series.map((s) => s.itemStyle.color)).toEqual([
      defaultPalette[JETON_ROLE.front],
      defaultPalette[JETON_ROLE.polyvalent],
      defaultPalette[JETON_ROLE.sniper],
      'transparent',
    ])
    expect(Object.values(JETON_ROLE)).toEqual(['range-role-front', 'range-role-mid', 'range-role-far'])
  })

  it('les cases : [match, ligne] par série de rôle ; sans rôle dans la série vide, sans hachure ni point creux', () => {
    const o = build()
    expect(o.series[0].data.map((d) => d.value.slice(0, 2))).toEqual([[1, 0]])
    expect(o.series[1].data.map((d) => d.value.slice(0, 2))).toEqual([[1, 1]])
    expect(o.series[2].data.map((d) => d.value.slice(0, 2))).toEqual([[2, 0]])
    expect(o.series[3].data.map((d) => d.value.slice(0, 2))).toEqual([[0, 0], [0, 1]])
    const dump = JSON.stringify(o)
    expect(dump).not.toContain('decal')
    expect(dump).not.toContain('dashed')
  })

  it('légende native : les trois rôles à leur teinte, pas de série vide', () => {
    const l = build().legend
    expect(l.data.map((d) => d.name)).toEqual(['Front', 'Poly', 'Sniper'])
  })

  it('infobulle d’une case avec rôle : joueur, match, rôle, écart signé, frags mesurés', () => {
    const o = build()
    expect(o.tooltip.formatter({ data: o.series[2].data[0] })).toBe(
      '<b>JGtm</b><br/>#3 · C<br/>rôle : Sniper<br/>écart +6 m<br/>9 frags',
    )
  })

  it('infobulle d’une case vide : fenêtre incomplète dite, petit échantillon dit dans l’infobulle seulement', () => {
    const o = build()
    const [fenetre, creux] = o.series[3].data
    expect(o.tooltip.formatter({ data: fenetre })).toBe('<b>JGtm</b><br/>#1 · A<br/>pas encore de rôle<br/>écart +1 m<br/>10 frags')
    expect(o.tooltip.formatter({ data: creux })).toBe('<b>Kaya</b><br/>#1 · A<br/>écart +4 m<br/>3 frags, données insuffisantes')
  })

  it('le gamertag est échappé', () => {
    const o = build([{ ...LIGNES[0], gamertag: '<i>x</i>' }])
    expect(o.tooltip.formatter({ data: o.series[0].data[0] })).toContain('<b>&lt;i&gt;x&lt;/i&gt;</b>')
  })

  it('hauteur : un pas par joueur au-dessus des marges, plancher de 200', () => {
    expect(hauteurRuban(1)).toBe(200)
    expect(hauteurRuban(4)).toBe(130 + 4 * HAUTEUR_LIGNE)
    expect(hauteurRuban(5)).toBeGreaterThan(hauteurRuban(4))
  })

  it('le rendu peint chaque case de rôle à la teinte de son jeton', () => {
    const svg = renderOptionToSvg(buildRangeRolesRibbonOption(LIGNES, OPTS), 900, hauteurRuban(2))
    const fills = svg.toLowerCase()
    expect(fills).toContain(defaultPalette['range-role-front'].toLowerCase())
    expect(fills).toContain(defaultPalette['range-role-mid'].toLowerCase())
    expect(fills).toContain(defaultPalette['range-role-far'].toLowerCase())
  })
})
