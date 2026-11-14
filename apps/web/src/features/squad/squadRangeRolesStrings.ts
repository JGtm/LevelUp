/**
 * squadRangeRolesStrings — les libellés du ruban « Rôles de portée », résolus depuis le
 * manifest i18n `squad.portee.*` (source unique FR/EN, parité vérifiée par le build
 * manifest, ADR 0003). Les clés sont typées (`SquadManifestKey`) : une clé absente du
 * manifest ne compile pas.
 *
 * Même forme que `squadFocusStrings` : un objet `t` ergonomique consommé par la carte et par
 * l'infobulle de ses cases. Ne pas remettre de littéral FR/EN en dur ici.
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
    help: (floor: number, window: number) => m('squad.portee.help', { floor, window }),
    xAxis: m('squad.portee.x_axis'),
    bandes,
    tooltipRole: (role: string) => m('squad.portee.tooltip_role', { role }),
    tooltipNoRole: m('squad.portee.tooltip_no_role'),
    tooltipDelta: (delta: string) => m('squad.portee.tooltip_delta', { delta }),
    tooltipMeasured: (n: number) => m('squad.portee.tooltip_measured', { n }),
    tooltipUnsure: (n: number) => m('squad.portee.tooltip_unsure', { n }),
    noRoleTitle: m('squad.portee.no_role_title'),
    noRoleDescription: (floor: number, window: number) => m('squad.portee.no_role_description', { floor, window }),
    emptyTitle: m('squad.portee.empty_title'),
    emptyDescription: m('squad.portee.empty_description'),
  }
}

export type SquadRangeRolesText = ReturnType<typeof getSquadRangeRolesText>
