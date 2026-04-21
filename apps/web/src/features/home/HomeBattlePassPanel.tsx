import { useEffect, useRef } from 'react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SeasonPassPageResponse, SeasonPassTierSummary, SeasonPassTrackSummary } from '@/lib/api/types'

function clampPercent(value?: number | null) {
  if (value == null) {
    return 0
  }
  return Math.max(0, Math.min(100, value))
}

function pickFeaturedPass(passes: SeasonPassTrackSummary[]) {
  return passes.find((pass) => pass.is_active)
    ?? passes.find((pass) => pass.status === 'in_progress')
    ?? passes[0]
    ?? null
}

function formatXPLabel(value: number, locale: string) {
  return `${Math.max(0, value).toLocaleString(locale)} XP`
}

function buildCompositeProgressEdgeLabels({
  partialProgress,
  xpPerRank,
  progressPercent,
  locale,
}: {
  partialProgress: number
  xpPerRank?: number | null
  progressPercent: number
  locale: string
}) {
  if (xpPerRank != null && xpPerRank > 0) {
    return {
      current: formatXPLabel(partialProgress, locale),
      target: formatXPLabel(xpPerRank, locale),
    }
  }

  return {
    current: `${progressPercent.toLocaleString(locale, { maximumFractionDigits: 0 })} %`,
    target: '100 %',
  }
}

function CompositeTierProgressBar({ value }: { value?: number | null }) {
  const width = clampPercent(value)

  return (
    <div
      className="h-3 w-full overflow-hidden rounded-full border border-slate-300/70 bg-slate-200/70"
      style={{
        backgroundImage:
          'repeating-linear-gradient(90deg, rgba(148,163,184,0.18) 0 18px, rgba(255,255,255,0.28) 18px 24px)',
      }}
    >
      <div
        data-testid="home-battle-pass-active-tier-progress-fill"
        className="h-full rounded-full bg-sky-500 transition-[width]"
        style={{
          width: `${width}%`,
          backgroundImage:
            'repeating-linear-gradient(90deg, rgba(255,255,255,0.22) 0 18px, rgba(14,165,233,0.92) 18px 24px)',
        }}
      />
    </div>
  )
}

function BattlePassTierCard({
  tier,
  activeRef,
}: {
  tier: SeasonPassTierSummary
  activeRef?: (node: HTMLDivElement | null) => void
}) {
  const imageClasses = [
    'relative overflow-hidden rounded-[1.2rem] border border-white/85 bg-slate-950/85 shadow-[0_18px_45px_-28px_rgba(15,23,42,0.85)]',
    tier.is_current ? 'ring-2 ring-sky-400/70 ring-offset-2 ring-offset-background' : '',
    tier.is_obtained && !tier.is_current ? 'opacity-60 grayscale-[0.82]' : '',
  ].filter(Boolean).join(' ')

  return (
    <div
      ref={activeRef}
      data-testid="home-battle-pass-tier-card"
      data-current={tier.is_current ? 'true' : 'false'}
      data-obtained={tier.is_obtained ? 'true' : 'false'}
      className="snap-center shrink-0"
    >
      <div className="w-32 space-y-3 sm:w-36 xl:w-40">
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
            <span>{tier.is_obtained ? 'Obtenu' : 'A venir'}</span>
          </div>
          <p className="line-clamp-2 text-sm font-semibold text-foreground">{tier.title}</p>
        </div>
      </div>
    </div>
  )
}

