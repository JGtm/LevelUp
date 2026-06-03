/**
 * SessionFdaRadar — radar 3 axes Frags / Morts / Assists (MOYENNES par match).
 * Remplace les barres FDA par-match (choix utilisateur). Paliers de référence
 * fixes par axe ; valeur brute affichée au survol via SessionStatsRadar.
 */
import { useMemo } from 'react'

import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { SessionStatsRadar, type StatRadarAxis } from './SessionStatsRadar'
import { useSessionT } from './_shared'

// Paliers de référence (100 % de l'axe) — niveaux indicatifs, ajustables.
const CAP = { frags: 25, deaths: 15, assists: 12 }

const round1 = (n: number) => Math.round(n * 10) / 10

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionFdaRadar({ title, matches, height }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields
  const t = useSessionT()

  const axes = useMemo<StatRadarAxis[]>(() => {
    if (matches.length === 0) return []
    const lbl = (key: string): string => fields?.[key]?.label ?? key
    const n = matches.length
    const sum = matches.reduce(
      (acc, m) => ({ k: acc.k + m.kills, d: acc.d + m.deaths, a: acc.a + m.assists }),
      { k: 0, d: 0, a: 0 },
    )
    return [
      { key: 'frags', label: lbl('kills'), raw: round1(sum.k / n), cap: CAP.frags },
      { key: 'deaths', label: lbl('deaths'), raw: round1(sum.d / n), cap: CAP.deaths },
      { key: 'assists', label: lbl('assists'), raw: round1(sum.a / n), cap: CAP.assists },
    ]
  }, [matches, fields])

  return (
    <SessionStatsRadar
      title={title}
      axes={axes}
      seriesName={t('session.detail.chart_fda_radar_series')}
      height={height}
    />
  )
}
