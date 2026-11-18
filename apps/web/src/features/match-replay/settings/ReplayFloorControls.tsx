/**
 * ReplayFloorControls — LE GROUPE « ÉTAGE » du panneau de la vue : la coupe du terrain par hauteur.
 *
 * Placé dans le groupe « Terrain », sur toute la largeur (maquette D.1, planches 1a à 1c) : une pastille
 * « Tout » (aucune coupe, le défaut), une pastille par étage détecté, un curseur de hauteur. Les étages sont
 * des RANGS (« Bas », « Étage 1 »…), jamais des hauteurs : une unité du jeu vaut environ 3 m (D-14). L'infobulle
 * d'une pastille dit combien d'anneaux porte un pion à cet étage — le langage d'étage des marqueurs.
 *
 * AU-DELÀ DE SIX ÉTAGES les pastilles défilent horizontalement au lieu de passer à la ligne (maquette D.1,
 * choix 11) : la rangée garde une hauteur, le curseur reste en dessous.
 *
 * CARTE SANS MAILLAGE DE NAVIGATION : un bloc d'explication en pointillé remplace la commande (D-7, maquette
 * D.1, choix 10) — jamais de pastille ni de curseur qui ne commanderaient rien.
 *
 * LA COUPE NE MASQUE AUCUN JOUEUR : le fond, ses traits et le relief suivent la coupe, les marqueurs restent
 * tous affichés. « Suivre le joueur ouvert » n'est PAS dans ce groupe (item D.6 du plan du fond Relief).
 */
import type { FloorChip, FloorState } from '../hooks/reliefLogic'
import { isChipSelected } from '../hooks/reliefLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { InfoMark } from './ReplaySettingsToggle'

/** Au-delà de ce nombre de pastilles d'étage, la rangée défile horizontalement. */
export const FLOOR_CHIPS_WRAP_MAX = 6

/** La commande d'étage : l'état de la commande, la coupe retenue et les gestes qui la changent. */
export type ReplayFloorControls = {
  state: FloorState
  /** La hauteur de la coupe ; `null` = « Tout ». */
  cut: number | null
  onSelectAll: () => void
  /** Retient l'étage de cette hauteur exacte. */
  onSelectRank: (z: number) => void
  /** Retient une hauteur libre (le curseur). */
  onHeight: (z: number) => void
}

const CHIP_CLASS =
  'h-[26px] shrink-0 rounded-full border border-border bg-transparent px-[11px] text-[11.5px] font-medium tabular-nums text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 aria-pressed:border-primary aria-pressed:bg-primary aria-pressed:text-primary-foreground'

/** L'étage le plus proche de la coupe, pour nommer la valeur du curseur sans unité. */
function nearestChip(chips: readonly FloorChip[], cut: number): FloorChip | null {
  let best: FloorChip | null = null
  for (const chip of chips) {
    if (best === null || Math.abs(chip.z - cut) < Math.abs(best.z - cut)) best = chip
  }
  return best
}

export function FloorGroup({ locale, floor }: { locale: ReplayLocale; floor: ReplayFloorControls }) {
  const t = REPLAY_TEXT[locale].relief
  const { state } = floor
  if (state.state === 'none') return null
  const title = (
    <p className="flex items-center gap-1 text-[11px] font-medium uppercase tracking-[0.06em] text-muted-foreground/70">
      {t.floorTitle}
      {state.state === 'ready' && <InfoMark text={t.floorInfo} />}
    </p>
  )
  if (state.state === 'noMesh') {
    return (
      <div className="col-span-2 space-y-0.5">
        {title}
        <p className="rounded-md border border-dashed border-input px-2.5 py-2 text-[11.5px] text-muted-foreground">{t.floorNoMesh}</p>
      </div>
    )
  }
  const { chips, min, max } = state
  const cut = floor.cut
  const named = cut === null ? null : nearestChip(chips, cut)
  return (
    <div className="col-span-2 space-y-1">
      {title}
      <div
        role="group"
        aria-label={t.floorTitle}
        className={`flex gap-1.5 py-0.5 ${chips.length > FLOOR_CHIPS_WRAP_MAX ? 'overflow-x-auto pb-1.5' : 'flex-wrap'}`}
      >
        <button type="button" className={CHIP_CLASS} aria-pressed={cut === null} title={t.floorAllHint} onClick={floor.onSelectAll}>
          {t.floorAll}
        </button>
        {chips.map((chip) => {
          const label = t.floorRankFmt(chip.rank)
          return (
            <button
              key={chip.rank}
              type="button"
              className={CHIP_CLASS}
              aria-pressed={isChipSelected(cut, chip)}
              title={t.floorChipHintFmt(label, chip.rings)}
              onClick={() => floor.onSelectRank(chip.z)}
            >
              {label}
            </button>
          )
        })}
      </div>
      <label className={`flex items-center gap-2.5 px-2 text-xs ${cut === null ? 'opacity-70' : ''}`}>
        <span className="shrink-0 text-muted-foreground">{t.floorHeight}</span>
        <input
          type="range"
          min={min}
          max={max}
          step={0.25}
          value={cut ?? min}
          aria-label={t.floorHeightAria}
          onChange={(e) => floor.onHeight(Number(e.target.value))}
          className="min-w-0 flex-1 accent-primary"
        />
        <output className="min-w-[6ch] shrink-0 text-right tabular-nums">
          {named === null ? t.floorAll : t.floorRankFmt(named.rank)}
        </output>
      </label>
    </div>
  )
}
