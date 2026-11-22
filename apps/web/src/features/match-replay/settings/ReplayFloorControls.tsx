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
 * « SUIVRE LE JOUEUR OUVERT » ferme le groupe, sous le curseur (maquette D.1) : un interrupteur, désactivé et
 * expliqué tant qu'aucune fiche de joueur n'est ouverte ; allumé, une note dit le rang de l'étage suivi. Un
 * choix manuel (« Tout », pastille, curseur) l'éteint : c'est l'état qui le décide, pas ce composant.
 *
 * LA COUPE NE MASQUE AUCUN JOUEUR : le fond, ses traits et le relief suivent la coupe, les marqueurs restent
 * tous affichés.
 */
import type { FloorRank } from '@/lib/replay/relief'
import { isCutAtLevel, nearestChip } from '@/lib/replay/reliefViewLogic'

import type { FloorState } from '../hooks/reliefLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReliefText } from '../i18n/relief.i18n'
import { InfoMark, SettingsToggle } from './ReplaySettingsToggle'

/**
 * Au-delà de ce nombre de pastilles d'étage, la rangée défile horizontalement. La brique ne cuit aujourd'hui que cinq
 * étages au plus : la branche attend un plafond plus haut, et jusque-là la rangée passe toujours à la ligne.
 */
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
  /** « Suivre le joueur ouvert » : `available` = une fiche de joueur est ouverte ; `on` = le suivi est allumé. */
  follow: { available: boolean; on: boolean; onToggle: () => void }
}

const CHIP_CLASS =
  'h-[26px] shrink-0 rounded-full border border-border bg-transparent px-[11px] text-[11.5px] font-medium tabular-nums text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40 aria-pressed:border-primary aria-pressed:bg-primary aria-pressed:text-primary-foreground'

/** La note sous l'interrupteur de suivi : l'explication du désactivé, ou le rang de l'étage suivi ; rien sinon. */
function followNote(t: ReliefText, follow: ReplayFloorControls['follow'], named: FloorRank | null): string | null {
  if (!follow.available) return t.floorFollowNone
  return follow.on && named ? t.floorFollowOnFmt(t.floorRankFmt(named.rank)) : null
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
  // Sans étage détecté, une hauteur libre n'a pas de rang à nommer (D-14 : jamais de mètres).
  const value = cut === null ? t.floorAll : named === null ? '' : t.floorRankFmt(named.rank)
  const note = followNote(t, floor.follow, named)
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
              aria-pressed={isCutAtLevel(cut, chip.z)}
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
          {value}
        </output>
      </label>
      <SettingsToggle
        label={t.floorFollow}
        pressed={floor.follow.on}
        onToggle={floor.follow.onToggle}
        hint={t.floorFollowHint}
        disabled={!floor.follow.available}
      />
      {note && <p className={`px-2 text-[11.5px] ${floor.follow.on ? 'text-foreground' : 'text-muted-foreground'}`}>{note}</p>}
    </div>
  )
}
