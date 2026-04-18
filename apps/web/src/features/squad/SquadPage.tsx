/**
 * SquadPage — Vue coéquipiers / escouade (Slice 6).
 * Multiselect jusqu'à 3 coéquipiers pour comparaison synergies/contributions.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useTeammates } from './queries'
import { PageHeader } from '@/components/shell/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import type { TeammateRow, TeammateKPIs, TeammatesQueryRequest, PlotlyFigurePayload } from '@/lib/api/types'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { useComparePrefetch } from '@/features/compare/queries'

const MAX_SELECTION = 3

type TabId = 'synergies' | 'contributions'
const TABS: { id: TabId; label: string }[] = [
  { id: 'synergies', label: 'Synergies' },
  { id: 'contributions', label: 'Contributions' },
]

// ─── Helpers graphiques ───────────────────────────────────────────────────────

const CHART_COLORS = ['#7C3AED', '#F59E0B', '#10B981']

function buildSynergiesChart(
  rows: TeammateRow[],
  soloRef: TeammateKPIs | null,
): PlotlyFigurePayload {
  const metrics = ['Win Rate', 'K/D', 'Kills/partie', 'Assists/partie']
  const extract = (k: TeammateKPIs) => [
    k.win_rate * 100,
    k.kd_ratio ?? 0,
    k.kills_per_game ?? 0,
    k.assists_per_game ?? 0,
  ]

  const traces: PlotlyFigurePayload['data'] = rows.map((row, i) => ({
    type: 'bar', name: `Avec ${row.gamertag}`,
    x: metrics, y: extract(row.with_kpis),
    marker: { color: CHART_COLORS[i % CHART_COLORS.length] },
  }))

  if (soloRef) traces.push({
    type: 'bar', name: 'Référence solo',
    x: metrics, y: extract(soloRef),
    marker: { color: '#94A3B8' },
  })

  return {
    data: traces,
    layout: {
      barmode: 'group', height: 320,
      margin: { l: 40, r: 20, t: 20, b: 60 },
      legend: { orientation: 'h', x: 0, y: -0.2 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
    },
  }
}

function buildRadarChart(
  rows: TeammateRow[],
  soloRef: TeammateKPIs | null,
): PlotlyFigurePayload {
  const axes = ['Win Rate', 'K/D', 'Kills/partie', 'Assists/partie', 'Précision']
  const norm = (v: number | null, max: number) => v != null ? Math.min(100, (v / max) * 100) : 0

  const makeVals = (k: TeammateKPIs) => [
    k.win_rate * 100,
    norm(k.kd_ratio, 3),
    norm(k.kills_per_game, 20),
    norm(k.assists_per_game, 10),
    norm(k.accuracy, 1) * 100,
  ]

  const traces: PlotlyFigurePayload['data'] = rows.map((row, i) => {
    const vals = makeVals(row.with_kpis)
    return {
      type: 'scatterpolar', name: `Avec ${row.gamertag}`,
      r: [...vals, vals[0]], theta: [...axes, axes[0]],
      fill: 'toself',
      marker: { color: CHART_COLORS[i % CHART_COLORS.length] },
      line: { color: CHART_COLORS[i % CHART_COLORS.length] },
    }
  })

  if (soloRef) {
    const vals = makeVals(soloRef)
    traces.push({
      type: 'scatterpolar', name: 'Solo ref',
      r: [...vals, vals[0]], theta: [...axes, axes[0]],
      fill: 'toself', opacity: 0.4,
      marker: { color: '#06B6D4' },
      line: { color: '#06B6D4', dash: 'dot' },
    })
  }

  return {
    data: traces,
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

interface KPIBlockProps { title: string; kpis: TeammateKPIs; color?: string }
function KPIBlock({ title, kpis, color = 'text-gray-600' }: KPIBlockProps) {
  return (
    <div>
      <h3 className={`text-sm font-medium mb-2 ${color}`}>{title}</h3>
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

interface TeammateRowItemProps {
  row: TeammateRow
  selectionIndex: number
  onSelect: () => void
  onCompare: (gamertag: string) => void
  onPrefetchCompare: (gamertag: string) => void
}
function TeammateRowItem({ row, selectionIndex, onSelect, onCompare, onPrefetchCompare }: TeammateRowItemProps) {
  const isSelected = selectionIndex >= 0
  const wr = (row.with_kpis.win_rate * 100).toFixed(0)
  const kd = row.with_kpis.kd_ratio?.toFixed(2) ?? '-'
  const dotColor = isSelected ? CHART_COLORS[selectionIndex % CHART_COLORS.length] : undefined
  return (
    <tr
      onClick={onSelect}
      className={`cursor-pointer transition-colors hover:bg-gray-50 ${isSelected ? 'bg-purple-50' : ''}`}
    >
      <td className="px-4 py-3 font-medium flex items-center gap-2">
        {dotColor && <span className="inline-block w-2 h-2 rounded-full" style={{ backgroundColor: dotColor }} />}
        {row.gamertag}
      </td>
      <td className="px-4 py-3 text-center">{row.encounter_count}</td>
      <td className="px-4 py-3 text-center">{row.with_kpis.wins}</td>
      <td className="px-4 py-3 text-center">{wr}%</td>
      <td className="px-4 py-3 text-center">{kd}</td>
      <td className="px-4 py-3 text-center text-xs text-gray-400">
        {row.last_seen_at ? new Date(row.last_seen_at).toLocaleDateString('fr-FR') : '-'}
      </td>
      {/* C4.4 : CTA Comparer avec prefetch onMouseEnter */}
      <td className="px-4 py-3 text-center">
        <button
          onClick={(e) => { e.stopPropagation(); onCompare(row.gamertag) }}
          onMouseEnter={() => onPrefetchCompare(row.gamertag)}
          className="text-xs text-purple-600 hover:underline"
        >
          Comparer
        </button>
      </td>
    </tr>
  )
}

