/**
 * playerNowReadings.ts — CE QUE LE BLOC « À L'INSTANT » D'UN JOUEUR LIT À UNE IMAGE, sans React.
 *
 * AUCUNE LECTURE N'EST ÉCRITE ICI : le bloc relit, par les mêmes fonctions et avec les mêmes âges,
 * ce que la fiche de la colonne lit — l'état vital (`playerStateAt`), les armes ordonnées
 * (`equippedWeapons`), l'inventaire et les grenades (`inventoryAt`, `grenadeBoxAt`), la capacité
 * et ses charges (`abilityAt`, `abilityChargesAt`, `carriedUsesAt`). Ce module les met en forme
 * pour des rangées nommées : il ne mesure rien, ne devine rien.
 *
 * LA RÈGLE DES FICHES VAUT ICI : une valeur non lue est ABSENTE (`null`, rangée absente), jamais un
 * zéro ni un mot qui dirait l'absence. Chaque groupe garde l'âge de SA lecture, pour que le rendu
 * l'estompe et le dise.
 *
 *  - la vitalité : la valeur lue est une FRACTION du plein (0..1) ; le bloc l'écrit en pour-cent du
 *    plein. Le document ne porte aucun maximum en points de vie ;
 *  - une arme dit « en main » ou « en réserve » seulement quand le sélecteur d'emplacement est lu ;
 *    ses munitions suivent la même condition (le sélecteur apparie l'arme et son emplacement) ;
 *  - l'équipement n'a un compte de charges que si le canal des charges ou l'équipement porté en
 *    établit un ; le document ne dit pas combien de charges font un plein.
 */
import type { AmmoHint } from '../i18n/i18nContract'
import { catalogText } from '../i18n/catalogLabel'
import type { ReplayLocale } from '../i18n/i18n'
import { abilityChargesAt, type AbilityChargeDisplay } from './abilityChargeLogic'
import { carriedUsesAt } from './carriedUsesLogic'
import { departuresOf, hasLeftAt } from './departureLogic'
import { equippedWeapons } from './equippedLogic'
import { ammoOfRecord } from './handCellHint'
import {
  grenadeBoxAt,
  grenadesCarriedFrom,
  inventoryAt,
  selectedGrenadeFrom,
  type GrenadeBox,
} from './inventoryReading'
import { weaponFullIcon, type FullWeaponIcon } from './weaponFullIcon'
import { formatSeconds, frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  abilityAt,
  LOADOUT_SRC_BIRTH,
  playerStateAt,
  type PlayerState,
  type ReplayPlayer,
  type VitalityPresence,
} from '../../../lib/replay/rosterLogic'

/** Une jauge lue : le pour-cent du plein (entier, borné à 0..100) et l'âge de la mesure, en images. */
export interface NowGauge {
  pct: number
  age: number
}

/** Où se trouve une arme : en main, rangée, ou `null` quand le sélecteur n'est pas lu (rien d'affirmé). */
export type NowWeaponPlace = 'hand' | 'stowed' | null

export interface NowWeapon {
  id: string
  /** Le nom du catalogue du document ; l'identifiant quand le catalogue n'en donne pas. */
  name: string
  place: NowWeaponPlace
  /** Munitions de l'emplacement ; `null` sans sélecteur lu ou sans compteur lu. */
  ammo: AmmoHint | null
  /** La vignette pleine (atlas silhouette, retournée) ; `null` sans image dans le catalogue. */
  icon: (FullWeaponIcon & { tinted: boolean }) | null
}

export interface NowGrenade {
  rank: number
  name: string
  count: number
  /** Vrai = un type est désigné ET ce n'est pas celui-ci : la rangée s'éteint. */
  stowed: boolean
  icon: { url: string; tinted: boolean } | null
}

export interface NowAbility {
  name: string
  icon: { url: string; tinted: boolean } | null
  /** Le compte ou le « plein » qualitatif ; `null` quand le canal ne mesure pas cette famille. */
  charges: AbilityChargeDisplay | null
  /** Âge de la lecture de la capacité, en images. */
  age: number
}

export interface PlayerNowReadings {
  state: PlayerState
  /** Vrai = le joueur est mort à l'instant lu (une vie close l'a précédé) : l'encadré « Éliminé » remplace le reste. */
  eliminated: boolean
  shield: NowGauge | null
  health: NowGauge | null
  weapons: NowWeapon[]
  /** Âge de la lecture des armes, de celle des munitions (inventaire), en images ; `null` sans lecture. */
  weaponsAge: number | null
  /** Vrai = les armes viennent de la dotation de naissance, pas d'une image-clé. */
  weaponsBirth: boolean
  ammoAge: number | null
  grenades: NowGrenade[]
  /** Âge de la lecture des grenades ; `null` sans lecture. */
  grenadesAge: number | null
  grenadeBox: GrenadeBox | null
  ability: NowAbility | null
}

export interface PlayerNowInput {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  presence: VitalityPresence
  locale: ReplayLocale
}

