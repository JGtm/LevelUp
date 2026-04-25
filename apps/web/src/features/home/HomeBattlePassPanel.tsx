import { useCallback, useEffect, useRef, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { buildCompositeProgressEdgeLabels, clampCompositeProgress } from '@/components/ui/composite-progress-bar'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SeasonPassItemSummary, SeasonPassPageResponse, SeasonPassTierSummary, SeasonPassTrackSummary } from '@/lib/api/types'
import { BattlePassRewardLightbox, type RewardLightboxData } from '@/features/palmares/BattlePassRewardLightbox'
import { normalizeRarity, rarityStyle } from '@/features/palmares/rarity'

function ChevronLeftIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m15 18-6-6 6-6" />
    </svg>
  )
}

function ChevronRightIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

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

function pickFeaturedPass(passes: SeasonPassTrackSummary[]) {
  return passes.find((pass) => pass.is_active)
    ?? passes.find((pass) => pass.status === 'in_progress')
    ?? passes[0]
    ?? null
}

/** Une reward individuelle dans un groupe de palier. */
interface RewardCard {
  key: string
  rank: number
  title: string
  image_url?: string | null
  description?: string | null
  quality?: string | null
  item_type?: string | null
  is_obtained: boolean
  is_current: boolean
  is_free: boolean
}

/** Groupe de rewards appartenant au même palier. */
interface TierGroup {
  rank: number
  is_current: boolean
  is_obtained: boolean
  cards: RewardCard[]
}

/**
 * Construit les groupes de paliers à afficher dans le carousel.
 * Chaque groupe contient toutes les rewards (paid + free) du palier.
 */
function buildTierGroups(tiers: SeasonPassTierSummary[]): TierGroup[] {
  const groups: TierGroup[] = []
  for (const tier of tiers) {
    const freeItems: SeasonPassItemSummary[] = tier.free_rewards ?? []
    const cards: RewardCard[] = []
    const base = { rank: tier.rank, is_obtained: tier.is_obtained, is_current: tier.is_current }

    if (tier.is_premium) {
      cards.push({
        ...base,
        key: `${tier.rank}-paid`,
        title: tier.title,
        image_url: tier.image_url,
        description: tier.description ?? null,
        quality: tier.quality ?? null,
        item_type: tier.item_type ?? null,
        is_free: false,
      })
    }

    for (let i = 0; i < freeItems.length; i++) {
      const r = freeItems[i]
      cards.push({
        ...base,
        key: `${tier.rank}-free-${i}`,
        title: r.title,
        image_url: r.image_url,
        description: r.description ?? null,
        quality: r.quality ?? null,
        item_type: r.item_type ?? null,
        is_free: true,
      })
    }

    if (cards.length === 0) {
      cards.push({
        ...base,
        key: `${tier.rank}-empty`,
        title: tier.title || `Palier ${tier.rank}`,
        image_url: tier.image_url,
        description: tier.description ?? null,
        quality: tier.quality ?? null,
        item_type: tier.item_type ?? null,
        is_free: false,
      })
    }

    groups.push({ rank: tier.rank, is_current: tier.is_current, is_obtained: tier.is_obtained, cards })
  }
  return groups
}

function BattlePassRewardCard({ card, onOpen }: { card: RewardCard; onOpen: (card: RewardCard) => void }) {
  const rarityTier = normalizeRarity(card.quality)
  const rarityStyles = rarityStyle(rarityTier)
  const imageBackground = rarityStyles
    ? `${rarityStyles.bg} ${rarityStyles.glow}`
    : 'bg-transparent'
  return (
    <button
      type="button"
      onClick={() => onOpen(card)}
      aria-label={`Voir le détail de ${card.title}`}
      data-rarity={rarityTier ?? 'none'}
      className="group block w-14 sm:w-16 xl:w-[4.5rem] space-y-1 text-left transition-transform duration-150 hover:scale-105 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-400/70 focus-visible:rounded-lg"
    >
      <div className={`relative aspect-[4/5] w-full overflow-hidden rounded-lg ${imageBackground}`}>
        {card.is_obtained && (
          <div className="absolute right-1 top-1 z-10 flex h-4 min-w-4 items-center justify-center rounded-full bg-emerald-500 px-1 text-[8px] font-semibold text-white shadow-sm">
            ✓
          </div>
        )}
        {card.is_free && (
          <div className="absolute bottom-1 left-1 z-10 rounded bg-amber-500/90 px-[3px] py-[1px] text-[6px] font-bold uppercase tracking-wide text-white">
            gratuit
          </div>
        )}
        {card.image_url ? (
          <img src={card.image_url} alt={card.title} className="h-full w-full object-cover" loading="lazy" />
        ) : (
          <div className="flex h-full w-full items-center justify-center bg-[radial-gradient(circle_at_top,rgba(14,165,233,0.22),transparent_55%),linear-gradient(180deg,rgba(15,23,42,0.92),rgba(30,41,59,0.84))] text-center text-white">
            <p className="text-3xl font-semibold">{card.rank}</p>
          </div>
        )}
      </div>
      <p className="line-clamp-2 text-[8px] font-medium leading-tight text-foreground/80">{card.title}</p>
    </button>
  )
}

