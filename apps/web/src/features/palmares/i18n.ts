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
    emptyTitle: string
    emptyDescription: string
    hero: {
      topAllyTitle: string
      topAllyEmpty: string
      topNemesisTitle: string
      topNemesisEmpty: string
      coreTitle: string
      coreUnit: string
      matchesPlayed: (count: string) => string
    }
    chips: {
      all: string
      core: string
      allies: string
      rivals: string
      recent: string
    }
    filters: {
      experience: string
      experienceAll: string
      experienceRanked: string
      experienceUnranked: string
      playlists: string
      modes: string
      view: string
      viewAll: string
      viewSolo: string
      viewSquad: string
      analyser: string
      reset: string
    }
    table: {
      player: string
      link: string
      encounters: string
      winRateAlly: string
      winRateEnemy: string
      fragsDeaths: string
      ratio: string
      lastSeen: string
      ratioTooltip: string
    }
    category: {
      ally: string
      enemy: string
      mixed: string
    }
    relative: {
      today: string
      yesterday: string
      daysAgo: (count: number) => string
      weeksAgo: (count: number) => string
      monthsAgo: (count: number) => string
      yearsAgo: (count: number) => string
    }
    tooltip: {
      matchesAlly: (count: string) => string
      matchesEnemy: (count: string) => string
      fragsDealt: (count: string) => string
      deathsSuffered: (count: string) => string
    }
    core: {
      sectionTitle: string
      sectionDescription: string
      empty: string
      together: (count: string) => string
    }
    filterEmptyTitle: string
    filterEmptyDescription: string
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
      emptyTitle: t(loc, 'palmares.relations.empty_title'),
      emptyDescription: t(loc, 'palmares.relations.empty_description'),
      hero: {
        topAllyTitle: t(loc, 'palmares.relations.hero.top_ally_title'),
        topAllyEmpty: t(loc, 'palmares.relations.hero.top_ally_empty'),
        topNemesisTitle: t(loc, 'palmares.relations.hero.top_nemesis_title'),
        topNemesisEmpty: t(loc, 'palmares.relations.hero.top_nemesis_empty'),
        coreTitle: t(loc, 'palmares.relations.hero.core_title'),
        coreUnit: t(loc, 'palmares.relations.hero.core_unit'),
        matchesPlayed: (count: string) =>
          formatMessage(palmaresManifest, 'palmares.relations.hero.matches_played', loc, { count }),
      },
      chips: {
        all: t(loc, 'palmares.relations.chip.all'),
        core: t(loc, 'palmares.relations.chip.core'),
        allies: t(loc, 'palmares.relations.chip.allies'),
        rivals: t(loc, 'palmares.relations.chip.rivals'),
        recent: t(loc, 'palmares.relations.chip.recent'),
      },
      filters: {
        experience: t(loc, 'palmares.relations.filters.experience'),
        experienceAll: t(loc, 'palmares.relations.filters.experience_all'),
        experienceRanked: t(loc, 'palmares.relations.filters.experience_ranked'),
        experienceUnranked: t(loc, 'palmares.relations.filters.experience_unranked'),
        playlists: t(loc, 'palmares.relations.filters.playlists'),
        modes: t(loc, 'palmares.relations.filters.modes'),
        view: t(loc, 'palmares.relations.filters.view'),
        viewAll: t(loc, 'palmares.relations.filters.view_all'),
        viewSolo: t(loc, 'palmares.relations.filters.view_solo'),
        viewSquad: t(loc, 'palmares.relations.filters.view_squad'),
        analyser: t(loc, 'palmares.relations.filters.analyser'),
        reset: t(loc, 'palmares.relations.filters.reset'),
      },
      table: {
        player: t(loc, 'palmares.relations.table.player'),
        link: t(loc, 'palmares.relations.table.link'),
        encounters: t(loc, 'palmares.relations.table.encounters'),
        winRateAlly: t(loc, 'palmares.relations.table.win_rate_ally'),
        winRateEnemy: t(loc, 'palmares.relations.table.win_rate_enemy'),
        fragsDeaths: t(loc, 'palmares.relations.table.frags_deaths'),
        ratio: t(loc, 'palmares.relations.table.ratio'),
        lastSeen: t(loc, 'palmares.relations.table.last_seen'),
        ratioTooltip: t(loc, 'palmares.relations.table.ratio_tooltip'),
      },
      category: {
        ally: t(loc, 'palmares.relations.category.ally'),
        enemy: t(loc, 'palmares.relations.category.enemy'),
        mixed: t(loc, 'palmares.relations.category.mixed'),
      },
      relative: {
        today: t(loc, 'palmares.relations.relative.today'),
        yesterday: t(loc, 'palmares.relations.relative.yesterday'),
        daysAgo: (count: number) =>
          formatMessage(palmaresManifest, 'palmares.relations.relative.days_ago', loc, { count }),
        weeksAgo: (count: number) =>
          formatMessage(palmaresManifest, 'palmares.relations.relative.weeks_ago', loc, { count }),
        monthsAgo: (count: number) =>
          formatMessage(palmaresManifest, 'palmares.relations.relative.months_ago', loc, { count }),
        yearsAgo: (count: number) =>
          formatMessage(palmaresManifest, 'palmares.relations.relative.years_ago', loc, { count }),
      },
      tooltip: {
        matchesAlly: (count: string) =>
          formatMessage(palmaresManifest, 'palmares.relations.tooltip.matches_ally', loc, { count }),
        matchesEnemy: (count: string) =>
          formatMessage(palmaresManifest, 'palmares.relations.tooltip.matches_enemy', loc, { count }),
        fragsDealt: (count: string) =>
          formatMessage(palmaresManifest, 'palmares.relations.tooltip.frags_dealt', loc, { count }),
        deathsSuffered: (count: string) =>
          formatMessage(palmaresManifest, 'palmares.relations.tooltip.deaths_suffered', loc, { count }),
      },
      core: {
        sectionTitle: t(loc, 'palmares.relations.core.section_title'),
        sectionDescription: t(loc, 'palmares.relations.core.section_description'),
        empty: t(loc, 'palmares.relations.core.empty'),
        together: (count: string) =>
          formatMessage(palmaresManifest, 'palmares.relations.core.together', loc, { count }),
      },
      filterEmptyTitle: t(loc, 'palmares.relations.filter_empty_title'),
      filterEmptyDescription: t(loc, 'palmares.relations.filter_empty_description'),
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
