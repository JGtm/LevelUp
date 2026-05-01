import { useCallback, useRef, useState } from 'react'

import { useParams } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SeasonPassStatus, SeasonPassTrackSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { BattlePassRewardCarousel, type RewardCard } from './BattlePassRewardCarousel'
import { BattlePassRewardLightbox, type RewardLightboxData } from './BattlePassRewardLightbox'
import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { PassContentSummary } from './PassContentSummary'
import { PalmaresShell } from './PalmaresShell'
import { useSeasonPassPage } from './queries'

function statusVariant(status: SeasonPassStatus) {
  switch (status) {
    case 'active': return 'default' as const
    case 'completed': return 'success' as const
    case 'in_progress': return 'secondary' as const
    default: return 'outline' as const
  }
}

// Extrait le dossier parent du reward_track_path pour identifier le type de pass
// (ex: "RewardTracks/Operations/foo.json" → "Operations").
function trackTypeLabel(rewardTrackPath: string): string | null {
  const parts = rewardTrackPath.replace(/\\/g, '/').split('/').filter(Boolean)
  return parts.length >= 2 ? parts[parts.length - 2] : null
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
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div className="h-full rounded-full bg-primary transition-[width]" style={{ width: `${width}%` }} />
    </div>
  )
}

type Text = ReturnType<typeof getPalmaresText>

function SeasonPassCard({ pass, intlLocale, statusLabel, labels, contentLabels, isSelected, onSelect }: {
  pass: SeasonPassTrackSummary
  intlLocale: string
  statusLabel: string
  labels: { premium: string; active: string; rank: string; progress: string; nowShowing: string }
  contentLabels: Parameters<typeof PassContentSummary>[0]['labels']
  isSelected: boolean
  onSelect: (pass: SeasonPassTrackSummary) => void
}) {
  const background = pass.background_image_url ?? pass.image_url ?? null
  const rankValue = pass.max_rank
    ? `${pass.current_rank} / ${pass.max_rank}`
    : pass.current_rank.toLocaleString(intlLocale)
  const typeLabel = trackTypeLabel(pass.reward_track_path)

  return (
    <button
      type="button"
      onClick={() => onSelect(pass)}
      aria-pressed={isSelected}
      className={[
        'group relative block w-full overflow-hidden rounded-xl border bg-card/95 text-left shadow-sm transition-all hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-400/70',
        isSelected ? 'border-sky-400/70 ring-2 ring-sky-400/40' : 'border-border/70',
      ].join(' ')}
    >
      {background && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-30 transition-opacity group-hover:opacity-40"
          style={{ backgroundImage: `url(${background})` }}
          aria-hidden="true"
        />
      )}
      <div className="relative flex h-full flex-col gap-4 p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            {typeLabel && (
              <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">{typeLabel}</p>
            )}
            <h3 className="mt-2 text-lg font-semibold text-foreground">{pass.name}</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            {isSelected && <Badge variant="default">{labels.nowShowing}</Badge>}
            {pass.is_active && <Badge variant="default">{labels.active}</Badge>}
            {pass.is_owned && <Badge variant="outline">{labels.premium}</Badge>}
            <Badge variant={statusVariant(pass.status)}>{statusLabel}</Badge>
          </div>
        </div>

        {pass.description && (
          <p className="line-clamp-2 text-sm leading-6 text-muted-foreground">{pass.description}</p>
        )}

        {pass.content && (
          <PassContentSummary content={pass.content} labels={contentLabels} locale={intlLocale} compact />
        )}

        <div className="mt-auto space-y-3">
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
        </div>
      </div>
    </button>
  )
}

