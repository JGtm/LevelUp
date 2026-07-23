/**
 * PostSyncTimeline — timeline du pipeline post-sync par joueur : barre
 * empilée horizontale (largeur des segments ∝ durée d'étape, couleur stable
 * par étape via tokens série), durée totale, top étapes lentes. Flat
 * hard-edge — l'analogue du clip-timeline csstat pour le sync.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { SchedulerPlayerOutcome } from '@/lib/api/types'
import { buildTimelineSegments, slowestSteps, stepToken, type TimelineSegment } from './timeline'
import { formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import type { Locale } from '@/lib/i18n/locale'

export function PostSyncTimeline({ players }: { players: SchedulerPlayerOutcome[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  const rows = players
    .map((p) => ({ player: p, segments: buildTimelineSegments(p.post_sync?.step_timings) }))
    .filter((r) => r.segments.length > 0)

  if (!rows.length) {
    return <EmptyStateNotice title={tA('admin.convergence.timeline_empty')} description="" />
  }

  // Légende : étapes présentes dans au moins une timeline (ordre canonique
  // préservé par l'ordre d'apparition des segments).
  const legendSteps = [...new Set(rows.flatMap((r) => r.segments.map((s) => s.step)))]

  return (
    <div className="space-y-4">
      {rows.map(({ player, segments }) => (
        <PlayerTimeline
          key={player.xuid || player.gamertag}
          gamertag={player.gamertag}
          totalMs={player.post_sync?.duration_ms ?? 0}
          segments={segments}
          slowestLabel={tA('admin.convergence.slowest')}
          totalLabel={tA('admin.convergence.timeline_total')}
          locale={locale}
        />
      ))}
      <div className="flex flex-wrap gap-x-3 gap-y-1">
        {legendSteps.map((step) => (
          <span key={step} className="inline-flex items-center gap-1 font-mono text-[10px] text-muted-foreground">
            <span aria-hidden className="h-2 w-2" style={{ backgroundColor: tokenCssVar(stepToken(step)) }} />
            {step}
          </span>
        ))}
      </div>
    </div>
  )
}

function PlayerTimeline({
  gamertag,
  totalMs,
  segments,
  slowestLabel,
  totalLabel,
  locale,
}: {
  gamertag: string
  totalMs: number
  segments: TimelineSegment[]
  slowestLabel: string
  totalLabel: string
  locale: Locale
}) {
  const top = slowestSteps(segments, 3)
  return (
    <div className="space-y-1">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-sm font-medium text-foreground">{gamertag}</span>
        <span className="font-mono text-xs tabular-nums text-muted-foreground">
          {formatDurationMs(totalMs, locale)} {totalLabel}
        </span>
      </div>
      <div className="flex h-4 w-full overflow-hidden border border-border" role="img" aria-label={gamertag}>
        {segments.map((s) => (
          <div
            key={s.step}
            className="h-full"
            style={{
              width: `${Math.max(s.pct, 0.5)}%`,
              backgroundColor: tokenCssVar(s.token),
            }}
            title={`${s.step} — ${formatDurationMs(s.durationMs, locale)} (${s.pct}%)${s.items > 0 ? ` — ${s.items} items` : ''}`}
          />
        ))}
      </div>
      <p className="font-mono text-[11px] text-muted-foreground">
        {slowestLabel} :{' '}
        {top
          .map((s) => `${s.step} ${formatDurationMs(s.durationMs, locale)} (${s.pct}%)`)
          .join(' · ')}
      </p>
    </div>
  )
}
