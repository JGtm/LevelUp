/**
 * SessionMatchesTable — table détaillée des matchs d'une session.
 *
 * Colonnes : ouvrir / heure / mode / playlist / K/D/A / accuracy / perf score / outcome.
 * Outcome libellé résolu via `useFieldMappings.outcomes` (TOML) avec fallback clé canonique.
 *
 * Navigation contextuelle (Phase 2 nav-context-unification) : un clic sur une
 * ligne ouvre la page match en propageant la liste des matchs de la session
 * (prev/next reste dans la session) + un descriptor `session`. `session_label`
 * sert d'identifiant de session côté filterSpec — cohérent avec
 * `filterContextToMatchFilterSpec` (sessions.picked → session_id).
 */
import { useCallback, useMemo } from 'react'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'

import {
  formatNumber,
  formatPercent,
  formatShortDateTime,
  matchOutcomeTone,
  useSessionT,
} from './_shared'

interface Props {
  matches: SessionDetailMatchRow[]
  playerSlug: string
}

export function SessionMatchesTable({ matches, playerSlug }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const t = useSessionT()
  const navigateToMatch = useNavigateToMatch(playerSlug)

  const allMatchIds = useMemo(() => matches.map((m) => m.match_id), [matches])
  // Début de session = match le plus ancien (start_time ISO UTC → tri lexical = chrono).
  const sessionStartUtc = useMemo(() => {
    if (matches.length === 0) return ''
    return matches.reduce(
      (min, m) => (m.start_time < min ? m.start_time : min),
      matches[0].start_time,
    )
  }, [matches])
  const sessionId = useMemo(
    () => matches.find((m) => m.session_label)?.session_label ?? undefined,
    [matches],
  )

  const goToMatch = useCallback(
    (matchId: string) => {
      navigateToMatch(matchId, {
        source: 'session',
        matchIds: allMatchIds,
        contextDescriptor: { kind: 'session', startTimeUtc: sessionStartUtc },
        filterSpec: sessionId ? { session_id: sessionId } : undefined,
      })
    },
    [navigateToMatch, allMatchIds, sessionStartUtc, sessionId],
  )

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
            <th className="w-8 px-3 py-3 font-medium" aria-hidden="true" />
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
                <td className="px-3 py-3">
                  <button
                    type="button"
                    className="group flex items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                    onClick={() => goToMatch(match.match_id)}
                    aria-label={t('session.detail.col_open')}
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3.5 w-3.5 opacity-50 transition-opacity group-hover:opacity-100" aria-hidden="true">
                      <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
                      <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
                    </svg>
                  </button>
                </td>
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