export function HomeBattlePassPanel({
  loading,
  data,
  errorHint,
}: {
  loading: boolean
  data?: SeasonPassPageResponse
  errorHint?: string | null
}) {
  const featuredPass = pickFeaturedPass(data?.passes ?? [])
  const activeTier = featuredPass?.tiers?.find((tier) => tier.is_current)
    ?? featuredPass?.tiers?.find((tier) => tier.rank === featuredPass.active_tier_rank)
    ?? null
  const activeTierRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    activeTierRef.current?.scrollIntoView?.({ behavior: 'smooth', block: 'nearest', inline: 'center' })
  }, [featuredPass?.active_tier_rank, featuredPass?.tiers])

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Pass de combat</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">Chargement du pass de combat...</p>
        </CardContent>
      </Card>
    )
  }

  if (!data?.available) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Pass de combat</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Non disponible ({errorHint ?? data?.error_hint ?? 'live API non configurée'})
          </p>
        </CardContent>
      </Card>
    )
  }

  if (!featuredPass) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Pass de combat</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyStateNotice
            title="Aucun pass détecté"
            description="Aucune progression de pass saisonnier n'a été renvoyée pour ce joueur."
          />
        </CardContent>
      </Card>
    )
  }

  const trackLabel = featuredPass.reward_track_path.split('/').slice(-1)[0]
  const rankValue = featuredPass.max_rank
    ? `${featuredPass.current_rank}/${featuredPass.max_rank}`
    : featuredPass.current_rank.toLocaleString('fr-FR')
  const completionPercent = clampPercent(featuredPass.completion_percent)
  const activeTierLabel = activeTier?.title ?? 'Palier a venir'
  const activeTierRank = featuredPass.active_tier_rank == null ? '—' : `#${featuredPass.active_tier_rank}`
  const tierProgress = clampPercent(featuredPass.active_tier_progress_percent)
  const tierProgressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: featuredPass.partial_progress,
    xpPerRank: featuredPass.xp_per_rank,
    progressPercent: tierProgress,
    locale: 'fr-FR',
  })

  return (
    <Card className="relative overflow-hidden border-border/70 bg-card/95 shadow-sm">
      {featuredPass.background_image_url && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-15"
          style={{ backgroundImage: `url(${featuredPass.background_image_url})` }}
          aria-hidden="true"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background/96 to-background/85" aria-hidden="true" />

      <CardHeader className="relative space-y-4 pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">Pass de combat</CardTitle>
            <p className="mt-3 text-[11px] uppercase tracking-[0.28em] text-muted-foreground">{trackLabel}</p>
            <h3 className="mt-2 text-xl font-semibold tracking-tight text-foreground sm:text-2xl">
              {featuredPass.name}
            </h3>
          </div>

          <div className="flex flex-wrap gap-2">
            {featuredPass.is_owned && <Badge variant="outline">Premium</Badge>}
            {featuredPass.is_active && <Badge variant="default">Actif</Badge>}
            <Badge variant="outline">{`Rang ${rankValue}`}</Badge>
            <Badge variant="secondary">{`${completionPercent.toLocaleString('fr-FR', { maximumFractionDigits: 0 })} %`}</Badge>
          </div>
        </div>

        <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
          {featuredPass.description ?? 'Aucune description disponible pour ce pass.'}
        </p>
      </CardHeader>

      <CardContent className="relative space-y-6">
        <div className="overflow-hidden rounded-[1.8rem] border border-white/15 bg-slate-950/80 shadow-[0_24px_72px_-44px_rgba(15,23,42,0.92)]">
          {featuredPass.image_url ? (
            <img
              src={featuredPass.image_url}
              alt={`Illustration de ${featuredPass.name}`}
              data-testid="home-battle-pass-image"
              className="h-44 w-full object-cover sm:h-52 xl:h-60"
            />
          ) : (
            <div className="flex h-44 w-full items-center justify-center bg-[radial-gradient(circle_at_top,rgba(14,165,233,0.18),transparent_45%),linear-gradient(135deg,rgba(15,23,42,1),rgba(51,65,85,0.95))] px-6 text-center text-white sm:h-52 xl:h-60">
              <div>
                <p className="text-xs uppercase tracking-[0.34em] text-slate-300">Pass actif</p>
                <p className="mt-3 text-2xl font-semibold sm:text-3xl">{featuredPass.name}</p>
              </div>
            </div>
          )}
        </div>

        {featuredPass.tiers && featuredPass.tiers.length > 0 && (
          <div className="space-y-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">Palier actif</p>
                <p className="mt-2 text-base font-semibold text-foreground sm:text-lg">{activeTierLabel}</p>
              </div>
              <Badge variant="outline" className="border-sky-200 bg-sky-50/80 text-sky-700">
                {activeTierRank}
              </Badge>
            </div>

            <div className="relative">
              <div className="pointer-events-none absolute inset-y-0 left-0 z-10 w-10 bg-gradient-to-r from-background to-transparent" aria-hidden="true" />
              <div className="pointer-events-none absolute inset-y-0 right-0 z-10 w-10 bg-gradient-to-l from-background to-transparent" aria-hidden="true" />
              <div className="flex gap-4 overflow-x-auto px-6 pb-4 pt-1 [scrollbar-width:none] snap-x snap-mandatory">
                {featuredPass.tiers.map((tier) => (
                  <BattlePassTierCard
                    key={tier.rank}
                    tier={tier}
                    activeRef={tier.is_current ? (node) => { activeTierRef.current = node } : undefined}
                  />
                ))}
              </div>
            </div>

            <div className="space-y-3 rounded-3xl border border-slate-200/70 bg-slate-50/80 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Progression du palier</p>
                  <p className="mt-1 text-base font-semibold text-foreground">{activeTierLabel}</p>
                  {activeTier?.description && (
                    <p className="mt-1 text-sm italic text-muted-foreground">{activeTier.description}</p>
                  )}
                </div>
                <p className="text-lg font-semibold text-sky-700">
                  {tierProgress.toLocaleString('fr-FR', { maximumFractionDigits: 0 })} %
                </p>
              </div>
              <div
                data-testid="home-battle-pass-active-tier-progress-row"
                className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3"
              >
                <span
                  data-testid="home-battle-pass-active-tier-progress-current"
                  className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
                >
                  {tierProgressLabels.current}
                </span>
                <div className="min-w-0">
                  <CompositeTierProgressBar value={tierProgress} />
                </div>
                <span
                  data-testid="home-battle-pass-active-tier-progress-target"
                  className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
                >
                  {tierProgressLabels.target}
                </span>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
