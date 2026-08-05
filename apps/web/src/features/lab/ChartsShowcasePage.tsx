/**
 * ChartsShowcasePage — galerie des wrappers ECharts du projet.
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
import {
  FirstBloodLanes,
  type FirstBloodPlayerSeries,
} from '@/components/charts/FirstBloodLanes'
import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import { HistogramChart } from '@/components/charts/HistogramChart'
import { OutcomeSequenceTape } from '@/components/charts/OutcomeSequenceTape'
import { RadarChart } from '@/components/charts/RadarChart'
import { ScatterChart } from '@/components/charts/ScatterChart'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'

import { TimeseriesKdaBars } from '@/features/timeseries/TimeseriesKdaBars'
import type { TimeseriesMatchRow } from '@/lib/api/types'
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

// Heatmap2DChart affiche `detail.count` dans chaque cellule et formate `value`
// comme un taux de victoire (`value × 100 %` dans le tooltip) : une fixture sans
// `detail` peignait donc « 0 » sur les 84 cellules, et une `value` en 0..10 y
// affichait « 800 % ». Valeurs DÉTERMINISTES (pas de Math.random) — cette page
// est la référence du harnais de régression visuelle : une fixture aléatoire rend
// toute baseline incomparable.
const HEATMAP_SERIES = [
  {
    key: 'demo.heatmap',
    meta: { gamertag: 'heatmap' },
    datapoints: Array.from({ length: 7 * 12 }, (_, i) => {
      const hour = i % 12
      const day = Math.floor(i / 12)
      // Motif reproductible : creux en début de journée et en milieu de semaine,
      // pic le week-end en soirée — plage de taux de victoire ~0,30 à ~0,70.
      const value = 0.5 + 0.2 * Math.sin((hour / 12) * Math.PI * 2 + day)
      return {
        x: String(hour + 8).padStart(2, '0'),
        y: DOW_LABELS_FR[day],
        value: Math.round(value * 1000) / 1000,
        detail: { count: 3 + ((hour * 5 + day * 3) % 18) },
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

// `dominance` sur quelques matchs : la vitrine doit montrer l'ENCOCHE traversante
// (drapeaux 1 domination / 3 remontada / 4 débandade), sinon la bande du Lab est la
// seule des 5 surfaces où le marqueur n'apparaît jamais.
const OUTCOME_TAPE = [
  { matchId: '1', outcome: 'win' as const, map: 'Streets', dominance: 1 as const },
  { matchId: '2', outcome: 'win' as const, map: 'Streets' },
  { matchId: '3', outcome: 'loss' as const, map: 'Argyle', dominance: 4 as const },
  { matchId: '4', outcome: 'tie' as const, map: 'Aquarius' },
  { matchId: '5', outcome: 'win' as const, map: 'Live Fire', dominance: 3 as const },
  { matchId: '6', outcome: 'win' as const, map: 'Streets' },
  { matchId: '7', outcome: 'loss' as const, map: 'Recharge' },
  { matchId: '8', outcome: 'win' as const, map: 'Bazaar' },
]

// Frags/Morts/Assists/Précision par match : valeurs DÉTERMINISTES (pas de
// Math.random) — même règle que HEATMAP_SERIES ci-dessus. Cette page est la
// référence du harnais de régression visuelle (e2e/visual/lab-charts.visual.spec.ts,
// baselines PNG versionnées) : une fixture aléatoire rend toute baseline incomparable.
const KDA_ROW_STATS: Array<{ kills: number; deaths: number; assists: number; accuracy: number }> = [
  { kills: 8, deaths: 6, assists: 3, accuracy: 0.52 },
  { kills: 12, deaths: 4, assists: 5, accuracy: 0.61 },
  { kills: 15, deaths: 9, assists: 2, accuracy: 0.47 },
  { kills: 9, deaths: 5, assists: 6, accuracy: 0.58 },
  { kills: 18, deaths: 11, assists: 4, accuracy: 0.65 },
  { kills: 11, deaths: 7, assists: 3, accuracy: 0.5 },
  { kills: 14, deaths: 8, assists: 7, accuracy: 0.55 },
  { kills: 10, deaths: 4, assists: 2, accuracy: 0.63 },
  { kills: 16, deaths: 10, assists: 5, accuracy: 0.49 },
  { kills: 13, deaths: 6, assists: 4, accuracy: 0.57 },
]

const KDA_ROWS: TimeseriesMatchRow[] = KDA_ROW_STATS.map((s, i) => ({
  match_id: String(i),
  index: i,
  start_time: new Date(2026, 0, 1 + i) as unknown as string,
  kills: s.kills,
  deaths: s.deaths,
  assists: s.assists,
  accuracy: s.accuracy,
  outcome: i % 3 === 0 ? 3 : 2,
  personal_score: null,
  damage_dealt: null,
  damage_taken: null,
  perf_score: null,
  rank: null,
  playlist_name: 'Ranked Arena',
  time_played_seconds: null,
})) as unknown as TimeseriesMatchRow[]

// Premier frag / première mort : 4 joueurs, 12 matchs, quelques `null` (événement
// absent du match — exclu des médianes, compté dans « n/12 »).
function firstBloodMatches(
  kills: (number | null)[],
  deaths: (number | null)[],
): FirstBloodPlayerSeries['matches'] {
  return kills.map((k, i) => ({
    matchId: `m${i + 1}`,
    firstKillSec: k,
    firstDeathSec: deaths[i] ?? null,
  }))
}

const FIRST_BLOOD_SERIES: FirstBloodPlayerSeries[] = [
  {
    player: 'Rusher',
    matches: firstBloodMatches(
      [12, 18, 9, 26, 15, 33, 11, 21, 17, null, 24, 14],
      [40, 22, 61, 35, 18, 74, 29, 52, 44, 31, 66, 27],
    ),
  },
  {
    player: 'Sniper',
    matches: firstBloodMatches(
      [58, 72, 44, 96, 63, 81, 55, 110, 68, 77, null, 90],
      [120, 145, 88, 190, 132, 165, 99, 210, 141, 158, 176, 205],
    ),
  },
  {
    player: 'Objectif',
    matches: firstBloodMatches(
      [35, 48, 29, 62, 41, 55, 38, 70, 44, 51, 33, 59],
      [28, 33, 21, 45, 30, 39, 25, 50, 31, 36, 24, 42],
    ),
  },
  {
    player: 'Support',
    matches: firstBloodMatches(
      [80, 105, 65, 130, 92, 118, 74, 145, 99, null, 87, 124],
      [95, 88, 140, 76, 112, 69, 155, 84, 101, 71, 133, 90],
    ),
  },
]

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
          title="FirstBloodLanes"
          description="Bandes par joueur : nuages premier frag / première mort + fenêtre d'avance médiane (custom + scatter)"
        >
          <FirstBloodLanes data={FIRST_BLOOD_SERIES} />
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
