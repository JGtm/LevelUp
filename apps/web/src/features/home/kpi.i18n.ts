/**
 * kpi.i18n.ts — adapter mince entre `home.toml` (manifest TOML) et le shape
 * historique `KPITextDict` consommé par HomePage.tsx.
 *
 * Phase 3 P3.C : la source de vérité est désormais
 * `apps/web/src/lib/i18n/manifests/home.toml`. Les libellés FR/EN sont
 * générés par `scripts/build_i18n_manifests.mjs` dans
 * `lib/i18n/generated/home.ts`. Ce module reconstruit l'ancienne forme
 * imbriquée pour minimiser la churn côté HomePage.
 */
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import type { Locale } from '@/lib/i18n/locale'

interface KPITextDict {
  labels: {
    totalTime: string
    favoritePlaylist: string
    offDef: string
    favoriteWeapon: string
    matches: string
    kda: string
    winRate: string
    accuracy: string
  }
  units: {
    year: string
    month: string
    day: string
    hour: string
    minute: string
  }
  /** Libellés d'issue (tooltip de la barre composite du KPI taux de victoire). */
  outcomes: {
    wins: string
    draws: string
    losses: string
    dnfs: string
  }
  matches: (count: number) => string
  kills: (count: number) => string
}

export function normalizeKPILocale(locale?: string | null): Locale {
  return locale === 'en' ? 'en' : 'fr'
}

function t(loc: Locale, key: HomeManifestKey): string {
  return formatMessage(homeManifest, key, loc)
}

export function getKPIText(locale?: string | null): KPITextDict {
  const loc = normalizeKPILocale(locale)
  return {
    labels: {
      totalTime: t(loc, 'home.kpi.total_time'),
      favoritePlaylist: t(loc, 'home.kpi.favorite_playlist'),
      offDef: t(loc, 'home.kpi.off_def'),
      favoriteWeapon: t(loc, 'home.kpi.favorite_weapon'),
      matches: t(loc, 'home.kpi.matches_label'),
      kda: t(loc, 'home.kpi.kda_label'),
      winRate: t(loc, 'home.kpi.win_rate_label'),
      accuracy: t(loc, 'home.kpi.accuracy_label'),
    },
    units: {
      year: t(loc, 'home.kpi.unit_year'),
      month: t(loc, 'home.kpi.unit_month'),
      day: t(loc, 'home.kpi.unit_day'),
      hour: t(loc, 'home.kpi.unit_hour'),
      minute: t(loc, 'home.kpi.unit_minute'),
    },
    outcomes: {
      wins: t(loc, 'home.kpi.outcome_wins'),
      draws: t(loc, 'home.kpi.outcome_draws'),
      losses: t(loc, 'home.kpi.outcome_losses'),
      dnfs: t(loc, 'home.kpi.outcome_dnfs'),
    },
    matches: () => t(loc, 'home.kpi.matches_word'),
    kills: () => t(loc, 'home.kpi.kills_word'),
  }
}
