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
 * DEUX AJUSTEMENTS DU 2026-09-21 (lot A2) :
 *   - « Matchs mesurés N/M » NE S'ÉCRIT PLUS (retour utilisateur) : le compteur doublait
 *     la couverture déjà dite par le reste de la page, en tête d'une carte sur deux ;
 *   - « Régularité » PASSE À DROITE de « Parts et parités », sur une rangée à deux
 *     colonnes, « Cadences » restant au-dessus. En COLONNE DIVISÉE (`compact`, drawer de
 *     comparaison ouvert) les cartes restent EMPILÉES : deux demi-colonnes dans une
 *     demi-colonne ne se lisent pas.
 *
 * Aucun calcul ici : projections dans `@/features/_shared/usage/`, chrome partagé dans
 * `SessionUsageShared.tsx`.
 */
import { useMemo } from 'react'

import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'

import { UsageBandLegend } from '@/features/_shared/usage/UsageBandLegend'
import { UsageGaugeGrid } from '@/features/_shared/usage/UsageForms'
import { UsageRegularityBand } from '@/features/_shared/usage/UsageRegularityBand'
import { buildCadenceGrid } from '@/features/_shared/usage/usageGrids'
import { usageCardTitle } from '@/features/_shared/usage/usageCardTitle'
import { equipmentMetrics, metricLabel } from '@/features/_shared/usage/usageMetricKinds'
import { buildRegularityBand } from '@/features/_shared/usage/usageRegularityBandModel'

import { bandAboveCaption, metricGaugeRows, useGridInks, type CardProps } from './SessionUsageShared'
import { sessionUsageCardsShown } from './sessionSectionVisibility'

export function EquipmentCards({ usage, meLabel, t, locale, compact }: CardProps) {
  const inks = useGridInks()
  const metrics = useMemo(() => equipmentMetrics(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const cadenceGrid = useMemo(
    () => buildCadenceGrid({ metrics, squadPlayers, meLabel, t, locale, ...inks }),
    [metrics, squadPlayers, meLabel, t, locale, inks],
  )
  const gaugeRows = useMemo(() => metricGaugeRows(metrics, usage, t, locale), [metrics, usage, t, locale])
  // La porte de la carte est celle que le TITRE DE SECTION interroge
  // (`sessionSectionVisibility`) : une seule ecriture, sinon le titre « Frags et usages »
  // finirait au-dessus du vide.
  if (!sessionUsageCardsShown(usage).equipment) return null

  return (
    <>
      {cadenceGrid && (
        <SectionCard
          title={t.viewCadences}
          label={t.viewCadences}
          titleAdornment={usageCardTitle(t.cardHintCadences)}
        >
          <div className="px-3 pb-3 pt-3">
            <ValueGrid model={cadenceGrid} dense={compact} />
          </div>
        </SectionCard>
      )}

      <div className={`grid grid-cols-1 gap-4${compact ? '' : ' lg:grid-cols-2'}`}>
        <SectionCard
          title={t.viewShares}
          label={t.viewShares}
          titleAdornment={usageCardTitle(t.cardHintShares)}
        >
          <div className="flex flex-1 flex-col justify-center px-3 pb-3 pt-3">
            <UsageGaugeGrid rows={gaugeRows} t={t} dense={compact} />
          </div>
        </SectionCard>

        <SectionCard
          title={t.viewRegularity}
          label={t.viewRegularity}
          titleAdornment={usageCardTitle(t.cardHintRegularity)}
        >
          <div className="flex flex-1 flex-col justify-center px-3 pb-3 pt-3">
            <div className="space-y-1.5">
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
            {/* UNE légende pour TOUTES les bandes de la carte — plus une phrase par ligne. */}
            <UsageBandLegend t={t} />
          </div>
        </SectionCard>
      </div>
    </>
  )
}
