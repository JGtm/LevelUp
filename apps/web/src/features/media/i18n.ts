/**
 * media/i18n.ts — adapter mince entre `media.toml` (manifest TOML) et le
 * shape historique `MediaText` consommé par MediaPage / MediaToolbar.
 *
 * Phase 3 P3.A : la source de vérité est désormais
 * `apps/web/src/lib/i18n/manifests/media.toml`. Les libellés FR/EN sont
 * générés par `scripts/build_i18n_manifests.mjs` dans
 * `lib/i18n/generated/media.ts`. Ce module reconstruit l'ancienne forme
 * imbriquée pour minimiser la churn côté consommateurs.
 */
import { formatMessage } from '@/lib/i18n/format'
import { mediaManifest, type MediaManifestKey } from '@/lib/i18n/generated/media'
import type { Locale } from '@/lib/i18n/locale'

export interface MediaText {
  title: string
  emptyState: string
  errorPrefix: string
  /** Message affiché quand le serveur refuse un like (garde anti-silence). */
  likeError: string
  previousPage: string
  nextPage: string
  pageLabel: (page: number, totalPages: number) => string
  navContextLabel: string
  thumbnail: {
    noMatchAssociated: string
    unknownMap: string
  }
  like: {
    /** aria-label du bouton coeur selon l'état courant. */
    ariaLike: string
    ariaUnlike: string
    /**
     * Infobulle du compteur : « Aimé par Alice, Bob et 3 autres ».
     * Retourne `null` quand personne n'a aimé le média (pas d'infobulle).
     */
    likersTooltip: (likers: string[] | undefined, totalLikers: number | undefined) => string | null
  }
  groupSection: {
    sessionOfPrefix: string
    likedSection: string
    notLikedSection: string
    unknownOwner: string
    unknownMap: string
    unknownMode: string
    unknownSession: string
  }
  toolbar: {
    filterLabel: string
    sortLabel: string
    kindAriaLabel: string
    playlistAriaLabel: string
    mapAriaLabel: string
    modeAriaLabel: string
    sortAriaLabel: string
    groupAriaLabel: string
    likedOnlyAriaLabel: string
    unassignedOnlyAriaLabel: string
    authorsAriaLabel: string
    allAuthorsToggle: string
    noAuthors: string
    allTypes: string
    screenshots: string
    clips: string
    allAuthors: string
    mine: string
    allPlaylists: string
    allMaps: string
    allModes: string
    allInCategory: (categoryLabel: string) => string
    modeCategories: {
      Assassin: string
      Fiesta: string
      'Super Fiesta': string
      'Husky Raid': string
      BTB: string
      Ranked: string
      Firefight: string
      Other: string
    }
    dateDesc: string
    dateAsc: string
    mapAsc: string
    modeAsc: string
    noGrouping: string
    byOwner: string
    byMap: string
    byMode: string
    bySession: string
  }
}

export function normalizeMediaLocale(locale?: string | null): Locale {
  return locale === 'en' ? 'en' : 'fr'
}

function t(locale: Locale, key: MediaManifestKey, values?: Record<string, string | number>): string {
  return formatMessage(mediaManifest, key, locale, values)
}

/**
 * Énumération des personnes ayant aimé un média, dans la langue courante :
 * « Alice », « Alice et Bob », « Alice, Bob et 3 autres ». Quand les noms ne
 * sont pas connus mais que le compte l'est : « 3 personnes ». Chaîne vide quand
 * personne n'a aimé — l'appelant n'affiche alors pas d'infobulle.
 */
function formatLikers(loc: Locale, names: string[], totalLikers: number): string {
  if (totalLikers <= 0) {
    return ''
  }
  if (names.length === 0) {
    return totalLikers === 1
      ? t(loc, 'media.like.anonymous_one')
      : t(loc, 'media.like.anonymous_many', { count: totalLikers })
  }
  const and = t(loc, 'media.like.conjunction')
  const rest = totalLikers - names.length
  if (rest > 0) {
    const others = rest === 1
      ? t(loc, 'media.like.others_one')
      : t(loc, 'media.like.others_many', { count: rest })
    return `${names.join(', ')} ${and} ${others}`
  }
  if (names.length === 1) {
    return names[0]
  }
  return `${names.slice(0, -1).join(', ')} ${and} ${names[names.length - 1]}`
}

