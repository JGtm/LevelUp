/**
 * spartanIdentity.i18n.ts — adapter mince entre `home.toml` et le shape
 * historique `SpartanIdentityTextDict` consommé par HomePage.tsx.
 *
 * Phase 3 P3.C : source de vérité = `lib/i18n/manifests/home.toml` section
 * `home.spartan.*`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import type { Locale } from '@/lib/i18n/locale'

interface SpartanIdentityTextDict {
  labels: {
    careerRank: string
    highestCsr: string
    highestLusr: string
    currentProgress: string
    rankPrefix: string
    maxRank: string
    progressTowardsRank: (name: string) => string
    peakReachedOn: (date: string) => string
  }
  emptyPanel: {
    titleUnavailable: string
    titleNone: string
    descriptionUnavailable: string
    descriptionNone: string
  }
}

export function normalizeSpartanIdentityLocale(
  locale?: string | null,
): Locale {
  return locale === 'en' ? 'en' : 'fr'
}

function t(
  loc: Locale,
  key: HomeManifestKey,
  values?: Record<string, string | number>,
): string {
  return formatMessage(homeManifest, key, loc, values)
}

export function getSpartanIdentityText(locale?: string | null): SpartanIdentityTextDict {
  const loc = normalizeSpartanIdentityLocale(locale)
  return {
    labels: {
      careerRank: t(loc, 'home.spartan.career_rank'),
      highestCsr: t(loc, 'home.spartan.highest_csr'),
      highestLusr: t(loc, 'home.spartan.highest_lusr'),
      currentProgress: t(loc, 'home.spartan.current_progress'),
      rankPrefix: t(loc, 'home.spartan.rank_prefix'),
      maxRank: t(loc, 'home.spartan.max_rank'),
      progressTowardsRank: (name: string) =>
        t(loc, 'home.spartan.progress_towards_rank', { name }),
      peakReachedOn: (date: string) =>
        t(loc, 'home.spartan.peak_reached_on', { date }),
    },
    emptyPanel: {
      titleUnavailable: t(loc, 'home.spartan.empty.title_unavailable'),
      titleNone: t(loc, 'home.spartan.empty.title_none'),
      descriptionUnavailable: t(loc, 'home.spartan.empty.description_unavailable'),
      descriptionNone: t(loc, 'home.spartan.empty.description_none'),
    },
  }
}
