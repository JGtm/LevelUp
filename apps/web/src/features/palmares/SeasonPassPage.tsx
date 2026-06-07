import { useCallback, useMemo, useRef, useState } from 'react'

import { useParams } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, clampCompositeProgress } from '@/components/ui/composite-progress-bar'
import { DataFreshnessIndicator } from '@/components/ui/data-freshness-indicator'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SeasonPassStatus, SeasonPassTrackSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { BattlePassRewardCarousel, buildTierGroups, type RewardCard } from './BattlePassRewardCarousel'
import { BattlePassRewardLightbox, type RewardLightboxData } from './BattlePassRewardLightbox'
import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { PassContentSummary, type ContentLabels } from './PassContentSummary'
import { useSeasonPassPage } from './queries'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
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
  remaining,
  labels,
  locale,
  palmaresLocale,
}: {
  content: SeasonPassContent
  /** Contenu restant (paliers non atteints). Fourni ⇒ affichage « restant/total » (XX/YY). */
  remaining?: SeasonPassContent | null
  labels: ContentLabels
  locale: string
  palmaresLocale: 'fr' | 'en'
}) {
  type Chip = { key: string; value: string; label: string }
  const showRemaining = remaining !== undefined
  const rem = remaining ?? null
  // Mode XX/YY : affiche l'ACQUIS (= total − restant) sur le total (cf. demande user :
  // « X obtenus sur Y », colle à la complétion). r = valeur des paliers NON atteints.
  const val = (total: number, r: number) =>
    showRemaining ? `${Math.max(0, total - r).toLocaleString(locale)}/${total.toLocaleString(locale)}` : total.toLocaleString(locale)
  const armorOf = (c: SeasonPassContent | null) => {
    let armor = 0
    if (c?.type_breakdown) for (const [type, count] of Object.entries(c.type_breakdown)) if (isArmorItemType(type)) armor += count
    return armor
  }

  const row1: Chip[] = []
  if (content.total_tiers > 0) row1.push({ key: 'tiers', value: val(content.total_tiers, rem?.total_tiers ?? 0), label: labels.tiersLabel })
  if (content.credits) row1.push({ key: 'cr', value: val(content.credits, rem?.credits ?? 0), label: labels.creditsLabel })
  if (content.spartan_points) row1.push({ key: 'sp', value: val(content.spartan_points, rem?.spartan_points ?? 0), label: labels.spartanPointsLabel })
  if (content.xp_boosts) row1.push({ key: 'xp', value: val(content.xp_boosts, rem?.xp_boosts ?? 0), label: labels.xpBoostsLabel })
  if (content.challenge_swaps) row1.push({ key: 'swap', value: val(content.challenge_swaps, rem?.challenge_swaps ?? 0), label: labels.challengeSwapsLabel })

  const row2: Chip[] = []
  if (content.cosmetics_total) {
    if (content.type_breakdown && Object.keys(content.type_breakdown).length > 0) {
      const armorT = armorOf(content)
      const cosmeticT = Math.max(0, content.cosmetics_total - armorT)
      const armorR = armorOf(rem)
      const cosmeticR = Math.max(0, (rem?.cosmetics_total ?? 0) - armorR)
      if (armorT > 0) row2.push({ key: 'armor', value: val(armorT, armorR), label: labels.armorLabel })
      if (cosmeticT > 0) row2.push({ key: 'cosmetic', value: val(cosmeticT, cosmeticR), label: labels.cosmeticsSplitLabel })
    } else {
      row2.push({ key: 'cosmetics', value: val(content.cosmetics_total, rem?.cosmetics_total ?? 0), label: labels.cosmeticsLabel })
    }
  }

  const rarities = RARITY_ORDER
    .map((tier) => ({ tier, count: content.rarity_breakdown?.[tier] ?? 0, r: rem?.rarity_breakdown?.[tier] ?? 0 }))
    .filter((e) => e.count > 0)

  if (row1.length === 0 && row2.length === 0 && rarities.length === 0) return null

  const chipRow = (chips: Chip[]) => chips.length === 0 ? null : (
    <div className="flex flex-wrap items-baseline gap-x-3 text-xs">
      {chips.map(({ key, value, label }) => (
        <span key={key}>
          <span className="font-semibold text-foreground tabular-nums">{value}</span>
          {' '}
          <span className="text-muted-foreground">{label}</span>
        </span>
      ))}
    </div>
  )

  return (
    <div className="mt-2 space-y-0.5">
      {/* Raretés EN PREMIER, puis items (cosmétiques), puis devises (cf. demande user). */}
      {rarities.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-2.5 text-xs">
          {rarities.map(({ tier, count, r }) => (
            <span key={tier} className="flex items-center gap-1">
              <span className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${rarityStyle(tier)?.segment ?? 'bg-muted-foreground/60'}`} />
              <span className="text-muted-foreground">{rarityLabel(tier, palmaresLocale)}</span>
              {' '}
              <span className="font-semibold text-foreground tabular-nums">{val(count, r)}</span>
            </span>
          ))}
        </div>
      )}
      {chipRow(row2)}
      {chipRow(row1)}
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="border-dashed">
      <CardContent className="pt-5">
        <p className="text-xs uppercase tracking-label-md text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
      </CardContent>
    </Card>
  )
}

function ProgressBar({ value }: { value?: number | null }) {
  const width = value == null ? 0 : Math.max(0, Math.min(100, value))
  return (
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div className="h-full rounded-full transition-[width]" style={{ width: `${width}%`, backgroundColor: width >= 100 ? tokenCssVar('success') : tokenCssVar('info') }} />
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
        'group relative block w-full min-h-[15rem] overflow-hidden rounded-xl border bg-card text-left shadow-sm transition-all hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        isSelected ? 'border-primary/70 ring-2 ring-primary/40' : 'border-border/70',
      ].join(' ')}
    >
      {background && (
        <>
          <div
            className="absolute inset-0 bg-cover bg-center transition-transform duration-500 group-hover:scale-105"
            style={{ backgroundImage: `url(${background})` }}
            aria-hidden="true"
          />
          {/* color-allow: gradient sombre fixe pour lisibilité du contenu sur image plein-cadre */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/40 to-black/50" aria-hidden="true" />
        </>
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

        {pass.content && (
          <PassContentSummary
            content={pass.content}
            remaining={pass.remaining_content ?? null}
            labels={contentLabels}
            locale={intlLocale}
            compact
          />
        )}

        <div className="mt-auto space-y-3">
          <div className="flex items-baseline justify-between">
            <p className="text-xl font-semibold text-foreground">{rankValue}</p>
            <p className="text-xl font-semibold text-foreground">
              {pass.completion_percent == null
                ? '—'
                : `${pass.completion_percent.toLocaleString(intlLocale, { maximumFractionDigits: 0 })} %`}
            </p>
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

  const allRewards = useMemo<RewardLightboxData[]>(
    () => allCards.map((card) => {
      const badges: RewardLightboxData['badges'] = []
      if (card.is_current) badges.push({ label: text.seasonPass.active, tone: 'current' })
      if (card.is_obtained) badges.push({ label: text.seasonPass.obtained, tone: 'obtained' })
      if (card.is_free) badges.push({ label: text.seasonPass.freeLabel, tone: 'free' })
      else badges.push({ label: text.seasonPass.premium, tone: 'premium' })
      return {
        title: card.title,
        rank: card.rank,
        imageUrl: card.image_url ?? null,
        description: card.description ?? null,
        quality: card.quality ?? null,
        itemType: card.item_type ?? null,
        badges,
      }
    }),
    [allCards, text.seasonPass.active, text.seasonPass.obtained, text.seasonPass.freeLabel, text.seasonPass.premium],
  )

  const handleOpenCard = useCallback((card: RewardCard) => {
    const idx = allCards.findIndex((c) => c.key === card.key)
    setSelectedIndex(idx >= 0 ? idx : null)
  }, [allCards])

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
    <div ref={showcaseRef} className="space-y-6">

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
          <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-black/5 to-transparent" /> {/* color-allow: gradient sombre fixe pour lisibilité du titre hero (overlay sur image map) */}
          <div className="absolute inset-x-0 bottom-0 p-4 sm:p-5">
            <div className="flex items-center gap-1.5">
              <h2 className="text-xl font-bold tracking-tight text-white sm:text-2xl"> {/* color-allow: blanc sur gradient sombre fixe (hero overlay) */}
                {pass.name}
              </h2>
              <DataFreshnessIndicator
                snapshotAt={pass.snapshot_at}
                buildLabel={text.seasonPass.freshnessLastSync}
                locale={text.intlLocale}
                className="text-white/50 hover:text-white/80"
              />
            </div>
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
                remaining={pass.remaining_content ?? null}
                labels={text.seasonPass.content}
                locale={text.intlLocale}
                palmaresLocale={locale}
              />
            )}
          </div>
        </div>
      ) : (
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex items-center gap-1.5">
            <h2 className="text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">{pass.name}</h2>
            <DataFreshnessIndicator
              snapshotAt={pass.snapshot_at}
              buildLabel={text.seasonPass.freshnessLastSync}
              locale={text.intlLocale}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            {pass.is_owned && <Badge variant="outline">{text.seasonPass.premium}</Badge>}
            {pass.is_active && <Badge variant="default">{text.seasonPass.active}</Badge>}
            <Badge variant={statusVariant(pass.status)}>{text.seasonPass.status[pass.status] ?? pass.status}</Badge>
          </div>
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

          <div className="flex justify-center">
            <div className="grid w-2/3 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-3xs text-muted-foreground">
              <span data-testid="season-pass-active-tier-progress-current" className="shrink-0 whitespace-nowrap">
                {progressLabels.current}
              </span>
              <div className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
                <div className="h-2 w-full">
                  <div
                    data-testid="season-pass-active-tier-progress-fill"
                    className="h-full rounded-full transition-all duration-300"
                    style={{ width: `${barPercent}%`, backgroundColor: barPercent >= 100 ? tokenCssVar('success') : tokenCssVar('info') }}
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

      {selectedIndex !== null && (
        <BattlePassRewardLightbox
          rewards={allRewards}
          startIndex={selectedIndex}
          onClose={() => setSelectedIndex(null)}
        />
      )}
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
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={text.seasonPass.unavailableTitle}
          description={error?.message ?? text.seasonPass.unavailableDescription}
          actionLabel={text.seasonPass.retry}
          onAction={() => refetch()}
        />
      </div>
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
    <div className="flex flex-col gap-6 p-6">
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
          <p className="text-xs uppercase tracking-label-2xl text-muted-foreground">{text.seasonPass.otherPassesTitle}</p>
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
    </div>
  )
}
