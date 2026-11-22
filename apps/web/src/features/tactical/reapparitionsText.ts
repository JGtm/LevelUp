/**
 * reapparitionsText — les mots de la lecture « Réapparitions adverses » (manifeste `tactical.reapp.*`) :
 * pilules, aide ⓘ, barres comparées, légende, états vides. Fusionné dans `getTacticalText` (i18n.ts) ; un
 * fichier à part pour ne pas grossir celui des autres surfaces de l'onglet.
 *
 * Les comptes arrivent en nombres : l'accord (singulier / pluriel) est celui du message ICU, jamais d'un
 * calcul ici.
 */
import type { TacticalManifestKey } from '@/lib/i18n/generated/tactical'

type Formateur = (key: TacticalManifestKey, values?: Record<string, unknown>) => string

/** Les deux planchers d'une répartition, tels que le serveur les publie. */
export interface PlanchersTexte {
  floorMatchs: number
  floorN: number
}

export function getReapparitionsText(m: Formateur) {
  const regle = (n: number) => m('tactical.reapp.rule', { n })
  return {
    reappQuestion: m('tactical.reapp.question') as string,
    reappPillHeld: m('tactical.reapp.pill_held'),
    reappHeldNone: m('tactical.reapp.held_none'),
    reappHeldOffFloor: (zone: string) => m('tactical.reapp.held_off_floor', { zone }),
    reappHeldOffFloorHint: m('tactical.reapp.held_off_floor_hint'),
    reappPlanLabel: (map: string) => m('tactical.reapp.plan_label', { map }),
    reappPlanInfo: (joueurs: number, matchs: number, reapparitions: number) =>
      m('tactical.reapp.plan_info', { rule: regle(joueurs), matchs, reapparitions }),
    reappMeta: (matchs: number, n: number) => m('tactical.reapp.meta', { matchs, n }),
    reappHeldCaption: m('tactical.reapp.held_caption'),
    reappHeldCounts: (matchs: number, n: number) => m('tactical.reapp.held_counts', { matchs, n }),
    reappChartTitle: (zone: string) => m('tactical.reapp.chart_title', { zone }),
    reappChartTitleReference: m('tactical.reapp.chart_title_reference'),
    reappChartInfo: (joueurs: number, n: number, matchs: number, planchers: PlanchersTexte) =>
      m('tactical.reapp.chart_info', { rule: regle(joueurs), n, matchs, ...planchers }),
    reappLegendGradient: (max: string) => m('tactical.reapp.legend_gradient', { max }),
    reappLegendGrey: m('tactical.reapp.legend_grey'),
    reappLegendGreyNoZone: m('tactical.reapp.legend_grey_no_zone'),
    reappLegendHeld: (n: number) => m('tactical.reapp.legend_held', { n }),
    reappLegendDelta: m('tactical.reapp.legend_delta'),
    reappBarHeld: (zone: string, n: number, matchs: number) => m('tactical.reapp.bar_held', { zone, n, matchs }),
    reappBarReference: (n: number, matchs: number) => m('tactical.reapp.bar_reference', { n, matchs }),
    reappLabelHeld: (zone: string) => m('tactical.reapp.label_held', { zone }),
    reappDeltaAbove: (n: number) => m('tactical.reapp.delta_above', { n }),
    reappDeltaBelow: (n: number) => m('tactical.reapp.delta_below', { n }),
    reappDeltaEqual: m('tactical.reapp.delta_equal'),
    reappEmptyFloorTitle: (zone: string) => m('tactical.reapp.empty_floor_title', { zone }),
    reappEmptyReferenceTitle: m('tactical.reapp.empty_reference_title'),
    reappEmptyFloorBody: (matchs: number, n: number, planchers: PlanchersTexte) =>
      m('tactical.reapp.empty_floor_body', { matchs, n, ...planchers }),
    reappGaugeMatchs: (n: number, floor: number) => m('tactical.reapp.gauge_matchs', { n, floor }),
    reappGaugeRespawns: (n: number, floor: number) => m('tactical.reapp.gauge_respawns', { n, floor }),
    reappEmptyNoneTitle: m('tactical.reapp.empty_none_title'),
    reappEmptyNoneBody: m('tactical.reapp.empty_none_body'),
    reappEmptyNoMatchTitle: m('tactical.reapp.empty_no_match_title'),
    reappEmptyNoMatchBody: m('tactical.reapp.empty_no_match_body'),
    reappEmptyTeamTitle: m('tactical.reapp.empty_team_title'),
    reappEmptyTeamBody: m('tactical.reapp.empty_team_body'),
    reappEmptyShapeTitle: m('tactical.reapp.empty_shape_title'),
    reappEmptyShapeBody: m('tactical.reapp.empty_shape_body'),
    reappEmptyNoZonesTitle: m('tactical.reapp.empty_no_zones_title'),
    reappEmptyNoZonesBody: m('tactical.reapp.empty_no_zones_body'),
    reappSubReference: m('tactical.reapp.sub_reference'),
    reappPlanEmpty: (zone: string) => m('tactical.reapp.plan_empty', { zone }),
    reappLoading: m('tactical.reapp.loading'),
    reappError: m('tactical.reapp.error'),
  }
}
