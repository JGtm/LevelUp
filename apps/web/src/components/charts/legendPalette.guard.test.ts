/**
 * legendPalette.guard.test — AUCUNE PASTILLE DE LÉGENDE NE DOIT PORTER UNE COULEUR D'ECHARTS.
 *
 * LE DÉFAUT QUE CE GARDE-RAIL FIGE (retours utilisateur 2026-09-09, « les couleurs des
 * légendes ne suivent pas les couleurs du graphe »). ECharts peint l'icône d'une entrée de
 * légende à partir du style de la SÉRIE. Une série `bar` dont la couleur ne vit que sur ses
 * POINTS — barres colorées par seuil, par palier de performance, par joueur — n'a pas de
 * style de série : le moteur retombe alors sur SA palette par défaut
 * (`tokens.color.theme` : `#5070dd`, `#b6d634`, …) et la légende annonce une couleur qui
 * n'existe nulle part dans le graphe.
 *
 * POURQUOI UN RENDU ET PAS UNE ASSERTION SUR L'OPTION. Le repli de palette n'est PAS visible
 * dans l'option : il se produit à la peinture. On rend donc chaque graphe hors navigateur
 * (`echarts.init(null, null, { ssr: true, renderer: 'svg' })`) et on lit les couleurs
 * réellement posées dans le SVG. Une assertion sur `legend.data` seule laisserait passer un
 * builder qui oublie une entrée.
 *
 * LA LISTE INTERDITE EST MESURÉE, jamais recopiée : on rend un graphe TÉMOIN de séries sans
 * aucune couleur et on relève ce que le moteur y pose de lui-même. Une montée de version
 * d'ECharts qui change sa palette met donc ce garde-rail à jour toute seule, sans dépendre
 * d'un chemin interne du paquet (`echarts/lib/visual/tokens.js` n'est pas typé).
 *
 * CE QUE CE TEST NE COUVRE PAS : les graphes dont l'option est construite en ligne dans un
 * composant React (page Solo, profil d'intensité…) — il faudrait les monter. Ils sont
 * couverts par les tests de leur feature.
 */
import * as echarts from 'echarts'
import { beforeEach, describe, expect, it } from 'vitest'

import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import type { EChartsCoreOption } from 'echarts/core'

import { buildSessionOcdrBarsOption } from '@/features/session-detail/SessionOcdrBars'
import { buildMapPerfVsHistoryOption } from '@/features/squad/charts/mapPerfVsHistoryChart'
import {
  buildPerformanceLineOption,
  buildTeamMMROption,
} from '@/features/squad/charts/squadPerformanceLineCharts'
import { buildSquadPerMinuteOption } from '@/features/squad/charts/squadPerMinuteChart'
import { buildSquadEfficiencyOption } from '@/features/squad/charts/squadEfficiencyChart'
import { buildWinRateVsHistoryBulletOption } from '@/features/squad/charts/winRateVsHistoryBulletChart'

/**
 * La palette par défaut du moteur, MESURÉE : douze séries `bar` sans la moindre couleur, et
 * l'on relève ce que le rendu y pose. C'est exactement le repli que ce garde-rail traque,
 * observé à sa source plutôt que recopié.
 */
function enginePalette(): string[] {
  const witnessFills = (seriesCount: number): Set<string> => {
    const chart = echarts.init(null, null, { renderer: 'svg', ssr: true, width: 320, height: 200 })
    chart.setOption({
      xAxis: { type: 'category', data: ['a'] },
      yAxis: { type: 'value' },
      series: Array.from({ length: seriesCount }, (_, i) => ({
        name: `s${i + 1}`,
        type: 'bar',
        data: [1],
      })),
    })
    const svg = chart.renderToSVGString()
    chart.dispose()
    const fills = new Set<string>()
    for (const m of svg.matchAll(/fill="(#[0-9a-f]{6})"/gi)) fills.add(m[1].toLowerCase())
    return fills
  }
  // Différence entre un témoin À SÉRIES et un témoin NU : ce que le moteur ajoute quand il
  // doit colorer des séries, et rien d'autre. Sans cette soustraction on garderait aussi
  // l'encre des axes et du texte, qui n'a rien d'interdit.
  const chrome = witnessFills(0)
  return [...witnessFills(12)].filter((c) => !chrome.has(c))
}

const ECHARTS_PALETTE: string[] = enginePalette()

