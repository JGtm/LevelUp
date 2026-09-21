/**
 * squadRangeRolesStrings — les libellés de la carte « Rôles de portée », résolus depuis le
 * manifest i18n `squad.portee.*` (source unique FR/EN, parité vérifiée par le build
 * manifest, ADR 0003).
 *
 * Même forme que `squadRiposteStrings` : un objet `t` ergonomique consommé par la carte. Ne
 * pas remettre de littéral FR/EN en dur ici.
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

import type { RoleDePortee } from './squadRangeRoles.logic'

export function getSquadRangeRolesText(locale: Locale) {
  const m = (key: SquadManifestKey, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, values)
  const bandes: Record<RoleDePortee, string> = {
    front: m('squad.portee.band_front'),
    polyvalent: m('squad.portee.band_polyvalent'),
    sniper: m('squad.portee.band_sniper'),
  }
  return {
    cardTitle: m('squad.portee.card_title'),
    sectionLabel: m('squad.portee.section_label'),
    help: (floor: number) => m('squad.portee.help', { floor }),
    xAxis: m('squad.portee.x_axis'),
    yAxis: m('squad.portee.y_axis'),
    lobbyLine: m('squad.portee.lobby_line'),
    bandes,
    legendLowSample: (floor: number) => m('squad.portee.legend_low_sample', { floor }),
    legendTrend: (n: number) => m('squad.portee.legend_trend', { n }),
    tooltipMedian: (mValue: string) => m('squad.portee.tooltip_median', { m: mValue }),
    tooltipDelta: (delta: string) => m('squad.portee.tooltip_delta', { delta }),
    tooltipMeasured: (n: number) => m('squad.portee.tooltip_measured', { n }),
    coverage: (measured: number, total: number) =>
      m('squad.portee.coverage', { measured, total }),
    emptyTitle: m('squad.portee.empty_title'),
    emptyDescription: m('squad.portee.empty_description'),
    foldTape: m('squad.portee.fold_tape'),
    tapeNoRole: m('squad.portee.tape_no_role'),
  }
}

export type SquadRangeRolesText = ReturnType<typeof getSquadRangeRolesText>
