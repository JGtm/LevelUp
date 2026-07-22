/**
 * SquadWeaponAccuracyBarsChart — « Précision par rôle » comparatif multi-joueurs (Escouade).
 *
 * Pendant précision de SquadWeaponKillsChart (teammates.09) : barres horizontales groupées,
 * 1 barre par joueur et par rôle d'arme, longueur = précision %. Hauteur dynamique identique
 * (`max(350, min(800, n * 38))`).
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadWeaponAccuracy, SquadWeaponAccuracyBar } from '@/lib/api/types'
import {
  buildSquadWeaponAccuracyBarsOption,
  type SquadWeaponAccuracyBarsOpts,
} from './charts/squadWeaponAccuracyBarsChart'

interface SquadWeaponAccuracyBarsChartProps extends SquadWeaponAccuracyBarsOpts {
  title?: string
  emptyMessage?: string
  data: SquadWeaponAccuracy | null | undefined
  /** Étire la carte pour remplir la cellule grille (aligne la hauteur sur le bloc frère). */
  fillHeight?: boolean
}

export function SquadWeaponAccuracyBarsChart({
  data,
  title,
  emptyMessage,
  fillHeight = false,
  ...opts
}: SquadWeaponAccuracyBarsChartProps) {
  const series = useMemo<ChartSeries<SquadWeaponAccuracyBar>[]>(() => {
    const bars = data?.bars ?? []
    return bars.length > 0 ? [{ key: 'weapon-accuracy', datapoints: bars }] : []
  }, [data])
  const buildOption = useCallback(
    () => buildSquadWeaponAccuracyBarsOption(data, opts),
    [data, opts],
  )
  const n = data?.bars?.length ?? 0
  const height = Math.max(350, Math.min(800, n * 38))
  return (
    <ChartCard title={title} series={series} buildOption={buildOption} height={height} emptyMessage={emptyMessage} fluid={fillHeight} />
  )
}