/**
 * Couleurs réellement VISIBLES dans le SVG rendu — remplissages et traits.
 *
 * Un trait d'épaisseur nulle est ignoré : zrender écrit systématiquement un `stroke` sur
 * chaque forme, y compris quand `stroke-width="0"` la rend invisible (c'est le cas des
 * zones de fond `markArea`). Le compter reviendrait à accuser un graphe pour une couleur
 * que personne ne voit.
 */
function paintedColors(option: EChartsCoreOption): string[] {
  const chart = echarts.init(null, null, { renderer: 'svg', ssr: true, width: 640, height: 360 })
  chart.setOption(option)
  const svg = chart.renderToSVGString()
  chart.dispose()
  const seen = new Set<string>()
  for (const [tag] of svg.matchAll(/<[a-z]+\s[^>]*>/g)) {
    const fill = /fill="([^"]+)"/.exec(tag)?.[1]
    if (fill && fill !== 'none') seen.add(fill.toLowerCase())
    const invisible = /stroke-width="0"/.test(tag)
    const stroke = /\sstroke="([^"]+)"/.exec(tag)?.[1]
    if (!invisible && stroke && stroke !== 'none') seen.add(stroke.toLowerCase())
  }
  return [...seen]
}

/** Les entrées de `legend.data` sous forme d'objets — celles qui portent leur couleur. */
function legendEntryColors(option: EChartsCoreOption): Array<string | undefined> {
  const legend = (option as { legend?: { data?: unknown } }).legend
  const data = Array.isArray(legend?.data) ? legend.data : []
  return data.map((e) => (e as { itemStyle?: { color?: string } })?.itemStyle?.color)
}

// ─── Jeux de données minimaux ────────────────────────────────────────────────

const COLOR_BY_PLAYER = { Me: '#aa3366', F1: '#22aa88' }
const PLAYERS = ['Me', 'F1']

const perfPoint = (order: number, o: Record<string, unknown> = {}) => ({
  match_order: order,
  kills: 12,
  deaths: 9,
  assists: 4,
  kda: 1.4,
  accuracy: 48,
  avg_life_seconds: 42,
  performance_score: 63,
  max_killing_spree: 5,
  damage_dealt: 3200,
  damage_taken: 2800,
  rendement_offensif: 1.1,
  resistance_defensive: 1.3,
  map_name: 'Streets',
  ...o,
})

/** Deux joueurs, deux matchs — dont un sous le seuil FDA pour déclencher la couleur opposée. */
const ROWS_BY_PLAYER = {
  Me: [perfPoint(0), perfPoint(1, { kda: 0.6, skill_rating: 1450, skill_rating_type: 'csr', skill_delta: -12, team_mmr: 1500 })],
  F1: [perfPoint(0, { skill_rating: 1300, skill_rating_type: 'lusr', skill_delta: 8, team_mmr: 1500 }), perfPoint(1)],
} as unknown as Parameters<typeof buildPerformanceLineOption>[0]

const MAP_ROWS = [
  { map_ui: 'streets', match_count: 6, win_rate: 0.66, historical_win_rate: 0.5, historical_match_count: 40, performance_avg: 71, historical_performance_avg: 58 },
  { map_ui: 'aquarius', match_count: 3, win_rate: 0, historical_win_rate: 0.42, historical_match_count: 25, performance_avg: 31, historical_performance_avg: 55 },
]

const MAP_SERIES = [{ key: 'maps', datapoints: MAP_ROWS }] as never
const MAP_OPTS = { mapLabelOf: (m: string) => m, sessionLabel: 'Session', historyLabel: 'Historique' }

const EFFICIENCY_LABELS = {
  offensiveMetric: 'Rendement',
  defensiveMetric: 'Résistance',
  oneLife: '1 vie',
  damageDealt: 'Dégâts infligés',
  damageTaken: 'Dégâts subis',
  perFrag: '/ frag effectif',
  perDeath: '/ mort',
}

