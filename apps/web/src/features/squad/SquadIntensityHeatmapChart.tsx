/**
 * SquadIntensityHeatmapChart — wrapper teammates.15.
 *
 * Toggle "all" / per-player via boutons segmentés en tête de la carte ; le
 * profile (rows par option) vient déjà bucket-isé du serveur.
 */
import { useCallback, useMemo, useState } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadIntensityProfile, SquadIntensityMatchRow } from '@/lib/api/types'
import {
  buildSquadIntensityHeatmapOption,
  type SquadIntensityOpts,
} from './charts/squadIntensityHeatmapChart'

interface SquadIntensityHeatmapChartProps extends SquadIntensityOpts {
  title?: string
  profile: SquadIntensityProfile
  /** gamertag → couleur hex résolue depuis semantic tokens. */
  colorByPlayer?: Record<string, string>
  /** Override du label de l'option "all" (par défaut, l'API renvoie déjà le libellé). */
  toggleLabel?: string
}

export function SquadIntensityHeatmapChart({
  profile,
  title,
  zLabel,
  colorByPlayer,
  toggleLabel,
}: SquadIntensityHeatmapChartProps) {
  const defaultKey = profile.options[0]?.key ?? 'all'
  const [selectedKey, setSelectedKey] = useState<string>(defaultKey)

  const rows = useMemo<SquadIntensityMatchRow[]>(
    // Inversé : l'API renvoie les matchs récent→ancien ; on passe ancien→récent
    // pour que (via yAxis.inverse) le plus récent soit en bas et le plus vieux en
    // haut, avec une numérotation #1 (haut, ancien) → #N (bas, récent).
    () => [...(profile.rows[selectedKey] ?? [])].reverse(),
    [profile.rows, selectedKey],
  )

  // Wrap pour ChartCard typé. On passe les rows comme datapoints d'une série unique.
  const series = useMemo<ChartSeries<SquadIntensityMatchRow>[]>(
    () => (rows.length > 0 ? [{ key: 'intensity', datapoints: rows }] : []),
    [rows],
  )

  const buildOption = useCallback(
    (s: ChartSeries<SquadIntensityMatchRow>[]) =>
      buildSquadIntensityHeatmapOption(s[0]?.datapoints ?? [], { zLabel }),
    [zLabel],
  )

  // Pas de `return null` quand aucune option : on garde le bloc titré et
  // ChartCard affiche son état vide (series vide ci-dessus).
  const matchCount = rows.length
  const height = Math.max(360, Math.min(600, matchCount * 28 + 120))

  return (
    <div className="space-y-2" data-testid="squad-intensity-heatmap">
      {toggleLabel && <p className="text-xs text-muted-foreground">{toggleLabel}</p>}
      <div className="flex flex-wrap gap-1">
        {profile.options.map((opt) => {
          // color-allow: hex résolu depuis colorByPlayer (semantic tokens via getSquadPlayerColors)
          const accentHex = opt.key !== 'all' ? (colorByPlayer?.[opt.key] ?? undefined) : undefined
          return (
            <button
              key={opt.key}
              type="button"
              onClick={() => setSelectedKey(opt.key)}
              className={[
                'inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors',
                opt.key === selectedKey
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-input bg-background hover:bg-muted',
              ].join(' ')}
            >
              {accentHex && (
                <span aria-hidden style={{ background: accentHex, width: 8, height: 8, display: 'inline-block', flexShrink: 0 }} />
              )}
              {opt.label}
            </button>
          )
        })}
      </div>
      <ChartCard
        title={title}
        series={series}
        buildOption={buildOption}
        height={height}
      />
    </div>
  )
}
