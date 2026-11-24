/**
 * ReplayRosterCells — LES TROIS CELLULES D'INVENTAIRE D'UNE LIGNE de la colonne des équipes : l'arme
 * en main, les grenades, l'équipement porté. Chacune tient la largeur FIXE de sa colonne
 * (`GABARIT_LIGNE`) et reste vide quand rien n'est lu : une colonne ne se décale jamais d'un joueur
 * à l'autre.
 *
 * LES LECTURES SONT CELLES DE LA FICHE : les armes ordonnées (`equippedWeapons`, calculées une fois par
 * la ligne), les grenades et l'équipement (`rowInventory`, qui relit `grenadeBoxAt`, `nowGrenades`,
 * `nowAbility`). CHAQUE CELLULE PÂLIT AVEC L'ÂGE DE SA LECTURE et le dit en infobulle : l'inventaire
 * ne se lit qu'aux images-clés, une toutes les ~20 s.
 *
 * L'ÉCHANGE D'ARME RESTE ANIMÉ (la vignette entrante glisse de la largeur de la cellule) : c'est le
 * même effet que la fiche, par la même vignette (`WeaponChip`).
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'

import { WeaponChip, SWAP_ANIM_MS } from './ReplayWeaponsRow'
import { GABARIT_LIGNE as G } from '../model/cardGabarit'
import { drawnSwapAt, type EquippedReading } from '../model/equippedLogic'
import { handCellHint } from '../model/handCellHint'
import { readingAgeTitle } from '../model/playerNowReadings'
import type { RowGrenade, RowInventory } from '../model/rosterRowReadings'
import { grenadeBoxHint } from '../model/inventoryReading'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { catalogText } from '../i18n/catalogLabel'
import { freshness, msToFrames, READING_FADE } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { LOADOUT_SRC_BIRTH } from '../../../lib/replay/slotReadings'

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

/** Ce que partagent les trois cellules : le document, l'image, la langue, l'estompage de lecture. */
interface CellsBase {
  doc: ReplayDocumentReady
  frame: number
  locale: ReplayLocale
  /** Âge, en images, auquel une lecture d'inventaire est au plancher d'opacité (`useCardScene`). */
  readingFull: number
}

/** La cellule vide d'une colonne : la place, rien d'autre. */
function EmptyCell({ width }: { width: number }) {
  return <span aria-hidden className="inline-block" style={{ width }} />
}

/**
 * WeaponCell — l'arme EN MAIN (la première de la lecture ordonnée) : sa vignette dans la cellule
 * fixe, l'infobulle disant l'arme rangée, les munitions et l'âge de la lecture. Sans loadout lu : cellule vide.
 */
export function WeaponCell({
  doc, frame, locale, readingFull, slot, equipped,
}: CellsBase & { slot: number; equipped: EquippedReading | null }) {
  const t = REPLAY_TEXT[locale]
  const hand = equipped?.weapons[0]
  if (!equipped || !hand) return <EmptyCell width={G.weaponColW} />
  const swapFrames = Math.max(1, msToFrames(SWAP_ANIM_MS, doc))
  const swapAge = drawnSwapAt(doc, slot, frame, swapFrames)
  const stowed = equipped.weapons[1]
  const stowedName = stowed ? (catalogText(doc.weaponLabels?.[stowed.id], locale) ?? stowed.id) : null
  const title = [
    stowedName === null ? null : equipped.drawn !== null ? t.weaponStowedFmt(stowedName) : stowedName,
    handCellHint(t, doc, slot, frame, equipped),
    equipped.src === LOADOUT_SRC_BIRTH ? t.loadoutBirth : null,
    readingAgeTitle(t.loadoutAge, t.loadoutAhead, equipped.age, doc),
  ]
    .filter((p): p is string => !!p)
    .join(' · ')
  return (
    <span
      className="inline-flex items-center justify-end"
      style={{ width: G.weaponColW, opacity: freshness(equipped.age, readingFull, READING_FADE) }}
      title={title}
    >
      <WeaponChip
        doc={doc}
        id={hand.id}
        inHand={hand.inHand}
        dimmed={false}
        swap={swapAge === null ? null : { cls: 'replay-wswap-l', age: swapAge, span: swapFrames, dx: G.weaponColW }}
        locale={locale}
        cellW={G.weaponColW}
        box={{ cellH: G.weaponCellH, iconW: G.iconWeaponW, iconH: G.iconWeaponH }}
      />
    </span>
  )
}

