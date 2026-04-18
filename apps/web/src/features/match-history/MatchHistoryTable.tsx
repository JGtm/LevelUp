/**
 * MatchHistoryTable — tableau paginé de l'historique des parties.
 * A1 NATIVE_COMPONENTS — couleur ligne par outcome + clic ligne → Match View.
 * Sprint exclusion : bouton ⊘ par ligne avec confirmation inline.
 */
import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { MatchHistoryRow, PaginationMeta } from '@/lib/api/types'
import { useSetMatchExclusion } from './queries'
import { queryKeys } from '@/lib/query/keys'

interface Props {
  rows: MatchHistoryRow[]
  pagination: PaginationMeta
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSort: (field: string) => void
  onPage: (page: number) => void
  onExport?: () => void
  exporting?: boolean
  playerSlug: string
  filterHash: string
  page: number
  /** Si true, n'affiche pas le bouton Export CSV (usage dans Explorer) */
  hideExport?: boolean
}

const OUTCOME_BADGE: Record<number, 'success' | 'destructive' | 'secondary' | 'outline'> = {
  1: 'secondary',    // Égalité
  2: 'success',      // Victoire
  3: 'destructive',  // Défaite
  4: 'outline',      // DNF
}

/** Couleur de fond de la ligne selon résultat — subtile, ton Halo foncé. */
const OUTCOME_ROW_BG: Record<number, string> = {
  1: 'bg-blue-950/20',   // Égalité — bleu très léger
  2: 'bg-green-950/20',  // Victoire — vert très léger
  3: 'bg-red-950/20',    // Défaite — rouge très léger
  4: 'bg-gray-900/20',   // DNF — gris
}

function SortIndicator({
  field,
  activeField,
  direction,
}: {
  field: string
  activeField: string
  direction: 'asc' | 'desc'
}) {
  if (field !== activeField) return <span className="ml-1 text-gray-600">⇅</span>
  return <span className="ml-1 text-purple-400">{direction === 'asc' ? '↑' : '↓'}</span>
}

