/**
 * SessionFragsRadar — radar 3 axes : Folie meurtrière (max) · Tirs à la tête (total) ·
 * Frags parfaits (total) sur la session. Agrégats fournis par le backend
 * (SessionCompareEntry). Paliers de référence fixes + valeur brute au survol (SessionStatsRadar).
 */
import { useMemo } from 'react'

import type { SessionCompareEntry } from '@/lib/api/types'

import { SessionStatsRadar, type StatRadarAxis } from './SessionStatsRadar'
import { useSessionT } from './_shared'

// Paliers de référence (100 % de l'axe) — indicatifs, ajustables.
const CAP = { spree: 15, headshots: 50, perfect: 20 }

interface Props {
  title: string
  entry: SessionCompareEntry | null
  height?: number
}

export function SessionFragsRadar({ title, entry, height }: Props) {
  const t = useSessionT()

  const axes = useMemo<StatRadarAxis[]>(() => {
    if (!entry) return []
    const spree = entry.max_killing_spree
    const headshots = entry.total_headshot_kills
    const perfect = entry.total_perfect_kills
    if (spree == null && headshots == null && perfect == null) return []
    return [
      { key: 'spree', label: t('session.detail.radar_spree'), raw: spree ?? 0, cap: CAP.spree },
      { key: 'headshots', label: t('session.detail.radar_headshots'), raw: headshots ?? 0, cap: CAP.headshots },
      { key: 'perfect', label: t('session.detail.radar_perfect'), raw: perfect ?? 0, cap: CAP.perfect },
    ]
  }, [entry, t])

  return (
    <SessionStatsRadar
      title={title}
      axes={axes}
      seriesName={t('session.detail.chart_frags_radar_series')}
      height={height}
    />
  )
}
