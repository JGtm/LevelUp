/**
 * Modèle des groupes de paliers du carrousel Battle Pass — extrait de
 * BattlePassRewardCarousel.tsx pour que le module de composant n'exporte que des
 * composants (react-refresh/only-export-components).
 */
import type { SeasonPassItemSummary, SeasonPassTierSummary } from '@/lib/api/types'

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
