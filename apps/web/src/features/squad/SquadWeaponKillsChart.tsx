/**
 * SquadWeaponKillsChart — wrapper des barres groupées par joueur (Outils de destruction,
 * Mécaniques de frag).
 *
 * Hauteur dynamique : `max(350, n_lignes * 38)` — SANS plafond depuis le lot L2 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : les Outils de destruction n'ont plus de
 * « Autres armes », chaque ligne garde sa hauteur de lecture.
 */
import { useCallback, useMemo, type ReactNode } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  buildSquadWeaponKillsOption,
  type SquadBarRow,
  type SquadBarRows,
  type SquadWeaponKillsOpts,
} from './charts/squadWeaponKillsChart'

interface SquadWeaponKillsChartProps extends SquadWeaponKillsOpts {
  title?: ReactNode
  emptyMessage?: string
  data: SquadBarRows | null | undefined
  /** Légende hors canvas (pied de carte). */
  legend?: ReactNode
}

export function SquadWeaponKillsChart({ data, title, emptyMessage, legend, ...opts }: SquadWeaponKillsChartProps) {
  const series = useMemo<ChartSeries<SquadBarRow>[]>(() => {
    const rows = data?.rows ?? []
    return rows.length > 0 ? [{ key: 'weapon-kills', datapoints: rows }] : []
  }, [data])
  const buildOption = useCallback(
    () => buildSquadWeaponKillsOption(data, opts),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [data, opts.colorByPlayer, opts.valueLabel, opts.valueText],
  )
  const n = data?.rows?.length ?? 0
  const height = Math.max(350, n * 38)
  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={height}
      emptyMessage={emptyMessage}
      legend={legend}
    />
  )
}
