/**
 * ResourceFilCard — « Contrôle des ressources au fil de la session » (Escouade › Emprise, bloc
 * « Bilan de la soirée », à droite de « Contrôle des ressources », même hauteur ; lot L5.2 du
 * plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette de l'onglet).
 *
 * Un graphe : une courbe cumulée par ressource du bilan (couleurs `resource-*`), les parts par
 * match en petits points pâles (taille = volume), point final grossi et valeur au bout, trait
 * 50 %, puis sous chaque match l'heure, la carte et la bande de résultats avec l'encoche de
 * dominance (`empriseCharts.ts`). Légende en pied de carte, centrée (S2).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import { formatMatchTime } from '../formes/format'
import { ObjectifLegend } from '../objectif/ObjectifFrame'
import type { ResourceFil, ResourceFilMatch } from './emprise.logic'
import { buildResourceFilOption, resolveEmpriseFilColors } from './empriseCharts'
import type { EmpriseText } from './empriseStrings'
import { resourceInk } from './resourceColors'

/** Hauteur du graphe (maquette : 520 × 246). */
const FIL_HEIGHT = 246

interface Props {
  fil: ResourceFil
  dominanceLabels: Record<DominanceValue, string>
  locale: Locale
  t: EmpriseText
}

export function ResourceFilCard({ fil, dominanceLabels, locale, t }: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.fil.title}
        items={[
          ...fil.resources.map((r) => ({ kind: 'square' as const, label: t.resources[r].label, color: resourceInk(r) })),
          { kind: 'pair', label: t.fil.winLoss, colors: [tokenCssVar('outcome-win'), tokenCssVar('outcome-loss')] },
          { kind: 'notch', label: t.fil.dominance, color: tokenCssVar(DOMINANCE_COLOR_TOKENS[1]) },
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
        ]}
      />
    ),
    [fil.resources, t],
  )

  const series = useMemo<ChartSeries<ResourceFilMatch>[]>(
    () => (fil.matches.length > 0 ? [{ key: 'emprise-fil', datapoints: fil.matches }] : []),
    [fil.matches],
  )
  const buildOption = useCallback(
    () =>
      buildResourceFilOption(fil, resolveEmpriseFilColors(), {
        resourceLabel: (r) => t.resources[r]?.label ?? r,
        pctFmt: t.pctFmt,
        pctIntFmt: t.pctIntFmt,
        timeOf: (iso) => formatMatchTime(iso, locale),
        outcomeOf: (m) => (m.outcome ? t.outcomeLower[m.outcome] : null),
        resultOf: (m) => (m.outcome ? `${t.outcome[m.outcome]}${m.score ? ` ${m.score}` : ''}` : null),
        dominanceLabel: (d) => dominanceLabels[d],
        pointTip: t.fil.pointTip,
        endTip: t.fil.endTip,
        bandTip: t.fil.bandTip,
      }),
    [fil, t, locale, dominanceLabels],
  )

  return (
    <div className="h-full min-w-0" data-testid="emprise-fil">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.fil.title}
            <InfoTooltip content={t.fil.info} />
          </span>
        }
        series={series}
        height={FIL_HEIGHT}
        // fluid : la carte s'étire à la hauteur de sa voisine (« Contrôle des ressources ») et
        // le graphe la remplit, centré (S2).
        fluid
        renderer="svg"
        buildOption={buildOption}
        legend={legend}
      />
    </div>
  )
}
