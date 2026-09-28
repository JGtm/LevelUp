/**
 * HabitCard — « Contrôle des ressources, soirée après soirée » (Escouade › Emprise, bloc « Par
 * rapport à d'habitude », à gauche ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26,
 * décision D5 ; maquette de l'onglet, `renderHabChart('habPrises', …)`).
 *
 * Un graphe (S6 : mêmes mesure et échelle) : notre part des prises des bonus et des armes
 * spéciales, une soirée comparable par point, ce soir à droite dans une colonne grisée, la
 * médiane des soirées précédentes en pointillé fin de la couleur de chaque courbe (trois soirées
 * précédentes au moins), le trait 50 %, la valeur au bout ; les dates sous l'axe, « ce soir » en
 * gras (`buildHabitOption`). Sans soirée précédente comparable, la carte le dit et donne les
 * parts de ce soir (même traitement que « Rapport de force, soirée après soirée »). Légende en
 * pied de carte, centrée (S2).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'

import { eveningDate } from '../objectif/objectif.logic'
import { ObjectifFrame, ObjectifLegend, ObjectifNote } from '../objectif/ObjectifFrame'
import { buildHabitOption, resolveHabitColors } from './empriseCharts'
import type { EmpriseText } from './empriseStrings'
import type { HabitPoint, HabitView } from './habit.logic'
import { resourceInk } from './resourceColors'

/** Hauteur du graphe (maquette : 520 × 220). */
const HABIT_HEIGHT = 220

interface Props {
  view: Exclude<HabitView, { kind: 'none' }>
  locale: Locale
  t: EmpriseText
}

export function HabitCard({ view, locale, t }: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.habit.title}
        items={[
          ...view.resources.map((r) => ({ kind: 'square' as const, label: t.resources[r].label, color: resourceInk(r) })),
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
        ]}
      />
    ),
    [view.resources, t],
  )

  const points = view.kind === 'chart' ? view.points : null
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
          })
        : {},
    [view, t, locale],
  )

  if (view.kind === 'noHistory') {
    const list = view.resources
      .flatMap((r) => (view.current.shares[r] == null ? [] : [t.habit.shareItem(t.resources[r].label, t.pctIntFmt(view.current.shares[r] as number))]))
      .join(', ')
    return (
      <ObjectifFrame title={t.habit.title} info={t.habit.info} testId="emprise-habit">
        <ObjectifNote note={t.habit.noHistory(list)} testId="emprise-habit-note" />
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
