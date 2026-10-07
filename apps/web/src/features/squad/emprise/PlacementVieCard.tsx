/**
 * PlacementVieCard — « Placement et rendement de chaque vie » (Escouade › Emprise, bloc
 * « Groupés ou isolés », pleine largeur ; lot V4 du plan PLAN_EMPRISE_VIES_2026-09-28, §2.1).
 *
 * Un nuage : un point par vie (couleur du joueur, taille = durée), un gros point par joueur
 * (médianes), repère de la portée du radar, frontière « au moins un frag », quatre quarts nommés
 * dont le seul « isolé et coûteux » est teinté (`placementCharts.ts`). Légende ECharts NATIVE en
 * bas, centrée : un clic isole le semis ET le gros point d'un joueur (même nom de série), ce que
 * la légende DOM des autres cartes ne fait pas. Hauteur 420.
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

/** Hauteur du graphe (spécification §2.1). */
const LIFE_HEIGHT = 420

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
  const { coverage } = placement
  return (
    <div className="min-w-0" data-testid="emprise-placement-vie">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.life.title}
            <InfoTooltip content={t.life.info(coverage.lives_measured, coverage.lives_total, coverage.matches_without_range)} />
          </span>
        }
        series={series}
        height={LIFE_HEIGHT}
        buildOption={buildOption}
      />
    </div>
  )
}
