/**
 * PlacementVieCard — « Distance au coéquipier le plus proche et frags, par vie » (Escouade ›
 * Emprise, bloc « Groupé ou isolé », pleine largeur ; spécification §2.1).
 *
 * Une carte de chaleur : 8 cases de distance (0,25 portée de radar) sur 6 lignes de frags, la
 * valeur d'une case étant sa part des vies de l'escouade ; un gros point par joueur (distance
 * médiane, frags moyens par vie), repère de la portée du radar, frontière « au moins un frag »,
 * quatre quarts nommés (`placementCharts.ts`). Légende ECharts NATIVE en bas, centrée : un clic
 * masque le gros point d'un joueur. Hauteur 520 : une ligne de frags garde de quoi loger le titre
 * d'un quart sans recouvrir la valeur de ses cases.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { SquadEmprisePlacementPlayer } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { useSquadPlayerPalette } from '../useSquadPlayerPalette'
import {
  buildPlacementLifeOption,
  placementFormats,
  resolvePlacementColors,
  type PlacementBlock,
} from './placementCharts'
import type { PlacementText } from './placementStrings'

/** Hauteur du graphe. */
const LIFE_HEIGHT = 520

export function PlacementVieCard({ placement, locale, t }: { placement: PlacementBlock; locale: Locale; t: PlacementText }) {
  const formats = useMemo(() => placementFormats(locale), [locale])
  const { tokenOf } = useSquadPlayerPalette()
  const series = useMemo<ChartSeries<SquadEmprisePlacementPlayer>[]>(
    () => (placement.players?.length ? [{ key: 'emprise-placement-vie', datapoints: placement.players }] : []),
    [placement.players],
  )
  const buildOption = useCallback(
    () => buildPlacementLifeOption(placement, resolvePlacementColors(tokenOf), { t, formats }),
    [placement, tokenOf, t, formats],
  )
  return (
    <div className="min-w-0" data-testid="emprise-placement-vie">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.life.title}
            <InfoTooltip content={t.life.info} />
          </span>
        }
        series={series}
        height={LIFE_HEIGHT}
        buildOption={buildOption}
      />
    </div>
  )
}
