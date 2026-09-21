/**
 * squadIsolementStrings — les libellés du nuage « Pourquoi la vengeance ne vient pas »,
 * résolus depuis le manifest i18n `squad.isolement.*` (source unique FR/EN, ADR 0003).
 *
 * Même forme que `squadEchangeStrings` : un objet `t` ergonomique (statiques + fonctions
 * d'interpolation ICU). Ne pas remettre de littéral FR/EN en dur ici : tout vit dans
 * `lib/i18n/manifests/squad.toml`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

export function getSquadIsolementText(locale: Locale) {
  const m = (key: SquadManifestKey, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, values)

  return {
    sectionTitle: m('squad.isolement.section_title'),
    sectionLabel: m('squad.isolement.section_label'),
    xAxis: m('squad.isolement.x_axis'),
    yAxis: m('squad.isolement.y_axis'),
    /** Infobulle (i) du titre — trois phrases, jamais un pavé (décision 9). */
    help: m('squad.isolement.help'),
    radarLine: m('squad.isolement.radar_line'),
    bandOutOfSight: m('squad.isolement.band_out_of_sight'),
    bandNever: m('squad.isolement.band_never'),
    lowSample: m('squad.isolement.low_sample'),
    /** Infobulle d'UNE mort, ligne par ligne : la couverture s'y dit en portée /
     *  proximité du radar. Le retour à la ligne HTML est posé par le composant. */
    tooltipDeathCoverage: (ratio: string) => m('squad.isolement.tooltip_death_coverage', { ratio }),
    tooltipDeathOutOfSight: m('squad.isolement.tooltip_death_out_of_sight'),
    tooltipRiposted: (seconds: string) => m('squad.isolement.tooltip_riposted', { seconds }),
    tooltipNever: m('squad.isolement.tooltip_never'),
    tooltipRepereHead: (v: { gamertag: string; n: number }) =>
      m('squad.isolement.tooltip_repere_head', { ...v }),
    tooltipRepereIso: (isoRate: string) => m('squad.isolement.tooltip_repere_iso', { isoRate }),
    tooltipRepereCov: (covRate: string) => m('squad.isolement.tooltip_repere_cov', { covRate }),
    cardTitle: m('squad.isolement.card_title'),
    figure: m('squad.isolement.figure'),
    legendDeaths: (n: number) => m('squad.isolement.legend_deaths', { n }),
    legendDeath: m('squad.isolement.legend_death'),
    legendLowSample: (floor: number) => m('squad.isolement.legend_low_sample', { floor }),
    emptyTitle: m('squad.isolement.empty_title'),
    emptyDescription: m('squad.isolement.empty_description'),
  }
}

export type SquadIsolementText = ReturnType<typeof getSquadIsolementText>