/** Un graphe migré : son nom, et son option construite au moment du test. */
const CHARTS: Array<{ name: string; build: () => EChartsCoreOption }> = [
  {
    name: 'Taux de victoire — Session vs Historique',
    build: () =>
      buildWinRateVsHistoryBulletOption(MAP_SERIES, {
        ...MAP_OPTS,
        parityLabel: 'Parité',
        zeroWinrateLabel: '0 %',
      }),
  },
  {
    name: 'Performance par carte — Session vs Historique',
    build: () => buildMapPerfVsHistoryOption(MAP_SERIES, MAP_OPTS),
  },
  {
    name: 'Escouade — FDA (barres colorées par seuil)',
    build: () =>
      buildPerformanceLineOption(ROWS_BY_PLAYER, {
        colorByPlayer: COLOR_BY_PLAYER,
        playerOrder: PLAYERS,
        metric: 'kda',
        decimals: 2,
        chartType: 'bar',
        complementBelowValue: 1,
        redReferenceLineAt: 1,
      }),
  },
  {
    name: 'Escouade — Performance (barres sur zones de palier)',
    build: () =>
      buildPerformanceLineOption(ROWS_BY_PLAYER, {
        colorByPlayer: COLOR_BY_PLAYER,
        playerOrder: PLAYERS,
        metric: 'performance_score',
        chartType: 'bar',
        showPerformanceZones: true,
      }),
  },
  {
    name: 'Escouade — Rang & MMR équipe',
    build: () =>
      buildTeamMMROption(ROWS_BY_PLAYER, {
        colorByPlayer: COLOR_BY_PLAYER,
        playerOrder: PLAYERS,
        mmrLabel: 'MMR équipe',
      }),
  },
  {
    name: 'Escouade — Stats par minute',
    build: () =>
      buildSquadPerMinuteOption(
        [
          {
            key: 'per-minute',
            datapoints: [
              { player: 'Me', kills_per_minute: 1.2, deaths_per_minute: 0.9, assists_per_minute: 0.4 },
              { player: 'F1', kills_per_minute: 0.8, deaths_per_minute: 1.1, assists_per_minute: 0.6 },
            ],
          },
        ] as never,
        {
          colorByPlayer: COLOR_BY_PLAYER,
          metricLabels: { frags: 'Frags/min', deaths: 'Morts/min', assists: 'Assistances/min' },
          perMinuteSuffix: ' /min',
        },
      ),
  },
  {
    name: 'Escouade — Rendement',
    build: () =>
      buildSquadEfficiencyOption(ROWS_BY_PLAYER, PLAYERS, {
        metric: 'offensive',
        colorByPlayer: COLOR_BY_PLAYER,
        labels: EFFICIENCY_LABELS,
      }),
  },
  {
    name: 'Sessions — Rendement · Résistance',
    build: () =>
      buildSessionOcdrBarsOption(
        [
          {
            key: 'ocdr',
            datapoints: [
              { label: '#1 · Streets', ocNorm: 96, drNorm: 78, ocRaw: 0.86, drRaw: 1.5 },
              { label: '#2 · Aquarius', ocNorm: 112, drNorm: 40, ocRaw: 1.01, drRaw: 1.26 },
            ],
          },
        ] as never,
        { ocLabel: 'Rendement', drLabel: 'Résistance', p80Label: 'P80', meanLabel: 'Moyenne' },
      ),
  },
]

describe('légendes — aucune couleur de la palette par défaut d’ECharts', () => {
  beforeEach(() => {
    // Palette RÉELLEMENT appliquée : sans elle `resolveToken` rend la chaîne vide et le
    // moteur repeindrait tout avec sa palette — le test passerait pour la mauvaise raison.
    _resetActivePalette()
    applyPalette(defaultPalette, 'default')
  })

  it('la liste interdite est bien lue dans le paquet (sentinelle)', () => {
    expect(ECHARTS_PALETTE.length).toBeGreaterThan(4)
    expect(ECHARTS_PALETTE.every((c) => /^#[0-9a-f]{6}$/i.test(c))).toBe(true)
    // Aucune de nos encres ne coïncide avec la palette du moteur : la détection est nette.
    const ours = Object.values(defaultPalette).map((c) => c.toLowerCase())
    expect(ECHARTS_PALETTE.filter((c) => ours.includes(c.toLowerCase()))).toEqual([])
  })

  for (const { name, build } of CHARTS) {
    it(`${name} — le rendu ne pose aucune couleur du moteur`, () => {
      const painted = paintedColors(build())
      const intruses = ECHARTS_PALETTE.filter((c) => painted.includes(c.toLowerCase()))
      expect(
        intruses,
        `couleurs de la palette ECharts peintes par « ${name} » : ${intruses.join(', ')} — ` +
          'une série sans couleur de série, ou une entrée de légende sans `legendEntries`.',
      ).toEqual([])
    })

    it(`${name} — chaque entrée de légende porte sa couleur`, () => {
      const colors = legendEntryColors(build())
      expect(colors.length).toBeGreaterThan(0)
      expect(colors.filter((c) => !c)).toEqual([])
    })
  }
})
