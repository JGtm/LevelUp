/**
 * SynthesisPage --- Vue synthese / bilan periodique (Slice 7).
 * Types ref: SynthesisPageResponse, SynthesisKPIs, ComparisonMetricItem
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useSynthesisPage } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import type { ComparisonMetricItem, PlotlyFigurePayload, SynthesisKPIs, SynthesisQueryRequest } from '@/lib/api/types'

const PERIOD_OPTIONS = [
  { value: 'all', label: 'Tout' },
  { value: '2y', label: '2 ans' },
  { value: '1y', label: '1 an' },
  { value: '1m', label: '1 mois' },
  { value: '1w', label: '1 semaine' },
] as const
type Period = typeof PERIOD_OPTIONS[number]['value']

/** Construit le graphique bipolaire Solo ← / → Escouade depuis les ComparisonMetricItem. */
function buildBipolaireChart(metrics: ComparisonMetricItem[]): PlotlyFigurePayload {
  // Ordre inversé : métrique la plus importante en haut
  const reversed = [...metrics].reverse()
  const labels = reversed.map((m) => m.label)
  const soloVals = reversed.map((m) => -Math.abs(m.solo_value))   // Solo à gauche (négatif)
  const squadVals = reversed.map((m) => Math.abs(m.squad_value))  // Escouade à droite (positif)
  const soloTexts = reversed.map((m) => m.solo_text)
  const squadTexts = reversed.map((m) => m.squad_text)

  const height = Math.max(320, 70 * metrics.length)

  return {
    data: [
      {
        type: 'bar',
        name: 'Solo',
        orientation: 'h',
        x: soloVals,
        y: labels,
        text: soloTexts,
        textposition: 'outside',
        marker: { color: '#06B6D4' },   // cyan-500
        hovertemplate: '<b>Solo</b>: %{text}<extra></extra>',
      },
      {
        type: 'bar',
        name: 'Escouade',
        orientation: 'h',
        x: squadVals,
        y: labels,
        text: squadTexts,
        textposition: 'outside',
        marker: { color: '#22C55E' },   // green-500
        hovertemplate: '<b>Escouade</b>: %{text}<extra></extra>',
      },
    ],
    layout: {
      height,
      barmode: 'overlay',
      bargap: 0.3,
      margin: { l: 110, r: 80, t: 20, b: 40 },
      xaxis: {
        tickformat: '.2f',
        zeroline: true,
        zerolinewidth: 2,
        zerolinecolor: 'rgba(100,116,139,0.8)',  // slate opacity 0.8
        showticklabels: false,
        fixedrange: true,
      },
      yaxis: { automargin: true, fixedrange: true },
      legend: { orientation: 'h', x: 0.5, xanchor: 'center', y: -0.08 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
      font: { size: 12 },
      shapes: [
        {
          type: 'line',
          x0: 0, x1: 0,
          y0: -0.5, y1: labels.length - 0.5,
          xref: 'x', yref: 'y',
          line: { color: 'rgba(100,116,139,0.8)', width: 2 },
        },
      ],
    },
  }
}

interface MetricRowProps { item: ComparisonMetricItem }
function MetricRow({ item }: MetricRowProps) {
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3 text-sm font-medium text-gray-700">{item.label}</td>
      <td className="px-4 py-3 text-center text-sm text-cyan-600">{item.solo_text}</td>
      <td className="px-4 py-3 text-center text-sm text-green-600">{item.squad_text}</td>
    </tr>
  )
}

interface KPISectionProps { title: string; kpis: SynthesisKPIs }
function KPISection({ title, kpis }: KPISectionProps) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-gray-600 mb-2">{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">Win Rate</span><span className="text-xl font-bold">{(kpis.win_rate * 100).toFixed(1)}%</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">K/D</span><span className="text-xl font-bold">{kpis.kd_ratio?.toFixed(2) ?? '-'}</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">Matchs</span><span className="text-xl font-bold">{kpis.match_count}</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">Perf.</span><span className="text-xl font-bold">{kpis.performance_score?.toFixed(0) ?? '-'}</span></div>
      </div>
    </div>
  )
}

