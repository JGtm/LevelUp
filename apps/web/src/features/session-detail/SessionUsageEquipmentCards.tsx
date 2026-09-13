/**
 * SessionUsageEquipmentCards — LES TROIS CARTES DE L'ÉQUIPEMENT de la page Sessions :
 * « Cadences par match », « Parts et parités », « Régularité match par match ».
 *
 * UNE VUE = UNE CARTE (demande utilisateur du 2026-09-13). Les trois vivaient dans une
 * seule carte « Usages d'équipement » et se lisaient comme un empilement : trois
 * questions différentes (à quelle fréquence ? quelle part ? régulièrement ?) sous un
 * seul bandeau, avec une seule aide pour les trois. Séparées, chacune porte son titre,
 * son aide et rien d'autre.
 *
 * LA COUVERTURE NE SE TRIPLE PAS : « Matchs mesurés N/M » n'est écrit que sur la
 * première carte — c'est le même dénominateur pour les trois, l'écrire trois fois ne
 * l'aurait pas rendu plus vrai.
 *
 * Aucun calcul ici : projections dans `@/features/_shared/usage/`, chrome partagé dans
 * `SessionUsageShared.tsx`.
 */
import { useMemo } from 'react'

import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'

import { buildCadenceGrid } from '@/features/_shared/usage/usageGrids'
import { equipmentMetrics, metricLabel } from '@/features/_shared/usage/usageMetricKinds'
import { buildRegularityBand } from '@/features/_shared/usage/usageRegularityBandModel'
import { UsageGaugeGrid, UsageRegularityBand } from '@/features/_shared/usage/UsageForms'

import {
  bandAboveCaption,
  cardTitleAdornment,
  metricGaugeRows,
  useGridInks,
  type CardProps,
} from './SessionUsageShared'

export function EquipmentCards({ usage, meLabel, t, locale, compact }: CardProps) {
  const inks = useGridInks()
  const metrics = useMemo(() => equipmentMetrics(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const cadenceGrid = useMemo(
    () => buildCadenceGrid({ metrics, squadPlayers, meLabel, t, locale, ...inks }),
    [metrics, squadPlayers, meLabel, t, locale, inks],
  )
  const gaugeRows = useMemo(() => metricGaugeRows(metrics, usage, t, locale), [metrics, usage, t, locale])
  if (metrics.length === 0) return null
  const measured = t.measuredFmt(usage.matches_measured, usage.matches_total)

  return (
    <>
      {cadenceGrid && (
        <SectionCard
          title={t.viewCadences}
          label={t.viewCadences}
          titleAdornment={cardTitleAdornment(measured, t.cardHintCadences)}
        >
          <div className="px-3 pb-3 pt-3">
            <ValueGrid model={cadenceGrid} dense={compact} />
          </div>
        </SectionCard>
      )}

      <SectionCard
        title={t.viewShares}
        label={t.viewShares}
        titleAdornment={cardTitleAdornment(null, t.cardHintShares)}
      >
        <div className="px-3 pb-3 pt-3">
          <UsageGaugeGrid rows={gaugeRows} t={t} dense={compact} />
        </div>
      </SectionCard>

      <SectionCard
        title={t.viewRegularity}
        label={t.viewRegularity}
        titleAdornment={cardTitleAdornment(null, t.cardHintRegularity)}
      >
        <div className="space-y-1.5 px-3 pb-3 pt-3">
          {metrics.map((m) => (
            <UsageRegularityBand
              key={m.key}
              label={metricLabel(m.key, t)}
              cells={buildRegularityBand(m.per_match, usage.team_parity_pct, t, locale)}
              caption={bandAboveCaption(m, usage.matches_measured, t)}
              dense={compact}
            />
          ))}
        </div>
      </SectionCard>
    </>
  )
}