// ─── Page principale ──────────────────────────────────────────────────────────

export function SquadPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/squad' })
  const { filterContext, filterContextHash } = useGlobalFilterStore()
  const [selectedGts, setSelectedGts] = useState<string[]>([])
  const [activeTab, setActiveTab] = useState<TabId>('synergies')
  const [compareTarget, setCompareTarget] = useState<string | null>(null)
  const prefetchCompare = useComparePrefetch(playerSlug)

  const request: TeammatesQueryRequest = {
    filters: filterContext,
    selected_gamertags: selectedGts.length > 0 ? selectedGts : undefined,
  }
  const { data, isLoading, isError, error } = useTeammates(playerSlug, request, filterContextHash)

  const toggleSelect = (gamertag: string) => {
    setSelectedGts((prev) => {
      if (prev.includes(gamertag)) return prev.filter((g) => g !== gamertag)
      if (prev.length >= MAX_SELECTION) return prev
      return [...prev, gamertag]
    })
  }

  if (isLoading) return <div className="flex items-center justify-center min-h-64"><Spinner size="lg" /></div>
  if (isError) return <div className="p-8 text-center text-red-600">Erreur : {String(error)}</div>
  if (!data) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Escouade"
          subtitle="Analyse des coéquipiers et des synergies"
        />
        <div className="px-6">
          <EmptyStateCard
            title="Données d'escouade indisponibles"
            description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie les filtres ou la disponibilité des matchs partagés."
          />
        </div>
      </div>
    )
  }

  const { teammates, solo_reference } = data
  const selectedRows = selectedGts.map((gt) => teammates.find((t) => t.gamertag === gt)).filter(Boolean) as TeammateRow[]

  const synergiesChart = selectedRows.length > 0 ? buildSynergiesChart(selectedRows, solo_reference) : null
  const radarChart = selectedRows.length > 0 ? buildRadarChart(selectedRows, solo_reference) : null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Escouade"
        subtitle={`${teammates.length} coéquipiers · ${data.total_matches} matchs`}
      />

      {/* KPI rapides si coéquipier(s) sélectionné(s) */}
      <Card>
        <CardHeader>
          <CardTitle>
            Stats avec les coéquipiers sélectionnés
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {selectedRows.length > 0 ? (
            <>
              {selectedRows.map((row, i) => (
                <KPIBlock
                  key={row.gamertag}
                  title={`Avec ${row.gamertag}`}
                  kpis={row.with_kpis}
                  color={`text-[${CHART_COLORS[i % CHART_COLORS.length]}]`}
                />
              ))}
              {solo_reference && <KPIBlock title="Référence solo" kpis={solo_reference} color="text-cyan-600" />}
            </>
          ) : (
            <EmptyStateNotice
              title="Aucune sélection"
              description="Sélectionne jusqu'à 3 coéquipiers dans le tableau pour activer la comparaison."
            />
          )}
        </CardContent>
      </Card>

      {/* Onglets Synergies / Contributions */}
      <Card>
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
          {selectedRows.length === 0 ? (
            <EmptyStateNotice
              title="Comparaison inactive"
              description="Sélectionne au moins un coéquipier pour afficher les synergies et les contributions."
            />
          ) : activeTab === 'synergies' ? (
            synergiesChart ? (
              <div className="space-y-4">
                <p className="text-sm text-gray-500">
                  Comparaison de tes stats <em>avec</em> chaque coéquipier vs ta référence solo.
                </p>
                <PlotlyChart figure={synergiesChart} />
              </div>
            ) : (
              <EmptyStateNotice
                title="Synergies indisponibles"
                description="Le graphique de synergie n'a pas pu être construit avec les données actuelles."
              />
            )
          ) : radarChart ? (
            <div className="space-y-4">
              <p className="text-sm text-gray-500">
                Profil de contribution normalisé (violet) vs ta référence solo (cyan pointillé).
              </p>
              <PlotlyChart figure={radarChart} />
            </div>
          ) : (
            <EmptyStateNotice
              title="Contributions indisponibles"
              description="Le radar de contribution n'a pas pu être calculé pour la sélection en cours."
            />
          )}
        </CardContent>
      </Card>

      {/* Table principale coéquipiers */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Coéquipiers</span>
            <div className="flex items-center gap-3">
              <span className="text-sm font-normal text-gray-500">
                {selectedGts.length}/{MAX_SELECTION} sélectionnés
              </span>
              {selectedGts.length > 0 && (
                <button
                  className="text-xs text-gray-400 hover:text-gray-700"
                  onClick={() => setSelectedGts([])}
                >
                  ✕ Effacer
                </button>
              )}
            </div>
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
                  {teammates.map((row) => {
                    const idx = selectedGts.indexOf(row.gamertag)
                    return (
                      <TeammateRowItem
                        key={row.xuid ?? row.gamertag}
                        row={row}
                        selectionIndex={idx}
                        onSelect={() => toggleSelect(row.gamertag)}
                        onCompare={(gt) => setCompareTarget(gt)}
                        onPrefetchCompare={prefetchCompare}
                      />
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* C4.4 : CompareDrawer déclenché depuis les lignes coéquipiers */}
      {compareTarget && (
        <CompareDrawer
          playerSlug={playerSlug}
          open={!!compareTarget}
          onClose={() => setCompareTarget(null)}
        />
      )}
    </div>
  )
}
