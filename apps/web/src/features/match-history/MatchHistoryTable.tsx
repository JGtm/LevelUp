/**
 * MatchHistoryTable — tableau paginé de l'historique des parties.
 * A1 NATIVE_COMPONENTS — couleur ligne par outcome + clic ligne → Match View.
 */
import { useNavigate } from '@tanstack/react-router'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { MatchHistoryRow, PaginationMeta } from '@/lib/api/types'

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
  hideExport,
}: Props) {
  const navigate = useNavigate()

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
  ] as const

  function navigateToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/explorer/matches/$matchId',
      params: { playerSlug, matchId },
    })
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
                </tr>
              )
            })}
            {rows.length === 0 && (
              <tr>
                <td colSpan={9} className="px-4 py-10 text-center text-gray-500">
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

import { Button } from '@/components/ui/button'
import type { MatchHistoryRow, PaginationMeta } from '@/lib/api/types'

interface Props {
  rows: MatchHistoryRow[]
  pagination: PaginationMeta
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSort: (field: string) => void
  onPage: (page: number) => void
  onExport?: () => void
  exporting?: boolean
}

const OUTCOME_TONE: Record<number, string> = {
  1: 'secondary', // Égalité
  2: 'success',   // Victoire
  3: 'destructive', // Défaite
  4: 'outline',   // DNF
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
  if (field !== activeField) return <span className="ml-1 text-gray-300">⇅</span>
  return <span className="ml-1 text-purple-500">{direction === 'asc' ? '↑' : '↓'}</span>
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
}: Props) {
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
    { key: 'detail', label: '', sortable: false },
  ] as const

  return (
    <div className="flex flex-col gap-2">
      {/* Barre d'actions */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-500">
          {pagination.total.toLocaleString('fr-FR')} partie{pagination.total !== 1 ? 's' : ''}
        </p>
        {onExport && (
          <Button variant="outline" size="sm" onClick={onExport} loading={exporting}>
            Exporter CSV
          </Button>
        )}
      </div>

      {/* Tableau */}
      <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-100 bg-gray-50 text-xs font-medium text-gray-500">
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={`px-4 py-2.5 text-left whitespace-nowrap ${
                    (col as { sortable?: boolean }).sortable !== false ? 'cursor-pointer hover:text-gray-900' : ''
                  }`}
                  onClick={() => (col as { sortable?: boolean }).sortable !== false && onSort(col.key)}
                >
                  {col.label}
                  {(col as { sortable?: boolean }).sortable !== false && (
                    <SortIndicator field={col.key} activeField={sortField} direction={sortDirection} />
                  )}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50">
            {rows.map((row) => (
              <tr key={row.match_id} className="hover:bg-purple-50/30 transition-colors">
                <td className="px-4 py-2 text-gray-400">
                  {new Date(row.start_time).toLocaleDateString('fr-FR')}
                </td>
                <td className="px-4 py-2">
                  <span className="font-medium text-gray-800">{row.map_ui}</span>
                  <span className="ml-1 text-xs text-gray-400">· {row.mode_ui}</span>
                  {row.playlist_label && (
                    <div className="text-xs text-gray-400 mt-0.5">{row.playlist_label}</div>
                  )}
                </td>
                <td className="px-4 py-2">
                  <Badge variant={(OUTCOME_TONE[row.outcome_code ?? 0] as 'success' | 'destructive' | 'secondary' | 'outline') ?? 'secondary'}>
                    {row.outcome_label}
                  </Badge>
                </td>
                <td className="px-4 py-2 text-gray-700">{row.score_label}</td>
                <td className="px-4 py-2 text-right font-mono text-purple-700">
                  {row.performance_score_relative != null ? row.performance_score_relative : '—'}
                </td>
                <td className="px-4 py-2 text-right font-mono text-sm">
                  {row.delta_mmr != null ? (
                    <span className={row.delta_mmr >= 0 ? 'text-green-600' : 'text-red-500'}>
                      {row.delta_mmr >= 0 ? '+' : ''}{row.delta_mmr.toFixed(0)}
                    </span>
                  ) : '—'}
                </td>
                <td className="px-4 py-2 text-right font-mono text-xs text-gray-500">
                  {row.team_mmr != null || row.enemy_mmr != null ? (
                    <span>
                      {row.team_mmr != null ? row.team_mmr.toFixed(0) : '?'}
                      <span className="text-gray-300">/</span>
                      {row.enemy_mmr != null ? row.enemy_mmr.toFixed(0) : '?'}
                    </span>
                  ) : '—'}
                </td>
                <td className="px-4 py-2 text-right text-gray-600">
                  {row.win_rate_hist != null ? (
                    <span>
                      {(row.win_rate_hist * 100).toFixed(0)}%
                      {row.win_rate_hist_total != null && (
                        <span className="text-xs text-gray-400 ml-1">({row.win_rate_hist_total})</span>
                      )}
                    </span>
                  ) : '—'}
                </td>
                <td className="px-4 py-2 text-right text-gray-500">{row.average_life_mmss}</td>
                <td className="px-4 py-2 text-center">
                  {row.match_url ? (
                    <a
                      href={row.match_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-purple-500 hover:text-purple-700 text-xs font-medium"
                    >
                      Détail →
                    </a>
                  ) : null}
                </td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td colSpan={10} className="px-4 py-8 text-center text-gray-400">
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
          Page {pagination.page} / {Math.ceil(pagination.total / pagination.page_size)}
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
