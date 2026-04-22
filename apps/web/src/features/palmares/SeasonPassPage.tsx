import { useEffect, useRef } from 'react'

import { useParams } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SeasonPassStatus, SeasonPassTierSummary, SeasonPassTrackSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { PalmaresShell } from './PalmaresShell'
import { useSeasonPassPage } from './queries'

function centerTierInRail(container: HTMLDivElement | null, item: HTMLDivElement | null) {
  if (!container || !item) {
    return
  }

  const maxScrollLeft = Math.max(0, container.scrollWidth - container.clientWidth)
  const targetLeft = Math.max(
    0,
    Math.min(item.offsetLeft - (container.clientWidth - item.offsetWidth) / 2, maxScrollLeft),
  )

  if (typeof container.scrollTo === 'function') {
    container.scrollTo({ left: targetLeft, behavior: 'smooth' })
    return
  }

  container.scrollLeft = targetLeft
}

function statusVariant(status: SeasonPassStatus) {
  switch (status) {
    case 'active':
      return 'default' as const
    case 'completed':
      return 'success' as const
    case 'in_progress':
      return 'secondary' as const
    default:
      return 'outline' as const
  }
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="border-dashed">
      <CardContent className="pt-5">
        <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
      </CardContent>
    </Card>
  )
}

function ProgressBar({ value }: { value?: number | null }) {
  const width = value == null ? 0 : Math.max(0, Math.min(100, value))
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
      <div className="h-full rounded-full bg-primary transition-[width]" style={{ width: `${width}%` }} />
    </div>
  )
}

