/**
 * i18n de l'onglet Tendances : accesseurs TYPÉS du manifeste `tendances.*` (source unique
 * FR/EN, parité vérifiée par le build des manifestes, ADR 0003). Aucun littéral FR/EN ici :
 * tout vit dans `lib/i18n/manifests/tendances.toml`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { tendancesManifest, type TendancesManifestKey } from '@/lib/i18n/generated/tendances'
import type { Locale } from '@/lib/i18n/locale'

import type { Step, TendancesView } from './tendances.logic'

/** Groupes de la matrice, dans l'ordre d'affichage. */
export const MATRIX_GROUPS = [
  'level',
  'results',
  'combat',
  'style',
  'objectives',
  'activity',
  'squad',
  'members',
] as const
export type MatrixGroup = (typeof MATRIX_GROUPS)[number]

/** Clés d'indicateur dont le libellé vit dans le manifeste (pas de clé de champ canonique). */
const MANIFEST_INDICATORS: ReadonlySet<string> = new Set([
  'damage_balance',
  'mmr_gap',
  'assists_per_match',
  'kills_per_match',
  'deaths_per_match',
  'headshot_share',
  'power_weapon_share',
  'equipment_used_share',
  'objective_take_share',
  'objective_defend_share',
  'objective_hold_share',
  'objective_parity',
  'hours_played',
  'days_played',
  'dnf_rate',
  'win_rate_alone',
  'squad_share_of_team_kills',
  'member_share_of_squad_kills',
])

/** Types de partie dont le libellé vit dans le manifeste. */
const MANIFEST_GAME_TYPES: ReadonlySet<string> = new Set([
  'ranked_slayer',
  'ranked_objectif',
  'other',
])

/** Groupes de file classée (variante d'une ligne CSR) dont le libellé vit dans le manifeste. */
const MANIFEST_VARIANTS: ReadonlySet<string> = new Set(['ranked'])

type Message = (key: TendancesManifestKey, values?: Record<string, unknown>) => string

/** Textes de la page, de ses états, du filtre et de la matrice avec ses infobulles. */
function pageMatrixTexts(m: Message) {
  return {
    gameTypeLabel: m('tendances.filter.game_type_label'),
    allGameTypes: m('tendances.filter.all_game_types'),
    loading: m('tendances.state.loading'),
    error: m('tendances.state.error'),
    retry: m('tendances.state.retry'),
    emptyTitle: m('tendances.state.empty_title'),
    emptyDescription: m('tendances.state.empty_description'),
    matrixTitle: m('tendances.matrix.title'),
    matrixInfo: m('tendances.matrix.info'),
    groupLabel: (group: MatrixGroup) => m(`tendances.group.${group}`),
    horizonLabel: m('tendances.horizon.label'),
    horizonAria: m('tendances.horizon.aria'),
    horizonInfo: m('tendances.horizon.info'),
    horizonShort: (days: number) => m('tendances.horizon.short', { days }),
    horizonLong: (days: number) => m('tendances.horizon.long', { days }),
    tooltipHeader: (label: string, period: string) =>
      m('tendances.tooltip.header', { label, period }),
    tooltipValue: (value: string, matches: number) =>
      m('tendances.tooltip.value', {
        value,
        matches: m('tendances.tooltip.matches', { n: matches }),
      }),
    tooltipPrevious: (value: string, matches: number) =>
      m('tendances.tooltip.previous', {
        value,
        matches: m('tendances.tooltip.matches', { n: matches }),
      }),
    tooltipVariation: (delta: string) => m('tendances.tooltip.variation', { delta }),
    tooltipNoComparison: (n: number, min: number) =>
      m('tendances.tooltip.no_comparison', { n, min }),
    tooltipNoComparisonCurrent: (n: number, min: number) =>
      m('tendances.tooltip.no_comparison_current', { n, min }),
  }
}

/** Textes de la section « Évolution » : pas, grilles, courbes et infobulles. */
function evolutionTexts(m: Message) {
  return {
    evolutionTitle: m('tendances.evolution.title'),
    evolutionInfo: m('tendances.evolution.info'),
    stepAria: m('tendances.evolution.step_aria'),
    stepLabel: (step: Step) => m(`tendances.evolution.step_${step}` as TendancesManifestKey),
    evolutionEmptyTitle: m('tendances.evolution.empty_title'),
    evolutionEmptyDescription: m('tendances.evolution.empty_description'),
    gridVersusMmr: m('tendances.evolution.grid_versus_mmr'),
    gridResults: m('tendances.evolution.grid_results'),
    gridLevel: m('tendances.evolution.grid_level'),
    chartVersusMmr: (label: string) => m('tendances.evolution.chart_versus_mmr', { label }),
    chartYieldResistance: m('tendances.evolution.chart_yield_resistance'),
    chartCsr: m('tendances.evolution.chart_csr'),
    chartLusr: m('tendances.evolution.chart_lusr'),
    chartMmrPair: m('tendances.evolution.chart_mmr_pair'),
    chartDamage: m('tendances.evolution.chart_damage'),
    chartObjectives: m('tendances.evolution.chart_objectives'),
    infoMmrPair: m('tendances.evolution.info_mmr_pair'),
    infoCsr: m('tendances.evolution.info_csr'),
    infoObjectives: m('tendances.evolution.info_objectives'),
    evolutionTooltipLine: (name: string, value: string, matches: number) =>
      m('tendances.evolution.tooltip_line', {
        name,
        value,
        matches: m('tendances.tooltip.matches', { n: matches }),
      }),
    evolutionWeekOf: (date: string) => m('tendances.evolution.tooltip_week_of', { date }),
  }
}

