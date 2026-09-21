/**
 * squadRiposteStrings — les libellés de la RIPOSTE (carte, frise, replis), résolus depuis
 * le manifest i18n `squad.riposte.*` (source unique FR/EN, parité vérifiée par le build
 * manifest, ADR 0003).
 *
 * Même forme que `squadFocusStrings` : un objet `t` ergonomique (statiques + fonctions
 * d'interpolation ICU) consommé par les composants. Ne pas remettre de littéral FR/EN en
 * dur ici : tout vit dans `lib/i18n/manifests/squad.toml`.
 *
 * VOCABULAIRE FR ARRÊTÉ (D19, 2026-09-21) : « riposte », « riposter », « taux de
 * riposte ». Jamais « échange », jamais « vengeance », jamais « trade ».
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

export function getSquadRiposteText(locale: Locale) {
  const m = (key: SquadManifestKey, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, values)
  return {
    // ── La section « Coordination » ──────────────────────────────────────────
    coordinationTitle: m('squad.riposte.coordination_title'),
    coordinationHelpRiposte: (seconds: number) =>
      m('squad.riposte.coordination_help_riposte', { seconds }),
    coordinationHelpAppui: m('squad.riposte.coordination_help_appui'),

    // ── La carte « Riposte » ─────────────────────────────────────────────────
    title: m('squad.riposte.title'),
    label: m('squad.riposte.label'),
    definition: (seconds: number) => m('squad.riposte.definition', { seconds }),
    helpUsual: m('squad.riposte.help_usual'),
    lowSample: m('squad.riposte.low_sample'),
    lowSampleHint: (floor: number) => m('squad.riposte.low_sample_hint', { floor }),
    emptyTitle: m('squad.riposte.empty_title'),

    // ── La phrase de lecteur ─────────────────────────────────────────────────
    sayFact: (v: { seconds: number; surCombien: number; delai: string }) =>
      m('squad.riposte.say_fact', { ...v }),
    sayFactNever: (seconds: number) => m('squad.riposte.say_fact_never', { seconds }),
    sayBelow: m('squad.riposte.say_below'),
    sayAbove: m('squad.riposte.say_above'),

    // ── Le chiffre d'appel ───────────────────────────────────────────────────
    appelLabel: (n: number) => m('squad.riposte.appel_label', { n }),
    appelUnit: m('squad.riposte.appel_unit'),
    appelBelow: (delta: string, usual: string) => m('squad.riposte.appel_below', { delta, usual }),
    appelAbove: (delta: string, usual: string) => m('squad.riposte.appel_above', { delta, usual }),
    delaiLabel: m('squad.riposte.delai_label'),
    delaiValue: (seconds: string) => m('squad.riposte.delai_value', { seconds }),
    delaiUnit: m('squad.riposte.delai_unit'),
    delaiSub: (ripostes: number, sans: number) => m('squad.riposte.delai_sub', { ripostes, sans }),
    noValue: m('squad.riposte.no_value'),

    // ── La frise, soirée par soirée ──────────────────────────────────────────
    friseTitle: (seconds: number) => m('squad.riposte.frise_title', { seconds }),
    friseAbove: m('squad.riposte.frise_above'),
    friseBelow: m('squad.riposte.frise_below'),
    friseTrend: (n: number) => m('squad.riposte.frise_trend', { n }),
    friseUsual: (rate: string) => m('squad.riposte.frise_usual', { rate }),
    friseYAxis: m('squad.riposte.frise_y_axis'),
    friseVolume: (n: number) => m('squad.riposte.frise_volume', { n }),
    friseVolumeAxis: m('squad.riposte.frise_volume_axis'),
    friseEmpty: m('squad.riposte.frise_empty'),

    // ── Les deux replis ──────────────────────────────────────────────────────
    foldDelay: m('squad.riposte.fold_delay'),
    foldMatrix: m('squad.riposte.fold_matrix'),

    // ── Repli « Combien de temps on met » ────────────────────────────────────
    delayXAxis: m('squad.riposte.delay_x_axis'),
    delayYAxis: m('squad.riposte.delay_y_axis'),
    delayBin: (start: number, end: number) => m('squad.riposte.delay_bin', { start, end }),
    delayBinOpen: (start: number) => m('squad.riposte.delay_bin_open', { start }),
    delayOutOfWindowSuffix: m('squad.riposte.delay_out_of_window_suffix'),
    delayWindow: (seconds: number) => m('squad.riposte.delay_window', { seconds }),
    delayNarrativeEmpty: m('squad.riposte.delay_narrative_empty'),
    delaySay: (v: { morts: number; dedans: number; dehors: number; pic: string }) =>
      m('squad.riposte.delay_say', { ...v }),
    delayFigure: (n: number, morts: number) => m('squad.riposte.delay_figure', { n, morts }),
    delayWindowMark: (seconds: number) => m('squad.riposte.delay_window_mark', { seconds }),
    delayFoot: (seconds: number) => m('squad.riposte.delay_foot', { seconds }),

    // ── Repli « Qui riposte pour qui » ───────────────────────────────────────
    matrixTooltip: (auteur: string, pour: string, n: number, perMatch: string) =>
      m('squad.riposte.matrix_tooltip', { auteur, pour, n, perMatch }),
    matrixReceived: (n: number) => m('squad.riposte.matrix_received', { n }),
    matrixHelp: (seconds: number) => m('squad.riposte.matrix_help', { seconds }),
    noPairs: m('squad.riposte.no_pairs'),
    badgeMostCovered: (player: string) => m('squad.riposte.badge_most_covered', { player }),
    badgeLeastCovered: (player: string) => m('squad.riposte.badge_least_covered', { player }),
  }
}

export type SquadRiposteText = ReturnType<typeof getSquadRiposteText>
