/**
 * ReplayFloorRow — LA COUPE PAR ÉTAGE dans la rubrique « Carte » : un segmenté « Tout » puis un rang par étage détecté
 * (« Bas », « Étage 1 »…), et un curseur « Hauteur » qui n'apparaît que lorsqu'une coupe est active. Les étages sont des
 * RANGS, jamais des hauteurs : une unité du jeu vaut environ 3 m. L'infobulle d'un étage dit combien d'anneaux porte
 * un pion à cet étage — le langage d'étage des marqueurs.
 *
 * AU-DELÀ DE SIX ÉTAGES le segmenté défile horizontalement : la rangée garde une hauteur.
 *
 * CARTE SANS MAILLAGE DE NAVIGATION : un bloc d'explication en pointillé remplace la commande — jamais de choix ni de
 * curseur qui ne commanderaient rien.
 *
 * LA COUPE NE MASQUE AUCUN JOUEUR : le fond, ses traits et le relief suivent la coupe, les marqueurs restent tous
 * affichés. « Étage de la fiche ouverte » est une ligne à part (`ToggleRow`, fille de celle-ci) ; la note qui l'explique
 * se compose dans `settingsText` (`followNote`).
 */
import { isCutAtLevel, nearestChip } from '@/lib/replay/reliefViewLogic'

import { REPLAY_TEXT } from '../i18n/i18n'
import { InfoMark } from './InfoMark'
import type { ReplayViewPanelModel } from './viewPanelModel'

/** Au-delà de ce nombre d'étages, le segmenté défile. La brique n'en cuit aujourd'hui que cinq au plus. */
const FLOOR_SEGMENTS_SCROLL_MAX = 6

const SEGMENT_CLASS =
  'flex min-h-6 shrink-0 cursor-pointer items-center justify-center rounded-[5px] px-2 py-0.5 text-center text-[11px] font-medium tabular-nums leading-tight transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'

export function FloorRow({ model }: { model: ReplayViewPanelModel }) {
  const t = REPLAY_TEXT[model.locale].relief
  const { floor } = model
  const { state, cut } = floor
  if (state.state === 'none') return null
  const title = (
    <div className="flex items-center gap-1 text-muted-foreground">
      {t.floorTitle}
      {state.state === 'ready' && <InfoMark text={t.floorInfo} />}
    </div>
  )
  if (state.state === 'noMesh') {
    return (
      <div data-setting="floor" className="flex flex-col gap-0.5 px-2 pb-1.5 pt-0.5 text-xs">
        {title}
        <p className="rounded-md border border-dashed border-placeholder-border px-2.5 py-2 text-[11.5px] text-muted-foreground">{t.floorNoMesh}</p>
      </div>
    )
  }
  const { chips, min, max } = state
  const named = cut === null ? null : nearestChip(chips, cut)
  const value = cut === null ? t.floorAll : named === null ? '' : t.floorRankFmt(named.rank)
  return (
    <div data-setting="floor" className="flex flex-col gap-1 px-2 pb-1.5 pt-0.5 text-xs">
      {title}
      <div
        role="radiogroup"
        aria-label={t.floorTitle}
        className={`flex gap-px rounded-md border border-border p-px ${chips.length > FLOOR_SEGMENTS_SCROLL_MAX ? 'overflow-x-auto' : ''}`}
      >
        <button
          type="button"
          role="radio"
          aria-checked={cut === null}
          title={t.floorAllHint}
          onClick={floor.selectAll}
          className={`${SEGMENT_CLASS} flex-1 ${cut === null ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'}`}
        >
          {t.floorAll}
        </button>
        {chips.map((chip) => {
          const label = t.floorRankFmt(chip.rank)
          const selected = isCutAtLevel(cut, chip.z)
          return (
            <button
              key={chip.rank}
              type="button"
              role="radio"
              aria-checked={selected}
              title={t.floorChipHintFmt(label, chip.rings)}
              onClick={() => floor.selectRank(chip.z)}
              className={`${SEGMENT_CLASS} flex-1 ${selected ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'}`}
            >
              {label}
            </button>
          )
        })}
      </div>
      {cut !== null && (
        <label className="flex items-center gap-2 text-[11.5px] text-muted-foreground">
          <span className="shrink-0">{t.floorHeight}</span>
          <input
            type="range"
            min={min}
            max={max}
            step={0.25}
            value={cut}
            aria-label={t.floorHeightAria}
            onChange={(e) => floor.setHeight(Number(e.target.value))}
            className="min-w-0 flex-1 accent-primary"
          />
          <output className="min-w-[6ch] shrink-0 text-right tabular-nums">{value}</output>
        </label>
      )}
    </div>
  )
}
