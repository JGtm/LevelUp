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
import type { ComparisonMetricItem, SynthesisKPIs, SynthesisQueryRequest } from '@/lib/api/types'

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
