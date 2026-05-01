/**
 * BattlePassRewardCarousel — carrousel partagé Home + Pass saisonnier.
 *
 * Pour chaque palier, regroupe toutes les rewards (paid + free) dans un même
 * "tier group" avec bordure, scale-up sur le palier actif, et navigation par
 * flèches. Les rewards individuelles ouvrent un `BattlePassRewardLightbox`
 * via le callback `onOpenCard` que le parent câble.
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import type { SeasonPassItemSummary, SeasonPassTierSummary } from '@/lib/api/types'

import { normalizeRarity, rarityStyle } from './rarity'

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
  if (!container || !item) return
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

/** Une reward individuelle dans un groupe de palier. */
export interface RewardCard {
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
export interface TierGroup {
  rank: number
  is_current: boolean
  is_obtained: boolean
  cards: RewardCard[]
}

/**
 * Construit les groupes de paliers à afficher dans le carrousel.
 * Chaque groupe contient toutes les rewards (paid + free) du palier.
 */
export function buildTierGroups(tiers: SeasonPassTierSummary[]): TierGroup[] {
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

function BattlePassRewardCard({ card, onOpen, freeLabel }: { card: RewardCard; onOpen: (card: RewardCard) => void; freeLabel: string }) {
  const rarityTier = normalizeRarity(card.quality)
  const rarityStyles = rarityStyle(rarityTier)
  const imageBackground = rarityStyles ? `${rarityStyles.bg} ${rarityStyles.glow}` : 'bg-transparent'

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
            {freeLabel}
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

function BattlePassTierGroupView({
  group,
  anchorRef,
  onOpenCard,
  freeLabel,
}: {
  group: TierGroup
  anchorRef?: (node: HTMLDivElement | null) => void
  onOpenCard: (card: RewardCard) => void
  freeLabel: string
}) {
  const borderClasses = [
    'flex gap-1.5 rounded-xl border p-1.5',
    group.is_current ? 'border-sky-400/60 shadow-[0_0_12px_-4px_rgba(56,189,248,0.5)]' : 'border-white/15',
    group.is_obtained && !group.is_current ? 'opacity-60 grayscale-[0.82]' : '',
  ].filter(Boolean).join(' ')

  return (
    <div
      ref={anchorRef}
      data-testid="battle-pass-tier-card"
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
          <BattlePassRewardCard key={card.key} card={card} onOpen={onOpenCard} freeLabel={freeLabel} />
        ))}
      </div>
    </div>
  )
}

export function BattlePassRewardCarousel({
  tiers,
  activeTierRank,
  onOpenCard,
  freeLabel = 'gratuit',
  prevAriaLabel = 'Paliers précédents',
  nextAriaLabel = 'Paliers suivants',
}: {
  tiers: SeasonPassTierSummary[]
  activeTierRank?: number | null
  onOpenCard: (card: RewardCard) => void
  freeLabel?: string
  prevAriaLabel?: string
  nextAriaLabel?: string
}) {
  const tiersRailRef = useRef<HTMLDivElement | null>(null)
  const activeTierRef = useRef<HTMLDivElement | null>(null)
  const [canLeft, setCanLeft] = useState(false)
  const [canRight, setCanRight] = useState(false)

  const updateScrollState = useCallback(() => {
    const el = tiersRailRef.current
    if (!el) return
    setCanLeft(el.scrollLeft > 4)
    setCanRight(el.scrollLeft + el.offsetWidth < el.scrollWidth - 4)
  }, [])

  useEffect(() => {
    centerTierInRail(tiersRailRef.current, activeTierRef.current)
  }, [activeTierRank, tiers])

  useEffect(() => {
    updateScrollState()
    const el = tiersRailRef.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(updateScrollState)
    ro.observe(el)
    return () => ro.disconnect()
  }, [updateScrollState, tiers])

  function scrollRail(direction: 'left' | 'right') {
    const el = tiersRailRef.current
    if (!el) return
    const step = el.offsetWidth * 0.6
    el.scrollBy({ left: direction === 'right' ? step : -step, behavior: 'smooth' })
  }

  if (tiers.length === 0) return null
  const groups = buildTierGroups(tiers)

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => scrollRail('left')}
        disabled={!canLeft}
        aria-label={prevAriaLabel}
        className="absolute left-0 inset-y-0 z-20 flex items-center justify-center w-8 rounded-l-lg border-y border-l border-border/60 bg-background/90 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-20"
      >
        <ChevronLeftIcon />
      </button>

      <div
        ref={tiersRailRef}
        onScroll={updateScrollState}
        onKeyDown={(e) => {
          if (e.key === 'ArrowLeft') { e.preventDefault(); scrollRail('left') }
          if (e.key === 'ArrowRight') { e.preventDefault(); scrollRail('right') }
        }}
        className="flex gap-4 overflow-x-auto px-10 pb-6 pt-3 [scrollbar-width:none] snap-x snap-mandatory [&::-webkit-scrollbar]:hidden"
      >
        {groups.map((group) => (
          <BattlePassTierGroupView
            key={group.rank}
            group={group}
            anchorRef={group.is_current ? (node) => { activeTierRef.current = node } : undefined}
            onOpenCard={onOpenCard}
            freeLabel={freeLabel}
          />
        ))}
      </div>

      <button
        type="button"
        onClick={() => scrollRail('right')}
        disabled={!canRight}
        aria-label={nextAriaLabel}
        className="absolute right-0 inset-y-0 z-20 flex items-center justify-center w-8 rounded-r-lg border-y border-r border-border/60 bg-background/90 backdrop-blur-sm transition-all duration-150 hover:bg-muted disabled:cursor-default disabled:opacity-20"
      >
        <ChevronRightIcon />
      </button>
    </div>
  )
}