/** Textes des blocs : calendrier, victoires et défaites, médailles, types de partie. */
function blockTexts(m: Message) {
  return {
    calendarTitle: m('tendances.calendar.title'),
    calendarInfo: m('tendances.calendar.info'),
    calendarEmptyTitle: m('tendances.calendar.empty_title'),
    calendarEmptyDescription: m('tendances.calendar.empty_description'),
    calendarWins: (n: number) => m('tendances.calendar.tooltip_wins', { n }),
    calendarLosses: (n: number) => m('tendances.calendar.tooltip_losses', { n }),
    calendarWinRate: (value: string) => m('tendances.calendar.tooltip_win_rate', { value }),
    calendarPerformance: (value: string) =>
      m('tendances.calendar.tooltip_performance', { value }),
    matchCount: (n: number) => m('tendances.tooltip.matches', { n }),
    outcomesSection: m('tendances.outcomes.section'),
    winLossTitle: m('tendances.winloss.title'),
    winLossInfo: m('tendances.winloss.info'),
    winLossSeriesLoss: m('tendances.winloss.series_loss'),
    winLossSeriesWin: m('tendances.winloss.series_win'),
    winLossTooltipLoss: (value: string) => m('tendances.winloss.tooltip_loss', { value }),
    winLossTooltipWin: (value: string) => m('tendances.winloss.tooltip_win', { value }),
    winLossTooltipR: (r: string) => m('tendances.winloss.tooltip_r', { r }),
    winLossEmptyTitle: m('tendances.winloss.empty_title'),
    winLossEmptyDescription: (n: number, required: number) =>
      m('tendances.winloss.empty_description', { n, required }),
    medalsTitle: m('tendances.medals.title'),
    medalsInfo: m('tendances.medals.info'),
    medalsSeriesPrevious: (days: number) => m('tendances.medals.series_previous', { days }),
    medalsTooltipCurrent: (name: string, value: string) =>
      m('tendances.medals.tooltip_current', { name, value }),
    medalsNew: m('tendances.medals.tooltip_new'),
    medalsEmptyTitle: m('tendances.medals.empty_title'),
    medalsEmptyDescription: m('tendances.medals.empty_description'),
    mixTitle: m('tendances.mix.title'),
    mixInfo: m('tendances.mix.info'),
    mixEmptyMessage: m('tendances.mix.empty_message'),
  }
}

/** Textes de la vue escouade, des unités et des libellés tirés du manifeste. */
function squadTexts(m: Message) {
  return {
    viewAria: m('tendances.view.aria'),
    viewLabel: (view: TendancesView) => m(`tendances.view.${view}` as TendancesManifestKey),
    squadEmptyTitle: m('tendances.squad.empty_title'),
    squadEmptyDescription: m('tendances.squad.empty_description'),
    squadSeriesWith: m('tendances.squad.series_with'),
    squadSeriesAlone: m('tendances.squad.series_alone'),
    squadChartWinRate: m('tendances.squad.chart_win_rate'),
    squadInfoWinRate: m('tendances.squad.info_win_rate'),
    squadChartKda: m('tendances.squad.chart_kda'),
    squadChartMemberShare: m('tendances.squad.chart_member_share'),
    squadInfoMemberShare: m('tendances.squad.info_member_share'),
    squadChartTeamShare: m('tendances.squad.chart_team_share'),
    squadInfoTeamShare: m('tendances.squad.info_team_share'),
    squadChartMmrGap: m('tendances.squad.chart_mmr_gap'),
    squadInfoMmrGap: m('tendances.squad.info_mmr_gap'),
    squadChartMatchCount: m('tendances.squad.chart_match_count'),
    unitSeconds: m('tendances.unit.seconds'),
    unitHours: m('tendances.unit.hours'),
    unitPoints: m('tendances.unit.points'),
    /** Libellé manifeste d'un indicateur, ou `undefined` quand la clé relève des champs canoniques. */
    indicatorLabel: (key: string): string | undefined =>
      MANIFEST_INDICATORS.has(key)
        ? m(`tendances.indicator.${key}` as TendancesManifestKey)
        : undefined,
    /** Libellé manifeste d'un type de partie, ou `undefined` si la clé n'y figure pas. */
    gameTypeManifestLabel: (key: string): string | undefined =>
      MANIFEST_GAME_TYPES.has(key)
        ? m(`tendances.game_type.${key}` as TendancesManifestKey)
        : undefined,
    /** Libellé manifeste d'un groupe de file classée, ou `undefined` si la clé n'y figure pas. */
    variantManifestLabel: (key: string): string | undefined =>
      MANIFEST_VARIANTS.has(key) ? m(`tendances.variant.${key}` as TendancesManifestKey) : undefined,
  }
}

export function getTendancesText(locale: Locale) {
  const m: Message = (key, values) => formatMessage(tendancesManifest, key, locale, values)
  return { ...pageMatrixTexts(m), ...evolutionTexts(m), ...blockTexts(m), ...squadTexts(m) }
}

export type TendancesText = ReturnType<typeof getTendancesText>
