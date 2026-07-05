/**
 * ChartsShowcasePage — galerie des 11 wrappers ECharts du projet.
 *
 * Phase 3 Option C : alternative légère à Storybook. Rend chaque wrapper
 * avec des données de démo statiques pour servir de :
 *   - Documentation visuelle des composants disponibles
 *   - Sandbox de régression visuelle manuelle
 *   - Référence d'usage pour les nouveaux écrans
 *
 * Aucune nouvelle dépendance ajoutée — utilise la stack existante
 * (TanStack Router, Tailwind, Card, ChartCard).
 *
 * Page exclusivement destinée au dev sandbox (`/lab/charts`). Les libellés
 * sont intentionnellement hardcodés (noms de composants, descriptions de
 * dev) — exemption de la règle no-hardcoded-strings.
 */
/* eslint-disable @levelup/no-hardcoded-strings */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { DonutChart } from '@/components/charts/DonutChart'
import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import { HistogramChart } from '@/components/charts/HistogramChart'
import { OutcomeSequenceTape } from '@/components/charts/OutcomeSequenceTape'
import { RadarChart } from '@/components/charts/RadarChart'
import { ScatterChart } from '@/components/charts/ScatterChart'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'

import { TimeseriesCombatYield } from '@/features/timeseries/TimeseriesCombatYield'
import { TimeseriesKdaBars } from '@/features/timeseries/TimeseriesKdaBars'
import type { MatchHistoryRow, TimeseriesMatchRow } from '@/lib/api/types'
import { DOW_LABELS_FR } from '@/lib/formatters'

// ─── Sample data ───────────────────────────────────────────────────────────

const TIMESERIES_LINE_SERIES = [
  {
    key: 'demo.kd',
    meta: { gamertag: 'K/D cumulé' },
    datapoints: Array.from({ length: 12 }, (_, i) => ({
      x: new Date(2026, 0, 1 + i * 2).toISOString(),
      y: 0.8 + Math.sin(i / 2) * 0.4 + i * 0.05,
    })),
  },
  {
    key: 'demo.score',
    meta: { gamertag: 'Score moyen' },
    datapoints: Array.from({ length: 12 }, (_, i) => ({
      x: new Date(2026, 0, 1 + i * 2).toISOString(),
      y: 1.5 + Math.cos(i / 3) * 0.3,
    })),
  },
]

const BAR_STACKED_SERIES = [
  {
    key: 'demo.outcomes',
    meta: { gamertag: 'outcomes' },
    datapoints: [
      { category: 'Lun', components: { win: 4, loss: 2, tie: 1 } as Record<string, number> },
      { category: 'Mar', components: { win: 3, loss: 4, tie: 0 } as Record<string, number> },
      { category: 'Mer', components: { win: 6, loss: 1, tie: 2 } as Record<string, number> },
      { category: 'Jeu', components: { win: 2, loss: 3, tie: 0 } as Record<string, number> },
      { category: 'Ven', components: { win: 5, loss: 2, tie: 1 } as Record<string, number> },
    ],
  },
]

const BAR_GROUPED_SERIES = [
  {
    key: 'demo.medals',
    meta: { gamertag: 'medals' },
    datapoints: [
      { category: 'Killjoy', components: { Filtré: 8, Total: 14 } },
      { category: 'Sniper', components: { Filtré: 5, Total: 9 } },
      { category: 'Quigley', components: { Filtré: 3, Total: 7 } },
    ],
  },
]

const DONUT_SERIES = [
  {
    key: 'demo.donut',
    meta: { gamertag: 'donut' },
    datapoints: [
      { name: 'Victoires', value: 12 },
      { name: 'Défaites', value: 5 },
      { name: 'Égalités', value: 2 },
    ],
  },
]

const HEATMAP_SERIES = [
  {
    key: 'demo.heatmap',
    meta: { gamertag: 'heatmap' },
    datapoints: Array.from({ length: 7 * 12 }, (_, i) => {
      const hour = i % 12
      const day = Math.floor(i / 12)
      return {
        x: String(hour + 8).padStart(2, '0'),
        y: DOW_LABELS_FR[day],
        value: Math.round(Math.random() * 10),
      }
    }),
  },
]

