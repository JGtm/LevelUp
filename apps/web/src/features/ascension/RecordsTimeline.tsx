/**
 * RecordsTimeline — affichage des Personal Bests + timeline historique.
 *
 * Décomposé en 2 sections :
 *  1. Cards "PB courants" groupées par métrique × période (30d/90d/all_time)
 *  2. Liste chronologique des records battus (record_history)
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §5.3.
 *
 * Note (commit 9) : pas de panneau "records proches" dédié — les alertes
 * `record_near_miss` arrivent via le centre de notifs existant, ce qui
 * évite un doublon UI et donne un feedback proactif plutôt que statique.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useMetricLabel } from '@/lib/i18n/metricLabel'
import { buildSoloFilterLink, dayWindowUTC } from '@/features/filters/filterLink'
import { useRecords } from './queries'
import { getAscensionText } from './i18n'
import { formatAscensionDate, formatMetricValue, interpolate } from './format'
import type { PersonalBest, RecordHistory, RecordPeriod } from './types'
import type { Locale } from '@/lib/i18n/locale'

export interface RecordsTimelineProps {
  playerSlug: string
}

const PERIOD_ORDER: RecordPeriod[] = ['30d', '90d', 'all_time']

/** Lien « voir la période » : Solo borné sur la journée (UTC) du record. Full-page
 *  nav (`<a href>`) — le `?f=` n'est décodé qu'au rehydrate du store solo. */
function SeePeriodLink({
  achievedAt,
  playerSlug,
  titleSlug,
  label,
}: {
  achievedAt: string
  playerSlug: string
  titleSlug: string
  label: string
}) {
  const href = buildSoloFilterLink({ playerSlug, titleSlug, period: dayWindowUTC(achievedAt) })
  return (
    <a href={href} className="text-2xs text-primary hover:underline">
      {label}
    </a>
  )
}

export function RecordsTimeline({ playerSlug }: RecordsTimelineProps) {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const t = getAscensionText(locale)
  const { data, isLoading, isError } = useRecords(playerSlug, { historyLimit: 50 })

  if (isLoading) {
    return (
      <p className="text-sm text-muted-foreground" role="status">
        {t.loading}
      </p>
    )
  }
  if (isError) {
    return (
      <p className="text-sm text-destructive" role="alert">
        {t.errorLoading}
      </p>
    )
  }

  const pbs = data?.personal_bests ?? []
  const history = data?.history ?? []

  if (pbs.length === 0 && history.length === 0) {
    return (
      <div className="rounded-md border border-border bg-card p-6 text-center text-muted-foreground">
        {t.recordsEmpty}
      </div>
    )
  }

  // Regroupe les PB par métrique (toutes périodes confondues dans la card).
  const pbByMetric = new Map<string, PersonalBest[]>()
  for (const pb of pbs) {
    const list = pbByMetric.get(pb.metric) ?? []
    list.push(pb)
    pbByMetric.set(pb.metric, list)
  }
  // Tri stable des périodes dans chaque card.
  for (const list of pbByMetric.values()) {
    list.sort(
      (a, b) =>
        PERIOD_ORDER.indexOf(a.period) - PERIOD_ORDER.indexOf(b.period),
    )
  }

  return (
    <section aria-labelledby="records-section-heading" className="space-y-4">
      <h2 id="records-section-heading" className="text-lg font-semibold">
        {t.recordsSectionTitle}
      </h2>

      {/* ─── PB courants ─────────────────────────────────────────────── */}
      {pbs.length > 0 && (
        <div>
          <h3 className="mb-2 text-sm font-medium text-muted-foreground">
            {t.recordsPersonalBestsTitle}
          </h3>
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {[...pbByMetric.entries()].map(([metric, list]) => (
              <li key={metric}>
                <PBCard
                  metric={metric}
                  pbs={list}
                  locale={locale}
                  t={t}
                  playerSlug={playerSlug}
                  titleSlug={titleSlug}
                />
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* ─── Timeline historique ─────────────────────────────────────── */}
      <div>
        <h3 className="mb-2 text-sm font-medium text-muted-foreground">
          {t.recordsTimelineTitle}
        </h3>
        {history.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.recordsHistoryEmpty}</p>
        ) : (
          <ol className="space-y-2" aria-label={t.recordsTimelineTitle}>
            {history.map((h) => (
              <li key={h.id}>
                <HistoryRow
                  entry={h}
                  locale={locale}
                  t={t}
                  playerSlug={playerSlug}
                  titleSlug={titleSlug}
                />
              </li>
            ))}
          </ol>
        )}
      </div>
    </section>
  )
}

interface PBCardProps {
  metric: string
  pbs: PersonalBest[]
  locale: Locale
  t: ReturnType<typeof getAscensionText>
  playerSlug: string
  titleSlug: string
}

function PBCard({ metric, pbs, locale, t, playerSlug, titleSlug }: PBCardProps) {
  const label = useMetricLabel(metric)
  return (
    <article className="rounded-md border border-border bg-card p-4">
      <h4 className="mb-3 text-sm font-medium">{label}</h4>
      <dl className="space-y-2 text-xs">
        {pbs.map((pb) => (
          <div key={pb.period} className="flex items-baseline justify-between gap-2">
            <dt className="text-muted-foreground">{t.period[pb.period]}</dt>
            <dd className="flex flex-col items-end">
              <span className="text-base font-semibold">
                {formatMetricValue(metric, pb.value)}
              </span>
              {pb.previous_value != null && (
                <span className="text-2xs text-muted-foreground">
                  {interpolate(t.recordsPreviousValue, {
                    value: formatMetricValue(metric, pb.previous_value),
                  })}
                </span>
              )}
              {pb.achieved_at && (
                <span className="text-2xs text-muted-foreground">
                  {interpolate(t.recordsAchievedAt, {
                    date: formatAscensionDate(pb.achieved_at, locale),
                  })}
                </span>
              )}
              {pb.achieved_at && (
                <SeePeriodLink
                  achievedAt={pb.achieved_at}
                  playerSlug={playerSlug}
                  titleSlug={titleSlug}
                  label={t.recordSeePeriod}
                />
              )}
            </dd>
          </div>
        ))}
      </dl>
    </article>
  )
}

interface HistoryRowProps {
  entry: RecordHistory
  locale: Locale
  t: ReturnType<typeof getAscensionText>
  playerSlug: string
  titleSlug: string
}

function HistoryRow({ entry, locale, t, playerSlug, titleSlug }: HistoryRowProps) {
  const label = useMetricLabel(entry.metric)
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-border bg-card px-3 py-2 text-sm">
      <div className="flex flex-1 items-baseline gap-2">
        <span className="text-xs text-muted-foreground">
          {formatAscensionDate(entry.achieved_at, locale)}
        </span>
        <span className="font-medium">{label}</span>
        <span className="text-xs text-muted-foreground">
          · {t.period[entry.period]}
        </span>
      </div>
      <div className="flex items-center gap-3">
        <SeePeriodLink
          achievedAt={entry.achieved_at}
          playerSlug={playerSlug}
          titleSlug={titleSlug}
          label={t.recordSeePeriod}
        />
        <span className="text-base font-semibold">
          {formatMetricValue(entry.metric, entry.value)}
        </span>
      </div>
    </div>
  )
}
