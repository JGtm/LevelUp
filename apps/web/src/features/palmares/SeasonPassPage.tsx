import { useCallback, useMemo, useRef, useState } from 'react'

import { useParams } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, clampCompositeProgress } from '@/components/ui/composite-progress-bar'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SeasonPassStatus, SeasonPassTrackSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { BattlePassRewardCarousel, buildTierGroups, type RewardCard } from './BattlePassRewardCarousel'
import { BattlePassRewardLightbox, type RewardLightboxData } from './BattlePassRewardLightbox'
import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { PassContentSummary, type ContentLabels } from './PassContentSummary'
import { PalmaresShell } from './PalmaresShell'
import { useSeasonPassPage } from './queries'
import { isArmorItemType, rarityLabel, rarityStyle, type RarityTier } from './rarity'

function statusVariant(status: SeasonPassStatus) {
  switch (status) {
    case 'active': return 'default' as const
    case 'completed': return 'success' as const
    case 'in_progress': return 'secondary' as const
    default: return 'outline' as const
  }
}

// Ordre décroissant de rareté — du plus rare au plus commun.
const RARITY_ORDER: RarityTier[] = ['mythic', 'legendary', 'epic', 'rare', 'common']

type SeasonPassContent = NonNullable<SeasonPassTrackSummary['content']>

