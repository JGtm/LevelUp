/**
 * SessionMatchesTable — table détaillée des matchs d'une session.
 *
 * Colonnes : heure / mode / playlist / K/D/A / accuracy / perf score / outcome.
 * Outcome libellé résolu via `useFieldMappings.outcomes` (TOML) avec fallback clé canonique.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import {
  formatNumber,
  formatPercent,
  formatShortDateTime,
  matchOutcomeTone,
  useSessionT,
} from './_shared'

interface Props {
  matches: SessionDetailMatchRow[]
}

export function SessionMatchesTable({ matches }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const t = useSessionT()
  const outcomeLabel = (outcome: number | null) => {
    const key =
      outcome === 2 ? 'win' : outcome === 3 ? 'loss' : outcome === 1 ? 'tie' : outcome === 4 ? 'dnf' : null
    if (!key) return '—'
    return fieldMappings?.outcomes?.[key]?.label ?? key
  }

  if (matches.length === 0) {
    return (
      <EmptyStateNotice
        title={t('session.detail.matches_empty_title')}
        description={t('session.detail.matches_empty_description')}
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs uppercase tracking-label text-muted-foreground">
            <th className="px-3 py-3 font-medium">{t('session.detail.col_time')}</th>
            <th className="px-3 py-3 font-medium">{t('session.detail.col_mode')}</th>
            <th className="px-3 py-3 font-medium">{t('session.detail.col_playlist')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_kda')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_accuracy')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_perf_score')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_outcome')}</th>
          </tr>
        </thead>
        <tbody>
          {matches.map((match) => {
            const tone = matchOutcomeTone(match.outcome)
            return (
              <tr key={match.match_id} className="border-b border-border/60 text-foreground last:border-0">
                <td className="px-3 py-3 text-muted-foreground">{formatShortDateTime(match.start_time)}</td>
                <td className="px-3 py-3 font-medium">{match.pair_name || '—'}</td>
                <td className="px-3 py-3 text-muted-foreground">{match.playlist_name || '—'}</td>
                <td className="px-3 py-3 text-right tabular-nums">{`${match.kills}/${match.deaths}/${match.assists}`}</td>
                <td className="px-3 py-3 text-right tabular-nums">{formatPercent(match.accuracy)}</td>
                <td className="px-3 py-3 text-right tabular-nums">{formatNumber(match.performance_score, 1)}</td>
                <td className="px-3 py-3 text-right">
                  <span className={tone.className} style={tone.style}>
                    {outcomeLabel(match.outcome)}
                  </span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
