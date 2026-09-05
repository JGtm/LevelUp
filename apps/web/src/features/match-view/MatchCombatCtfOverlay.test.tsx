/**
 * MatchCombatCtfOverlay.test.tsx — LES REPÈRES DE CAPTURE sur les deux graphes de combat.
 *
 * CE QUE CE FICHIER NE FAISAIT PAS (registre 2026-09-05, N2). Ses quatre cas montaient les
 * composants et vérifiaient que le stub ECharts existait : rien ne lisait l'option, et le
 * mock la tronquait d'ailleurs à 80 caractères. Un repère posé au mauvais instant — ou plus
 * posé du tout — restait vert. L'option est désormais capturée ENTIÈRE et les abscisses sont
 * calculées à la main.
 *
 * LES DEUX GRAPHES NE PLACENT PAS LEURS REPÈRES DE LA MÊME FAÇON, et c'est le fond du sujet :
 *  - « Frags cumulés » a un axe en MILLISECONDES : la verticale tombe sur l'instant même.
 *  - « Dominance » a un axe de CATÉGORIES (une par intervalle) : la verticale s'interpole
 *    DANS son intervalle, `index − 0,5 + fraction`, faute de quoi trois captures d'un même
 *    intervalle se superposeraient sur sa borne.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'

import { MatchKDCumulChart } from './MatchKDCumulChart'
import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import { MATCH_VIEW_TEXT } from './i18n'
import type {
  MatchHighlightEvent,
  MatchObjectiveEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

// Un chart ECharts monté en jsdom crashe le canvas — on stubbe le wrapper. L'option est
// sérialisée EN ENTIER : c'est elle que les cas ci-dessous lisent.
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

const t = MATCH_VIEW_TEXT.fr

function sbRow(partial: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return {
    xuid: 'x', gamertag: 'GT', team_side: null, is_me: false, rank: null, score: null,
    kills: null, deaths: null, assists: null, shots_fired: null, shots_hit: null,
    accuracy: null, damage_dealt: null, damage_taken: null, average_life: null,
    headshot_kills: null, max_killing_spree: null, perfect_kills: null,
    power_weapon_kills: null, melee_kills: null, outcome_label: '', ...partial,
  } as MatchScoreboardRow
}

const SCOREBOARD: MatchScoreboardRow[] = [
  sbRow({ xuid: 'me', gamertag: 'Me', team_side: '0', is_me: true }),
  sbRow({ xuid: 'ally1', gamertag: 'AllyOne', team_side: '0' }),
  sbRow({ xuid: 'enemy1', gamertag: 'EnemyOne', team_side: '1' }),
]

/** Trois frags : moi à 5 s, l'adversaire à 12 s, mon allié à 20 s. */
const KILLS: MatchHighlightEvent[] = [
  { event_time_ms: 5000, event_type: 'kill', actor_xuid: 'me', target_xuid: 'enemy1', weapon_id: null },
  { event_time_ms: 12000, event_type: 'kill', actor_xuid: 'enemy1', target_xuid: 'ally1', weapon_id: null },
  { event_time_ms: 20000, event_type: 'kill', actor_xuid: 'ally1', target_xuid: 'enemy1', weapon_id: null },
]

/** Deux captures : la mienne à 8 s, celle d'en face à 18 s. */
const CAPTURES: MatchObjectiveEvent[] = [
  { matchId: 'm1', seq: 0, timeMs: 8000, objectiveType: 'flag', eventType: 'capture', teamId: 0, value: 1, source: 'film', confidence: 'high', players: [{ xuid: 'me', role: 'scorer' }] },
  { matchId: 'm1', seq: 1, timeMs: 18000, objectiveType: 'flag', eventType: 'capture', teamId: 1, value: 1, source: 'film', confidence: 'high', players: [{ xuid: 'enemy1', role: 'scorer' }] },
]

/** Trois intervalles de dix secondes : [0,10), [10,20), [20,30). */
const BINS: MatchTugOfWarBin[] = [
  { bin_start: 0, bin_end: 10, team_kills: 1, enemy_kills: 0, net_kills: 1 },
  { bin_start: 10, bin_end: 20, team_kills: 1, enemy_kills: 1, net_kills: 0 },
  { bin_start: 20, bin_end: 30, team_kills: 1, enemy_kills: 0, net_kills: 1 },
]

interface SerieOption {
  name?: string
  data?: unknown[]
  markLine?: { data?: Array<{ xAxis: number; lineStyle?: { color?: string } }> }
}