export function MatchHistoryTable({
  rows,
  pagination,
  sortField,
  sortDirection,
  onSort,
  onPage,
  onExport,
  exporting,
  playerSlug,
  filterHash,
  page,
  hideExport,
}: Props) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [confirmingId, setConfirmingId] = useState<string | null>(null)
  const excludeMutation = useSetMatchExclusion(playerSlug)

  const columns = [
    { key: 'start_time', label: 'Date' },
    { key: 'map_mode', label: 'Carte / Mode', sortable: false },
    { key: 'outcome_label', label: 'Résultat', sortable: false },
    { key: 'score_label', label: 'Score', sortable: false },
    { key: 'performance_score_relative', label: 'Perf.' },
    { key: 'delta_mmr', label: 'ΔMMR' },
    { key: 'team_enemy_mmr', label: 'MMR T/A', sortable: false },
    { key: 'win_rate_hist', label: 'Win%' },
    { key: 'average_life_mmss', label: 'Vie moy.', sortable: false },
    { key: 'actions', label: '', sortable: false },
  ] as const

  function navigateToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/explorer/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  function handleExclude(e: React.MouseEvent, matchId: string) {
    e.stopPropagation()
    setConfirmingId(matchId)
  }

  function handleConfirm(e: React.MouseEvent, matchId: string) {
    e.stopPropagation()
    excludeMutation.mutate(
      { matchId, excluded: true },
      {
        onSuccess: () => {
          setConfirmingId(null)
          void queryClient.invalidateQueries({
            queryKey: queryKeys.matchHistory(playerSlug, filterHash, page),
          })
        },
      },
    )
  }

  function handleCancel(e: React.MouseEvent) {
    e.stopPropagation()
    setConfirmingId(null)
  }

  return (
    <div className="flex flex-col gap-2">
      {/* Barre d'actions */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-400">
          {pagination.total.toLocaleString('fr-FR')} partie{pagination.total !== 1 ? 's' : ''}
        </p>
        {!hideExport && onExport && (
          <Button variant="outline" size="sm" onClick={onExport} loading={exporting}>
            Exporter CSV
          </Button>
        )}
      </div>

      {/* Tableau */}
      <div className="overflow-x-auto rounded-lg border border-gray-700 bg-[#1d2328]">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-700 bg-gray-900/60 text-xs font-medium text-gray-400">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-2.5 text-left whitespace-nowrap ${
                    (col as { sortable?: boolean }).sortable !== false
                      ? 'cursor-pointer hover:text-white'
                      : ''
                  }`}
                  onClick={() =>
                    (col as { sortable?: boolean }).sortable !== false && onSort(col.key)
                  }
                >
                  {col.label}
                  {(col as { sortable?: boolean }).sortable !== false && (
                    <SortIndicator
                      field={col.key}
                      activeField={sortField}
                      direction={sortDirection}
                    />
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {rows.map((row) => {
              const bg = OUTCOME_ROW_BG[row.outcome_code ?? 0] ?? ''
              const isConfirming = confirmingId === row.match_id
              const isPending = excludeMutation.isPending && confirmingId === row.match_id
              return (
                <tr
                  key={row.match_id}
                  className={`cursor-pointer transition-colors hover:brightness-125 ${bg}`}
                  onClick={() => navigateToMatch(row.match_id)}
                  title="Voir le détail du match"
                >
                  <td className="px-4 py-2 text-gray-400 whitespace-nowrap">
                    {row.start_time_label ?? new Date(row.start_time).toLocaleDateString('fr-FR')}
                  </td>
                  <td className="px-4 py-2">
                    <span className="font-medium text-white">{row.map_ui}</span>
                    <span className="ml-1 text-xs text-gray-400">· {row.mode_ui}</span>
                    {row.playlist_label && (
                      <div className="text-xs text-gray-500 mt-0.5">{row.playlist_label}</div>
                    )}
                  </td>
                  <td className="px-4 py-2">
                    <Badge variant={OUTCOME_BADGE[row.outcome_code ?? 0] ?? 'secondary'}>
                      {row.outcome_label}
                    </Badge>
                  </td>
                  <td className="px-4 py-2 text-gray-200">{row.score_label}</td>
                  <td className="px-4 py-2 text-right font-mono text-purple-300">
                    {row.performance_score_relative != null
                      ? row.performance_score_relative
                      : '—'}
                  </td>
                  <td className="px-4 py-2 text-right font-mono text-sm">
                    {row.delta_mmr != null ? (
                      <span
                        className={row.delta_mmr >= 0 ? 'text-[#00DC82]' : 'text-[#FF4B4B]'}
                      >
                        {row.delta_mmr >= 0 ? '+' : ''}
                        {row.delta_mmr.toFixed(0)}
                      </span>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-4 py-2 text-right font-mono text-xs text-gray-500">
                    {row.team_mmr != null || row.enemy_mmr != null ? (
                      <span>
                        {row.team_mmr != null ? row.team_mmr.toFixed(0) : '?'}
                        <span className="text-gray-600">/</span>
                        {row.enemy_mmr != null ? row.enemy_mmr.toFixed(0) : '?'}
                      </span>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-4 py-2 text-right text-gray-300">
                    {row.win_rate_hist != null ? (
                      <span>
                        {(row.win_rate_hist * 100).toFixed(0)}%
                        {row.win_rate_hist_total != null && (
                          <span className="text-xs text-gray-500 ml-1">
                            ({row.win_rate_hist_total})
                          </span>
                        )}
                      </span>
                    ) : (
                      '—'
                    )}
                  </td>
                  <td className="px-4 py-2 text-right text-gray-400">
                    {row.average_life_mmss}
                  </td>
                  <td className="px-4 py-2 text-right whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                    {isConfirming ? (
                      <span className="inline-flex items-center gap-1">
                        <button
                          className="rounded px-2 py-0.5 text-xs bg-red-900/60 text-red-300 hover:bg-red-800/60 disabled:opacity-50"
                          onClick={(e) => handleConfirm(e, row.match_id)}
                          disabled={isPending}
                        >
                          Ignorer
                        </button>
                        <button
                          className="rounded px-2 py-0.5 text-xs bg-gray-800 text-gray-400 hover:bg-gray-700"
                          onClick={handleCancel}
                          disabled={isPending}
                        >
                          Annuler
                        </button>
                      </span>
                    ) : (
                      <button
                        className="rounded p-1 text-gray-600 hover:text-gray-300 hover:bg-gray-800/60 transition-colors"
                        title="Ignorer ce match"
                        onClick={(e) => handleExclude(e, row.match_id)}
                      >
                        ⊘
                      </button>
                    )}
                  </td>
                </tr>
              )
            })}
            {rows.length === 0 && (
              <tr>
                <td colSpan={10} className="px-4 py-10 text-center text-gray-500">
                  Aucun match trouvé.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-gray-500">
          Page {pagination.page} /{' '}
          {Math.ceil(pagination.total / pagination.page_size)}
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={!pagination.has_prev}
            onClick={() => onPage(pagination.page - 1)}
          >
            ← Précédent
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={!pagination.has_next}
            onClick={() => onPage(pagination.page + 1)}
          >
            Suivant →
          </Button>
        </div>
      </div>
    </div>
  )
}
