/**
 * SessionFragsRadar — radar 3 axes, AGRÉGATS DE SESSION (le "compte" de la session) :
 *   - Folie meurtrière (max) : plus longue série de frags atteinte sur la session.
 *   - Tirs à la tête : TOTAL des frags à la tête de la session.
 *   - Frags parfaits : TOTAL des frags parfaits (PerfectKills) de la session.
 *
 * Valeurs brutes fournies par le backend (SessionCompareEntry.max_killing_spree /
 * total_headshot_kills / total_perfect_kills). Le tooltip affiche ces comptes bruts
 * (rawInTooltip) ; le radar normalise seulement la FORME contre des paliers fixes.
 *
 * Paliers FIXES (niveau "excellente session") — surtout pas de cap dynamique qui se
 * calerait sur la valeur et saturerait toujours l'axe.
 */
import { useMemo } from 'react'

import type { SessionCompareEntry } from '@/lib/api/types'

import { SessionStatsRadar, type StatRadarAxis } from './SessionStatsRadar'
import { useSessionT } from './_shared'

/** Paliers fixes "excellente session" pour la FORME du radar (le tooltip montre le compte brut). */
const CAP = { spree: 12, headshots: 40, perfect: 12 }

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
      { key: 'spree',     label: t('session.detail.radar_spree'),     raw: spree     ?? 0, cap: CAP.spree },
      { key: 'headshots', label: t('session.detail.radar_headshots'), raw: headshots ?? 0, cap: CAP.headshots },
      { key: 'perfect',   label: t('session.detail.radar_perfect'),   raw: perfect   ?? 0, cap: CAP.perfect },
    ]
  }, [entry, t])

  return (
    <SessionStatsRadar
      title={title}
      axes={axes}
      seriesName={t('session.detail.chart_frags_radar_series')}
      height={height}
      rawInTooltip
    />
  )
}
