// cross-feature-allow: section Historique Réalisations — agrège les défis
// terminaux (features/prestige) + arcs + campagnes closes (features/ascension).
/**
 * HistorySection — mémoire complète datée de l'onglet Réalisations (Lot C).
 *
 * Trois blocs chronologiques sous les jalons, distincts du bloc « célébration »
 * (cartes moments) :
 *   1. Objectifs passés   — défis terminaux (réussi/expiré/abandonné/retiré) + date
 *   2. Arcs terminés      — nom + date de complétion
 *   3. Campagnes closes    — axe, delta snapshot→final, playlist, dates, statut
 *
 * Principe : Objectifs = actif uniquement ; Réalisations = liste exhaustive datée.
 */
import type { ReactNode } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { tokenCssVar } from '@/lib/accessibility'
import { metricLabel } from '@/lib/i18n/metricLabel'
import { useChallengeHistory, useArcs } from '@/features/prestige/hooks'
import { useCampaignHistory } from './profile/queries'
import { useProfileI18n } from './profile/useProfileI18n'
import { getAscensionText, type AscensionText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'
import type { Challenge, ChallengeStatus, Arc } from '@/lib/prestige'
import type { CampaignHistoryItem } from '@/lib/playerProfile'
import type { ProfileManifestKey } from '@/lib/i18n/generated/profile'

interface HistorySectionProps {
  playerSlug: string
}

export function HistorySection({ playerSlug }: HistorySectionProps) {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const t = getAscensionText(locale)

  const { data: challengeData } = useChallengeHistory(playerSlug, titleSlug)
  const { data: arcData } = useArcs(playerSlug, titleSlug)
  const { data: campaignData } = useCampaignHistory(playerSlug)

  const pastChallenges = [...(challengeData?.challenges ?? [])].sort((a, b) =>
    terminalDate(b).localeCompare(terminalDate(a)),
  )
  const completedArcs = (arcData?.arcs ?? [])
    .filter((a) => !!a.completed_at)
    .sort((a, b) => (b.completed_at ?? '').localeCompare(a.completed_at ?? ''))
  const closedCampaigns = campaignData ?? []

  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {t.historyTitle}
      </h2>
      <div className="space-y-6">
        <PastObjectivesBlock challenges={pastChallenges} locale={locale} t={t} />
        <CompletedArcsBlock arcs={completedArcs} locale={locale} t={t} />
        <ClosedCampaignsBlock campaigns={closedCampaigns} locale={locale} t={t} />
      </div>
    </section>
  )
}

// ─── Bloc : objectifs passés ─────────────────────────────────────────────────

interface PastObjectivesBlockProps {
  challenges: Challenge[]
  locale: Locale
  t: AscensionText
}

function PastObjectivesBlock({ challenges, locale, t }: PastObjectivesBlockProps) {
  return (
    <HistoryBlock title={t.historyObjectivesTitle}>
      {challenges.length === 0 ? (
        <EmptyRow message={t.historyObjectivesEmpty} />
      ) : (
        <ul className="divide-y divide-border">
          {challenges.map((c) => (
            <li key={c.id} className="flex items-center justify-between gap-3 py-2">
              <span className="min-w-0 truncate text-sm font-medium">
                {c.label || metricLabel(c.metric, locale)}
              </span>
              <span className="flex shrink-0 items-center gap-2">
                <ResultBadge status={c.status} t={t} />
                <time className="text-xs text-muted-foreground">
                  {formatDate(terminalDate(c), locale)}
                </time>
              </span>
            </li>
          ))}
        </ul>
      )}
    </HistoryBlock>
  )
}

// ─── Bloc : arcs terminés ────────────────────────────────────────────────────

interface CompletedArcsBlockProps {
  arcs: Arc[]
  locale: Locale
  t: AscensionText
}

function CompletedArcsBlock({ arcs, locale, t }: CompletedArcsBlockProps) {
  return (
    <HistoryBlock title={t.historyArcsTitle}>
      {arcs.length === 0 ? (
        <EmptyRow message={t.historyArcsEmpty} />
      ) : (
        <ul className="divide-y divide-border">
          {arcs.map((a) => (
            <li key={a.id} className="flex items-center justify-between gap-3 py-2">
              <span className="min-w-0 truncate text-sm font-medium">{a.title}</span>
              <time className="shrink-0 text-xs text-muted-foreground">
                {t.historyArcCompletedOn.replace('{date}', formatDate(a.completed_at, locale))}
              </time>
            </li>
          ))}
        </ul>
      )}
    </HistoryBlock>
  )
}

