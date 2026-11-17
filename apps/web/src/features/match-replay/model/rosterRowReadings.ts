/**
 * rosterRowReadings.ts — CE QU'UNE LIGNE DE LA COLONNE DES ÉQUIPES LIT EN PLUS DE LA FICHE, sans React.
 *
 * Aucune lecture n'est écrite ici : la ligne relit, par les fonctions de la fiche et du bloc
 * « À l'instant », l'état vital, les armes, les grenades et l'équipement. Ce module ne fait que les
 * mettre en forme pour des cellules de largeur fixe (une grenade, une capacité, un compte à rebours).
 *
 * LA RÈGLE DES FICHES VAUT ICI : une valeur non lue est ABSENTE (`null`, cellule vide), jamais un zéro
 * ni un mot qui dirait l'absence. Chaque cellule garde l'âge de SA lecture, pour que le rendu
 * l'estompe.
 */
import type { ReplayText } from '../i18n/i18nContract'
import type { ReplayLocale } from '../i18n/i18n'
import { grenadeBoxAt, inventoryAt, type GrenadeBox } from './inventoryReading'
import {
  nowAbility,
  nowGrenades,
  type NowAbility,
  type NowGrenade,
  type NowWeapon,
} from './playerNowReadings'
import { formatSeconds, frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { PlayerState } from '../../../lib/replay/rosterLogic'

/**
 * respawnText — CE QUE LA LIGNE D'UN JOUEUR MORT ÉCRIT À LA PLACE DES JAUGES : « Réapparition dans
 * 5,8 s », le retour étant l'image de départ de la vie suivante du même joueur (lu, jamais déduit
 * d'une constante). Sans vie suivante le joueur ne revient plus dans le film : la ligne dit
 * « Hors film », « ne revient plus » — jamais un délai deviné.
 */
export function respawnText(
  state: Pick<PlayerState, 'respawnFrame'>,
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
  frame: number,
  t: Pick<ReplayText, 'respawnIn' | 'goneLabel' | 'goneValue'>,
): string {
  if (state.respawnFrame < 0) return `${t.goneLabel} · ${t.goneValue}`
  return `${t.respawnIn} ${formatSeconds(frameToMs(state.respawnFrame - frame, doc))}`
}

/** La cellule de grenades : le type qui partira (icône, nom, stock) ou, à défaut, le total porté. */
export interface RowGrenade {
  /** Le nombre affiché : le stock du type désigné, ou le total des types portés. */
  count: number
  /** Le type désigné (lu ou déduit d'un seul type porté) ; `null` = plusieurs types sans sélecteur lu. */
  selected: NowGrenade | null
}

/**
 * rowGrenade — la cellule FIXE de grenades d'une ligne. Un type désigné (`stowed` faux pour lui, vrai
 * pour les autres) s'affiche avec sa vignette et son stock ; un seul type porté est désigné par
 * déduction (`selectedGrenadeFrom`). Plusieurs types sans sélecteur lu : on ne devine pas lequel partira,
 * la cellule montre le TOTAL porté sans vignette, et l'infobulle de la cellule donne le stock par type.
 * Aucune grenade portée : cellule vide.
 */
export function rowGrenade(grenades: readonly NowGrenade[]): RowGrenade | null {
  if (grenades.length === 0) return null
  const open = grenades.filter((g) => !g.stowed)
  if (open.length === 1) return { count: open[0].count, selected: open[0] }
  return { count: grenades.reduce((n, g) => n + g.count, 0), selected: null }
}

/** Tout ce que les cellules d'inventaire d'une ligne lisent d'un joueur vivant à une image. */
export interface RowInventory {
  grenade: RowGrenade | null
  grenades: NowGrenade[]
  /** La lecture de grenades (âge, stock par type), pour l'infobulle et l'estompage ; `null` sans lecture. */
  box: GrenadeBox | null
  equipment: NowAbility | null
}

/**
 * rowInventory — les grenades et l'équipement porté d'un joueur vivant, par les lectures du bloc
 * « À l'instant » (`nowGrenades`, `nowAbility`) et la boîte de grenades de la fiche (`grenadeBoxAt`,
 * qui départage l'axe des grenades et l'inventaire aux images-clés).
 */
export function rowInventory(
  doc: ReplayDocumentReady,
  locale: ReplayLocale,
  slot: number,
  frame: number,
): RowInventory {
  const box = grenadeBoxAt(doc, slot, frame, inventoryAt(doc, slot, frame))
  const grenades = nowGrenades(doc, locale, box)
  return { grenade: rowGrenade(grenades), grenades, box, equipment: nowAbility(doc, locale, slot, frame) }
}

/**
 * ammoParts — les munitions d'une arme en deux morceaux : le compte, et sa réserve (« /108 ») ; ou la
 * charge restante en pour-cent pour une arme à charge. « Pleines » ne s'écrit qu'en infobulle
 * (`null`), comme dans le bloc « À l'instant ».
 */
export function ammoParts(w: Pick<NowWeapon, 'ammo'>): { value: string; reserve: string } | null {
  if (!w.ammo || w.ammo.kind === 'full') return null
  if (w.ammo.kind === 'charge') return { value: `${w.ammo.pct} %`, reserve: '' }
  return { value: `${w.ammo.mag}`, reserve: w.ammo.res === undefined ? '' : `/${w.ammo.res}` }
}