export function SynthesisPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/synthesis' })
  const { filterContext } = useGlobalFilterStore()
  const [period, setPeriod] = useState<Period>('all')
  const request: SynthesisQueryRequest = { filters: filterContext, period }
  const { data, isLoading, isError, error } = useSynthesisPage(playerSlug, period, request)

  if (isLoading) return <div className="flex items-center justify-center min-h-64"><Spinner size="lg" /></div>
  if (isError) return <div className="p-8 text-center text-red-600">Erreur : {String(error)}</div>
  if (!data) return null

  const bipolaireChart = data.comparison_metrics.length > 0
    ? buildBipolaireChart(data.comparison_metrics)
    : null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Synthese"
        subtitle="Bilan global et comparaison solo / escouade"
        actions={
          <div className="flex gap-1 rounded-lg border p-1">
            {PERIOD_OPTIONS.map((opt) => (
              <button key={opt.value} onClick={() => setPeriod(opt.value)} className={`rounded px-3 py-1 text-sm transition-colors ${period === opt.value ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>{opt.label}</button>
            ))}
          </div>
        }
      />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader><CardTitle>Solo ({data.solo_kpis.match_count} matchs)</CardTitle></CardHeader>
          <CardContent><KPISection title="" kpis={data.solo_kpis} /></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Escouade ({data.squad_kpis.match_count} matchs)</CardTitle></CardHeader>
          <CardContent><KPISection title="" kpis={data.squad_kpis} /></CardContent>
        </Card>
      </div>

      {/* Graphique bipolaire Solo vs Escouade */}
      {bipolaireChart ? (
        <Card>
          <CardHeader>
            <CardTitle>
              Solo
              <span className="mx-2 text-cyan-500">←</span>
              vs
              <span className="mx-2 text-green-500">→</span>
              Escouade
            </CardTitle>
          </CardHeader>
          <CardContent>
            <PlotlyChart figure={bipolaireChart} />
            <p className="mt-2 text-xs text-gray-400 text-center">
              Solo : {data.solo_kpis.match_count} matchs · Escouade : {data.squad_kpis.match_count} matchs
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="py-6 text-center text-gray-400 text-sm">
            Pas assez de données des deux contextes pour afficher la comparaison.
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle>Comparaison detaillee</CardTitle></CardHeader>
        <CardContent className="p-0">
          {data.comparison_metrics.length === 0 ? (
            <p className="p-6 text-center text-gray-500">Pas assez de donnees pour cette periode.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="px-4 py-3 text-left">Metrique</th>
                    <th className="px-4 py-3 text-center text-cyan-600">Solo</th>
                    <th className="px-4 py-3 text-center text-green-600">Escouade</th>
                  </tr>
                </thead>
                <tbody>
                  {data.comparison_metrics.map((item, idx) => <MetricRow key={idx} item={item} />)}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}


const PERIOD_OPTIONS = [
  { value: 'all', label: 'Tout' },
  { value: '2y', label: '2 ans' },
  { value: '1y', label: '1 an' },
  { value: '1m', label: '1 mois' },
  { value: '1w', label: '1 semaine' },
] as const
type Period = typeof PERIOD_OPTIONS[number]['value']

interface MetricRowProps { item: ComparisonMetricItem }
function MetricRow({ item }: MetricRowProps) {
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3 text-sm font-medium text-gray-700">{item.label}</td>
      <td className="px-4 py-3 text-center text-sm">{item.solo_text}</td>
      <td className="px-4 py-3 text-center text-sm">{item.squad_text}</td>
    </tr>
  )
}

interface KPISectionProps { title: string; kpis: SynthesisKPIs }
function KPISection({ title, kpis }: KPISectionProps) {
  return (
    <div>
      <h3 className="text-sm font-semibold text-gray-600 mb-2">{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">Win Rate</span><span className="text-xl font-bold">{(kpis.win_rate * 100).toFixed(1)}%</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">K/D</span><span className="text-xl font-bold">{kpis.kd_ratio?.toFixed(2) ?? '-'}</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">Matchs</span><span className="text-xl font-bold">{kpis.match_count}</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-gray-500 block">Perf.</span><span className="text-xl font-bold">{kpis.performance_score?.toFixed(0) ?? '-'}</span></div>
      </div>
    </div>
  )
}

export function SynthesisPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/synthesis' })
  const { filterContext } = useGlobalFilterStore()
  const [period, setPeriod] = useState<Period>('all')
  const request: SynthesisQueryRequest = { filters: filterContext, period }
  const { data, isLoading, isError, error } = useSynthesisPage(playerSlug, period, request)

  if (isLoading) return <div className="flex items-center justify-center min-h-64"><Spinner size="lg" /></div>
  if (isError) return <div className="p-8 text-center text-red-600">Erreur : {String(error)}</div>
  if (!data) return null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Synthese"
        subtitle="Bilan global et comparaison solo / escouade"
        actions={
          <div className="flex gap-1 rounded-lg border p-1">
            {PERIOD_OPTIONS.map((opt) => (
              <button key={opt.value} onClick={() => setPeriod(opt.value)} className={`rounded px-3 py-1 text-sm transition-colors ${period === opt.value ? 'bg-blue-600 text-white' : 'text-gray-600 hover:bg-gray-100'}`}>{opt.label}</button>
            ))}
          </div>
        }
      />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader><CardTitle>Solo ({data.solo_kpis.match_count} matchs)</CardTitle></CardHeader>
          <CardContent><KPISection title="" kpis={data.solo_kpis} /></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Escouade ({data.squad_kpis.match_count} matchs)</CardTitle></CardHeader>
          <CardContent><KPISection title="" kpis={data.squad_kpis} /></CardContent>
        </Card>
      </div>
      <Card>
        <CardHeader><CardTitle>Comparaison detaillee</CardTitle></CardHeader>
        <CardContent className="p-0">
          {data.comparison_metrics.length === 0 ? (
            <p className="p-6 text-center text-gray-500">Pas assez de donnees pour cette periode.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="px-4 py-3 text-left">Metrique</th>
                    <th className="px-4 py-3 text-center">Solo</th>
                    <th className="px-4 py-3 text-center">Escouade</th>
                  </tr>
                </thead>
                <tbody>
                  {data.comparison_metrics.map((item, idx) => <MetricRow key={idx} item={item} />)}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
