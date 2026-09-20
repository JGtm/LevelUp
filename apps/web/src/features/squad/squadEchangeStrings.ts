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
    definition: (seconds: number) => m('squad.echange.definition', { seconds }),
    narrative: (v: EchangeNarrativeVars) => m('squad.echange.narrative', { ...v }),
    lowSample: m('squad.echange.low_sample'),
    lowSampleHint: (floor: number) => m('squad.echange.low_sample_hint', { floor }),
    badgeMostCovered: (player: string) => m('squad.echange.badge_most_covered', { player }),
    badgeLeastCovered: (player: string) => m('squad.echange.badge_least_covered', { player }),
    noPairs: m('squad.echange.no_pairs'),
    emptyTitle: m('squad.echange.empty_title'),
    matrixTooltip: (avenger: string, avenged: string, n: number, perMatch: string) =>
      m('squad.echange.matrix_tooltip', { avenger, avenged, n, perMatch }),

    delayTitle: m('squad.echange.delay_title'),
    delayLabel: m('squad.echange.delay_label'),
    delayNarrativeEmpty: m('squad.echange.delay_narrative_empty'),
    delayXAxis: m('squad.echange.delay_x_axis'),
    delayYAxis: m('squad.echange.delay_y_axis'),
    delayBin: (start: number, end: number) => m('squad.echange.delay_bin', { start, end }),
    delayBinOpen: (start: number) => m('squad.echange.delay_bin_open', { start }),
    delayOutOfWindowSuffix: m('squad.echange.delay_out_of_window_suffix'),
    delayWindow: (seconds: number) => m('squad.echange.delay_window', { seconds }),

    kpiLabel: m('squad.echange.kpi_label'),
    kpiVsUsual: (delta: string) => m('squad.echange.kpi_vs_usual', { delta }),

    constatTitle: m('squad.echange.constat_title'),
    constatConsolidate: (delta: string, rate: string, usual: string) =>
      m('squad.echange.constat_consolidate', { delta, rate, usual }),
    constatAttention: (delta: string, rate: string, usual: string) =>
      m('squad.echange.constat_attention', { delta, rate, usual }),
    constatBasis: (n: number, matches: number) => m('squad.echange.constat_basis', { n, matches }),

    // ── Maquette 4c520da6 : « Le compte » ────────────────────────────────────
    compteTitle: m('squad.echange.compte_title'),
    compteLabel: m('squad.echange.compte_label'),
    compteSay: m('squad.echange.compte_say'),
    compteFoot: m('squad.echange.compte_foot'),
    kpiRateSub: (seconds: number) => m('squad.echange.kpi_rate_sub', { seconds }),
    kpiDelayLabel: m('squad.echange.kpi_delay_label'),
    kpiDelaySub: m('squad.echange.kpi_delay_sub'),
    kpiDelayValue: (seconds: string) => m('squad.echange.kpi_delay_value', { seconds }),
    kpiUnansweredLabel: m('squad.echange.kpi_unanswered_label'),
    kpiUnansweredSub: (n: number) => m('squad.echange.kpi_unanswered_sub', { n }),
    kpiUnansweredRateLabel: m('squad.echange.kpi_unanswered_rate_label'),
    kpiUnansweredRateSub: m('squad.echange.kpi_unanswered_rate_sub'),
    kpiNoValue: m('squad.echange.kpi_no_value'),

    // ── « Combien, et à quelle vitesse » ─────────────────────────────────────
    delaySay: (v: { morts: number; dedans: number; dehors: number; pic: string }) =>
      m('squad.echange.delay_say', { ...v }),
    delayFigure: (n: number, morts: number) => m('squad.echange.delay_figure', { n, morts }),
    delayWindowMark: (seconds: number) => m('squad.echange.delay_window_mark', { seconds }),
    delayFoot: (seconds: number) => m('squad.echange.delay_foot', { seconds }),

    // ── « Qui couvre qui » ───────────────────────────────────────────────────
    matrixReceived: (n: number) => m('squad.echange.matrix_received', { n }),
    matrixHelp: (seconds: number) => m('squad.echange.matrix_help', { seconds }),

    // ── « Donné et reçu, par coéquipier » ────────────────────────────────────
    donneRecuTitle: m('squad.echange.donne_recu_title'),
    donneRecuLabel: m('squad.echange.donne_recu_label'),
    donneRecuSay: (v: { joueur: string; recu: number; donne: number }) =>
      m('squad.echange.donne_recu_say', { ...v }),
    donneRecuSayEquilibre: m('squad.echange.donne_recu_say_equilibre'),
    donneRecuGiven: m('squad.echange.donne_recu_given'),
    donneRecuReceived: m('squad.echange.donne_recu_received'),

    // ── « Taux d'échange par session » ───────────────────────────────────────
    sessionRateTitle: m('squad.echange.session_rate_title'),
    sessionRateLabel: m('squad.echange.session_rate_label'),
    sessionRateSay: (v: { bas: string; haut: string; taux: string; n: number }) =>
      m('squad.echange.session_rate_say', { ...v }),
    sessionRateFigure: (seconds: number) => m('squad.echange.session_rate_figure', { seconds }),
    /** Sous le plancher de tendance : la ou les soirées retenues face à l'habituel. */
    sessionRateFewSay: (v: { n: number; floor: number }) =>
      m('squad.echange.session_rate_few_say', { ...v }),
    sessionRateSelection: m('squad.echange.session_rate_selection'),
    sessionRateUsual: m('squad.echange.session_rate_usual'),
    sessionRateUsualSub: (matchs: number) =>
      m('squad.echange.session_rate_usual_sub', { matchs }),
    sessionRateYAxis: m('squad.echange.session_rate_y_axis'),
  }
}

export type SquadEchangeText = ReturnType<typeof getSquadEchangeText>
