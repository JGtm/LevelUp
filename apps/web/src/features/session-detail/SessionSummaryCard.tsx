/**
 * SessionSummaryCard — bloc résumé pour une session (active ou comparée).
 *
 * Tone "primary" (couleur thème) pour la session active, "compare" pour la comparée.
 * Affiche 4 stats clés : matchs, W/L, KDA, performance score.
 */
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SessionCompareEntry } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { formatNumber, useSessionT } from './_shared'

interface Props {
  title: string
  entry: SessionCompareEntry | null
  tone: 'primary' | 'compare'
}

export function SessionSummaryCard({ title, entry, tone }: Props) {
  const toneClass =
    tone === 'primary' ? 'border-primary/20 bg-primary/5' : 'border-compare-b bg-compare-b/10'
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string => fieldMappings?.fields[key]?.label ?? key
  const t = useSessionT()

  if (!entry) {
    return (
      <Card className={toneClass}>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyStateNotice
            title={t('session.detail.summary_unavailable_title')}
            description={t('session.detail.summary_unavailable_description')}
          />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={toneClass}>
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">{title}</CardTitle>
            <p className="mt-1 text-xs text-muted-foreground">{entry.session_label}</p>
          </div>
          {entry.dominant_category && <Badge variant="secondary">{entry.dominant_category}</Badge>}
        </div>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <SessionStat label={t('session.detail.stat_matches')} value={entry.total_matches.toString()} />
        <SessionStat label={t('session.detail.stat_wins_losses')} value={`${entry.wins} / ${entry.losses}`} />
        <SessionStat label={labelOf('kda')} value={formatNumber(entry.kda, 2)} />
        <SessionStat label={t('session.detail.stat_perf_score')} value={formatNumber(entry.performance_score, 1)} />
      </CardContent>
    </Card>
  )
}

function SessionStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-border/60 bg-background/70 p-3">
      <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold text-foreground">{value}</p>
    </div>
  )
}
