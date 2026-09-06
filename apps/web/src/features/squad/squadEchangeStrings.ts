/**
 * squadEchangeStrings — les libellés de L'ÉCHANGE (matrice, délais, KPI, cap du
 * moment), résolus depuis le manifest i18n `squad.echange.*` (source unique FR/EN,
 * parité vérifiée par le build manifest, ADR 0003).
 *
 * Même forme que `squadFocusStrings` : un objet `t` ergonomique (statiques +
 * fonctions d'interpolation ICU) consommé par les composants. Ne pas remettre de
 * littéral FR/EN en dur ici : tout vit dans `lib/i18n/manifests/squad.toml`.
 *
 * VOCABULAIRE FR ARRÊTÉ : « échange », « vengeance », « riposte ». Jamais l'anglais.
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

export interface EchangeNarrativeVars {
  matches: number
  brut: number
  n: number
  seconds: number
  rate: string
}

export function getSquadEchangeText(locale: Locale) {
  const m = (key: SquadManifestKey, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, values)
  return {
    sectionTitle: m('squad.echange.section_title'),
    sectionLabel: m('squad.echange.section_label'),
    axisAvenger: m('squad.echange.axis_avenger'),
    axisAvenged: m('squad.echange.axis_avenged'),
    definition: (seconds: number) => m('squad.echange.definition', { seconds }),
    coverage: (measured: number, total: number) =>
      m('squad.echange.coverage', { measured, total }),
    coverageHint: m('squad.echange.coverage_hint'),
    narrative: (v: EchangeNarrativeVars) => m('squad.echange.narrative', { ...v }),
    lowSample: m('squad.echange.low_sample'),
    lowSampleHint: (floor: number) => m('squad.echange.low_sample_hint', { floor }),
    badgeMostCovered: (player: string) => m('squad.echange.badge_most_covered', { player }),
    badgeLeastCovered: (player: string) => m('squad.echange.badge_least_covered', { player }),
    badgeCoveredDetail: (n: number) => m('squad.echange.badge_covered_detail', { n }),
    noPairs: m('squad.echange.no_pairs'),
    emptyTitle: m('squad.echange.empty_title'),
    matrixTooltip: (avenger: string, avenged: string, n: number, perMatch: string) =>
      m('squad.echange.matrix_tooltip', { avenger, avenged, n, perMatch }),

    delayTitle: m('squad.echange.delay_title'),
    delayLabel: m('squad.echange.delay_label'),
    delayNarrative: (inWindow: number, outside: number, total: number, seconds: number) =>
      m('squad.echange.delay_narrative', { inWindow, outside, total, seconds }),
    delayNarrativeEmpty: m('squad.echange.delay_narrative_empty'),
    delayXAxis: m('squad.echange.delay_x_axis'),
    delayYAxis: m('squad.echange.delay_y_axis'),
    delayBin: (start: number, end: number) => m('squad.echange.delay_bin', { start, end }),
    delayBinOpen: (start: number) => m('squad.echange.delay_bin_open', { start }),
    delayOutOfWindowSuffix: m('squad.echange.delay_out_of_window_suffix'),
    delayWindow: (seconds: number) => m('squad.echange.delay_window', { seconds }),

    kpiLabel: m('squad.echange.kpi_label'),
    kpiSecondary: (brut: number, n: number) => m('squad.echange.kpi_secondary', { brut, n }),
    kpiVsUsual: (delta: string) => m('squad.echange.kpi_vs_usual', { delta }),

    constatTitle: m('squad.echange.constat_title'),
    constatConsolidate: (delta: string, rate: string, usual: string) =>
      m('squad.echange.constat_consolidate', { delta, rate, usual }),
    constatAttention: (delta: string, rate: string, usual: string) =>
      m('squad.echange.constat_attention', { delta, rate, usual }),
    constatBasis: (n: number, matches: number) => m('squad.echange.constat_basis', { n, matches }),
  }
}

export type SquadEchangeText = ReturnType<typeof getSquadEchangeText>
