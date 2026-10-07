/**
 * ObjectiveEveningsCard — « Rapport de force, soirée après soirée » (Escouade › Emprise,
 * section Objectif ; lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, D6 / D7,
 * maquette C3EW).
 *
 * Un graphe, trois courbes de rôle, une soirée par point : les soirées précédentes de la
 * composition d'au moins trois matchs à objectif (tous modes, drapeau neutre exclu — calcul Go),
 * puis ce soir à droite dans une colonne grisée ; la médiane des soirées précédentes en
 * pointillé fin de la couleur de chaque courbe ; le trait 50 % ; la valeur au bout ; sous chaque
 * soirée, la date, la barre victoires / défaites, « x sur y » et les modes (abréviations
 * expliquées dans la légende). Une soirée sous trois matchs à objectif n'a pas de point : la carte
 * le dit dans le bloc placeholder (`EmptyStateNotice` tiretée), sans graphe ; une première soirée à
 * objectif n'a pas d'historique : idem.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { roleToken } from '@/features/_shared/usage/usageMetricKinds'
import { tokenCssVar } from '@/lib/accessibility'
import type { SquadObjectiveHistory } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { buildEveningsView, eveningDate, familyMix, type EveningPoint } from './objectif.logic'
import { buildEveningsOption } from './eveningsChart'
import { resolveObjectifColors } from './objectifCharts'
import { ObjectifFrame, ObjectifLegend, ObjectifPlaceholder, type ObjectifLegendItem } from './ObjectifFrame'
import type { ObjectifNotice, ObjectifText } from './objectifStrings'

/** Hauteur du graphe (maquette : 760 × 290). */
const EVENINGS_HEIGHT = 290

interface Props {
  history: SquadObjectiveHistory
  familyLabel: (family: string) => string
  locale: Locale
  t: ObjectifText
}

export function ObjectiveEveningsCard({ history, familyLabel, locale, t }: Props) {
  const view = useMemo(() => buildEveningsView(history), [history])
  const abbr = t.evenings.familyAbbr
  const mixOf = useCallback((p: EveningPoint) => familyMix(p.families, abbr), [abbr])

  const points = view.kind === 'chart' ? view.points : null
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.evenings.title}
        items={[
          { kind: 'square', label: t.roles.take, color: tokenCssVar(roleToken('take')) },
          { kind: 'square', label: t.roles.defend, color: tokenCssVar(roleToken('defend')) },
          { kind: 'square', label: t.roles.hold, color: tokenCssVar(roleToken('hold')) },
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
          { kind: 'median', label: t.evenings.median },
          { kind: 'pair', label: t.evenings.winsLosses, colors: [tokenCssVar('outcome-win'), tokenCssVar('outcome-loss')] },
          ...abbreviationItems(points ?? [], abbr, familyLabel, t),
        ]}
      />
    ),
    [t, points, abbr, familyLabel],
  )

  const series = useMemo<ChartSeries<EveningPoint>[]>(
    () => (points ? [{ key: 'objective-evenings', datapoints: points }] : []),
    [points],
  )
  const buildOption = useCallback(
    () =>
      view.kind === 'chart'
        ? buildEveningsOption(view.points, view.medians, resolveObjectifColors(), {
            roles: t.roles,
            pctFmt: t.pctFmt,
            tonight: t.evenings.tonight,
            dateOf: (iso) => eveningDate(iso, locale),
            outOfFmt: t.evenings.outOfFmt,
            mixOf,
            pointTip: t.evenings.pointTip,
            bandTip: t.evenings.bandTip,
            eveningOf: t.evenings.eveningOf,
            medianTip: (role, value) => `${role}\n${t.evenings.median} : ${value}`,
          })
        : {},
    [view, t, locale, mixOf],
  )

  if (view.kind === 'belowMinimum') {
    return <EveningsPlaceholderCard notice={t.evenings.belowMinimum(view.matches, view.below, view.withObjective)} t={t} />
  }
  if (view.kind === 'noHistory') {
    const c = view.current
    const pct = (v: number | null) => (v == null ? '—' : t.pctFmt(v))
    const notice = t.evenings.noHistory(pct(c.shares.take), pct(c.shares.defend), pct(c.shares.hold), c.wins, c.matches)
    return <EveningsPlaceholderCard notice={notice} t={t} />
  }

  return (
    <div className="min-w-0" data-testid="objective-evenings">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.evenings.title}
            <InfoTooltip content={t.evenings.info} />
          </span>
        }
        series={series}
        height={EVENINGS_HEIGHT}
        renderer="svg"
        buildOption={buildOption}
        legend={legend}
      />
    </div>
  )
}

/** La carte sans graphe : le bloc placeholder dit pourquoi (soirée sous le minimum, pas d'historique). */
function EveningsPlaceholderCard({ notice, t }: { notice: ObjectifNotice; t: ObjectifText }) {
  return (
    <ObjectifFrame title={t.evenings.title} info={t.evenings.info} testId="objective-evenings">
      <ObjectifPlaceholder notice={notice} testId="objective-evenings-note" />
    </ObjectifFrame>
  )
}

/**
 * La clé des abréviations de modes écrites sous chaque soirée, en entrées de légende, dans l'ordre
 * alphabétique des abréviations (maquette : « B : Bases », « D : Drapeau »).
 */
function abbreviationItems(
  points: EveningPoint[],
  abbr: Record<string, string>,
  familyLabel: (family: string) => string,
  t: ObjectifText,
): ObjectifLegendItem[] {
  const families = [...new Set(points.flatMap((p) => p.families.map((f) => f.family)))]
  return families
    .map((f) => ({ a: abbr[f] ?? f, name: familyLabel(f) }))
    .sort((x, y) => x.a.localeCompare(y.a))
    .map((x) => ({ kind: 'text' as const, label: t.evenings.abbrItem(x.a, x.name) }))
}