const HISTOGRAM_SERIES = [
  {
    key: 'demo.kda_dist',
    meta: { gamertag: 'kda' },
    datapoints: [
      { binStart: 0, binEnd: 0.5, count: 1 },
      { binStart: 0.5, binEnd: 1.0, count: 4 },
      { binStart: 1.0, binEnd: 1.5, count: 9 },
      { binStart: 1.5, binEnd: 2.0, count: 7 },
      { binStart: 2.0, binEnd: 2.5, count: 3 },
      { binStart: 2.5, binEnd: 3.0, count: 1 },
    ],
  },
]

const SCATTER_SERIES = [
  {
    key: 'outcome.win',
    meta: { gamertag: 'Victoires' },
    datapoints: [
      { x: 8, y: 1.6 },
      { x: 12, y: 2.4 },
      { x: 6, y: 1.2 },
      { x: 15, y: 3.0 },
    ],
  },
  {
    key: 'outcome.loss',
    meta: { gamertag: 'Défaites' },
    datapoints: [
      { x: 3, y: 0.6 },
      { x: 5, y: 0.8 },
      { x: 4, y: 0.4 },
    ],
  },
]

const RADAR_SERIES = [
  {
    key: 'demo.player_a',
    meta: { gamertag: 'Joueur A' },
    axes: [
      { axis: 'Combat', value: 78, raw: 78 },
      { axis: 'Survie', value: 62, raw: 62 },
      { axis: 'Support', value: 84, raw: 84 },
      { axis: 'Score', value: 71, raw: 71 },
      { axis: 'Objectif', value: 55, raw: 55 },
      { axis: 'Impact', value: 69, raw: 69 },
    ],
  },
  {
    key: 'demo.player_b',
    meta: { gamertag: 'Joueur B' },
    axes: [
      { axis: 'Combat', value: 65, raw: 65 },
      { axis: 'Survie', value: 80, raw: 80 },
      { axis: 'Support', value: 50, raw: 50 },
      { axis: 'Score', value: 75, raw: 75 },
      { axis: 'Objectif', value: 88, raw: 88 },
      { axis: 'Impact', value: 60, raw: 60 },
    ],
  },
]

const OUTCOME_TAPE = [
  { matchId: '1', outcome: 'win' as const, map: 'Streets' },
  { matchId: '2', outcome: 'win' as const, map: 'Streets' },
  { matchId: '3', outcome: 'loss' as const, map: 'Argyle' },
  { matchId: '4', outcome: 'tie' as const, map: 'Aquarius' },
  { matchId: '5', outcome: 'win' as const, map: 'Live Fire' },
  { matchId: '6', outcome: 'win' as const, map: 'Streets' },
  { matchId: '7', outcome: 'loss' as const, map: 'Recharge' },
  { matchId: '8', outcome: 'win' as const, map: 'Bazaar' },
]

const COMBAT_YIELD_ROWS: MatchHistoryRow[] = Array.from({ length: 8 }, (_, i) => ({
  match_id: String(i),
  start_time: new Date(2026, 0, 1 + i).toISOString(),
  outcome: i % 3 === 0 ? 3 : 2,
  outcome_label: '',
  outcome_tone: 'positive',
  detail: '',
  is_favorite: false,
  offensive_conversion: 0.6 + Math.sin(i / 2) * 0.3,
  defensive_resistance: 1.2 + Math.cos(i / 2) * 0.5,
})) as unknown as MatchHistoryRow[]

const KDA_ROWS: TimeseriesMatchRow[] = Array.from({ length: 10 }, (_, i) => ({
  match_id: String(i),
  index: i,
  start_time: new Date(2026, 0, 1 + i) as unknown as string,
  kills: 8 + Math.floor(Math.random() * 12),
  deaths: 4 + Math.floor(Math.random() * 8),
  assists: 2 + Math.floor(Math.random() * 6),
  accuracy: 0.45 + Math.random() * 0.2,
  outcome: i % 3 === 0 ? 3 : 2,
  personal_score: null,
  damage_dealt: null,
  damage_taken: null,
  perf_score: null,
  rank: null,
  playlist_name: 'Ranked Arena',
  time_played_seconds: null,
})) as unknown as TimeseriesMatchRow[]

// ─── Page ──────────────────────────────────────────────────────────────────

interface ShowcaseSectionProps {
  title: string
  description: string
  children: React.ReactNode
}

