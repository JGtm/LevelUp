/**
 * HistoryTable — tableau historique des matchs partages du squad (chunk S9).
 *
 * 1 ligne par match, colonnes : date, mode, carte, durée, K/D/A par joueur,
 * outcome global. Le tri est fait côté backend (date desc).
 */
import { tokenCssVar } from '@/lib/accessibility'

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
}

function formatDate(iso: string, locale: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString(locale, { day: '2-digit', month: '2-digit', year: '2-digit' })
}

function formatDuration(seconds?: number): string {
  if (!seconds) return '-'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${s.toString().padStart(2, '0')}`
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

export function HistoryTable({ rows, squadOrder, locale, labels }: HistoryTableProps) {
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
            <tr key={row.match_id}>
              <td className="px-3 py-2">{formatDate(row.started_at_utc, locale)}</td>
              <td className="px-3 py-2">{row.mode_label ?? '-'}</td>
              <td className="px-3 py-2">{row.map_label ?? '-'}</td>
              <td className="px-3 py-2 text-center">{formatDuration(row.duration_seconds)}</td>
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
