/**
 * SquadPage --- Vue coequipiers / escouade (Slice 6).
 * Types ref: TeammateRow, TeammateKPIs, TeammatesPageResponse
 */
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
