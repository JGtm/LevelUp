/**
 * OccupationReliefControls — LA RANGÉE DE COMMANDES DU FOND d'« Occupation du terrain », au-dessus de la carte
 * (maquette D.1, planche 4) : la pilule « Fond » (Plan, Rendu, Relief), le bouton « Traits du relief » et la pilule
 * « Étage », celles de la Tactique (`components/map-relief`). Rien n'est posé quand aucune commande ne commande
 * quelque chose (carte sans plan ni relief : le rendu est servi, sans choix).
 */
import { ReliefBackgroundPill, ReliefFloorPill, ReliefLinesButton } from '@/components/map-relief/ReliefPills'
import type { ReliefViewText } from '@/components/map-relief/text'
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import type { OccupationRelief } from './useOccupationRelief'

interface OccupationReliefControlsProps {
  text: ReliefViewText
  relief: OccupationRelief
  onSelectBackground: (style: MapBackgroundVariant) => void
}

export function OccupationReliefControls({ text, relief, onSelectBackground }: OccupationReliefControlsProps) {
  const hasBackground = relief.options.length > 0
  const hasFloor = relief.floor.state.state === 'ready'
  if (!hasBackground && !relief.linesAvailable && !hasFloor) return null
  return (
    <div className="mb-2.5 flex flex-wrap items-center gap-2" data-testid="match-positions-relief">
      <ReliefBackgroundPill
        text={text}
        options={relief.options}
        selected={relief.selected}
        onSelect={onSelectBackground}
        meshPercent={relief.meshPercent}
        testId="match-positions-fond-pilule"
      />
      <ReliefLinesButton
        text={text}
        available={relief.linesAvailable}
        show={relief.linesOn}
        onToggle={relief.toggleLines}
        testId="match-positions-traits"
      />
      <ReliefFloorPill text={text} floor={relief.floor} testId="match-positions-etage" />
    </div>
  )
}