function SeasonPassCard({ pass, intlLocale, statusLabel, labels }: {
  pass: SeasonPassTrackSummary
  intlLocale: string
  statusLabel: string
  labels: { premium: string; active: string; rank: string; progress: string }
}) {
  const background = pass.background_image_url ?? pass.image_url ?? null
  const rankValue = pass.max_rank ? `${pass.current_rank}/${pass.max_rank}` : pass.current_rank.toLocaleString(intlLocale)

  return (
    <Card className="relative overflow-hidden border-border/70 bg-card/95 shadow-sm">
      {background && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-15"
          style={{ backgroundImage: `url(${background})` }}
          aria-hidden="true"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background/92 to-background/75" aria-hidden="true" />
      <CardContent className="relative flex h-full flex-col gap-4 pt-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">{pass.reward_track_path.split('/').slice(-1)[0]}</p>
            <h3 className="mt-2 text-lg font-semibold text-foreground">{pass.name}</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            {pass.is_active && <Badge variant="default">{labels.active}</Badge>}
            {pass.is_owned && <Badge variant="outline">{labels.premium}</Badge>}
            <Badge variant={statusVariant(pass.status)}>{statusLabel}</Badge>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-xs text-muted-foreground">{labels.rank}</p>
            <p className="mt-1 text-xl font-semibold text-foreground">{rankValue}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{labels.progress}</p>
            <p className="mt-1 text-xl font-semibold text-foreground">
              {pass.completion_percent == null
                ? '—'
                : `${pass.completion_percent.toLocaleString(intlLocale, { maximumFractionDigits: 0 })} %`}
            </p>
          </div>
        </div>

        <ProgressBar value={pass.completion_percent} />
      </CardContent>
    </Card>
  )
}

function SeasonPassTierCard({ tier, labels, activeRef }: {
  tier: SeasonPassTierSummary
  labels: { obtained: string; upcoming: string }
  activeRef?: (node: HTMLDivElement | null) => void
}) {
  const imageClasses = [
    'relative overflow-hidden rounded-[1.35rem] border border-white/80 bg-slate-950/85 shadow-[0_18px_45px_-28px_rgba(15,23,42,0.85)]',
    tier.is_current ? 'ring-2 ring-sky-400/70 ring-offset-2 ring-offset-background' : '',
    tier.is_obtained && !tier.is_current ? 'opacity-65 grayscale-[0.7]' : '',
  ].filter(Boolean).join(' ')

  return (
    <div
      ref={activeRef}
      data-testid="season-pass-tier-card"
      data-current={tier.is_current ? 'true' : 'false'}
      data-obtained={tier.is_obtained ? 'true' : 'false'}
      className="snap-center shrink-0"
    >
      <div className="w-40 space-y-3">
        <div className={imageClasses}>
          {tier.is_obtained && (
            <div className="absolute right-2 top-2 z-10 flex h-7 min-w-7 items-center justify-center rounded-full bg-emerald-500 px-2 text-[11px] font-semibold text-white shadow-sm">
              ✓
            </div>
          )}
          <div className="aspect-[4/5] w-full bg-gradient-to-br from-slate-200 via-slate-100 to-white">
            {tier.image_url ? (
              <img
                src={tier.image_url}
                alt={tier.title}
                className="h-full w-full object-cover"
                loading="lazy"
              />
            ) : (
              <div className="flex h-full w-full items-center justify-center bg-[radial-gradient(circle_at_top,rgba(14,165,233,0.22),transparent_55%),linear-gradient(180deg,rgba(15,23,42,0.92),rgba(30,41,59,0.84))] text-center text-white">
                <div>
                  <p className="text-[11px] uppercase tracking-[0.26em] text-slate-300">Tier</p>
                  <p className="mt-2 text-4xl font-semibold">{tier.rank}</p>
                </div>
              </div>
            )}
          </div>
        </div>
        <div className="space-y-1 px-1">
          <div className="flex items-center justify-between gap-2 text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
            <span>{`#${tier.rank}`}</span>
            <span>{tier.is_obtained ? labels.obtained : labels.upcoming}</span>
          </div>
          <p className="line-clamp-2 text-sm font-semibold text-foreground">{tier.title}</p>
        </div>
      </div>
    </div>
  )
}

function ActivePassShowcase({
  pass,
  text,
}: {
  pass: SeasonPassTrackSummary
  text: ReturnType<typeof getPalmaresText>
}) {
  const tiersRailRef = useRef<HTMLDivElement | null>(null)
  const activeTier = pass.tiers?.find((tier) => tier.is_current)
    ?? pass.tiers?.find((tier) => tier.rank === pass.active_tier_rank)
    ?? null
  const activeTierRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    centerTierInRail(tiersRailRef.current, activeTierRef.current)
  }, [pass.active_tier_rank, pass.tiers])

  const tierProgress = pass.active_tier_progress_percent ?? 0
  const tierLabel = activeTier?.title ?? text.seasonPass.activeTierFallback
  const rankLabel = pass.active_tier_rank == null ? '—' : `#${pass.active_tier_rank}`
  const progressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: pass.partial_progress,
    xpPerRank: pass.xp_per_rank,
    progressPercent: tierProgress,
    locale: text.intlLocale,
  })

  return (
    <Card className="relative overflow-hidden border-border/70 bg-card/95 shadow-sm">
      {pass.background_image_url && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-15"
          style={{ backgroundImage: `url(${pass.background_image_url})` }}
          aria-hidden="true"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background/96 to-background/85" aria-hidden="true" />
      <CardContent className="relative space-y-8 p-6 lg:p-8">
        <div className="space-y-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-[0.28em] text-muted-foreground">{text.seasonPass.activePassTitle}</p>
              <h2 className="mt-3 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">{pass.name}</h2>
            </div>
            <div className="flex flex-wrap gap-2">
              {pass.is_owned && <Badge variant="outline">{text.seasonPass.premium}</Badge>}
              <Badge variant="default">{text.seasonPass.active}</Badge>
              <Badge variant={statusVariant(pass.status)}>{text.seasonPass.status[pass.status] ?? pass.status}</Badge>
            </div>
          </div>
          {pass.description && (
            <p className="max-w-3xl text-sm leading-7 text-muted-foreground sm:text-base">
              {pass.description}
            </p>
          )}
        </div>

        {pass.background_image_url && (
          <div className="overflow-hidden rounded-[2rem] border border-white/15 bg-slate-950/80 shadow-[0_30px_90px_-50px_rgba(15,23,42,0.9)]">
            <img
              src={pass.background_image_url}
              alt={`Illustration de ${pass.name}`}
              className="aspect-[986/248] w-full object-cover"
            />
          </div>
        )}

        {pass.tiers && pass.tiers.length > 0 && (
          <div className="space-y-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">{text.seasonPass.activeTierTitle}</p>
                <p className="mt-2 text-xl font-semibold text-foreground">{tierLabel}</p>
              </div>
              <Badge variant="outline" className="border-sky-200 bg-sky-50/80 text-sky-700">
                {rankLabel}
              </Badge>
            </div>

            <div className="relative">
              <div className="pointer-events-none absolute inset-y-0 left-0 z-10 w-10 bg-gradient-to-r from-background to-transparent" aria-hidden="true" />
              <div className="pointer-events-none absolute inset-y-0 right-0 z-10 w-10 bg-gradient-to-l from-background to-transparent" aria-hidden="true" />
              <div
                ref={tiersRailRef}
                className="flex gap-4 overflow-x-auto pb-4 pr-6 pt-1 [scrollbar-width:none]"
              >
                {pass.tiers.map((tier) => (
                  <SeasonPassTierCard
                    key={tier.rank}
                    tier={tier}
                    labels={{ obtained: text.seasonPass.obtained, upcoming: text.seasonPass.upcoming }}
                    activeRef={tier.is_current ? (node) => { activeTierRef.current = node } : undefined}
                  />
                ))}
              </div>
            </div>

            <div className="space-y-3 rounded-3xl border border-slate-200/70 bg-slate-50/80 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{text.seasonPass.activeTierProgress}</p>
                  <p className="mt-1 text-base font-semibold text-foreground">{tierLabel}</p>
                  {activeTier?.description && (
                    <p className="mt-1 text-sm italic text-muted-foreground">{activeTier.description}</p>
                  )}
                </div>
                <p className="text-lg font-semibold text-sky-700">
                  {tierProgress.toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })} %
                </p>
              </div>
              <div
                data-testid="season-pass-active-tier-progress-row"
                className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3"
              >
                <span
                  data-testid="season-pass-active-tier-progress-current"
                  className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
                >
                  {progressLabels.current}
                </span>
                <div className="min-w-0">
                  <CompositeProgressBar value={tierProgress} fillTestId="season-pass-active-tier-progress-fill" />
                </div>
                <span
                  data-testid="season-pass-active-tier-progress-target"
                  className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
                >
                  {progressLabels.target}
                </span>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function SeasonPassPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const { data, isLoading, isError, error, refetch } = useSeasonPassPage(playerSlug)

  if (isLoading) {
    return (
      <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
        <div className="flex items-center justify-center py-24">
          <Spinner size="lg" />
        </div>
      </PalmaresShell>
    )
  }

  if (isError || !data) {
    return (
      <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
        <EmptyStateCard
          title={text.seasonPass.unavailableTitle}
          description={error?.message ?? text.seasonPass.unavailableDescription}
          actionLabel={text.seasonPass.retry}
          onAction={() => refetch()}
        />
      </PalmaresShell>
    )
  }

  const completedCount = data.passes.filter((pass) => pass.status === 'completed').length
  const inProgressCount = data.passes.filter((pass) => pass.status === 'in_progress').length
  const activePass = data.passes.find((pass) => pass.is_active)
  const otherPasses = data.passes.filter((pass) => !pass.is_active)

  return (
    <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
      <div className="grid gap-4 xl:grid-cols-4">
        <StatCard label={text.seasonPass.activeCard} value={activePass?.name ?? '—'} />
        <StatCard label={text.seasonPass.completedCard} value={completedCount.toLocaleString(text.intlLocale)} />
        <StatCard label={text.seasonPass.inProgressCard} value={inProgressCount.toLocaleString(text.intlLocale)} />
        <StatCard label={text.seasonPass.challengesCard} value={(data.challenges.total ?? 0).toLocaleString(text.intlLocale)} />
      </div>

      {activePass ? (
        <ActivePassShowcase pass={activePass} text={text} />
      ) : data.passes.length > 0 ? null : (
        <EmptyStateCard title={text.seasonPass.noPassesTitle} description={text.seasonPass.noPassesDescription} />
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{text.seasonPass.challengesTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          {data.challenges.available ? (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.completed}</p>
                <p className="mt-1 text-lg font-semibold">{(data.challenges.completed ?? 0).toLocaleString(text.intlLocale)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.total}</p>
                <p className="mt-1 text-lg font-semibold">{(data.challenges.total ?? 0).toLocaleString(text.intlLocale)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.xpAvailable}</p>
                <p className="mt-1 text-lg font-semibold">{(data.challenges.xp_available ?? 0).toLocaleString(text.intlLocale)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.nextExpiry}</p>
                <p className="mt-1 text-lg font-semibold">
                  {data.challenges.next_expiry
                    ? new Date(data.challenges.next_expiry).toLocaleString(text.intlLocale)
                    : text.seasonPass.noExpiry}
                </p>
              </div>
            </div>
          ) : (
            <EmptyStateNotice title={text.seasonPass.challengesUnavailable} description={data.challenges.error_hint ?? text.seasonPass.unavailableDescription} />
          )}
        </CardContent>
      </Card>

      {otherPasses.length > 0 && (
        <div className="space-y-4">
          <div>
            <p className="text-xs uppercase tracking-[0.26em] text-muted-foreground">{text.seasonPass.otherPassesTitle}</p>
          </div>
        <div className="grid gap-4 xl:grid-cols-2">
          {otherPasses.map((pass) => (
            <SeasonPassCard
              key={pass.reward_track_path}
              pass={pass}
              intlLocale={text.intlLocale}
              statusLabel={text.seasonPass.status[pass.status] ?? pass.status}
              labels={{
                premium: text.seasonPass.premium,
                active: text.seasonPass.active,
                rank: text.seasonPass.cardRank,
                progress: text.seasonPass.cardProgress,
              }}
            />
          ))}
        </div>
        </div>
      )}
    </PalmaresShell>
  )
}