function OverlayContentRows({
  content,
  labels,
  locale,
  palmaresLocale,
}: {
  content: SeasonPassContent
  labels: ContentLabels
  locale: string
  palmaresLocale: 'fr' | 'en'
}) {
  type Chip = { key: string; value: string; label: string }

  const row1: Chip[] = []
  if (content.total_tiers > 0) row1.push({ key: 'tiers', value: String(content.total_tiers), label: labels.tiersLabel })
  if (content.credits) row1.push({ key: 'cr', value: content.credits.toLocaleString(locale), label: labels.creditsLabel })
  if (content.spartan_points) row1.push({ key: 'sp', value: content.spartan_points.toLocaleString(locale), label: labels.spartanPointsLabel })
  if (content.xp_boosts) row1.push({ key: 'xp', value: String(content.xp_boosts), label: labels.xpBoostsLabel })
  if (content.challenge_swaps) row1.push({ key: 'swap', value: String(content.challenge_swaps), label: labels.challengeSwapsLabel })

  const row2: Chip[] = []
  if (content.cosmetics_total) {
    if (content.type_breakdown && Object.keys(content.type_breakdown).length > 0) {
      let armor = 0
      for (const [type, count] of Object.entries(content.type_breakdown)) {
        if (isArmorItemType(type)) armor += count
      }
      const cosmetic = Math.max(0, content.cosmetics_total - armor)
      if (armor > 0) row2.push({ key: 'armor', value: String(armor), label: labels.armorLabel })
      if (cosmetic > 0) row2.push({ key: 'cosmetic', value: String(cosmetic), label: labels.cosmeticsSplitLabel })
    } else {
      row2.push({ key: 'cosmetics', value: String(content.cosmetics_total), label: labels.cosmeticsLabel })
    }
  }

  const rarities = RARITY_ORDER
    .map((tier) => ({ tier, count: content.rarity_breakdown?.[tier] ?? 0 }))
    .filter((e) => e.count > 0)

  if (row1.length === 0 && row2.length === 0 && rarities.length === 0) return null

  const chipRow = (chips: Chip[]) => chips.length === 0 ? null : (
    <div className="flex flex-wrap items-baseline gap-x-3 text-xs">
      {chips.map(({ key, value, label }) => (
        <span key={key}>
          <span className="font-semibold text-white tabular-nums">{value}</span>
          {' '}
          <span className="text-white/60">{label}</span>
        </span>
      ))}
    </div>
  )

  return (
    <div className="mt-2 space-y-0.5">
      {chipRow(row1)}
      {chipRow(row2)}
      {rarities.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-2.5 text-xs">
          {rarities.map(({ tier, count }) => (
            <span key={tier} className="flex items-center gap-1">
              <span className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${rarityStyle(tier)?.segment ?? 'bg-muted-foreground/60'}`} />
              <span className="text-white/60">{rarityLabel(tier, palmaresLocale)}</span>
              {' '}
              <span className="font-semibold text-white tabular-nums">{count}</span>
            </span>
          ))}
        </div>
      )}
    </div>
  )
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

  return (
    <button
      type="button"
      onClick={() => onSelect(pass)}
      aria-pressed={isSelected}
      className={[
        'group relative block w-full overflow-hidden rounded-xl border bg-card/95 text-left shadow-sm transition-all hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        isSelected ? 'border-primary/70 ring-2 ring-primary/40' : 'border-border/70',
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
          <h3 className="text-lg font-semibold text-foreground">{pass.name}</h3>
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
  locale,
  isViewingActive,
  onBackToActive,
  showcaseRef,
}: {
  pass: SeasonPassTrackSummary
  text: Text
  locale: 'fr' | 'en'
  isViewingActive: boolean
  onBackToActive: () => void
  showcaseRef?: React.Ref<HTMLDivElement>
}) {
  const allCards = useMemo(
    () => (pass.tiers ? buildTierGroups(pass.tiers).flatMap((g) => g.cards) : []),
    [pass.tiers],
  )
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null)

  const selectedReward = useMemo<RewardLightboxData | null>(() => {
    if (selectedIndex == null) return null
    const card = allCards[selectedIndex]
    if (!card) return null
    const badges: RewardLightboxData['badges'] = []
    if (card.is_current) badges.push({ label: text.seasonPass.active, tone: 'current' })
    if (card.is_obtained) badges.push({ label: text.seasonPass.obtained, tone: 'obtained' })
    if (card.is_free) badges.push({ label: text.seasonPass.freeLabel, tone: 'free' })
    else badges.push({ label: text.seasonPass.premium, tone: 'premium' })
    return {
      title: card.title, rank: card.rank,
      imageUrl: card.image_url ?? null, description: card.description ?? null,
      quality: card.quality ?? null, itemType: card.item_type ?? null, badges,
    }
  }, [selectedIndex, allCards, text.seasonPass.active, text.seasonPass.obtained, text.seasonPass.freeLabel, text.seasonPass.premium])

  const handleOpenCard = useCallback((card: RewardCard) => {
    const idx = allCards.findIndex((c) => c.key === card.key)
    setSelectedIndex(idx >= 0 ? idx : null)
  }, [allCards])

  const handlePrev = useCallback(() => setSelectedIndex((i) => (i != null && i > 0 ? i - 1 : i)), [])
  const handleNext = useCallback(() => setSelectedIndex((i) => (i != null && i < allCards.length - 1 ? i + 1 : i)), [allCards.length])

  const tierProgress = pass.active_tier_progress_percent ?? 0
  // Pour les passes sans palier actif (complété, non commencé), rabat sur completion_percent.
  const barPercent = pass.active_tier_rank != null
    ? clampCompositeProgress(tierProgress)
    : clampCompositeProgress(pass.completion_percent ?? 0)
  const progressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: pass.partial_progress,
    xpPerRank: pass.xp_per_rank,
    progressPercent: barPercent,
    locale: text.intlLocale,
  })

  return (
    <div ref={showcaseRef}>
      <Card className="overflow-hidden border-border/70 bg-card/95 shadow-sm">
        <CardContent className="space-y-6 p-6 lg:p-8">

          {!isViewingActive && (
            <button
              type="button"
              onClick={onBackToActive}
              className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:rounded"
            >
              <span aria-hidden="true">←</span>
              <span>{text.seasonPass.backToActive}</span>
            </button>
          )}

          {/* Hero : image du pass avec infos en overlay */}
          {pass.background_image_url ? (
            <div className="relative overflow-hidden rounded-2xl">
              <img
                src={pass.background_image_url}
                alt={pass.name}
                className="aspect-[986/248] w-full object-cover"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/50 to-transparent" />
              <div className="absolute inset-x-0 bottom-0 p-4 sm:p-5">
                <h2 className="text-xl font-bold tracking-tight text-white sm:text-2xl">
                  {pass.name}
                </h2>
                <div className="mt-1 flex flex-wrap gap-1.5">
                  <Badge variant={statusVariant(pass.status)}>
                    {text.seasonPass.status[pass.status] ?? pass.status}
                  </Badge>
                  {pass.is_active && <Badge variant="default">{text.seasonPass.active}</Badge>}
                  {pass.is_owned && <Badge variant="outline">{text.seasonPass.premium}</Badge>}
                </div>
                {pass.content && (
                  <OverlayContentRows
                    content={pass.content}
                    labels={text.seasonPass.content}
                    locale={text.intlLocale}
                    palmaresLocale={locale}
                  />
                )}
              </div>
            </div>
          ) : (
            <div className="flex flex-wrap items-start justify-between gap-3">
              <h2 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">{pass.name}</h2>
              <div className="flex flex-wrap gap-2">
                {pass.is_owned && <Badge variant="outline">{text.seasonPass.premium}</Badge>}
                {pass.is_active && <Badge variant="default">{text.seasonPass.active}</Badge>}
                <Badge variant={statusVariant(pass.status)}>{text.seasonPass.status[pass.status] ?? pass.status}</Badge>
              </div>
            </div>
          )}

          {pass.description && (
            <p className="max-w-3xl text-sm leading-7 text-muted-foreground sm:text-base">{pass.description}</p>
          )}

          {pass.tiers && pass.tiers.length > 0 && (
            <div className="space-y-5">
              <BattlePassRewardCarousel
                tiers={pass.tiers}
                activeTierRank={pass.active_tier_rank}
                onOpenCard={handleOpenCard}
                freeLabel={text.seasonPass.freeLabel}
              />

              <div className="flex justify-center">
                <div className="grid w-2/3 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-[11px] text-muted-foreground">
                  <span data-testid="season-pass-active-tier-progress-current" className="shrink-0 whitespace-nowrap">
                    {progressLabels.current}
                  </span>
                  <div className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
                    <div className="h-2 w-full">
                      <div
                        data-testid="season-pass-active-tier-progress-fill"
                        className="h-full rounded-full bg-primary transition-all duration-300"
                        style={{ width: `${barPercent}%` }}
                      />
                    </div>
                  </div>
                  <span data-testid="season-pass-active-tier-progress-target" className="shrink-0 whitespace-nowrap text-right">
                    {progressLabels.target}
                  </span>
                </div>
              </div>
            </div>
          )}
        </CardContent>
        <BattlePassRewardLightbox
          reward={selectedReward}
          onClose={() => setSelectedIndex(null)}
          onPrev={selectedIndex != null && selectedIndex > 0 ? handlePrev : undefined}
          onNext={selectedIndex != null && selectedIndex < allCards.length - 1 ? handleNext : undefined}
        />
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
          locale={locale}
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
