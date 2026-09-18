/**
 * squadIntensityProfileChart.test.ts — « Intensité » profil médian + enveloppe.
 *
 * Couvre : layout des grilles (N=1 pleine largeur, 2/3/4 → 2 colonnes), échelle
 * Y partagée entre panneaux, panneau absent si aucune manche exploitable, médiane
 * seule (pas d'enveloppe) sous MIN_MATCHES_FOR_ENVELOPE manches, courbes de
 * référence équipe / lobby (styles, ordre du tooltip, borne Y).
 */
import { describe, it, expect } from 'vitest'

import {
  buildSquadIntensityProfileOption,
  computeGrids,
  type IntensityOverlay,
  type IntensityPanelInput,
} from './squadIntensityProfileChart'

/** N manches concentrées sur `phaseIdx` (part = 1 sur cette phase). */
function rows(n: number, phaseIdx = 0): Array<{ phases: number[] }> {
  return Array.from({ length: n }, () => {
    const p = new Array<number>(10).fill(0)
    p[phaseIdx] = 1
    return { phases: p }
  })
}

function panel(key: string, n: number, phaseIdx = 0): IntensityPanelInput {
  return { key, label: key, color: '#ff8800', rows: rows(n, phaseIdx) }
}

const OPTS = {
  medianLabel: 'Médiane',
  envelopeLabel: 'Enveloppe P25–P75',
  refLabel: '10 %',
  axisLabels: { start: 'Début', mid: 'Milieu', end: 'Fin', rangeSuffix: 'du match' },
}

function seriesIds(opt: ReturnType<typeof buildSquadIntensityProfileOption>): string[] {
  return (opt.series as Array<{ id?: string }>).map((s) => String(s.id ?? ''))
}

describe('computeGrids', () => {
  it('N=1 → un panneau pleine largeur (1 colonne)', () => {
    const g = computeGrids(1)
    expect(g).toHaveLength(1)
    // 1 colonne : largeur = 100 - left(6) - right(3) = 91 %.
    expect(g[0].width).toBe('91%')
  })

  it('N=2 → une rangée de 2 (même top)', () => {
    const g = computeGrids(2)
    expect(g).toHaveLength(2)
    expect(g[0].top).toBe(g[1].top)
    expect(g[0].left).not.toBe(g[1].left)
  })

  it('N=3 → 2+1 (le 3e panneau sur une 2e rangée)', () => {
    const g = computeGrids(3)
    expect(g).toHaveLength(3)
    expect(g[0].top).toBe(g[1].top)
    expect(g[2].top).not.toBe(g[0].top)
  })

  it('N=4 → 2×2 (deux rangées de 2)', () => {
    const g = computeGrids(4)
    expect(g).toHaveLength(4)
    expect(g[0].top).toBe(g[1].top)
    expect(g[2].top).toBe(g[3].top)
    expect(g[2].top).not.toBe(g[0].top)
  })
})

