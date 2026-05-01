/**
 * HistoryTable — tableau historique des matchs partages du squad (chunk S9).
 *
 * 1 ligne par match, colonnes : date, mode, carte, durée, K/D/A par joueur,
 * outcome global. Le tri est fait côté backend (date desc).
 */
import { useNavigate } from '@tanstack/react-router'
import { tokenCssVar } from '@/lib/accessibility'
import { formatDate, formatDurationMMSS } from '@/lib/formatters'

import type { HistoryTableRow, Outcome } from '../types'

export interface HistoryTableProps {
  rows: HistoryTableRow[]
  /** Ordre stable des colonnes joueurs (main + coequipiers). */
  squadOrder: string[]
  /** Locale utilisee pour formater les dates ("fr-FR" / "en-US"). */
  locale: string
  /** Labels d'en-tete deja localises par le caller. */
  labels: {
    date: string
    mode: string
    map: string
    outcome: string
    duration: string
    kdaSuffix: string
  }
  /** playerSlug pour naviguer vers le détail du match au clic. */
  playerSlug?: string
}

// formatDate et formatDurationMMSS importés depuis @/lib/formatters
// (revue 2026-04-29 P2.6bis — centralisation helpers).
// Le format date local utilise day/month/year 2-digit ; passer ce format
// explicite au helper canonique.
const HISTORY_DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: '2-digit',
}

function outcomeColorVar(o: Outcome): string {
  switch (o) {
    case 'win':
      return tokenCssVar('outcome-win')
    case 'loss':
      return tokenCssVar('outcome-loss')
    case 'tie':
      return tokenCssVar('outcome-draw')
    default:
      return tokenCssVar('outcome-dnf')
  }
}

export function HistoryTable({ rows, squadOrder, locale, labels, playerSlug }: HistoryTableProps) {
  const navigate = useNavigate()

  function goToMatch(matchId: string) {
    if (!playerSlug) return
    void navigate({
      to: '/players/$playerSlug/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  if (rows.length === 0) {
    return null
  }
  return (
    <div className="overflow-x-auto" data-testid="history-table">
      <table className="w-full text-sm">
        <thead className="bg-muted border-b">
          <tr>
            <th className="px-3 py-2 text-left">{labels.date}</th>
            <th className="px-3 py-2 text-left">{labels.mode}</th>
            <th className="px-3 py-2 text-left">{labels.map}</th>
            <th className="px-3 py-2 text-center">{labels.duration}</th>
            <th className="px-3 py-2 text-center">{labels.outcome}</th>
            {squadOrder.map((gt) => (
              <th key={gt} className="px-3 py-2 text-center">
                {gt} {labels.kdaSuffix}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y">
          {rows.map((row) => (
            <tr
              key={row.match_id}
              className={playerSlug ? 'cursor-pointer hover:bg-primary/10 transition-colors' : ''}
              onClick={playerSlug ? () => goToMatch(row.match_id) : undefined}
              role={playerSlug ? 'button' : undefined}
              tabIndex={playerSlug ? 0 : undefined}
              onKeyDown={playerSlug ? (e) => e.key === 'Enter' && goToMatch(row.match_id) : undefined}
            >
              <td className="px-3 py-2">{formatDate(row.started_at_utc, locale, HISTORY_DATE_OPTS)}</td>
              <td className="px-3 py-2">{row.mode_label ?? '-'}</td>
              <td className="px-3 py-2">{row.map_label ?? '-'}</td>
              <td className="px-3 py-2 text-center">{formatDurationMMSS(row.duration_seconds)}</td>
              <td
                className="px-3 py-2 text-center font-medium"
                style={{ color: outcomeColorVar(row.main_outcome) }}
              >
                {row.main_outcome}
              </td>
              {squadOrder.map((gt) => {
                const cell = row.player_stats[gt]
                if (!cell) {
                  return (
                    <td key={gt} className="px-3 py-2 text-center text-muted-foreground">
                      -
                    </td>
                  )
                }
                return (
                  <td key={gt} className="px-3 py-2 text-center">
                    {cell.kills ?? '-'}/{cell.deaths ?? '-'}/{cell.assists ?? '-'}
                  </td>
                )
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
