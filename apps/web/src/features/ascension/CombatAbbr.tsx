/**
 * CombatAbbr — abréviation OC / DR avec tooltip explicatif (B6).
 *
 * OC (Conversion offensive) et DR (Résistance défensive) sont affichés en
 * abrégé dans les grilles compactes (PatternContextGrid, SquadVsSoloCard). Le
 * libellé complet localisé vient de `AscensionText.lusrComponent` — source
 * unique, jamais de string en dur ici.
 */
import { Tooltip } from '@/components/ui/tooltip'
import type { AscensionText } from './i18n'

type CombatMetric = 'oc' | 'dr'

export function CombatAbbr({ metric, t }: { metric: CombatMetric; t: AscensionText }) {
  const abbr = metric === 'oc' ? 'OC' : 'DR'
  const full =
    metric === 'oc'
      ? t.lusrComponent.offensive_conversion
      : t.lusrComponent.defensive_resistance
  return (
    <Tooltip content={full}>
      <span className="cursor-help underline decoration-dotted underline-offset-2">{abbr}</span>
    </Tooltip>
  )
}
