/**
 * timeseriesRangeRolesStrings — les libellés de la carte « Rôles de portée » du Résumé,
 * résolus depuis le manifest `timeseries.portee.*` (source unique FR/EN, parité vérifiée
 * par le build manifest, ADR 0003).
 *
 * Même forme que `squadRangeRolesStrings` : un objet `t` ergonomique consommé par la carte.
 * Les clés sont CELLES DE TIMESERIES, pas celles de l'Escouade — la carte est solo, son
 * infobulle et sa légende ne disent pas la même chose. Ne pas remettre de littéral FR/EN ici.
 */
import { formatMessage } from '@/lib/i18n/format'
import { timeseriesManifest, type TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'
import type { RoleDePortee } from '@/features/squad/squadRangeRoles.logic'

export function getTimeseriesRangeRolesText(locale: Locale) {
  const m = (key: TimeseriesManifestKey, values?: Record<string, unknown>) =>
    formatMessage(timeseriesManifest, key, locale, values)
  const bandes: Record<RoleDePortee, string> = {
    front: m('timeseries.portee.band_front'),
    polyvalent: m('timeseries.portee.band_polyvalent'),
    sniper: m('timeseries.portee.band_sniper'),
  }
  return {
    cardTitle: m('timeseries.portee.card_title'),
    sectionLabel: m('timeseries.portee.section_label'),
    help: (floor: number) => m('timeseries.portee.help', { floor }),
    xAxis: m('timeseries.portee.x_axis'),
    yAxis: m('timeseries.portee.y_axis'),
    lobbyLine: m('timeseries.portee.lobby_line'),
    bandes,
    legendMatch: m('timeseries.portee.legend_match'),
    legendLowSample: (floor: number) => m('timeseries.portee.legend_low_sample', { floor }),
    legendTrend: (n: number) => m('timeseries.portee.legend_trend', { n }),
    tooltipMedian: (mValue: string) => m('timeseries.portee.tooltip_median', { m: mValue }),
    tooltipDelta: (delta: string) => m('timeseries.portee.tooltip_delta', { delta }),
    tooltipMeasured: (n: number) => m('timeseries.portee.tooltip_measured', { n }),
    coverage: (measured: number, total: number) =>
      m('timeseries.portee.coverage', { measured, total }),
    emptyTitle: m('timeseries.portee.empty_title'),
    emptyDescription: m('timeseries.portee.empty_description'),
  }
}
