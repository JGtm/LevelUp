/**
 * SessionFragsRadar — radar 3 axes, MOYENNES PAR MATCH (aligné page Escouade) :
 *   - Folie meurtrière (moy.) : moyenne des MaxKillingSpree par match de la session.
 *   - Tirs à la tête / match : moyenne des headshot_kills par match.
 *   - Œil de lynx / match : moyenne des perfect_kills (medal Eagle Eye) par match.
 *
 * Agrégats fournis par le backend (SessionCompareEntry.avg_* / *_per_match),
 * calculés comme `extKPIAcc.applyTo` (analysis/squad_breakdown.go) = sum/n.
 *
 * Paliers de référence FIXES (un "excellent par match", esprit des seuils Escouade).
 * Volontairement PAS de cap dynamique : un cap qui se cale sur la valeur observée
 * sature toujours l'axe à 75-100 % et détruit toute lecture comparative.
 * Caps calibrables ci-dessous si le ressenti terrain évolue.
 */
import { useMemo } from 'react'

import type { SessionCompareEntry } from '@/lib/api/types'

import { SessionStatsRadar, type StatRadarAxis } from './SessionStatsRadar'
import { useSessionT } from './_shared'

/** Paliers fixes "excellent par match" — un bon match top-frag tutoie ces valeurs. */
const CAP = { spree: 10, headshots: 12, perfect: 5 }

interface Props {
  title: string
  entry: SessionCompareEntry | null
  height?: number
}

export function SessionFragsRadar({ title, entry, height }: Props) {
  const t = useSessionT()

  const axes = useMemo<StatRadarAxis[]>(() => {
    if (!entry) return []
    const spree = entry.avg_max_killing_spree
    const headshots = entry.headshots_per_match
    const perfect = entry.perfect_kills_per_match
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
