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
    recordLabel: (wins: number, losses: number, n: number) =>
      m('tactical.maps.record_label', { wins, losses, n }),
    select: (map: string) => m('tactical.maps.select', { map }),
    selected: m('tactical.maps.selected'),
    floorReason: (n: number, floor: number) => m('tactical.maps.floor_reason', { n, floor }),
    // Phrase d'introduction de la grille (maquette 034b1915) : cartes, matchs et
    // plancher en UNE phrase — elle a remplacé la paire couverture + note de plancher.
    intro: (maps: number, matches: number, floor: number) =>
      m('tactical.maps.intro', { maps, matches, floor }),
    // Résumé d'une vignette sur une seule ligne : « N matchs · V V / D D ».
    tileSummary: (n: number, wins: number, losses: number) =>
      m('tactical.maps.tile_summary', { n, wins, losses }),
    // Bascule d'écran (grille / analyse).
    screenLabel: m('tactical.screen.label'),
    screenGrid: m('tactical.screen.grid'),
    screenMap: m('tactical.screen.map'),

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
    // ORDRE DE LA MAQUETTE 034b1915, et il n'est pas arbitraire : on va du plus large au
    // plus précis (où je passe mon temps, où je meurs, où je tue), puis des lectures
    // dérivées (isolement, écart victoires/défaites), puis des routes, qui ne sont pas une
    // grandeur par cellule.
    analysisQuestions: [
      { id: 'temps' as const, label: m('tactical.analysis.questions.temps') as string },
      { id: 'morts' as const, label: m('tactical.analysis.questions.morts') as string },
      { id: 'kills' as const, label: m('tactical.analysis.questions.kills') as string },
      { id: 'isole' as const, label: m('tactical.analysis.questions.isole') as string },
      { id: 'gagne' as const, label: m('tactical.analysis.questions.gagne') as string },
      { id: 'routes' as const, label: m('tactical.analysis.questions.routes') as string },
    ],
    kpiMatchsRetained: m('tactical.kpi.matches_retained'),
    kpiCoverage: m('tactical.kpi.coverage'),
    kpiRiposte: m('tactical.kpi.riposte'),
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
    // Sous-titres des quatre tuiles (maquette 034b1915).
    kpiMatchsRetainedSecondary: (n: number) => m('tactical.kpi.matches_retained_secondary', { n }),
    kpiCoverageReplay: m('tactical.kpi.coverage_replay'),
    kpiCoverageShared: m('tactical.kpi.coverage_shared'),
    kpiRiposteWindow: (secondes: number) => m('tactical.kpi.riposte_window', { secondes }),
    kpiIsolationRadius: (rayon: string) => m('tactical.kpi.isolation_radius', { rayon }),
    kpiLowerIsBetter: m('tactical.kpi.lower_is_better'),
    planTitle: m('tactical.plan.title'),
    planLegendLabel: (lo: string, hi: string) => m('tactical.plan.legend_label', { lo, hi }),
    planScaleQuantile: m('tactical.plan.scale_quantile'),
    planScaleDivergent: (plancher: number) => m('tactical.plan.scale_divergent', { plancher }),
    planFooterRetained: (retenus: number, filtres: number, source: string) =>
      m('tactical.plan.footer_retained', { retenus, filtres, source }),
    footerFloor: (n: number) => m('tactical.plan.footer_floor', { n }),
    sourceReplay: m('tactical.plan.source_replay'),
    sourceJournal: m('tactical.plan.source_journal'),
    // Pas de la grille : PUBLIÉ par le serveur (`pas_m`), affiché tel quel — un plan à
    // 2 m est plus grossier qu'un plan à 0,5 m, et cela doit se lire au pied de la carte.
    footerGrid: (pas: number) => m('tactical.plan.footer_grid', { pas }),
    footerOffFrame: (hors: number, total: number) =>
      m('tactical.plan.footer_off_frame', { hors, total }),
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
    cellContributionOpen: (instant: string) => m('tactical.cell.contribution_open', { instant }),
    // « Mes routes de spawn » : la lecture n'a pas de cellule à détailler.
    cellPlaceholderRoutes: m('tactical.cell.placeholder_routes'),
    cellPlaceholderRoutesDescription: m('tactical.cell.placeholder_routes_description'),

    // ── Titre de la vue, barre d'outils ──────────────────────────────────────
    analysisPageTitle: (map: string, question: string) =>
      m('tactical.analysis.page_title', { map, question }),
    analysisErrorTitle: m('tactical.analysis.error_title'),
    analysisErrorDescription: m('tactical.analysis.error_description'),
    // Relecture : l'ancien calque reste affiché, estompé, sous cette mention (Q26).
    analysisUpdating: m('tactical.analysis.updating'),
    questionLabel: m('tactical.toolbar.question_label'),
    whoLabel: m('tactical.toolbar.who_label'),
    whoMe: m('tactical.toolbar.who_me'),
    whoSquad: m('tactical.toolbar.who_squad'),
    whoOpponents: m('tactical.toolbar.who_opponents'),
    spawnLabel: m('tactical.toolbar.spawn_label'),
    spawnAll: m('tactical.toolbar.spawn_all'),

    // ── Section « Coordination d'équipe » ────────────────────────────────────
    coordinationTitle: m('tactical.coordination.title'),
    coordinationMedian: m('tactical.coordination.median'),
    coordinationChartTitle: m('tactical.coordination.chart_title'),
    coordinationChartY: m('tactical.coordination.chart_y'),
    coordinationBucket: (min: number, max: number) =>
      m('tactical.coordination.bucket', { min, max }),
    coordinationBucketLast: (min: number) => m('tactical.coordination.bucket_last', { min }),
    coordinationThresholdLabel: m('tactical.coordination.threshold_label'),
    coordinationNoteRules: (secondes: number, rayon: string) =>
      m('tactical.coordination.note_rules', { secondes, rayon }),
    coordinationNoteCoverage: (retenus: number, filtres: number) =>
      m('tactical.coordination.note_coverage', { retenus, filtres }),
    coordinationNoDistance: (n: number) => m('tactical.coordination.no_distance', { n }),
    coordinationEmpty: m('tactical.coordination.empty'),
    coordinationEmptyDescription: m('tactical.coordination.empty_description'),
    // La valeur arrive DEJA FORMATEE (`formatNumber`, lib/formatters) : l'arrondi est une
    // regle de presentation du depot, pas une regle de message — « 9,905 m » etait le
    // symptome d'un nombre passe brut a ICU.
    radiusValue: (rayon: string) => m('tactical.coordination.radius_value', { rayon }),
    radiusJoin: m('tactical.coordination.radius_join') as string,

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
