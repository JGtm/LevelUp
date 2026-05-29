/**
 * SessionStatsRadar — radar présentationnel réutilisable à N axes pour UNE session.
 *
 * Le RadarChart du projet normalise chaque axe sur 0..100 ; pour une seule session
 * (1 polygone) on normalise chaque axe contre un PALIER de référence fixe (`cap`),
 * et on affiche la valeur BRUTE (`raw`) au survol (meta.raw_by_axis). Le radar
 * devient donc un "profil indicatif" — choix validé avec l'utilisateur.
 */
import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'

export interface StatRadarAxis {
  key: string
  label: string
  raw: number
  /** Palier de référence (= 100 % de l'axe). */
  cap: number
}

interface Props {
  title: string
  axes: StatRadarAxis[]
  seriesName: string
  height?: number
}

export function SessionStatsRadar({ title, axes, seriesName, height = 280 }: Props) {
  const series: RadarSeriesPayload[] =
    axes.length === 0
      ? []
      : [
          {
            key: 'session',
            axes: axes.map((a) => ({
              axis: a.key,
              value: a.cap > 0 ? Math.max(0, Math.min(100, (a.raw / a.cap) * 100)) : 0,
              raw: a.raw,
            })),
            meta: { raw_by_axis: Object.fromEntries(axes.map((a) => [a.key, a.raw])) },
          },
        ]

  const axisLabels = Object.fromEntries(axes.map((a) => [a.key, a.label]))

  return (
    <RadarChart
      title={title}
      series={series}
      axisLabels={axisLabels}
      height={height}
      seriesNameResolver={() => seriesName}
    />
  )
}