/** L'infobulle de la cellule de grenades : le stock par type et l'âge de la lecture ; `undefined` sans lecture. */
function grenadesTitle(t: ReplayText, doc: ReplayDocumentReady, inv: RowInventory): string | undefined {
  return inv.box ? grenadeBoxHint(t, inv.box, inv.grenades, doc) : undefined
}

/** Le contenu de la cellule : la vignette du type qui partira et son stock, ou le total sans vignette. */
function GrenadeContent({ grenade }: { grenade: RowGrenade }) {
  const icon = grenade.selected?.icon
  return (
    <>
      {icon && (
        <WeaponIcon
          imageUrl={icon.url}
          tinted={icon.tinted}
          label={grenade.selected?.name ?? ''}
          width={G.iconGrenadePx}
          height={G.iconGrenadePx}
        />
      )}
      <span className="tabular-nums" style={{ fontSize: G.countPx }}>{grenade.count}</span>
    </>
  )
}

/**
 * GrenadeCell — le type de grenade qui partira (vignette) et son stock ; plusieurs types sans
 * sélecteur lu : le total porté, sans vignette (on ne devine pas lequel partira). Aucune grenade
 * portée, ou pas de lecture : cellule vide.
 */
export function GrenadeCell({
  doc, locale, readingFull, inventory,
}: Pick<CellsBase, 'doc' | 'locale' | 'readingFull'> & { inventory: RowInventory }) {
  const t = REPLAY_TEXT[locale]
  const { grenade, box } = inventory
  if (!grenade || !box) return <EmptyCell width={G.grenadeColW} />
  return (
    <span
      className="inline-flex items-center justify-end gap-0.5 text-foreground"
      style={{ width: G.grenadeColW, opacity: freshness(box.age, readingFull, READING_FADE) }}
      title={grenadesTitle(t, doc, inventory)}
    >
      <GrenadeContent grenade={grenade} />
    </span>
  )
}

/**
 * EquipmentCell — la vignette de l'équipement porté ; son nom, ses charges et l'âge de sa lecture en
 * infobulle. Un rang hors de la table du document n'est pas rendu (aucun nom à écrire) : cellule vide.
 */
export function EquipmentCell({
  doc, locale, readingFull, inventory,
}: Pick<CellsBase, 'doc' | 'locale' | 'readingFull'> & { inventory: RowInventory }) {
  const t = REPLAY_TEXT[locale]
  const a = inventory.equipment
  if (!a || !a.icon) return <EmptyCell width={G.equipmentColW} />
  const charges = a.charges?.kind === 'count' ? t.playerInstant.equipmentCharges(a.charges.charges) : null
  const full = a.charges?.kind === 'full' ? t.abilityChargesFullHint : null
  const title = [`${t.abilityLabel} — ${a.name}`, charges, full, readingAgeTitle(t.abilityAge, t.abilityAhead, a.age, doc)]
    .filter((p): p is string => !!p)
    .join(' · ')
  return (
    <span
      className="inline-flex items-center justify-end text-foreground"
      style={{ width: G.equipmentColW, opacity: freshness(a.age, readingFull, READING_FADE) }}
      title={title}
    >
      <WeaponIcon
        imageUrl={a.icon.url}
        tinted={a.icon.tinted}
        label={a.name}
        width={G.iconEquipmentPx}
        height={G.iconEquipmentPx}
      />
    </span>
  )
}
