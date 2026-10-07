/**
 * HabitCard — « Contrôle des ressources, soirée après soirée » (Escouade › Emprise, bloc « Par
 * rapport à d'habitude », à gauche ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26,
 * décision D5 ; maquette de l'onglet, `renderHabChart('habPrises', …)`).
 *
 * Un graphe (S6 : mêmes mesure et échelle) : notre part des prises des bonus et des armes
 * spéciales, une soirée de la composition par point, ce soir à droite dans une colonne grisée ;
 * une soirée sans aucun des modes de ce soir reste un point, gris, que la courbe enjambe, raison
 * au survol, hors médiane ; la médiane des soirées précédentes comparables en pointillé fin de la
 * couleur de chaque courbe (trois au moins), le trait 50 %, la valeur au bout ; les dates sous
 * l'axe, « ce soir » en gras (`buildHabitOption`). Sans aucune part sur aucune soirée : le bloc
 * placeholder. Légende en pied de carte, centrée.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'

import { eveningDate } from '../objectif/objectif.logic'
import { ObjectifFrame, ObjectifLegend, ObjectifPlaceholder } from '../objectif/ObjectifFrame'
import { buildHabitOption, resolveHabitColors } from './empriseCharts'
import type { EmpriseText } from './empriseStrings'
import type { HabitPoint, HabitView } from './habit.logic'
import { resourceInk } from './resourceColors'

/** Hauteur du graphe (maquette : 520 × 220). */
const HABIT_HEIGHT = 220

/** Le gris des soirées hors comparaison : l'encre atténuée du thème (celle que lit le graphe). */
const MUTED_INK = 'var(--muted-foreground)' // color-allow: encre atténuée du thème, même valeur que resolveHabitColors().muted

interface Props {
  view: Exclude<HabitView, { kind: 'none' }>
  locale: Locale
  t: EmpriseText
}

export function HabitCard({ view, locale, t }: Props) {
  const points = view.kind === 'chart' ? view.points : null
  const resources = view.kind === 'chart' ? view.resources : null
  const legend = useMemo(
    () =>
      resources && points ? (
        <ObjectifLegend
          ariaLabel={t.habit.title}
          items={[
            ...resources.map((r) => ({ kind: 'square' as const, label: t.resources[r].label, color: resourceInk(r) })),
            ...(points.some((p) => !p.comparable) ? [{ kind: 'dot' as const, label: t.habit.notComparable, color: MUTED_INK }] : []),
            { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
          ]}
        />
      ) : null,
    [resources, points, t],
  )

  const series = useMemo<ChartSeries<HabitPoint>[]>(
    () => (points ? [{ key: 'emprise-habit', datapoints: points }] : []),
    [points],
  )
  const buildOption = useCallback(
    () =>
      view.kind === 'chart'
        ? buildHabitOption(view.resources, view.points, view.medians, resolveHabitColors(), {
            resourceLabel: (r) => t.resources[r]?.label ?? r,
            pctFmt: t.pctFmt,
            pctIntFmt: t.pctIntFmt,
            dateOf: (iso) => eveningDate(iso, locale),
            tonight: t.habit.tonight,
            eveningOf: t.habit.eveningOf,
            pointTip: t.habit.pointTip,
            medianTip: t.habit.medianTip,
            notComparableTip: t.habit.notComparableTip,
          })
        : {},
    [view, t, locale],
  )

  if (view.kind === 'empty') {
    return (
      <ObjectifFrame title={t.habit.title} info={t.habit.info} testId="emprise-habit">
        <ObjectifPlaceholder notice={t.habit.empty} testId="emprise-habit-note" />
      </ObjectifFrame>
    )
  }

  return (
    <div className="h-full min-w-0" data-testid="emprise-habit">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.habit.title}
            <InfoTooltip content={t.habit.info} />
          </span>
        }
        series={series}
        height={HABIT_HEIGHT}
        renderer="svg"
        buildOption={buildOption}
        legend={legend}
      />
    </div>
  )
}