function BattlePassTierGroup({
  group,
  anchorRef,
  onOpenCard,
}: {
  group: TierGroup
  anchorRef?: (node: HTMLDivElement | null) => void
  onOpenCard: (card: RewardCard) => void
}) {
  const borderClasses = [
    'flex gap-1.5 rounded-xl border p-1.5',
    group.is_current ? 'border-sky-400/60 shadow-[0_0_12px_-4px_rgba(56,189,248,0.5)]' : 'border-white/15',
    group.is_obtained && !group.is_current ? 'opacity-60 grayscale-[0.82]' : '',
  ].filter(Boolean).join(' ')

  return (
    <div
      ref={anchorRef}
      data-testid="home-battle-pass-tier-card"
      data-current={group.is_current ? 'true' : 'false'}
      data-obtained={group.is_obtained ? 'true' : 'false'}
      className={[
        'snap-center shrink-0 space-y-1 transition-transform duration-200',
        group.is_current ? 'z-10 relative scale-[1.10]' : '',
      ].filter(Boolean).join(' ')}
    >
      <p className="px-0.5 text-[8px] font-semibold text-muted-foreground">#{group.rank}</p>
      <div className={borderClasses}>
        {group.cards.map((card) => (
          <BattlePassRewardCard key={card.key} card={card} onOpen={onOpenCard} />
        ))}
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
  const tiersRailRef = useRef<HTMLDivElement | null>(null)
  const featuredPass = pickFeaturedPass(data?.passes ?? [])
  const activeTierRef = useRef<HTMLDivElement | null>(null)

  const [canLeft, setCanLeft] = useState(false)
  const [canRight, setCanRight] = useState(false)
  const [activeReward, setActiveReward] = useState<RewardLightboxData | null>(null)

  const handleOpenCard = useCallback((card: RewardCard) => {
    const badges: RewardLightboxData['badges'] = []
    if (card.is_current) badges.push({ label: 'Palier actuel', tone: 'current' })
    if (card.is_obtained) badges.push({ label: 'Obtenu', tone: 'obtained' })
    if (card.is_free) badges.push({ label: 'Gratuit', tone: 'free' })
    else badges.push({ label: 'Premium', tone: 'premium' })
    setActiveReward({
      title: card.title,
      rank: card.rank,
      imageUrl: card.image_url ?? null,
      description: card.description ?? null,
      quality: card.quality ?? null,
      itemType: card.item_type ?? null,
      badges,
    })
  }, [])

  const updateScrollState = useCallback(() => {
    const el = tiersRailRef.current
    if (!el) return
    setCanLeft(el.scrollLeft > 4)
    setCanRight(el.scrollLeft + el.offsetWidth < el.scrollWidth - 4)
  }, [])

  useEffect(() => {
    centerTierInRail(tiersRailRef.current, activeTierRef.current)
  }, [featuredPass?.active_tier_rank, featuredPass?.tiers])

  // Met à jour l'état des flèches après le centrage et au resize
  useEffect(() => {
    updateScrollState()
    const el = tiersRailRef.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(updateScrollState)
    ro.observe(el)
    return () => ro.disconnect()
  }, [updateScrollState, featuredPass?.tiers])

  function scrollRail(direction: 'left' | 'right') {
    const el = tiersRailRef.current
    if (!el) return
    const step = el.offsetWidth * 0.6
    el.scrollBy({ left: direction === 'right' ? step : -step, behavior: 'smooth' })
  }

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

  const tierProgress = clampCompositeProgress(featuredPass.active_tier_progress_percent)
  const tierProgressLabels = buildCompositeProgressEdgeLabels({
    partialProgress: featuredPass.partial_progress,
    xpPerRank: featuredPass.xp_per_rank,
    progressPercent: tierProgress,
    locale: 'fr-FR',
  })
  const tierGroups = featuredPass.tiers && featuredPass.tiers.length > 0
    ? buildTierGroups(featuredPass.tiers)
    : []

  return (
    <Card className="relative flex min-h-[14rem] flex-col overflow-hidden border-border/70 bg-card/95 shadow-sm">
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background/96 to-background/85" aria-hidden="true" />

      <CardHeader className="relative space-y-4 pb-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">Pass de combat</CardTitle>
            <h3 className="mt-3 text-xl font-semibold tracking-tight text-foreground sm:text-2xl">
              {featuredPass.name}
            </h3>
          </div>

          <div className="flex flex-wrap gap-2">
            {featuredPass.is_owned && <Badge variant="outline">Premium</Badge>}
            {featuredPass.is_active && <Badge variant="default">Actif</Badge>}
          </div>
        </div>

        <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
          {featuredPass.description ?? 'Aucune description disponible pour ce pass.'}
        </p>
      </CardHeader>

      <CardContent className="relative space-y-6">
        <div className="overflow-hidden rounded-xl border border-white/15 bg-slate-950/80 shadow-[0_24px_72px_-44px_rgba(15,23,42,0.92)]">
          {(featuredPass.background_image_url ?? featuredPass.image_url) ? (
            <img
              src={featuredPass.background_image_url ?? featuredPass.image_url!}
              alt={`Illustration de ${featuredPass.name}`}
              data-testid="home-battle-pass-image"
              className="aspect-[986/248] w-full object-cover"
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

        {tierGroups.length > 0 ? (
          <div className="space-y-5">

            <div className="relative">
              {/* Flèche gauche */}
              <button
                type="button"
                onClick={() => scrollRail('left')}
                disabled={!canLeft}
                aria-label="Paliers précédents"
                className="absolute left-0 inset-y-0 z-20 flex items-center justify-center w-8 rounded-l-lg border-y border-l border-border/60 bg-background/90 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-20"
              >
                <ChevronLeftIcon />
              </button>

              <div
                ref={tiersRailRef}
                onScroll={updateScrollState}
                className="flex gap-4 overflow-x-auto px-10 pb-6 pt-3 [scrollbar-width:none] snap-x snap-mandatory [&::-webkit-scrollbar]:hidden"
              >
                {tierGroups.map((group) => (
                  <BattlePassTierGroup
                    key={group.rank}
                    group={group}
                    anchorRef={group.is_current ? (node) => { activeTierRef.current = node } : undefined}
                    onOpenCard={handleOpenCard}
                  />
                ))}
              </div>

              {/* Flèche droite */}
              <button
                type="button"
                onClick={() => scrollRail('right')}
                disabled={!canRight}
                aria-label="Paliers suivants"
                className="absolute right-0 inset-y-0 z-20 flex items-center justify-center w-8 rounded-r-lg border-y border-r border-border/60 bg-background/90 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-20"
              >
                <ChevronRightIcon />
              </button>
            </div>

            <div className="flex justify-center">
              <div
                data-testid="home-battle-pass-active-tier-progress-row"
                className="grid w-2/3 grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-[11px] text-muted-foreground"
              >
                <span data-testid="home-battle-pass-active-tier-progress-current" className="shrink-0 whitespace-nowrap">
                  {tierProgressLabels.current}
                </span>
                <div data-testid="home-battle-pass-active-tier-progress-track" className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
                  <div className="h-2 w-full">
                    <div
                      data-testid="home-battle-pass-active-tier-progress-fill"
                      className="h-full rounded-full bg-sky-500 transition-all duration-300"
                      style={{ width: `${clampCompositeProgress(tierProgress)}%` }}
                    />
                  </div>
                </div>
                <span data-testid="home-battle-pass-active-tier-progress-target" className="shrink-0 whitespace-nowrap text-right">
                  {tierProgressLabels.target}
                </span>
              </div>
            </div>
          </div>
        ) : (
          <div className="flex min-h-[8.5rem] items-center justify-center">
            <p className="text-sm text-muted-foreground">Aucun palier disponible pour ce pass.</p>
          </div>
        )}
      </CardContent>
      <BattlePassRewardLightbox reward={activeReward} onClose={() => setActiveReward(null)} />
    </Card>
  )
}