describe('buildSquadIntensityProfileOption', () => {
  it('aucun panneau exploitable → option minimale (fond seul)', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [{ key: 'A', label: 'A', color: '#fff', rows: [{ phases: null }] }],
      ...OPTS,
    })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('1 joueur (≥4 manches) → 1 grille + base/bande/médiane', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    expect(opt.grid).toHaveLength(1)
    expect(opt.title).toHaveLength(1)
    expect(seriesIds(opt)).toEqual(['base-0', 'band-0', 'median-0'])
  })

  it('4 joueurs → 4 grilles, 4 axes X/Y, titres = gamertags', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5), panel('B', 5), panel('C', 5), panel('D', 5)],
      ...OPTS,
    })
    expect(opt.grid).toHaveLength(4)
    expect(opt.xAxis).toHaveLength(4)
    expect(opt.yAxis).toHaveLength(4)
    const titles = (opt.title as Array<{ text: string }>).map((t) => t.text)
    expect(titles).toEqual(['A', 'B', 'C', 'D'])
  })

  it('échelle Y PARTAGÉE : tous les panneaux ont le même max', () => {
    // A concentré phase 0 (part médiane haute), B étalé → max commun.
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5, 0), panel('B', 5, 3)],
      ...OPTS,
    })
    const maxes = (opt.yAxis as Array<{ max: number }>).map((y) => y.max)
    expect(maxes[0]).toBe(maxes[1])
    expect(maxes[0]).toBeGreaterThan(0)
  })

  it('panneau absent si le joueur n a aucune manche exploitable', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [
        panel('A', 5),
        { key: 'B', label: 'B', color: '#0af', rows: [{ phases: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0] }] },
      ],
      ...OPTS,
    })
    expect(opt.grid).toHaveLength(1)
    expect((opt.title as Array<{ text: string }>)[0].text).toBe('A')
  })

  it('médiane seule (< 4 manches) : pas de base/bande d enveloppe', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 3)], ...OPTS })
    expect(seriesIds(opt)).toEqual(['median-0'])
  })

  it('repère 10 % : markLine sur la médiane à 1/PHASE_COUNT', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 3)], ...OPTS })
    const median = (opt.series as Array<{ id?: string; markLine?: { data: Array<{ yAxis: number }> } }>).find(
      (s) => s.id === 'median-0',
    )
    expect(median?.markLine?.data[0].yAxis).toBeCloseTo(0.1)
  })

  it('axe X : 3 repères seulement (Début/Milieu/Fin aux index 0/5/9)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const xAxis = (opt.xAxis as Array<{
      data: string[]
      axisLabel: {
        interval: (i: number) => boolean
        formatter: (v: string, i: number) => string
      }
    }>)[0]
    // Les 10 tranches restent en data (tooltip précis).
    expect(xAxis.data).toHaveLength(10)
    // interval : vrai seulement aux index 0, 5, 9.
    const shown = [...Array(10).keys()].filter((i) => xAxis.axisLabel.interval(i))
    expect(shown).toEqual([0, 5, 9])
    // formatter : libellés i18n aux 3 repères, vide ailleurs.
    expect(xAxis.axisLabel.formatter('', 0)).toBe('Début')
    expect(xAxis.axisLabel.formatter('', 5)).toBe('Milieu')
    expect(xAxis.axisLabel.formatter('', 9)).toBe('Fin')
    expect(xAxis.axisLabel.formatter('', 4)).toBe('')
  })

  it('axe Y : formatter en pourcentage (0.1 → « 10% »)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const yAxis = (opt.yAxis as Array<{ axisLabel: { formatter: (v: number) => string } }>)[0]
    expect(yAxis.axisLabel.formatter(0.1)).toBe('10%')
  })

  it('tooltip : tranche précise via dataIndex + suffixe (« 40-50% du match »)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const formatter = (opt.tooltip as { formatter: (p: unknown) => string }).formatter
    const html = formatter([{ seriesId: 'median-0', seriesName: 'A', dataIndex: 4, value: 0.3 }])
    expect(html).toContain('40-50% du match')
    expect(html).toContain('Médiane')
  })

  // Affordance de survol (v7.3 lot 2, item 2.3c) : sans symbole survolable, le
  // canvas se lit comme une image figée. Symbole caché au repos (showSymbol:false)
  // mais DÉFINI (symbol ≠ 'none'), sinon ECharts n'affiche rien à l'emphase.
  it('médiane : symbole caché au repos mais révélé au survol', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const median = (opt.series as Array<{ id?: string; showSymbol?: boolean; symbol?: string }>).find(
      (s) => s.id === 'median-0',
    )
    expect(median?.showSymbol).toBe(false)
    expect(median?.symbol).toBe('circle')
  })
})