// ─── Bloc : campagnes closes ─────────────────────────────────────────────────

interface ClosedCampaignsBlockProps {
  campaigns: CampaignHistoryItem[]
  locale: Locale
  t: AscensionText
}

function ClosedCampaignsBlock({ campaigns, locale, t }: ClosedCampaignsBlockProps) {
  const { t: pt } = useProfileI18n()
  const axisLabel = (c: CampaignHistoryItem) => {
    const key = (c.axis_kind === 'radar'
      ? `profile.axis.${c.axis}`
      : `profile.lusr.${c.axis}`) as ProfileManifestKey
    return pt(key)
  }
  const playlistLabel = (c: CampaignHistoryItem) =>
    c.playlist_group === 'all' ? pt('campaign.tracker.playlist_all') : c.playlist_group

  return (
    <HistoryBlock title={t.historyCampaignsTitle}>
      {campaigns.length === 0 ? (
        <EmptyRow message={t.historyCampaignsEmpty} />
      ) : (
        <ul className="divide-y divide-border">
          {campaigns.map((c) => (
            <li key={c.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{axisLabel(c)}</p>
                <p className="text-xs text-muted-foreground">
                  {playlistLabel(c)} · {pt(`campaign.status.${c.status}` as ProfileManifestKey)}
                </p>
              </div>
              <div className="flex shrink-0 items-center gap-3">
                {c.delta !== undefined && (
                  <span className="text-xs">
                    <span className="text-muted-foreground">{t.historyCampaignProgress} </span>
                    <DeltaValue delta={c.delta} />
                  </span>
                )}
                <time className="text-xs text-muted-foreground">
                  {formatDate(c.ended_at ?? c.started_at, locale)}
                </time>
              </div>
            </li>
          ))}
        </ul>
      )}
    </HistoryBlock>
  )
}

// ─── Primitives ──────────────────────────────────────────────────────────────

function HistoryBlock({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div>
      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        {title}
      </h3>
      {children}
    </div>
  )
}

function EmptyRow({ message }: { message: string }) {
  return (
    <p className="rounded-md border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
      {message}
    </p>
  )
}

function ResultBadge({ status, t }: { status: ChallengeStatus; t: AscensionText }) {
  const label = challengeResultLabel(status, t)
  const positive = status === 'completed'
  return (
    <span
      className="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground"
      style={
        positive
          ? {
              backgroundColor: `color-mix(in srgb, ${tokenCssVar('outcome-win')} 18%, transparent)`,
              color: tokenCssVar('outcome-win'),
            }
          : undefined
      }
    >
      {label}
    </span>
  )
}

function DeltaValue({ delta }: { delta: number }) {
  const rounded = Math.round(delta * 100) / 100
  const accent = rounded > 0 ? 'outcome-win' : rounded < 0 ? 'outcome-loss' : undefined
  const sign = rounded > 0 ? '+' : rounded < 0 ? '−' : ''
  return (
    <span
      className="font-mono font-semibold"
      style={accent ? { color: tokenCssVar(accent) } : undefined}
    >
      {sign}
      {Math.abs(rounded).toFixed(2)}
    </span>
  )
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

/** Date terminale d'un défi : la plus spécifique disponible, sinon création. */
function terminalDate(c: Challenge): string {
  return c.completed_at ?? c.abandoned_at ?? c.expired_at ?? c.created_at
}

function challengeResultLabel(status: ChallengeStatus, t: AscensionText): string {
  switch (status) {
    case 'completed':
      return t.historyResultCompleted
    case 'expired':
      return t.historyResultExpired
    case 'abandoned':
      return t.historyResultAbandoned
    case 'archived':
      return t.historyResultArchived
    default:
      return status
  }
}

function formatDate(iso: string | undefined, locale: Locale): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString(intlLocale(locale), {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  })
}
