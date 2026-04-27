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
export type PalmaresTab = 'leaderboard' | 'relations' | 'season-pass' | 'compare'

export interface PalmaresText {
  intlLocale: string
  page: {
    title: string
    subtitle: string
  }
  tabs: Record<PalmaresTab, string>
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
    activeTierTitle: string
    activeTierProgress: string
    activeTierFallback: string
    obtained: string
    upcoming: string
    otherPassesTitle: string
    noDescription: string
    noPassesTitle: string
    noPassesDescription: string
    status: Record<string, string>
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
    tabs: {
      leaderboard: t(loc, 'palmares.tabs.leaderboard'),
      relations: t(loc, 'palmares.tabs.relations'),
      'season-pass': t(loc, 'palmares.tabs.season_pass'),
      compare: t(loc, 'palmares.tabs.compare'),
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
      activeTierTitle: t(loc, 'palmares.season_pass.active_tier_title'),
      activeTierProgress: t(loc, 'palmares.season_pass.active_tier_progress'),
      activeTierFallback: t(loc, 'palmares.season_pass.active_tier_fallback'),
      obtained: t(loc, 'palmares.season_pass.obtained'),
      upcoming: t(loc, 'palmares.season_pass.upcoming'),
      otherPassesTitle: t(loc, 'palmares.season_pass.other_passes_title'),
      noDescription: t(loc, 'palmares.season_pass.no_description'),
      noPassesTitle: t(loc, 'palmares.season_pass.no_passes_title'),
      noPassesDescription: t(loc, 'palmares.season_pass.no_passes_description'),
      status: {
        active: t(loc, 'palmares.season_pass.status.active'),
        in_progress: t(loc, 'palmares.season_pass.status.in_progress'),
        completed: t(loc, 'palmares.season_pass.status.completed'),
        not_started: t(loc, 'palmares.season_pass.status.not_started'),
      },
    },
  }
}