describe('buildSquadIntensityProfileOption — courbes de référence équipe / lobby', () => {
  const TEAM: IntensityOverlay = { key: 'team', label: 'Équipe', rows: rows(8, 5) }
  const LOBBY: IntensityOverlay = { key: 'lobby', label: 'Lobby', rows: rows(8, 7) }
  const THREE = [panel('A', 5), panel('B', 5), panel('C', 5)]

  /** Séries de référence (team-* / lobby-*) avec leur style. */
  function overlaySeries(opt: ReturnType<typeof buildSquadIntensityProfileOption>) {
    return (
      opt.series as Array<{
        id?: string
        name?: string
        xAxisIndex?: number
        data?: number[]
        lineStyle?: { type?: string; width?: number; opacity?: number }
      }>
    ).filter((s) => /^(team|lobby)-/.test(String(s.id ?? '')))
  }

  it('overlays absent : rendu inchangé (aucune série team-/lobby-, ids identiques)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: THREE, ...OPTS })
    expect(seriesIds(opt)).toEqual([
      'base-0', 'band-0', 'median-0',
      'base-1', 'band-1', 'median-1',
      'base-2', 'band-2', 'median-2',
    ])
  })

  it('overlays vide : même rendu que sans overlays', () => {
    const sans = buildSquadIntensityProfileOption({ panels: THREE, ...OPTS })
    const vide = buildSquadIntensityProfileOption({ panels: THREE, ...OPTS, overlays: [] })
    expect(seriesIds(vide)).toEqual(seriesIds(sans))
    expect((vide.yAxis as Array<{ max: number }>)[0].max).toBe((sans.yAxis as Array<{ max: number }>)[0].max)
  })

  it('deux overlays : une série équipe ET une série lobby PAR panneau, liées à leur grille', () => {
    const opt = buildSquadIntensityProfileOption({ panels: THREE, ...OPTS, overlays: [TEAM, LOBBY] })
    const refs = overlaySeries(opt)
    expect(refs.map((s) => s.id)).toEqual(['team-0', 'lobby-0', 'team-1', 'lobby-1', 'team-2', 'lobby-2'])
    expect(refs.map((s) => s.xAxisIndex)).toEqual([0, 0, 1, 1, 2, 2])
    expect(refs[0].name).toBe('Équipe')
    expect(refs[1].name).toBe('Lobby')
  })

  it('styles neutres : équipe en trait plein, lobby en pointillé plus fin et plus discret', () => {
    const opt = buildSquadIntensityProfileOption({ panels: THREE, ...OPTS, overlays: [TEAM, LOBBY] })
    const [team, lobby] = overlaySeries(opt)
    expect(team.lineStyle?.type).toBe('solid')
    expect(lobby.lineStyle?.type).toBe('dashed')
    expect(lobby.lineStyle?.width as number).toBeLessThan(team.lineStyle?.width as number)
    expect(lobby.lineStyle?.opacity as number).toBeLessThan(team.lineStyle?.opacity as number)
  })

  it('un seul overlay (lobby seul, escouade < 3) : une seule série de référence par panneau', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS, overlays: [LOBBY] })
    expect(overlaySeries(opt).map((s) => s.id)).toEqual(['lobby-0'])
  })

  it('les médianes viennent du helper canonique (manches agrégées, pas une moyenne de médianes)', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5, 0), panel('B', 5, 0), panel('C', 5, 0)],
      ...OPTS,
      overlays: [TEAM, LOBBY],
    })
    const [team, lobby] = overlaySeries(opt)
    // TEAM = 8 manches concentrées sur la phase 5, LOBBY sur la phase 7 (une moyenne
    // des médianes joueur aurait donné le profil de la phase 0 pour les deux).
    expect(team.data?.[5]).toBeCloseTo(1)
    expect(team.data?.[0]).toBeCloseTo(0)
    expect(lobby.data?.[7]).toBeCloseTo(1)
    expect(lobby.data?.[0]).toBeCloseTo(0)
  })

  it('échelle Y partagée : la borne inclut les médianes des DEUX overlays', () => {
    // Panneau et équipe ÉTALÉS (part 0,1 sur chaque phase → médiane basse) ; seul
    // le lobby culmine à 1 → la borne doit le contenir même s'il n'est pas le
    // premier overlay.
    const spread = Array.from({ length: 5 }, () => ({ phases: new Array<number>(10).fill(1) }))
    const flat = { key: 'A', label: 'A', color: '#ff8800', rows: spread }
    const lowTeam: IntensityOverlay = { key: 'team', label: 'Équipe', rows: spread }
    const sans = buildSquadIntensityProfileOption({ panels: [flat], ...OPTS, overlays: [lowTeam] })
    const avec = buildSquadIntensityProfileOption({ panels: [flat], ...OPTS, overlays: [lowTeam, LOBBY] })
    const yMax = (opt: ReturnType<typeof buildSquadIntensityProfileOption>) =>
      (opt.yAxis as Array<{ max: number }>)[0].max
    // Sans lobby : ~0,112 (0,1 + marge de tête) ; avec : 1 (le lobby entre dans le cadre).
    expect(yMax(sans)).toBeLessThan(0.5)
    expect(yMax(avec)).toBeGreaterThanOrEqual(1)
  })

  it('overlay sans frag : aucune courbe plate, l autre overlay reste tracé', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: THREE,
      ...OPTS,
      overlays: [{ key: 'team', label: 'Équipe', rows: [{ phases: new Array<number>(10).fill(0) }] }, LOBBY],
    })
    expect(overlaySeries(opt).map((s) => s.id)).toEqual(['lobby-0', 'lobby-1', 'lobby-2'])
  })

  it('tooltip : lignes « Équipe » puis « Lobby » sous la médiane du joueur, dans cet ordre', () => {
    const opt = buildSquadIntensityProfileOption({ panels: THREE, ...OPTS, overlays: [TEAM, LOBBY] })
    const formatter = (opt.tooltip as { formatter: (p: unknown) => string }).formatter
    // Le lobby arrive AVANT l'équipe dans les params : l'ordre du tooltip ne
    // dépend pas de l'ordre des séries survolées.
    const html = formatter([
      { seriesId: 'lobby-0', seriesName: 'Lobby', dataIndex: 4, value: 0.15 },
      { seriesId: 'median-0', seriesName: 'A', dataIndex: 4, value: 0.3 },
      { seriesId: 'team-0', seriesName: 'Équipe', dataIndex: 4, value: 0.2 },
    ])
    const iPlayer = html.indexOf('Médiane')
    const iTeam = html.indexOf('Équipe : 20%')
    const iLobby = html.indexOf('Lobby : 15%')
    expect(iPlayer).toBeGreaterThanOrEqual(0)
    expect(iTeam).toBeGreaterThan(iPlayer)
    expect(iLobby).toBeGreaterThan(iTeam)
  })
})
