/**
 * i18n de l'onglet Tactique — accesseurs TYPÉS du manifeste `tactical.*`
 * (source unique FR/EN, parité vérifiée par le build manifest, ADR 0003).
 *
 * Même forme que `squadEchangeStrings` : un objet `t` ergonomique (statiques +
 * fonctions d'interpolation ICU) consommé par les composants. Ne JAMAIS remettre de
 * littéral FR/EN en dur ici : tout vit dans `lib/i18n/manifests/tactical.toml`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { tacticalManifest, type TacticalManifestKey } from '@/lib/i18n/generated/tactical'
import type { Locale } from '@/lib/i18n/locale'

export function getTacticalText(locale: Locale) {
  const m = (key: TacticalManifestKey, values?: Record<string, unknown>) =>
    formatMessage(tacticalManifest, key, locale, values)
  return {
    mapsTitle: m('tactical.maps.title'),
    mapsLabel: m('tactical.maps.label'),
    loading: m('tactical.maps.loading'),
    error: m('tactical.maps.error'),
    emptyTitle: m('tactical.maps.empty_title'),
    emptyDescription: m('tactical.maps.empty_description'),
    matches: (n: number) => m('tactical.maps.matches', { n }),
    record: (wins: number, losses: number) => m('tactical.maps.record', { wins, losses }),
    recordLabel: (wins: number, losses: number, n: number) =>
      m('tactical.maps.record_label', { wins, losses, n }),
    select: (map: string) => m('tactical.maps.select', { map }),
    selected: m('tactical.maps.selected'),
    floorReason: (n: number, floor: number) => m('tactical.maps.floor_reason', { n, floor }),
    floorNote: (floor: number) => m('tactical.maps.floor_note', { floor }),
    coverage: (maps: number, matches: number) =>
      m('tactical.maps.coverage', { maps, matches }),

    // ── Barre de filtres L2 ─────────────────────────────────────────────────
    filterLabels: {
      experience: m('tactical.filter.experience'),
      experienceAll: m('tactical.filter.experience_all'),
      experienceRanked: m('tactical.filter.experience_ranked'),
      experienceUnranked: m('tactical.filter.experience_unranked'),
      playlists: m('tactical.filter.playlists'),
      modes: m('tactical.filter.modes'),
      reset: m('tactical.filter.reset'),
    },
    viewLabels: {
      view: m('tactical.filter.view'),
      viewAll: m('tactical.filter.view_all'),
      viewSolo: m('tactical.filter.view_solo'),
      viewSquad: m('tactical.filter.view_squad'),
    },
    sessions: m('tactical.filter.sessions'),
    sessionsHorsListe: (n: number, names: string) =>
      m('tactical.filter.sessions_off_list', { n, names }),
    squadPlaceholder: (n: number) => m('tactical.filter.squad_placeholder', { n }),
    unknownTeammateTitle: m('tactical.filter.unknown_teammate_title'),
    unknownTeammateDescription: (names: string) =>
      m('tactical.filter.unknown_teammate_description', { names }),
  }
}

export type TacticalText = ReturnType<typeof getTacticalText>
