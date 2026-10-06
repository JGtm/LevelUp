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
    // ── Colonne « Cartes jouées » ───────────────────────────────────────────
    mapsTitle: m('tactical.maps.title'),
    loading: m('tactical.maps.loading'),
    error: m('tactical.maps.error'),
    emptyTitle: m('tactical.maps.empty_title'),
    recordLabel: (wins: number, losses: number, n: number) =>
      m('tactical.maps.record_label', { wins, losses, n }),
    select: (map: string) => m('tactical.maps.select', { map }),
    selected: m('tactical.maps.selected'),
    // Résumé d'une vignette sur une seule ligne : « 54 · 30 V / 24 D ».
    tileSummary: (n: number, wins: number, losses: number) =>
      m('tactical.maps.tile_summary', { n, wins, losses }),
    mapsSearchPlaceholder: m('tactical.maps.search_placeholder'),
    mapsSearchLabel: m('tactical.maps.search_label'),
    mapsNoMatch: m('tactical.maps.no_match'),
    mapsNoneOpenable: m('tactical.maps.none_openable'),
    floorFold: (n: number) => m('tactical.maps.floor_fold', { n }),
    floorFoldFiltered: (shown: number, n: number) => m('tactical.maps.floor_fold_filtered', { shown, n }),
    floorCount: (n: number, floor: number) => m('tactical.maps.floor_count', { n, floor }),

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

    // ── Lectures du plan, dans l'ordre de la pilule « Lecture » ─────────────
    // Les grandeurs de base (morts, frags, leur solde), puis les lectures signées et dérivées
    // (victoires − défaites), puis celles des artefacts de rejeu (temps, trajets), et la lecture
    // de placement (morts seul).
    analysisQuestions: [
      { id: 'morts' as const, label: m('tactical.analysis.questions.morts') as string },
      { id: 'kills' as const, label: m('tactical.analysis.questions.kills') as string },
      { id: 'solde' as const, label: m('tactical.analysis.questions.solde') as string },
      { id: 'gagne' as const, label: m('tactical.analysis.questions.gagne') as string },
      { id: 'temps' as const, label: m('tactical.analysis.questions.temps') as string },
      { id: 'routes' as const, label: m('tactical.analysis.questions.routes') as string },
      { id: 'isole' as const, label: m('tactical.analysis.questions.isole') as string },
    ],
    // Unité de la légende et de la zone, une par lecture.
    units: {
      morts: m('tactical.unit.morts') as string,
      kills: m('tactical.unit.kills') as string,
      solde: m('tactical.unit.solde') as string,
      gagne: m('tactical.unit.gagne') as string,
      temps: m('tactical.unit.temps') as string,
      routes: m('tactical.unit.routes') as string,
      isole: m('tactical.unit.isole') as string,
    },

    // ── Carte du plan : aide ⓘ du titre et réglages en pilules ──────────────
    planInfo: (retenus: number, filtres: number, source: string, pas: number, plancher: number) =>
      m('tactical.plan.info', { retenus, filtres, source, pas, plancher }),
    planInfoSourceJournal: m('tactical.plan.info_source_journal'),
    planInfoSourceReplay: m('tactical.plan.info_source_replay'),
    planInfoGagne: (v: number, d: number, plancher: number) => m('tactical.plan.info_gagne', { v, d, plancher }),
    planInfoIsole: (portee: string) => m('tactical.plan.info_isole', { portee }),
    planInfoNoRange: (n: number) => m('tactical.plan.info_no_range', { n }),
    planInfoTeamDown: (n: number) => m('tactical.plan.info_team_down', { n }),
    planInfoOffFrame: (hors: number, total: number) => m('tactical.plan.info_off_frame', { hors, total }),
    // La valeur arrive DÉJÀ FORMATÉE (au plus au dixième) : l'arrondi est une règle de présentation
    // du dépôt, pas une règle de message.
    radiusValue: (rayon: string) => m('tactical.plan.radius_value', { rayon }),
    radiusJoin: m('tactical.plan.radius_join') as string,
    pillReading: m('tactical.plan.pill_reading'),
    pillPlayers: m('tactical.plan.pill_players'),
    whoMe: m('tactical.plan.who_me'),
    whoSquad: m('tactical.plan.who_squad'),
    whoOpponents: m('tactical.plan.who_opponents'),
    planSquadDisabled: m('tactical.plan.squad_disabled'),
    pillRespawn: m('tactical.plan.pill_respawn'),
    pillRespawnAll: m('tactical.plan.pill_respawn_all'),

    // ── Carte du plan : états posés sur le fond, légende, bandeau d'état ────
    // Les trois états vides ne disent PAS la même chose : périmètre vide, aucune mesure, ou
    // mesures trop dispersées (cf. `planEmptyReason`). Un titre seul, aucun conseil.
    planEmptyNoMatchTitle: m('tactical.plan.empty_no_match_title'),
    planEmptyTitle: m('tactical.plan.empty_title'),
    planEmptyDensityTitle: m('tactical.plan.empty_density_title'),
    analysisErrorTitle: m('tactical.analysis.error_title'),
    // Relecture : l'ancien calque reste affiché, estompé, sous cette mention (Q26).
    analysisUpdating: m('tactical.analysis.updating'),
    planLegendLabel: (lo: string, hi: string) => m('tactical.plan.legend_label', { lo, hi }),
    statusPending: (n: number) => m('tactical.status.pending', { n }),
    statusUnavailable: (n: number) => m('tactical.status.unavailable', { n }),

    // ── Zone sélectionnée ───────────────────────────────────────────────────
    zoneTitle: m('tactical.zone.title'),
    zoneNone: m('tactical.zone.none'),
    zoneUnnamed: m('tactical.zone.unnamed'),
    zoneCoords: (x0: string, x1: string, y0: string, y1: string) => m('tactical.zone.coords', { x0, x1, y0, y1 }),
    zoneMatches: (n: number) => m('tactical.zone.matches', { n }),
    zoneWinsLosses: (v: number, d: number) => m('tactical.zone.wins_losses', { v, d }),
    zoneKillsDeaths: (f: number, morts: number) => m('tactical.zone.kills_deaths', { f, m: morts }),
    zoneReplayHeading: m('tactical.zone.replay_heading'),
    zoneContributionsLoading: m('tactical.zone.contributions_loading'),
    zoneContributionsEmpty: m('tactical.zone.contributions_empty'),
    zoneNotOpenable: (n: number) => m('tactical.zone.not_openable', { n }),

    // ── Mini-tuile « Rejeu » ────────────────────────────────────────────────
    tileKilledBy: (gt: string) => m('tactical.tile.killed_by', { gt }),
    tileKilled: (gt: string) => m('tactical.tile.killed', { gt }),
    tileDeath: m('tactical.tile.death'),
    tileFrag: m('tactical.tile.frag'),
    tileEntry: m('tactical.tile.entry'),
    tileRespawn: m('tactical.tile.respawn'),
    // Les catégories de source de dégât traduites ; les autres ne s'écrivent pas.
    tileCategories: {
      Headshot: m('tactical.tile.cat_headshot') as string,
      AttachedDamage: m('tactical.tile.cat_attached') as string,
      SilentMelee: m('tactical.tile.cat_silent_melee') as string,
      ChainedProjectile: m('tactical.tile.cat_chained') as string,
    } as Readonly<Record<string, string>>,
    tileAlone: m('tactical.tile.alone'),
    tileAloneAt: (d: string) => m('tactical.tile.alone_at', { d }),
    tileNearAt: (d: string) => m('tactical.tile.near_at', { d }),
    tileOpenReplay: (instant: string) => m('tactical.tile.open_replay', { instant }),
  }
}

export type TacticalText = ReturnType<typeof getTacticalText>
