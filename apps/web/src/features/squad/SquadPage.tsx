/**
 * SquadPage --- Vue coequipiers / escouade (Slice 6).
 * Types ref: TeammateRow, TeammateKPIs, TeammatesPageResponse
 *
 * Features :
 * - Table de coéquipiers avec sélection
 * - Onglet Synergies : comparaison Avec/Sans/Solo en barres groupées
 * - Onglet Contributions : radar de performance
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useTeammates } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import type { TeammateRow, TeammateKPIs, TeammatesQueryRequest, PlotlyFigurePayload } from '@/lib/api/types'

type TabId = 'synergies' | 'contributions'
const TABS: { id: TabId; label: string }[] = [
  { id: 'synergies', label: 'Synergies' },
  { id: 'contributions', label: 'Contributions' },
]

// ─── Helpers graphiques ───────────────────────────────────────────────────────

function buildSynergiesChart(
  gamertag: string,
  withKpis: TeammateKPIs,
  withoutKpis: TeammateKPIs | null,
  soloRef: TeammateKPIs | null,
): PlotlyFigurePayload {
  const metrics = ['Win Rate', 'K/D', 'Kills/partie', 'Assists/partie']

  const extract = (k: TeammateKPIs | null) => k ? [
    k.win_rate * 100,
    k.kd_ratio ?? 0,
    k.kills_per_game ?? 0,
    k.assists_per_game ?? 0,
  ] : null

  const withVals = extract(withKpis)
  const withoutVals = extract(withoutKpis)
  const soloVals = extract(soloRef)

  const traces: PlotlyFigurePayload['data'] = [
    {
      type: 'bar', name: `Avec ${gamertag}`, orientation: 'v',
      x: metrics, y: withVals,
      marker: { color: '#7C3AED' },
    },
  ]
  if (withoutVals) traces.push({
    type: 'bar', name: 'Sans ce coéquipier', orientation: 'v',
    x: metrics, y: withoutVals,
    marker: { color: '#94A3B8' },
  })
  if (soloVals) traces.push({
    type: 'bar', name: 'Référence solo', orientation: 'v',
    x: metrics, y: soloVals,
    marker: { color: '#06B6D4' },
  })

  return {
    data: traces,
    layout: {
      barmode: 'group',
      height: 320,
      margin: { l: 40, r: 20, t: 20, b: 60 },
      xaxis: { automargin: true },
      yaxis: { automargin: true },
      legend: { orientation: 'h', x: 0, y: -0.2 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
    },
  }
}

function buildRadarChart(
  gamertag: string,
  withKpis: TeammateKPIs,
  soloRef: TeammateKPIs | null,
): PlotlyFigurePayload {
  const axes = ['Win Rate', 'K/D', 'Kills/partie', 'Assists/partie', 'Précision']
  const norm = (v: number | null, max: number) => v != null ? Math.min(100, (v / max) * 100) : 0

  const withVals = [
    withKpis.win_rate * 100,
    norm(withKpis.kd_ratio, 3),
    norm(withKpis.kills_per_game, 20),
    norm(withKpis.assists_per_game, 10),
    norm(withKpis.accuracy, 1) * 100,
  ]
  const soloVals = soloRef ? [
    soloRef.win_rate * 100,
    norm(soloRef.kd_ratio, 3),
    norm(soloRef.kills_per_game, 20),
    norm(soloRef.assists_per_game, 10),
    norm(soloRef.accuracy, 1) * 100,
  ] : null

  const data: PlotlyFigurePayload['data'] = [
    {
      type: 'scatterpolar',
      name: `Avec ${gamertag}`,
      r: [...withVals, withVals[0]],
      theta: [...axes, axes[0]],
      fill: 'toself',
      marker: { color: '#7C3AED' },
      line: { color: '#7C3AED' },
    },
  ]
  if (soloVals) data.push({
    type: 'scatterpolar',
    name: 'Solo ref',
    r: [...soloVals, soloVals[0]],
    theta: [...axes, axes[0]],
    fill: 'toself',
    opacity: 0.4,
    marker: { color: '#06B6D4' },
    line: { color: '#06B6D4', dash: 'dot' },
  })

  return {
    data,
    layout: {
      height: 380,
      polar: { radialaxis: { visible: true, range: [0, 100] } },
      legend: { orientation: 'h', x: 0.5, xanchor: 'center', y: -0.08 },
      margin: { l: 40, r: 40, t: 30, b: 60 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
    },
  }
}

// ─── KPI Block ────────────────────────────────────────────────────────────────

interface KPICardProps { label: string; value: number | null; unit?: string }
function KPICard({ label, value, unit = '' }: KPICardProps) {
  const display = value == null ? '-' : `${Number.isInteger(value) ? value : value.toFixed(2)}${unit}`
  return (
    <div className="flex flex-col gap-1 rounded-lg border p-3">
      <span className="text-xs text-gray-500 uppercase tracking-wide">{label}</span>
      <span className="text-xl font-bold">{display}</span>
    </div>
  )
}

interface KPIBlockProps { title: string; kpis: TeammateKPIs }
function KPIBlock({ title, kpis }: KPIBlockProps) {
  return (
    <div>
      <h3 className="text-sm font-medium text-gray-600 mb-2">{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <KPICard label="Matchs" value={kpis.match_count} />
        <KPICard label="Win Rate" value={kpis.win_rate * 100} unit="%" />
        <KPICard label="K/D" value={kpis.kd_ratio} />
        <KPICard label="Kills / match" value={kpis.kills_per_game} />
      </div>
    </div>
  )
}

// ─── Tableau coéquipiers ──────────────────────────────────────────────────────

interface TeammateRowItemProps { row: TeammateRow; isSelected: boolean; onSelect: () => void }
function TeammateRowItem({ row, isSelected, onSelect }: TeammateRowItemProps) {
  const wr = (row.with_kpis.win_rate * 100).toFixed(0)
  const kd = row.with_kpis.kd_ratio?.toFixed(2) ?? '-'
  return (
    <tr onClick={onSelect} className={`cursor-pointer transition-colors hover:bg-gray-50 ${isSelected ? 'bg-purple-50 border-l-2 border-purple-500' : ''}`}>
      <td className="px-4 py-3 font-medium">{row.gamertag}</td>
      <td className="px-4 py-3 text-center">{row.encounter_count}</td>
      <td className="px-4 py-3 text-center">{row.with_kpis.wins}</td>
      <td className="px-4 py-3 text-center">{wr}%</td>
      <td className="px-4 py-3 text-center">{kd}</td>
      <td className="px-4 py-3 text-center text-xs text-gray-400">
        {row.last_seen_at ? new Date(row.last_seen_at).toLocaleDateString('fr-FR') : '-'}
      </td>
    </tr>
  )
}

// ─── Page principale ──────────────────────────────────────────────────────────

export function SquadPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/squad' })
  const { filterContext, filterContextHash } = useGlobalFilterStore()
  const [selectedGt, setSelectedGt] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<TabId>('synergies')

  const request: TeammatesQueryRequest = {
    filters: filterContext,
    selected_gamertags: selectedGt ? [selectedGt] : undefined,
  }
  const { data, isLoading, isError, error } = useTeammates(playerSlug, request, filterContextHash)

  if (isLoading) return <div className="flex items-center justify-center min-h-64"><Spinner size="lg" /></div>
  if (isError) return <div className="p-8 text-center text-red-600">Erreur : {String(error)}</div>
  if (!data) return null

  const { teammates, solo_reference } = data
  const selectedRow = selectedGt ? teammates.find((t) => t.gamertag === selectedGt) : null

  const synergiesChart = selectedRow
    ? buildSynergiesChart(selectedRow.gamertag, selectedRow.with_kpis, selectedRow.without_kpis, solo_reference)
    : null
  const radarChart = selectedRow
    ? buildRadarChart(selectedRow.gamertag, selectedRow.with_kpis, solo_reference)
    : null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Escouade" subtitle={`${teammates.length} coequipiers · ${data.total_matches} matchs`} />

      {/* KPI rapides si joueur sélectionné */}
      {selectedRow && (
        <Card>
          <CardHeader>
            <CardTitle>
              Statistiques avec <span className="text-purple-600">{selectedRow.gamertag}</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <KPIBlock title="Avec ce coéquipier" kpis={selectedRow.with_kpis} />
            {selectedRow.without_kpis && <KPIBlock title="Sans ce coéquipier" kpis={selectedRow.without_kpis} />}
            {solo_reference && <KPIBlock title="Référence solo" kpis={solo_reference} />}
          </CardContent>
        </Card>
      )}

      {/* Onglets Synergies / Contributions — visibles seulement si un joueur est sélectionné */}
      {selectedRow && (
        <Card>
          {/* Tab bar */}
          <div className="flex gap-0 border-b px-4">
            {TABS.map((tab) => (
              <Button
                key={tab.id}
                variant="ghost"
                size="sm"
                onClick={() => setActiveTab(tab.id)}
                className={`rounded-none border-b-2 px-4 py-3 text-sm ${
                  activeTab === tab.id
                    ? 'border-purple-600 font-semibold text-purple-700'
                    : 'border-transparent text-gray-500 hover:text-gray-800'
                }`}
              >
                {tab.label}
              </Button>
            ))}
          </div>

          <CardContent className="pt-4">
            {activeTab === 'synergies' && synergiesChart && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">
                  Comparaison de tes stats <em>avec</em> {selectedRow.gamertag} vs <em>sans</em> lui vs ta référence solo.
                </p>
                <PlotlyChart figure={synergiesChart} />
              </div>
            )}
            {activeTab === 'contributions' && radarChart && (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">
                  Profil de contribution normalisé de <em>{selectedRow.gamertag}</em> (violet) vs ta référence solo (cyan pointillé).
                </p>
                <PlotlyChart figure={radarChart} />
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Table principale coéquipiers */}
      <Card>
        <CardHeader>
          <CardTitle>
            Coéquipiers
            {selectedGt && (
              <button className="ml-3 text-xs text-gray-400 hover:text-gray-700" onClick={() => setSelectedGt(null)}>
                ✕ Effacer
              </button>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {teammates.length === 0 ? (
            <p className="p-6 text-center text-gray-500">Aucun coéquipier trouvé pour cette période.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="px-4 py-3 text-left">Gamertag</th>
                    <th className="px-4 py-3 text-center">Matchs</th>
                    <th className="px-4 py-3 text-center">Victoires</th>
                    <th className="px-4 py-3 text-center">Win%</th>
                    <th className="px-4 py-3 text-center">K/D</th>
                    <th className="px-4 py-3 text-center">Dernière rencontre</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {teammates.map((row) => (
                    <TeammateRowItem
                      key={row.xuid ?? row.gamertag}
                      row={row}
                      isSelected={row.gamertag === selectedGt}
                      onSelect={() => setSelectedGt(row.gamertag === selectedGt ? null : row.gamertag)}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useTeammates } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import type { TeammateRow, TeammateKPIs, TeammatesQueryRequest } from '@/lib/api/types'

interface KPICardProps { label: string; value: number | null; unit?: string }
function KPICard({ label, value, unit = '' }: KPICardProps) {
  const display = value == null ? '-' : `${Number.isInteger(value) ? value : value.toFixed(2)}${unit}`
  return (
    <div className="flex flex-col gap-1 rounded-lg border p-3">
      <span className="text-xs text-gray-500 uppercase tracking-wide">{label}</span>
      <span className="text-xl font-bold">{display}</span>
    </div>
  )
}

interface KPIBlockProps { title: string; kpis: TeammateKPIs }
function KPIBlock({ title, kpis }: KPIBlockProps) {
  return (
    <div>
      <h3 className="text-sm font-medium text-gray-600 mb-2">{title}</h3>
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <KPICard label="Matchs" value={kpis.match_count} />
        <KPICard label="Win Rate" value={kpis.win_rate * 100} unit="%" />
        <KPICard label="K/D" value={kpis.kd_ratio} />
        <KPICard label="Kills / match" value={kpis.kills_per_game} />
      </div>
    </div>
  )
}

interface TeammateRowItemProps { row: TeammateRow; isSelected: boolean; onSelect: () => void }
function TeammateRowItem({ row, isSelected, onSelect }: TeammateRowItemProps) {
  const wr = (row.with_kpis.win_rate * 100).toFixed(0)
  const kd = row.with_kpis.kd_ratio?.toFixed(2) ?? '-'
  return (
    <tr onClick={onSelect} className={`cursor-pointer transition-colors hover:bg-gray-50 ${isSelected ? 'bg-blue-50 border-l-2 border-blue-500' : ''}`}>
      <td className="px-4 py-3 font-medium">{row.gamertag}</td>
      <td className="px-4 py-3 text-center">{row.encounter_count}</td>
      <td className="px-4 py-3 text-center">{row.with_kpis.wins}</td>
      <td className="px-4 py-3 text-center">{wr}%</td>
      <td className="px-4 py-3 text-center">{kd}</td>
      <td className="px-4 py-3 text-center text-xs text-gray-400">
        {row.last_seen_at ? new Date(row.last_seen_at).toLocaleDateString('fr-FR') : '-'}
      </td>
    </tr>
  )
}

export function SquadPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/squad' })
  const { filterContext, filterContextHash } = useGlobalFilterStore()
  const [selectedGt, setSelectedGt] = useState<string | null>(null)
  const request: TeammatesQueryRequest = { filters: filterContext, selected_gamertags: selectedGt ? [selectedGt] : undefined }
  const { data, isLoading, isError, error } = useTeammates(playerSlug, request, filterContextHash)

  if (isLoading) return <div className="flex items-center justify-center min-h-64"><Spinner size="lg" /></div>
  if (isError) return <div className="p-8 text-center text-red-600">Erreur : {String(error)}</div>
  if (!data) return null

  const { teammates, solo_reference } = data
  const selectedRow = selectedGt ? teammates.find((t) => t.gamertag === selectedGt) : null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Escouade" subtitle={`${teammates.length} coequipiers - ${data.total_matches} matchs`} />
      {selectedRow && (
        <Card>
          <CardHeader><CardTitle>Statistiques avec <span className="text-blue-600">{selectedRow.gamertag}</span></CardTitle></CardHeader>
          <CardContent className="flex flex-col gap-4">
            <KPIBlock title="Avec ce coequipier" kpis={selectedRow.with_kpis} />
            {selectedRow.without_kpis && <KPIBlock title="Sans ce coequipier" kpis={selectedRow.without_kpis} />}
            {solo_reference && <KPIBlock title="Reference solo" kpis={solo_reference} />}
          </CardContent>
        </Card>
      )}
      <Card>
        <CardHeader>
          <CardTitle>
            Coequipiers
            {selectedGt && <button className="ml-3 text-xs text-gray-400 hover:text-gray-700" onClick={() => setSelectedGt(null)}>x Effacer</button>}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {teammates.length === 0 ? (
            <p className="p-6 text-center text-gray-500">Aucun coequipier trouve pour cette periode.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="px-4 py-3 text-left">Gamertag</th>
                    <th className="px-4 py-3 text-center">Matchs</th>
                    <th className="px-4 py-3 text-center">Victoires</th>
                    <th className="px-4 py-3 text-center">Win%</th>
                    <th className="px-4 py-3 text-center">K/D</th>
                    <th className="px-4 py-3 text-center">Derniere rencontre</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {teammates.map((row) => (
                    <TeammateRowItem key={row.xuid ?? row.gamertag} row={row} isSelected={row.gamertag === selectedGt} onSelect={() => setSelectedGt(row.gamertag === selectedGt ? null : row.gamertag)} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