function PassShowcase({
  pass,
  text,
  isViewingActive,
  onBackToActive,
  showcaseRef,
}: {
  pass: SeasonPassTrackSummary
  text: Text
  isViewingActive: boolean
  onBackToActive: () => void
  showcaseRef?: React.Ref<HTMLDivElement>
}) {
  const [selectedReward, setSelectedReward] = useState<RewardLightboxData | null>(null)

  const handleOpenCard = useCallback((card: RewardCard) => {
    const badges: RewardLightboxData['badges'] = []
    if (card.is_current) badges.push({ label: text.seasonPass.active, tone: 'current' })
    if (card.is_obtained) badges.push({ label: text.seasonPass.obtained, tone: 'obtained' })
    if (card.is_free) badges.push({ label: text.seasonPass.freeLabel, tone: 'free' })
    else badges.push({ label: text.seasonPass.premium, tone: 'premium' })
    setSelectedReward({
      title: card.title,
      rank: card.rank,
      imageUrl: card.image_url ?? null,
      description: card.description ?? null,
      quality: card.quality ?? null,
      itemType: card.item_type ?? null,
      badges,
    })
  }, [text.seasonPass.active, text.seasonPass.obtained, text.seasonPass.freeLabel, text.seasonPass.premium])

  const tierProgress = pass.active_tier_progress_percent ?? 0
  const progressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: pass.partial_progress,
    xpPerRank: pass.xp_per_rank,
    progressPercent: tierProgress,
    locale: text.intlLocale,
  })

  return (
    <div ref={showcaseRef}>
      <Card className="relative overflow-hidden border-border/70 bg-card/95 shadow-sm">
        {pass.background_image_url && (
          <div
            className="absolute inset-0 bg-cover bg-center opacity-25"
            style={{ backgroundImage: `url(${pass.background_image_url})` }}
            aria-hidden="true"
          />
        )}
        <CardContent className="relative space-y-7 p-6 lg:p-8">

        {!isViewingActive && (
          <button
            type="button"
            onClick={onBackToActive}
            className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-400/70 focus-visible:rounded"
          >
            <span aria-hidden="true">←</span>
            <span>{text.seasonPass.backToActive}</span>
          </button>
        )}

        <div className="space-y-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-xs uppercase tracking-[0.28em] text-muted-foreground">
                {pass.is_active ? text.seasonPass.activePassTitle : trackTypeLabel(pass.reward_track_path) ?? ''}
              </p>
              <h2 className="mt-3 text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">{pass.name}</h2>
            </div>
            <div className="flex flex-wrap gap-2">
              {pass.is_owned && <Badge variant="outline">{text.seasonPass.premium}</Badge>}
              {pass.is_active && <Badge variant="default">{text.seasonPass.active}</Badge>}
              <Badge variant={statusVariant(pass.status)}>{text.seasonPass.status[pass.status] ?? pass.status}</Badge>
            </div>
          </div>
          {pass.description && (
            <p className="max-w-3xl text-sm leading-7 text-muted-foreground sm:text-base">{pass.description}</p>
          )}
        </div>

        {pass.content && (
          <div className="rounded-2xl border border-border/60 bg-background/60 p-4">
            <PassContentSummary content={pass.content} labels={text.seasonPass.content} locale={text.intlLocale} />
          </div>
        )}

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
            <BattlePassRewardCarousel
              tiers={pass.tiers}
              activeTierRank={pass.active_tier_rank}
              onOpenCard={handleOpenCard}
              freeLabel={text.seasonPass.freeLabel}
            />

            {pass.is_active && pass.active_tier_rank != null && (
              <div className="space-y-3 rounded-3xl border border-slate-200/70 bg-slate-50/80 p-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{text.seasonPass.activeTierProgress}</p>
                    <p className="mt-1 text-base font-semibold text-foreground">#{pass.active_tier_rank}</p>
                  </div>
                  <p className="text-lg font-semibold text-sky-700">
                    {tierProgress.toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })} %
                  </p>
                </div>
                <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
                  <span className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs">
                    {progressLabels.current}
                  </span>
                  <div className="min-w-0">
                    <CompositeProgressBar value={tierProgress} />
                  </div>
                  <span className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs">
                    {progressLabels.target}
                  </span>
                </div>
              </div>
            )}
          </div>
        )}
        </CardContent>
        <BattlePassRewardLightbox reward={selectedReward} onClose={() => setSelectedReward(null)} />
      </Card>
    </div>
  )
}

export function SeasonPassPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const { data, isLoading, isError, error, refetch } = useSeasonPassPage(playerSlug)
  const [selectedPassPath, setSelectedPassPath] = useState<string | null>(null)
  const showcaseRef = useRef<HTMLDivElement | null>(null)

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

  const completedCount = data.passes.filter((p) => p.status === 'completed').length
  const inProgressCount = data.passes.filter((p) => p.status === 'in_progress').length
  const activePass = data.passes.find((p) => p.is_active) ?? null
  const selectedPass = (selectedPassPath
    ? data.passes.find((p) => p.reward_track_path === selectedPassPath)
    : null) ?? activePass
  const otherPasses = data.passes.filter((p) => p.reward_track_path !== selectedPass?.reward_track_path)
  const isViewingActive = !selectedPass || selectedPass.is_active

  function selectPass(pass: SeasonPassTrackSummary) {
    setSelectedPassPath(pass.reward_track_path)
    requestAnimationFrame(() => {
      showcaseRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  function backToActive() {
    setSelectedPassPath(null)
    requestAnimationFrame(() => {
      showcaseRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  return (
    <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
      <div className="grid gap-4 xl:grid-cols-3">
        <StatCard label={text.seasonPass.activeCard} value={activePass?.name ?? '—'} />
        <StatCard label={text.seasonPass.completedCard} value={completedCount.toLocaleString(text.intlLocale)} />
        <StatCard label={text.seasonPass.inProgressCard} value={inProgressCount.toLocaleString(text.intlLocale)} />
      </div>

      {selectedPass ? (
        <PassShowcase
          pass={selectedPass}
          text={text}
          isViewingActive={isViewingActive}
          onBackToActive={backToActive}
          showcaseRef={showcaseRef}
        />
      ) : data.passes.length === 0 ? (
        <EmptyStateCard title={text.seasonPass.noPassesTitle} description={text.seasonPass.noPassesDescription} />
      ) : null}

      {otherPasses.length > 0 && (
        <div className="space-y-4">
          <p className="text-xs uppercase tracking-[0.26em] text-muted-foreground">{text.seasonPass.otherPassesTitle}</p>
          <div className="grid gap-4 xl:grid-cols-2">
            {otherPasses.map((pass) => (
              <SeasonPassCard
                key={pass.reward_track_path}
                pass={pass}
                intlLocale={text.intlLocale}
                statusLabel={text.seasonPass.status[pass.status] ?? pass.status}
                contentLabels={text.seasonPass.content}
                isSelected={false}
                onSelect={selectPass}
                labels={{
                  premium: text.seasonPass.premium,
                  active: text.seasonPass.active,
                  rank: text.seasonPass.cardRank,
                  progress: text.seasonPass.cardProgress,
                  nowShowing: text.seasonPass.nowShowing,
                }}
              />
            ))}
          </div>
        </div>
      )}
    </PalmaresShell>
  )
}
