/**
 * ReliefPills — LES COMMANDES DU FOND RELIEF des vues 2D sans panneau « Vue » : la pilule « Fond » (Plan, Rendu,
 * Relief), le bouton « Traits du relief » et la pilule « Étage » (Tout, une pastille par étage, curseur de hauteur).
 * Le bandeau du plan de l'onglet Tactique et « Occupation du terrain » les posent à l'identique (maquette D.1,
 * planches 3 et 4) ; elles ne savent rien de la carte, seulement de ce que `useReliefView` leur donne.
 *
 * LES ÉTAGES SONT DES RANGS (« Bas », « Étage 1 »…), jamais des hauteurs (D-14). Au-delà de six pastilles la rangée
 * défile horizontalement (maquette D.1, choix 11) ; le curseur reste à côté (choix 12).
 *
 * CARTE SANS MAILLAGE (D-7) : la pilule « Étage » n'est pas posée (planche 2 : « la pilule Étage n'est pas posée »),
 * jamais de pastille ni de curseur qui ne commanderaient rien. La pilule « Fond » et les traits restent.
 */
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'
import type { ReliefViewFloor } from '@/lib/replay/useReliefView'
import { isCutAtLevel, nearestChip } from '@/lib/replay/reliefViewLogic'

import { BOUTON_REGLAGE, ETIQUETTE, PILULE, VALEUR } from './pillStyles'
import type { ReliefViewText } from './text'

/** Au-delà de ce nombre de pastilles d'étage, la rangée défile horizontalement. */
export const FLOOR_CHIPS_SCROLL_MIN = 6

export interface ReliefBackgroundPillProps {
  text: ReliefViewText
  /** Les fonds proposés (`ReliefView.options`) ; vide = pas de choix, rien n'est posé. */
  options: readonly MapBackgroundVariant[]
  selected: MapBackgroundVariant
  onSelect: (style: MapBackgroundVariant) => void
  /** La part du relief tirée du maillage (D-16), pour l'infobulle de « Relief » ; `null` = inconnue. */
  meshPercent: number | null
  testId?: string
}

/** ReliefBackgroundPill — la pilule « Fond », dans l'ordre Plan, Rendu, Relief. */
export function ReliefBackgroundPill({ text, options, selected, onSelect, meshPercent, testId }: ReliefBackgroundPillProps) {
  if (options.length === 0) return null
  const label: Record<MapBackgroundVariant, string> = { plan: text.plan, rendu: text.render, relief: text.relief }
  const hint = meshPercent === null ? text.reliefHint : `${text.reliefHint} ${text.reliefMeshFmt(meshPercent)}`
  return (
    <span role="group" aria-label={text.backgroundLabel} className={PILULE} data-testid={testId}>
      <span className={ETIQUETTE}>{text.backgroundLabel}</span>
      {options.map((value, i) => (
        <button
          key={value}
          type="button"
          aria-pressed={selected === value}
          title={value === 'relief' ? hint : undefined}
          onClick={() => onSelect(value)}
          className={`${VALEUR}${i > 0 ? ' border-l border-input' : ''}`}
        >
          {label[value]}
        </button>
      ))}
    </span>
  )
}

export interface ReliefLinesButtonProps {
  text: ReliefViewText
  /** Les données du relief sont lues : sans elles, le bouton ne se pose pas. */
  available: boolean
  show: boolean
  onToggle: () => void
  testId?: string
}

/** ReliefLinesButton — le bouton « Traits du relief », de la forme de « Zones nommées ». */
export function ReliefLinesButton({ text, available, show, onToggle, testId }: ReliefLinesButtonProps) {
  if (!available) return null
  return (
    <button type="button" aria-pressed={show} title={text.linesHint} onClick={onToggle} className={BOUTON_REGLAGE} data-testid={testId}>
      {text.lines}
    </button>
  )
}

export interface ReliefFloorPillProps {
  text: ReliefViewText
  floor: ReliefViewFloor
  testId?: string
}

/** ReliefFloorPill — la pilule « Étage » : « Tout », une pastille par étage, curseur ; rien sans maillage. */
export function ReliefFloorPill({ text, floor, testId }: ReliefFloorPillProps) {
  const { state, cut } = floor
  if (state.state !== 'ready') return null
  const { chips, min, max } = state
  const named = cut === null ? null : nearestChip(chips, cut)
  return (
    <span role="group" aria-label={text.floorLabel} className={PILULE} data-testid={testId}>
      <span className={ETIQUETTE} title={text.floorInfo}>
        {text.floorLabel}
      </span>
      <span className={`flex min-w-0 items-stretch ${chips.length > FLOOR_CHIPS_SCROLL_MIN ? 'max-w-[16rem] overflow-x-auto' : ''}`}>
        <button type="button" aria-pressed={cut === null} title={text.floorAllHint} onClick={floor.selectAll} className={`${VALEUR} shrink-0`}>
          {text.floorAll}
        </button>
        {chips.map((chip) => (
          <button
            key={chip.rank}
            type="button"
            aria-pressed={isCutAtLevel(cut, chip.z)}
            onClick={() => floor.selectRank(chip.z)}
            className={`${VALEUR} shrink-0 whitespace-nowrap border-l border-input`}
          >
            {text.floorRankFmt(chip.rank)}
          </button>
        ))}
      </span>
      <label className={`flex shrink-0 items-center gap-1.5 border-l border-input px-1.5 ${cut === null ? 'opacity-70' : ''}`}>
        <input
          type="range"
          min={min}
          max={max}
          step={0.25}
          value={cut ?? min}
          aria-label={text.floorHeightAria}
          onChange={(e) => floor.setHeight(Number(e.target.value))}
          className="h-1 w-[84px] accent-primary"
        />
        <output className="min-w-[4.5ch] text-right tabular-nums text-foreground">
          {named === null ? text.floorAll : text.floorRankFmt(named.rank)}
        </output>
      </label>
    </span>
  )
}