/**
 * Monte le graphe, lit l'option ENTIÈRE, puis démonte — deux montages dans un même cas
 * laisseraient deux stubs dans le document et la lecture deviendrait ambiguë.
 */
async function optionDe(ui: React.ReactElement): Promise<{ series: SerieOption[]; xAxis: { max?: number } }> {
  const vue = render(ui)
  const stub = await waitFor(() => vue.getByTestId('echarts-stub'))
  const option = JSON.parse(stub.textContent ?? '{}')
  vue.unmount()
  return option
}

function kdCumul(objectiveEvents: MatchObjectiveEvent[] | null) {
  return (
    <MatchKDCumulChart
      events={KILLS}
      badges={[]}
      scoreboard={SCOREBOARD}
      meXUID="me"
      objectiveEvents={objectiveEvents}
      t={t}
    />
  )
}

function tugOfWar(objectiveEvents: MatchObjectiveEvent[] | null) {
  return (
    <MatchTugOfWarChart
      bins={BINS}
      events={KILLS}
      scoreboard={SCOREBOARD}
      meXUID="me"
      objectiveEvents={objectiveEvents}
      t={t}
    />
  )
}

describe('Frags cumulés — les captures sur un axe en millisecondes', () => {
  it('trace les deux courbes de frags, chacune à son instant', async () => {
    const opt = await optionDe(kdCumul(CAPTURES))
    const [allie, adverse] = opt.series
    expect(allie.name).toBe(t.combatTeamLabel)
    // Mon frag à 5 s, celui de mon allié à 20 s : cumul 1 puis 2.
    expect(allie.data).toEqual([[5000, 1], [20_000, 2]])
    expect(adverse.name).toBe(t.combatEnemyLabel)
    expect(adverse.data).toEqual([[12_000, 1]])
    // Trois frags sur vingt secondes : l'axe garde son plancher d'une minute.
    expect(opt.xAxis.max).toBe(60_000)
  })

  it('POSE UNE VERTICALE À L’INSTANT de chaque capture, teintée du camp qui marque', async () => {
    const opt = await optionDe(kdCumul(CAPTURES))
    const reperes = opt.series.find((s) => s.name === t.combatCtfCaptureLabel)!
    expect(reperes.markLine?.data?.map((d) => d.xAxis)).toEqual([8000, 18_000])
    expect(reperes.markLine?.data?.map((d) => d.lineStyle?.color)).toEqual([
      'var(team-ally)',
      'var(team-enemy)',
    ])
  })

  it('N’AJOUTE AUCUNE SÉRIE sans capture : hors CTF, le graphe est celui d’avant', async () => {
    const avec = await optionDe(kdCumul(CAPTURES))
    const sans = await optionDe(kdCumul(null))
    expect(sans.series).toHaveLength(avec.series.length - 1)
    expect(sans.series.some((s) => s.name === t.combatCtfCaptureLabel)).toBe(false)
  })
})

describe('Dominance — les captures sur un axe de catégories', () => {
  it('INTERPOLE la verticale DANS son intervalle : index − 0,5 + fraction', async () => {
    const opt = await optionDe(tugOfWar(CAPTURES))
    const reperes = opt.series.find((s) => s.name === t.combatCtfCaptureLabel)!
    // 8 s tombe dans l'intervalle 0 ([0,10)), aux 8/10 : 0 − 0,5 + 0,8 = 0,3.
    // 18 s tombe dans l'intervalle 1 ([10,20)), aux 8/10 : 1 − 0,5 + 0,8 = 1,3.
    const abscisses = reperes.markLine?.data?.map((d) => d.xAxis) ?? []
    expect(abscisses).toHaveLength(2)
    expect(abscisses[0]).toBeCloseTo(0.3, 10)
    expect(abscisses[1]).toBeCloseTo(1.3, 10)
    expect(reperes.markLine?.data?.map((d) => d.lineStyle?.color)).toEqual([
      'var(team-ally)',
      'var(team-enemy)',
    ])
  })

  it('ÉCARTE une capture hors des intervalles servis, plutôt que de la poser au hasard', async () => {
    const horsPiste: MatchObjectiveEvent[] = [{ ...CAPTURES[0], timeMs: 99_000 }]
    const opt = await optionDe(tugOfWar(horsPiste))
    expect(opt.series.some((s) => s.name === t.combatCtfCaptureLabel)).toBe(false)
  })

  it('N’AJOUTE AUCUNE SÉRIE sans capture', async () => {
    const avec = await optionDe(tugOfWar(CAPTURES))
    const sans = await optionDe(tugOfWar(null))
    expect(sans.series).toHaveLength(avec.series.length - 1)
  })
})
