/**
 * DebriefDelayTable — le délai du débrief des matchs récents : pour chaque match, l'écart entre
 * sa fin et chaque étape de son chemin (détection, stats, morts lues, rejeu, dérivations,
 * citations), et si l'objectif de 5 min est tenu. Les délais sont calculés par le serveur ;
 * la logique d'affichage vit dans debriefDelay.logic.ts.
 *
 * Couleurs exclusivement via tokens sémantiques (StatusBadge).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { MatchDelay } from '@/lib/api/types'
import { AdminTable, AdminTd, AdminTh, AdminTr } from '../components/AdminTable'
import { StatusBadge } from '../components/StatusBadge'
import { adminAbsoluteTime, type AdminLocale } from '../format'
import type { TAdmin } from '../useAdminText'
import { DEBRIEF_COLUMNS, formatOffset, objectiveStatus, stepForColumn } from './debriefDelay.logic'

const OBJECTIVE_BADGE = {
  met: { status: 'ok', key: 'admin.buildqueue.delays_objective_met' },
  missed: { status: 'failed', key: 'admin.buildqueue.delays_objective_missed' },
  pending: { status: 'idle', key: 'admin.buildqueue.delays_objective_pending' },
} as const

export function DebriefDelayTable({ delays, tA, locale }: { delays: MatchDelay[]; tA: TAdmin; locale: AdminLocale }) {
  if (!delays.length) {
    return (
      <EmptyStateNotice
        title={tA('admin.buildqueue.delays_empty_title')}
        description={tA('admin.buildqueue.delays_empty_desc')}
      />
    )
  }
  return (
    <AdminTable
      head={
        <>
          <AdminTh>{tA('admin.buildqueue.col_match')}</AdminTh>
          <AdminTh>{tA('admin.buildqueue.delays_col_ended')}</AdminTh>
          {DEBRIEF_COLUMNS.map((c) => (
            <AdminTh key={c.key} className="text-right">
              {tA(c.key)}
            </AdminTh>
          ))}
          <AdminTh className="text-right">{tA('admin.buildqueue.delays_col_total')}</AdminTh>
        </>
      }
    >
      {delays.map((d) => {
        const badge = OBJECTIVE_BADGE[objectiveStatus(d)]
        return (
          <AdminTr key={d.match_id}>
            <AdminTd className="font-mono text-xs text-foreground">{d.match_id}</AdminTd>
            <AdminTd className="text-xs text-muted-foreground">{adminAbsoluteTime(d.ended_at, locale) || '—'}</AdminTd>
            {DEBRIEF_COLUMNS.map((c) => {
              const step = stepForColumn(d, c)
              return (
                <AdminTd
                  key={c.key}
                  className="text-right font-mono text-xs tabular-nums text-muted-foreground"
                  title={step ? adminAbsoluteTime(step.at, locale) : undefined}
                >
                  {step ? formatOffset(step.offset_seconds) : '—'}
                </AdminTd>
              )
            })}
            <AdminTd className="text-right">
              <span className="mr-2 font-mono text-xs tabular-nums text-foreground">{formatOffset(d.total_seconds)}</span>
              <StatusBadge status={badge.status} label={tA(badge.key)} />
            </AdminTd>
          </AdminTr>
        )
      })}
    </AdminTable>
  )
}
