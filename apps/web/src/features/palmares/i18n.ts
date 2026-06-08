/**
 * palmares/i18n.ts — adapter mince entre `palmares.toml` (manifest TOML) et
 * le shape historique `PalmaresText` consommé par les pages Palmarès.
 *
 * Phase 3 P3.C : la source de vérité est désormais
 * `apps/web/src/lib/i18n/manifests/palmares.toml`. Les libellés FR/EN sont
 * générés par `scripts/build_i18n_manifests.mjs` dans
 * `lib/i18n/generated/palmares.ts`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { palmaresManifest, type PalmaresManifestKey } from '@/lib/i18n/generated/palmares'

export type PalmaresLocale = 'fr' | 'en'

export interface PalmaresText {
  intlLocale: string
  page: {
    title: string
    subtitle: string
  }
  relations: {
    retry: string
    unavailableTitle: string
    unavailableDescription: string
    overview: {
      distinctPlayers: string
      frequentAllies: string
      repeatRivals: string
      closedCircle: string
    }
    sections: {
      frequentAlliesTitle: string
      frequentAlliesDescription: string
      bestSynergiesTitle: string
      bestSynergiesDescription: string
      nemesesTitle: string
      nemesesDescription: string
      victimsTitle: string
      victimsDescription: string
      closedCircleTitle: string
      closedCircleDescription: string
    }
    labels: {
      with: string
      against: string
      winRate: string
      avgKDA: string
      lastSeen: string
    }
    emptyTitle: string
    emptyDescription: string
  }
  seasonPass: {
    retry: string
    unavailableTitle: string
    unavailableDescription: string
    activeCard: string
    completedCard: string
    inProgressCard: string
    remainingPassesCard: string
    cosmeticsUnlockedCard: string
    xpUnlockedCard: string
    lootCard: string
    challengesCard: string
    challengesTitle: string
    challengesUnavailable: string
    completed: string
    total: string
    xpAvailable: string
    nextExpiry: string
    noExpiry: string
    premium: string
    active: string
    cardRank: string
    cardProgress: string
    activePassTitle: string
    selectedPassTitle: string
    activeTierTitle: string
    activeTierProgress: string
    activeTierFallback: string
    freshnessLastSync: (date: string) => string
    obtained: string
    upcoming: string
    otherPassesTitle: string
    backToActive: string
    nowShowing: string
    freeLabel: string
    noDescription: string
    noPassesTitle: string
    noPassesDescription: string
    status: Record<string, string>
    content: {
      creditsLabel: string
      spartanPointsLabel: string
      xpBoostsLabel: string
      challengeSwapsLabel: string
      cosmeticsLabel: string
      armorLabel: string
      cosmeticsSplitLabel: string
      tiersLabel: string
      rarityTitle: string
      typeTitle: string
    }
  }
}

export function normalizePalmaresLocale(locale?: string | null): PalmaresLocale {
  return locale === 'en' ? 'en' : 'fr'
}

function t(loc: PalmaresLocale, key: PalmaresManifestKey): string {
  return formatMessage(palmaresManifest, key, loc)
}

export function getPalmaresText(locale?: string | null): PalmaresText {
  const loc = normalizePalmaresLocale(locale)
  return {
    intlLocale: t(loc, 'palmares.intl_locale'),
    page: {
      title: t(loc, 'palmares.page.title'),
      subtitle: t(loc, 'palmares.page.subtitle'),
    },
    relations: {
      retry: t(loc, 'palmares.errors.retry'),
      unavailableTitle: t(loc, 'palmares.relations.unavailable_title'),
      unavailableDescription: t(loc, 'palmares.relations.unavailable_description'),
      overview: {
        distinctPlayers: t(loc, 'palmares.relations.overview.distinct_players'),
        frequentAllies: t(loc, 'palmares.relations.overview.frequent_allies'),
        repeatRivals: t(loc, 'palmares.relations.overview.repeat_rivals'),
        closedCircle: t(loc, 'palmares.relations.overview.closed_circle'),
      },
      sections: {
        frequentAlliesTitle: t(loc, 'palmares.relations.section.frequent_allies_title'),
        frequentAlliesDescription: t(loc, 'palmares.relations.section.frequent_allies_description'),
        bestSynergiesTitle: t(loc, 'palmares.relations.section.best_synergies_title'),
        bestSynergiesDescription: t(loc, 'palmares.relations.section.best_synergies_description'),
        nemesesTitle: t(loc, 'palmares.relations.section.nemeses_title'),
        nemesesDescription: t(loc, 'palmares.relations.section.nemeses_description'),
        victimsTitle: t(loc, 'palmares.relations.section.victims_title'),
        victimsDescription: t(loc, 'palmares.relations.section.victims_description'),
        closedCircleTitle: t(loc, 'palmares.relations.section.closed_circle_title'),
        closedCircleDescription: t(loc, 'palmares.relations.section.closed_circle_description'),
      },
      labels: {
        with: t(loc, 'palmares.relations.label.with'),
        against: t(loc, 'palmares.relations.label.against'),
        winRate: t(loc, 'palmares.relations.label.win_rate'),
        avgKDA: t(loc, 'palmares.relations.label.avg_kda'),
        lastSeen: t(loc, 'palmares.relations.label.last_seen'),
      },
      emptyTitle: t(loc, 'palmares.relations.empty_title'),
      emptyDescription: t(loc, 'palmares.relations.empty_description'),
    },
    seasonPass: {
      retry: t(loc, 'palmares.errors.retry'),
      unavailableTitle: t(loc, 'palmares.season_pass.unavailable_title'),
      unavailableDescription: t(loc, 'palmares.season_pass.unavailable_description'),
      activeCard: t(loc, 'palmares.season_pass.active_card'),
      completedCard: t(loc, 'palmares.season_pass.completed_card'),
      inProgressCard: t(loc, 'palmares.season_pass.in_progress_card'),
      remainingPassesCard: t(loc, 'palmares.season_pass.remaining_passes_card'),
      cosmeticsUnlockedCard: t(loc, 'palmares.season_pass.cosmetics_unlocked_card'),
      xpUnlockedCard: t(loc, 'palmares.season_pass.xp_unlocked_card'),
      lootCard: t(loc, 'palmares.season_pass.loot_card'),
      challengesCard: t(loc, 'palmares.season_pass.challenges_card'),
      challengesTitle: t(loc, 'palmares.season_pass.challenges_title'),
      challengesUnavailable: t(loc, 'palmares.season_pass.challenges_unavailable'),
      completed: t(loc, 'palmares.season_pass.completed'),
      total: t(loc, 'palmares.season_pass.total'),
      xpAvailable: t(loc, 'palmares.season_pass.xp_available'),
      nextExpiry: t(loc, 'palmares.season_pass.next_expiry'),
      noExpiry: t(loc, 'palmares.season_pass.no_expiry'),
      premium: t(loc, 'palmares.season_pass.premium'),
      active: t(loc, 'palmares.season_pass.active'),
      cardRank: t(loc, 'palmares.season_pass.card_rank'),
      cardProgress: t(loc, 'palmares.season_pass.card_progress'),
      activePassTitle: t(loc, 'palmares.season_pass.active_pass_title'),
      selectedPassTitle: t(loc, 'palmares.season_pass.selected_pass_title'),
      activeTierTitle: t(loc, 'palmares.season_pass.active_tier_title'),
      activeTierProgress: t(loc, 'palmares.season_pass.active_tier_progress'),
      activeTierFallback: t(loc, 'palmares.season_pass.active_tier_fallback'),
      freshnessLastSync: (date: string) =>
        formatMessage(palmaresManifest, 'palmares.season_pass.freshness_last_sync', loc, { date }),
      obtained: t(loc, 'palmares.season_pass.obtained'),
      upcoming: t(loc, 'palmares.season_pass.upcoming'),
      otherPassesTitle: t(loc, 'palmares.season_pass.other_passes_title'),
      backToActive: t(loc, 'palmares.season_pass.back_to_active'),
      nowShowing: t(loc, 'palmares.season_pass.now_showing'),
      freeLabel: t(loc, 'palmares.season_pass.free_label'),
      noDescription: t(loc, 'palmares.season_pass.no_description'),
      noPassesTitle: t(loc, 'palmares.season_pass.no_passes_title'),
      noPassesDescription: t(loc, 'palmares.season_pass.no_passes_description'),
      status: {
        active: t(loc, 'palmares.season_pass.status.active'),
        in_progress: t(loc, 'palmares.season_pass.status.in_progress'),
        completed: t(loc, 'palmares.season_pass.status.completed'),
        not_started: t(loc, 'palmares.season_pass.status.not_started'),
      },
      content: {
        creditsLabel: t(loc, 'palmares.season_pass.content.credits_label'),
        spartanPointsLabel: t(loc, 'palmares.season_pass.content.spartan_points_label'),
        xpBoostsLabel: t(loc, 'palmares.season_pass.content.xp_boosts_label'),
        challengeSwapsLabel: t(loc, 'palmares.season_pass.content.challenge_swaps_label'),
        cosmeticsLabel: t(loc, 'palmares.season_pass.content.cosmetics_label'),
        armorLabel: t(loc, 'palmares.season_pass.content.armor_label'),
        cosmeticsSplitLabel: t(loc, 'palmares.season_pass.content.cosmetics_split_label'),
        tiersLabel: t(loc, 'palmares.season_pass.content.tiers_label'),
        rarityTitle: t(loc, 'palmares.season_pass.content.rarity_title'),
        typeTitle: t(loc, 'palmares.season_pass.content.type_title'),
      },
    },
  }
}
