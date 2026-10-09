/**
 * ObjectiveSessionFilCard — « Rapport de force au fil de la session » (Escouade ›
 * Emprise, section Objectif ; lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26,
 * maquette C3EW).
 *
 * Un graphe, trois courbes de rôle cumulées (moyenne des parts, chaque match pèse pareil, D7),
 * aux couleurs `objective-role-*` ; les parts de chaque match en petits points pâles (taille =
 * volume) ; le point final grossi et la valeur au bout de chaque courbe ; le trait 50 % ; sous
 * l'axe, l'heure, la bande de résultats (encoche de dominance quand le drapeau existe, S9), la
 * carte et le mode. Sous trois matchs à objectif, la courbe ne se trace pas : la carte garde sa
 * place et dit pourquoi, dans le bloc placeholder (`EmptyStateNotice` tiretée).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import { roleToken } from '@/features/_shared/usage/usageMetricKinds'

import { formatMatchTime } from '../formes/format'
import type { Locale } from '@/lib/i18n/locale'
import { OBJECTIVE_MIN_MATCHES, type FilMatch } from './objectif.logic'
import { buildFilOption, resolveObjectifColors } from './objectifCharts'
import { ObjectifFrame, ObjectifLegend, ObjectifPlaceholder } from './ObjectifFrame'
import type { ObjectifText } from './objectifStrings'

/** Hauteur du graphe (maquette : 480 × 262). */
const FIL_HEIGHT = 262

interface Props {
  matches: FilMatch[]
  familyLabel: (family: string) => string
  dominanceLabels: Record<DominanceValue, string>
  locale: Locale
  t: ObjectifText
}

export function ObjectiveSessionFilCard({ matches, familyLabel, dominanceLabels, locale, t }: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.fil.title}
        items={[
          { kind: 'square', label: t.roles.take, color: tokenCssVar(roleToken('take')) },
          { kind: 'square', label: t.roles.defend, color: tokenCssVar(roleToken('defend')) },
          { kind: 'square', label: t.roles.hold, color: tokenCssVar(roleToken('hold')) },
          { kind: 'pair', label: t.fil.winLoss, colors: [tokenCssVar('outcome-win'), tokenCssVar('outcome-loss')] },
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
        ]}
      />
    ),
    [t],
  )

  const series = useMemo<ChartSeries<FilMatch>[]>(
    () => (matches.length > 0 ? [{ key: 'objective-fil', datapoints: matches }] : []),
    [matches],
  )
  const buildOption = useCallback(
    () =>
      buildFilOption(matches, resolveObjectifColors(), {
        roles: t.roles,
        pctFmt: (v) => t.pctFmt(v),
        countFmt: (v, duration) => (duration ? t.durationFmt(v) : String(Math.round(v * 10) / 10)),
        timeOf: (iso) => formatMatchTime(iso, locale),
        familyLabel,
        contextOf: (m) =>
          t.fil.contextFmt(familyLabel(m.family), m.outcome ? t.outcome[m.outcome] : null, m.score),
        dominanceLabel: (d) => dominanceLabels[d],
        pointTip: t.fil.pointTip,
        bandTip: t.fil.bandTip,
      }),
    [matches, t, locale, familyLabel, dominanceLabels],
  )

  if (matches.length < OBJECTIVE_MIN_MATCHES) {
    return (
      <ObjectifFrame title={t.fil.title} info={t.fil.info} testId="objective-fil">
        <ObjectifPlaceholder notice={t.fil.belowMinimum(matches.length)} testId="objective-fil-note" />
      </ObjectifFrame>
    )
  }

  return (
    <div className="h-full min-w-0" data-testid="objective-fil">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.fil.title}
            <InfoTooltip content={t.fil.info} />
          </span>
        }
        series={series}
        height={FIL_HEIGHT}
        // fluid : la carte s'étire à la hauteur de sa voisine (« Rapport de force par famille
        // de mode ») et le graphe la remplit, centré (S2).
        fluid
        renderer="svg"
        buildOption={buildOption}
        legend={legend}
      />
    </div>
  )
}