/** Une fraction du plein devient un pour-cent entier borné : la barre se borne de même. */
export function gaugePct(value: number): number {
  return Math.round(Math.max(0, Math.min(1, value)) * 100)
}

function gaugeOf(reading: { value: number; age: number } | null): NowGauge | null {
  return reading ? { pct: gaugePct(reading.value), age: reading.age } : null
}

/**
 * nowWeapons — les deux armes de la rangée, l'arme en main d'abord. L'emplacement d'inventaire d'une
 * arme est `equipped.order[k]` (le même appariement que la cellule de munitions de la fiche, pour
 * la main) ; les munitions ne se lisent que si le sélecteur est lu.
 */
function nowWeapons(
  doc: ReplayDocumentReady,
  locale: ReplayLocale,
  equipped: ReturnType<typeof equippedWeapons>,
  am: ReadonlyArray<{ mag?: number; res?: number; gauge?: number }>,
): NowWeapon[] {
  if (!equipped) return []
  const selector = equipped.drawn !== null
  return equipped.weapons.slice(0, 2).map((w, k) => {
    const label = doc.weaponLabels?.[w.id]
    const icon = label?.img ? weaponFullIcon(label.img) : null
    const record = am[equipped.order[k]]
    return {
      id: w.id,
      name: catalogText(label, locale) ?? w.id,
      place: !selector ? null : w.inHand ? 'hand' : 'stowed',
      ammo: selector && am.length > 0 ? ammoOfRecord(doc, w.id, record ?? {}) : null,
      icon: icon ? { ...icon, tinted: !!label?.tinted } : null,
    }
  })
}

export function nowGrenades(
  doc: ReplayDocumentReady,
  locale: ReplayLocale,
  box: GrenadeBox | null,
): NowGrenade[] {
  if (!box) return []
  const selected = selectedGrenadeFrom(box.g, box.gs)
  const known = typeof selected === 'object' && selected !== null ? selected : null
  return grenadesCarriedFrom(box.g, doc.grenadeLabels, locale).map((g) => {
    const label = doc.grenadeLabels?.[g.rank]
    return {
      ...g,
      stowed: known !== null && known.rank !== g.rank,
      icon: label?.img ? { url: label.img, tinted: !!label.tinted } : null,
    }
  })
}

/**
 * nowAbility — l'équipement porté : son nom du catalogue et ses charges. Un rang hors table n'est
 * pas rendu (aucun nom à écrire, la fiche de la colonne le dit en infobulle seulement).
 */
export function nowAbility(
  doc: ReplayDocumentReady,
  locale: ReplayLocale,
  slot: number,
  frame: number,
): NowAbility | null {
  const read = abilityAt(doc, slot, frame)
  if (!read) return null
  const label = doc.abilityLabels?.[String(read.rank)]
  const name = catalogText(label, locale)
  if (!name) return null
  return {
    name,
    icon: label?.img ? { url: label.img, tinted: !!label.tinted } : null,
    charges: abilityChargesAt(doc, slot, frame, read.rank) ?? carriedUsesAt(doc, slot, frame, read.rank),
    age: read.age,
  }
}

export function playerNowReadings({ player, doc, frame, presence, locale }: PlayerNowInput): PlayerNowReadings {
  const state = playerStateAt(player, frame, presence, hasLeftAt(departuresOf(doc), player.xuid, frame))
  const base = {
    state,
    eliminated: !state.alive && state.sinceDeath >= 0,
    shield: gaugeOf(state.shield),
    health: gaugeOf(state.health),
  }
  if (!state.life) {
    return {
      ...base, weapons: [], weaponsAge: null, weaponsBirth: false, ammoAge: null,
      grenades: [], grenadesAge: null, grenadeBox: null, ability: null,
    }
  }
  const slot = state.life.slot
  const equipped = equippedWeapons(doc, slot, frame)
  const inventory = inventoryAt(doc, slot, frame)
  const box = grenadeBoxAt(doc, slot, frame, inventory)
  return {
    ...base,
    weapons: nowWeapons(doc, locale, equipped, inventory?.state.am ?? []),
    weaponsAge: equipped ? equipped.age : null,
    weaponsBirth: equipped?.src === LOADOUT_SRC_BIRTH,
    ammoAge: inventory ? inventory.age : null,
    grenades: nowGrenades(doc, locale, box),
    grenadesAge: box ? box.age : null,
    grenadeBox: box,
    ability: nowAbility(doc, locale, slot, frame),
  }
}

/**
 * readingAgeTitle — la phrase d'âge d'une lecture : « <libellé> 8,4 s », ou « <libellé à venir> 3,0 s » quand
 * l'âge est NÉGATIF (lecture d'une image-clé à venir, dite comme telle). Les libellés sont ceux de la
 * fiche de la colonne (`loadoutAge`, `inventoryAge`, `abilityAge`…).
 */
export function readingAgeTitle(
  past: string,
  ahead: string,
  age: number,
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
): string {
  return `${age < 0 ? ahead : past} ${formatSeconds(frameToMs(Math.abs(age), doc))}`
}
