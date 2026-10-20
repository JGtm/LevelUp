/**
 * TendancesMatrix — la carte « Indicateurs par mois et par horizon » : un `Heatmap2DChart`
 * par groupe d'indicateurs, sous un intertitre, dans une `SectionCard`.
 *
 * Seules les briques existantes : `SectionCard` + `titleWithInfo` pour la carte, `SectionTitle`
 * pour l'intertitre d'un groupe, le wrapper de grille canonique en rampe divergente. Pas de
 * groupe repliable, pas de clic, pas de réglette de couleur (le sens des teintes est dit
 * dans la bulle d'information de la carte).
 *
 * LES COLONNES S'ALIGNENT D'UN GROUPE À L'AUTRE : même marge gauche pour toutes les grilles
 * (`gridOverride`), calculée sur la plus longue étiquette de lignes de la page.
 */
import { useMemo } from 'react'

import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import { getEChartsThemeColors } from '@/components/charts/_utils'
import { SectionTitle } from '@/components/ui/detail-section'
import { SectionCard } from '@/components/ui/section-card'
import { titleWithInfo } from '@/components/ui/title-with-info'
import type { TrendsPageResponse } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { useIndicatorLabeler } from './labels'
import { Z_LIMIT } from './tendances.logic'
import {
  buildMatrixGrids,
  formatMatrixTooltip,
  matrixGridHeight,
  matrixLeftMargin,
} from './tendancesMatrix.logic'

const VALUE_RANGE: [number, number] = [-Z_LIMIT, Z_LIMIT]
/** Marges de trace communes (hors marge gauche) : haut, bas (axe des colonnes), droite. */
const GRID_MARGINS = { top: 8, bottom: 32, right: 16 } as const

export interface TendancesMatrixProps {
  locale: Locale
  data: TrendsPageResponse
}

export function TendancesMatrix({ locale, data }: TendancesMatrixProps) {
  const t = getTendancesText(locale)
  const labelOf = useIndicatorLabeler(locale)
  const grids = useMemo(
    () => buildMatrixGrids(data.indicators, data.months, locale, labelOf),
    [data.indicators, data.months, locale, labelOf],
  )
  const gridOverride = useMemo(
    () => ({ ...GRID_MARGINS, left: matrixLeftMargin(grids) }),
    [grids],
  )
  const formatTooltip = useMemo(
    () => (point: ChartPointHeatmap) => formatMatrixTooltip(point, locale),
    [locale],
  )
  // Encre des nombres : celle du thème, lisible sur toute la rampe divergente.
  const cellLabelColor = getEChartsThemeColors().text

  return (
    <SectionCard title={t.matrixTitle} titleAdornment={titleWithInfo(t.matrixInfo)}>
      <div className="space-y-4 p-3" data-testid="tendances-matrix">
        {grids.map((grid) => (
          <div key={grid.group} className="space-y-1" data-testid={`tendances-matrix-${grid.group}`}>
            <SectionTitle>{t.groupLabel(grid.group)}</SectionTitle>
            <Heatmap2DChart
              series={grid.series}
              height={matrixGridHeight(grid.rows)}
              frameless
              paletteMode="divergent"
              valueRange={VALUE_RANGE}
              showVisualMap={false}
              emptyCells="blank"
              yAxisInverse
              cellLabelColor={cellLabelColor}
              formatTooltip={formatTooltip}
              gridOverride={gridOverride}
            />
          </div>
        ))}
      </div>
    </SectionCard>
  )
}
