/**
 * squadRangeRolesStrings — les libellés du nuage des rôles, résolus depuis le manifest i18n
 * (source unique FR/EN, parité vérifiée par le build manifest, ADR 0003).
 *
 * UN PRÉFIXE PAR GRANDEUR, LES MÊMES SUFFIXES : `squad.portee.*` (lot R, distance) et
 * `squad.hauteur.*` (E1, dénivelé). C'est ce qui permet à la même carte, au même nuage et à
 * la même bande de servir les deux lectures sans dupliquer un composant — la grandeur ne
 * change que ce qu'elle nomme. Ajouter une grandeur = un préfixe ici et le bloc de clés
 * correspondant dans `manifests/squad.toml` (le test de parité en vérifie la complétude).
 *
 * Même forme que `squadRiposteStrings` : un objet `t` ergonomique consommé par la carte. Ne
 * pas remettre de littéral FR/EN en dur ici.
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

import type { GrandeurProfil, RoleDePortee } from './squadRangeRoles.logic'

/** Le préfixe de clés du manifest, par grandeur. */
export const PREFIXE_GRANDEUR: Record<GrandeurProfil, string> = {
  portee: 'squad.portee',
  hauteur: 'squad.hauteur',
}

/** Les suffixes attendus sous chaque préfixe — la liste que le test de parité balaye. */
export const SUFFIXES_ROLES = [
  'card_title',
  'section_label',
  'help',
  'x_axis',
  'y_axis',
  'lobby_line',
  'band_front',
  'band_polyvalent',
  'band_sniper',
  'legend_low_sample',
  'legend_trend',
  'tooltip_median',
  'tooltip_delta',
  'tooltip_measured',
  'coverage',
  'empty_title',
  'empty_description',
  'fold_tape',
  'tape_no_role',
] as const

export function getSquadRangeRolesText(locale: Locale, grandeur: GrandeurProfil = 'portee') {
  const prefixe = PREFIXE_GRANDEUR[grandeur]
  const m = (suffixe: string, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, `${prefixe}.${suffixe}` as SquadManifestKey, locale, values)
  const bandes: Record<RoleDePortee, string> = {
    front: m('band_front'),
    polyvalent: m('band_polyvalent'),
    sniper: m('band_sniper'),
  }
  return {
    cardTitle: m('card_title'),
    sectionLabel: m('section_label'),
    help: (floor: number) => m('help', { floor }),
    xAxis: m('x_axis'),
    yAxis: m('y_axis'),
    lobbyLine: m('lobby_line'),
    bandes,
    legendLowSample: (floor: number) => m('legend_low_sample', { floor }),
    legendTrend: (n: number) => m('legend_trend', { n }),
    tooltipMedian: (mValue: string) => m('tooltip_median', { m: mValue }),
    tooltipDelta: (delta: string) => m('tooltip_delta', { delta }),
    tooltipMeasured: (n: number) => m('tooltip_measured', { n }),
    coverage: (measured: number, total: number) => m('coverage', { measured, total }),
    emptyTitle: m('empty_title'),
    emptyDescription: m('empty_description'),
    foldTape: m('fold_tape'),
    tapeNoRole: m('tape_no_role'),
  }
}

export type SquadRangeRolesText = ReturnType<typeof getSquadRangeRolesText>
