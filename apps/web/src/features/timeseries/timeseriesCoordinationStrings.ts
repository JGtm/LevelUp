/**
 * timeseriesCoordinationStrings — les libellés des deux cartes de coordination des Séries
 * temporelles, résolus depuis le manifest `timeseries.progression.coord_*` (source unique
 * FR/EN, parité vérifiée par le build manifest, ADR 0003).
 *
 * Même forme que `getSquadRiposteText` : un objet `t` ergonomique consommé par le
 * composant. Aucun littéral FR/EN ici — tout vit dans le TOML.
 */
import { formatMessage } from '@/lib/i18n/format'
import { timeseriesManifest, type TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'

export function getTimeseriesCoordinationText(locale: Locale) {
  const m = (key: TimeseriesManifestKey, values?: Record<string, unknown>) =>
    formatMessage(timeseriesManifest, key, locale, values)
  return {
    riposteTitle: m('timeseries.progression.coord_riposte_title'),
    riposteTooltip: (seconds: number) =>
      m('timeseries.progression.coord_riposte_tooltip', { seconds }),
    appuiTitle: m('timeseries.progression.coord_appui_title'),
    appuiTooltip: m('timeseries.progression.coord_appui_tooltip'),

    covered: m('timeseries.progression.coord_covered'),
    iRiposte: m('timeseries.progression.coord_i_riposte'),
    prepared: m('timeseries.progression.coord_prepared'),
    myShare: m('timeseries.progression.coord_my_share'),

    usual: (rate: string) => m('timeseries.progression.coord_usual', { rate }),
    parity: (rate: string) => m('timeseries.progression.coord_parity', { rate }),

    // Mode ÉCART (3.B) : l'axe compte des points, la ligne zéro NOMME les deux repères
    // qu'elle remplace, et chaque tendance nomme la série qu'elle suit (deux courbes de
    // même libellé seraient indiscernables en légende).
    yAxisDelta: m('timeseries.progression.coord_y_axis_delta'),
    zeroLabel: (a: string, b: string) => m('timeseries.progression.coord_zero_label', { a, b }),
    trend: (name: string) => m('timeseries.progression.coord_trend', { name }),
    hollow: m('timeseries.progression.coord_hollow'),
    points: m('timeseries.progression.coord_points'),
    tooltipZero: m('timeseries.progression.coord_tt_zero'),
    tooltipTrend: m('timeseries.progression.coord_tt_trend'),

    volMyDeaths: (n: number) => m('timeseries.progression.coord_vol_my_deaths', { n }),
    volTeamDeaths: (n: number) => m('timeseries.progression.coord_vol_team_deaths', { n }),
    volMyKills: (n: number) => m('timeseries.progression.coord_vol_my_kills', { n }),
    volTeamAssists: (n: number) => m('timeseries.progression.coord_vol_team_assists', { n }),

    delay: m('timeseries.progression.coord_delay'),
    coverage: (measured: number, total: number) =>
      m('timeseries.progression.coord_coverage', { measured, total }),
    empty: m('timeseries.progression.coord_empty'),
    unavailable: m('timeseries.progression.coord_unavailable'),
  }
}

export type TimeseriesCoordinationText = ReturnType<typeof getTimeseriesCoordinationText>
