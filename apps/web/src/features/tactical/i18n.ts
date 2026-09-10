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

    // ── Vue d'analyse (Phase 5) ──────────────────────────────────────────────
    analysisQuestions: [
      { id: 'morts' as const, label: m('tactical.analysis.questions.morts') as string },
      { id: 'kills' as const, label: m('tactical.analysis.questions.kills') as string },
      { id: 'gagne' as const, label: m('tactical.analysis.questions.gagne') as string },
      { id: 'temps' as const, label: m('tactical.analysis.questions.temps') as string },
      { id: 'routes' as const, label: m('tactical.analysis.questions.routes') as string },
      { id: 'isole' as const, label: m('tactical.analysis.questions.isole') as string },
    ],
    kpiMatchsRetained: m('tactical.kpi.matches_retained'),
    kpiCoverage: m('tactical.kpi.coverage'),
    kpiTrade: m('tactical.kpi.trade'),
    kpiIsolation: m('tactical.kpi.isolation'),
    kpiSecondary: (brut: number, n: number) => m('tactical.kpi.secondary', { brut, n }),
    // Réserve d'échantillon des tuiles KPI « Échange » / « Isolement » : même clé
    // et même forme (accolée au secondaire) que `SquadEchangeKpi` pour la même
    // mesure — le drapeau `echantillon_faible` interdit de comparer, il ne cache
    // pas la valeur.
    lowSample: m('tactical.kpi.low_sample'),
    // Note de couverture de la tuile « Morts en isolement » : `matchs_sans_rayon`
    // est déjà publié par le contrat (aucun calcul côté web).
    kpiNoRadiusNote: (n: number) => m('tactical.kpi.no_radius_note', { n }),
    planTitle: m('tactical.plan.title'),
    footerFloor: (n: number) => m('tactical.plan.footer_floor', { n }),
    sourceReplay: m('tactical.plan.source_replay'),
    sourceJournal: m('tactical.plan.source_journal'),
    // Pas de la grille : PUBLIÉ par le serveur (`pas_m`), affiché tel quel — un plan à
    // 2 m est plus grossier qu'un plan à 0,5 m, et cela doit se lire au pied de la carte.
    footerGrid: (pas: number) => m('tactical.plan.footer_grid', { pas }),
    // Les trois états vides du plan. Ils ne disent PAS la même chose : périmètre vide,
    // aucune mesure, ou mesures trop dispersées (cf. `planEmptyReason`).
    planEmptyNoMatchTitle: m('tactical.plan.empty_no_match_title'),
    planEmptyNoMatchDescription: m('tactical.plan.empty_no_match_description'),
    planEmptyTitle: m('tactical.plan.empty_title'),
    planEmptyDescription: m('tactical.plan.empty_description'),
    planEmptyDensityTitle: m('tactical.plan.empty_density_title'),
    planEmptyDensityDescription: (matchs: number, plancher: number, pas: number) =>
      m('tactical.plan.empty_density_description', { matchs, plancher, pas }),
    statusPending: (n: number) => m('tactical.status.pending', { n }),
    statusUnavailable: (n: number) => m('tactical.status.unavailable', { n }),
    cellTitle: m('tactical.cell.title'),
    cellPlaceholder: m('tactical.cell.placeholder'),
    cellPlaceholderDescription: m('tactical.cell.placeholder_description'),
    cellMatches: (n: number) => m('tactical.cell.matches', { n }),
    // Contributions et lien vers le rejeu (lot M1, Tactique S.1).
    cellContributionsTitle: m('tactical.cell.contributions_title'),
    cellContributionsLoading: m('tactical.cell.contributions_loading'),
    cellContributionsEmpty: m('tactical.cell.contributions_empty'),
    cellContributionLabel: (date: string, instant: string) =>
      m('tactical.cell.contribution_label', { date, instant }),
    cellFooterNotOpenable: (n: number) => m('tactical.cell.footer_not_openable', { n }),

    // ── Titre de la vue, barre d'outils ──────────────────────────────────────
    analysisPageTitle: (map: string, question: string) =>
      m('tactical.analysis.page_title', { map, question }),
    analysisErrorTitle: m('tactical.analysis.error_title'),
    analysisErrorDescription: m('tactical.analysis.error_description'),
    questionLabel: m('tactical.toolbar.question_label'),
    whoLabel: m('tactical.toolbar.who_label'),
    whoMe: m('tactical.toolbar.who_me'),
    whoSquad: m('tactical.toolbar.who_squad'),
    whoOpponents: m('tactical.toolbar.who_opponents'),
    spawnLabel: m('tactical.toolbar.spawn_label'),
    spawnAll: m('tactical.toolbar.spawn_all'),

    // ── Unité de la légende, une par question ────────────────────────────────
    units: {
      morts: m('tactical.unit.morts') as string,
      kills: m('tactical.unit.kills') as string,
      gagne: m('tactical.unit.gagne') as string,
      temps: m('tactical.unit.temps') as string,
      routes: m('tactical.unit.routes') as string,
      isole: m('tactical.unit.isole') as string,
    },
  }
}

export type TacticalText = ReturnType<typeof getTacticalText>
