/**
 * PlacementQuartsCard — « Part des vies par placement » (Escouade › Emprise, bloc « Groupés ou
 * isolés », sous le nuage ; lot V4 du plan PLAN_EMPRISE_VIES_2026-09-28, §2.2).
 *
 * Une barre horizontale empilée à 100 % par joueur (le premier en haut), les quatre quarts du
 * nuage aux mêmes teintes, valeur « v % » dans le segment à partir de 8 % (`placementCharts.ts`).
 * Légende ECharts en bas, centrée. Hauteur 230.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { SquadEmprisePlacementPlayer } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { useSquadPlayerPalette } from '../useSquadPlayerPalette'
import {
  buildPlacementQuartsOption,
  placementFormats,
  resolvePlacementColors,
  type PlacementBlock,
} from './placementCharts'
import type { PlacementText } from './placementStrings'

/** Hauteur du graphe (spécification §2.2). */
const QUARTS_HEIGHT = 230

export function PlacementQuartsCard({ placement, locale, t }: { placement: PlacementBlock; locale: Locale; t: PlacementText }) {
  const formats = useMemo(() => placementFormats(locale), [locale])
  const { tokenOf } = useSquadPlayerPalette()
  const series = useMemo<ChartSeries<SquadEmprisePlacementPlayer>[]>(
    () => (placement.players?.length ? [{ key: 'emprise-placement-quarts', datapoints: placement.players }] : []),
    [placement.players],
  )
  const buildOption = useCallback(
    () => buildPlacementQuartsOption(placement, resolvePlacementColors(tokenOf), { t, formats }),
    [placement, tokenOf, t, formats],
  )
  const iso = placement.isolated_from_ratio
  return (
    <div className="min-w-0" data-testid="emprise-placement-quarts">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.quarts.title}
            <InfoTooltip content={t.quarts.info(formats.count.format(iso), iso, placement.productive_from_kills)} />
          </span>
        }
        series={series}
        height={QUARTS_HEIGHT}
        buildOption={buildOption}
      />
    </div>
  )
}