export function getMediaText(locale?: string | null): MediaText {
  const loc = normalizeMediaLocale(locale)
  return {
    title: t(loc, 'media.page.title'),
    emptyState: t(loc, 'media.page.empty_state'),
    errorPrefix: t(loc, 'media.page.error_prefix'),
    likeError: t(loc, 'media.like.error'),
    previousPage: t(loc, 'media.pagination.previous'),
    nextPage: t(loc, 'media.pagination.next'),
    pageLabel: (page, totalPages) =>
      t(loc, 'media.pagination.page_label', { page, totalPages }),
    navContextLabel: t(loc, 'media.nav_context_label'),
    thumbnail: {
      noMatchAssociated: t(loc, 'media.thumbnail.no_match_associated'),
      unknownMap: t(loc, 'media.group.unknown_map'),
    },
    like: {
      ariaLike: t(loc, 'media.like.aria_like'),
      ariaUnlike: t(loc, 'media.like.aria_unlike'),
      likersTooltip: (likers, totalLikers) => {
        const names = formatLikers(loc, likers ?? [], totalLikers ?? 0)
        return names ? t(loc, 'media.like.tooltip', { names }) : null
      },
    },
    groupSection: {
      sessionOfPrefix: t(loc, 'media.group.session_of_prefix'),
      likedSection: t(loc, 'media.group.liked_section'),
      notLikedSection: t(loc, 'media.group.not_liked_section'),
      unknownOwner: t(loc, 'media.group.unknown_owner'),
      unknownMap: t(loc, 'media.group.unknown_map'),
      unknownMode: t(loc, 'media.group.unknown_mode'),
      unknownSession: t(loc, 'media.group.unknown_session'),
    },
    toolbar: {
      filterLabel: t(loc, 'media.toolbar.filter_label'),
      sortLabel: t(loc, 'media.toolbar.sort_label'),
      kindAriaLabel: t(loc, 'media.toolbar.kind_aria'),
      playlistAriaLabel: t(loc, 'media.toolbar.playlist_aria'),
      mapAriaLabel: t(loc, 'media.toolbar.map_aria'),
      modeAriaLabel: t(loc, 'media.toolbar.mode_aria'),
      sortAriaLabel: t(loc, 'media.toolbar.sort_aria'),
      groupAriaLabel: t(loc, 'media.toolbar.group_aria'),
      likedOnlyAriaLabel: t(loc, 'media.toolbar.liked_only_aria'),
      unassignedOnlyAriaLabel: t(loc, 'media.toolbar.unassigned_only_aria'),
      authorsAriaLabel: t(loc, 'media.toolbar.authors_aria'),
      allAuthorsToggle: t(loc, 'media.toolbar.all_authors_toggle'),
      noAuthors: t(loc, 'media.toolbar.no_authors'),
      allTypes: t(loc, 'media.toolbar.all_types'),
      screenshots: t(loc, 'media.toolbar.screenshots'),
      clips: t(loc, 'media.toolbar.clips'),
      allAuthors: t(loc, 'media.toolbar.all_authors'),
      mine: t(loc, 'media.toolbar.mine'),
      allPlaylists: t(loc, 'media.toolbar.all_playlists'),
      allMaps: t(loc, 'media.toolbar.all_maps'),
      allModes: t(loc, 'media.toolbar.all_modes'),
      allInCategory: (categoryLabel: string) =>
        t(loc, 'media.toolbar.all_in_category', { category: categoryLabel }),
      modeCategories: {
        Assassin: t(loc, 'media.toolbar.mode_categories.assassin'),
        Fiesta: t(loc, 'media.toolbar.mode_categories.fiesta'),
        'Super Fiesta': t(loc, 'media.toolbar.mode_categories.super_fiesta'),
        'Husky Raid': t(loc, 'media.toolbar.mode_categories.husky_raid'),
        BTB: t(loc, 'media.toolbar.mode_categories.btb'),
        Ranked: t(loc, 'media.toolbar.mode_categories.ranked'),
        Firefight: t(loc, 'media.toolbar.mode_categories.firefight'),
        Other: t(loc, 'media.toolbar.mode_categories.other'),
      },
      dateDesc: t(loc, 'media.toolbar.sort.date_desc'),
      dateAsc: t(loc, 'media.toolbar.sort.date_asc'),
      mapAsc: t(loc, 'media.toolbar.sort.map_asc'),
      modeAsc: t(loc, 'media.toolbar.sort.mode_asc'),
      noGrouping: t(loc, 'media.toolbar.group.no_grouping'),
      byOwner: t(loc, 'media.toolbar.group.by_owner'),
      byMap: t(loc, 'media.toolbar.group.by_map'),
      byMode: t(loc, 'media.toolbar.group.by_mode'),
      bySession: t(loc, 'media.toolbar.group.by_session'),
    },
  }
}
