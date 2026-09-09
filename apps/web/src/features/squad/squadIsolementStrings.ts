/**
 * squadIsolementStrings — les libellés du nuage « isolement x couverture » (item 7.7),
 * résolus depuis le manifest i18n `squad.isolement.*` (source unique FR/EN, ADR 0003).
 *
 * Même forme que `squadEchangeStrings` : un objet `t` ergonomique (statiques + fonctions
 * d'interpolation ICU). Ne pas remettre de littéral FR/EN en dur ici : tout vit dans
 * `lib/i18n/manifests/squad.toml`.
 */
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import type { Locale } from '@/lib/i18n/locale'

import type { QuadrantIsolement } from './squadIsolement.logic'

export function getSquadIsolementText(locale: Locale) {
  const m = (key: SquadManifestKey, values?: Record<string, unknown>) =>
    formatMessage(squadManifest, key, locale, values)

  const quadrantKeys: Record<QuadrantIsolement, SquadManifestKey> = {
    procheCouvert: 'squad.isolement.quadrant_proche_couvert',
    loinCouvert: 'squad.isolement.quadrant_loin_couvert',
    procheSeul: 'squad.isolement.quadrant_proche_seul',
    loinSansSecours: 'squad.isolement.quadrant_loin_sans_secours',
  }

  return {
    sectionTitle: m('squad.isolement.section_title'),
    sectionLabel: m('squad.isolement.section_label'),
    xAxis: m('squad.isolement.x_axis'),
    yAxis: m('squad.isolement.y_axis'),
    definition: m('squad.isolement.definition'),
    floor: (sessionFloor: number, sampleFloor: number) =>
      m('squad.isolement.floor', { sessionFloor, sampleFloor }),
    lowSample: m('squad.isolement.low_sample'),
    tooltip: (v: {
      gamertag: string
      session: string
      isoRate: string
      isoBrut: number
      isoN: number
      covRate: string
      covBrut: number
      covN: number
    }) => m('squad.isolement.tooltip', { ...v }),
    /** Tooltip du GROS point par joueur (D4, lot C3) : la médiane toutes sessions
     *  confondues, sans session à nommer (distinct de `tooltip` ci-dessus). */
    tooltipMedian: (v: { gamertag: string; n: number; isoRate: string; covRate: string }) =>
      m('squad.isolement.tooltip_median', { ...v }),
    quadrant: (q: QuadrantIsolement) => m(quadrantKeys[q]),
    /** Rebranche `quadrantDuPoint` (lot C3) : nomme le quadrant d'UN point dans son
     *  tooltip — les quatre libellés eux-mêmes étaient déjà affichés dans les coins
     *  (`quadrant` ci-dessus, consommé par `markArea`). */
    pointQuadrant: (q: QuadrantIsolement) => m('squad.isolement.point_quadrant', { quadrant: m(quadrantKeys[q]) }),
    emptyTitle: m('squad.isolement.empty_title'),
    emptyDescription: m('squad.isolement.empty_description'),
  }
}

export type SquadIsolementText = ReturnType<typeof getSquadIsolementText>