function ShowcaseSection({ title, description, children }: ShowcaseSectionProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <p className="text-xs text-muted-foreground">{description}</p>
      </CardHeader>
      <CardContent className="pb-4">{children}</CardContent>
    </Card>
  )
}

export function ChartsShowcasePage() {
  return (
    <div className="space-y-6 p-6">
      <header>
        <h1 className="text-2xl font-bold">Charts showcase</h1>
        <p className="text-sm text-muted-foreground">
          Galerie de tous les wrappers ECharts du projet avec données de démo. Sert de
          documentation visuelle et de sandbox de régression manuelle.
        </p>
      </header>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <ShowcaseSection
          title="TimeseriesLineChart"
          description="Multi-séries temporelles avec axe X time/category/value"
        >
          <TimeseriesLineChart series={TIMESERIES_LINE_SERIES} height={260} xAxisType="time" />
        </ShowcaseSection>

        <ShowcaseSection
          title="BarStackedChart"
          description="Barres empilées avec couleurs par composante"
        >
          <BarStackedChart
            series={BAR_STACKED_SERIES}
            height={260}
            componentColors={{
              win: 'outcome-win',
              loss: 'outcome-loss',
              tie: 'outcome-draw',
            }}
            componentOrder={['win', 'tie', 'loss']}
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="BarGroupedChart"
          description="Barres groupées (côte-à-côte)"
        >
          <BarGroupedChart
            series={BAR_GROUPED_SERIES}
            height={260}
            componentColors={{ Filtré: 'chart-series-1', Total: 'chart-series-3' }}
            componentOrder={['Filtré', 'Total']}
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="DonutChart"
          description="Pie/donut avec couleurs par slice"
        >
          <DonutChart
            series={DONUT_SERIES}
            height={260}
            sliceColors={{
              Victoires: 'outcome-win',
              Défaites: 'outcome-loss',
              Égalités: 'outcome-draw',
            }}
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="Heatmap2DChart"
          description="Heatmap 2D avec gradient sequential"
        >
          <Heatmap2DChart series={HEATMAP_SERIES} height={260} paletteMode="sequential" />
        </ShowcaseSection>

        <ShowcaseSection
          title="HistogramChart"
          description="Histogramme depuis buckets {binStart, binEnd, count}"
        >
          <HistogramChart
            series={HISTOGRAM_SERIES}
            height={260}
            colorToken="perf-tier-2"
            xAxisLabel="K/D"
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="ScatterChart"
          description="Scatter multi-séries avec couleurs par série"
        >
          <ScatterChart
            series={SCATTER_SERIES}
            height={260}
            xAxisLabel="Kills"
            yAxisLabel="K/D"
            seriesColorTokens={{
              'outcome.win': 'outcome-win',
              'outcome.loss': 'outcome-loss',
            }}
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="RadarChart"
          description="Radar 6 axes multi-joueurs (0..100 normalisés)"
        >
          <RadarChart series={RADAR_SERIES} height={280} />
        </ShowcaseSection>

        <ShowcaseSection
          title="OutcomeSequenceTape"
          description="Bande narrative outcomes (custom series)"
        >
          <OutcomeSequenceTape
            matches={OUTCOME_TAPE}
            labels={{ win: 'Win', loss: 'Loss', tie: 'Tie', dnf: 'DNF' }}
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="TimeseriesCombatYield"
          description="2 courbes OC + DR avec markLine références p80"
        >
          <TimeseriesCombatYield
            rows={COMBAT_YIELD_ROWS}
            height={260}
            labels={{
              ocSeries: 'Offensive (OC)',
              drSeries: 'Defensive (DR)',
              ocReference: 'p80 OC',
              drReference: 'p80 DR',
              emptyTitle: '',
              emptyDescription: '',
            }}
          />
        </ShowcaseSection>

        <ShowcaseSection
          title="TimeseriesKdaBars"
          description="Bars K colorées par outcome + bars D négatives + line K/D"
        >
          <TimeseriesKdaBars
            rows={KDA_ROWS}
            height={260}
            labels={{
              kills: 'Kills',
              deaths: 'Deaths',
              kdRatio: 'K/D',
              yAxisLeft: 'Kills / Deaths',
              yAxisRight: 'K/D',
              emptyTitle: '',
              emptyDescription: '',
            }}
          />
        </ShowcaseSection>
      </div>
    </div>
  )
}
