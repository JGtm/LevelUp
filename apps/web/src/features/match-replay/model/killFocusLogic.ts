/**
 * killFocusLogic.ts — LA MISE EN VALEUR D'UNE ÉLIMINATION SUR LA CARTE : ce qu'un clic demande,
 * quand elle s'efface, et ce qu'elle met EFFECTIVEMENT en valeur.
 *
 * LE GESTE. Un clic sur une ligne de frag du fil pose la lecture en pause à l'instant du frag
 * (le canal de saut du magasin de lecture, `playbackStore`) et demande de mettre le tueur et la
 * victime en valeur : voile léger, équerres qui se verrouillent, noms en étiquettes pleines.
 * Cette demande est un ÉTAT, et rien qu'un état : elle est tenue par le magasin de lecture (le
 * fil la pose, le canvas la lit pour peindre), jamais mémorisée entre deux visites.
 *
 * QUAND ELLE S'EFFACE. Par la Lecture, par un saut manuel (frise, raccourcis clavier, repères),
 * par un autre clic qui la REMPLACE. Un clic sur la ligne déjà choisie la REJOUE : chaque clic
 * porte un jeton neuf, et l'animation repart à chaque jeton — c'est le jeton, pas la ligne, qui
 * identifie « un clic ».
 *
 * CE QU'ELLE MET EN VALEUR. Seulement ce qui est POSITIONNÉ à l'image du frag : la victime peut ne
 * pas avoir de position (aucune vie lisible), le tueur non plus (bot relevé, mort hors carte). On
 * met alors en valeur ce qui l'est, jamais une équerre à une position devinée ; l'étiquette suit
 * la même règle que l'équerre, par acteur.
 *
 * Pas de React, pas de canvas : logique pure, testée (killFocusLogic.test.ts).
 */
import { frameToMs, msToFrames, type XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayFeedEntry } from './killFeedLogic'

/** L'arme du frag, telle que le fil la porte : l'icône et si elle est un masque à teindre. */
export interface KillFocusWeapon {
  /** Vide = arme non identifiée : l'étiquette du tueur n'a pas d'icône, jamais celle d'une autre. */
  imageUrl: string
  tinted: boolean
}

/** Ce que demande un clic sur une ligne de frag. */
export interface KillFocusTarget {
  /** Clé de la ligne du fil : elle porte l'état « sélectionnée ». */
  entryKey: string
  /** L'image du frag sur la grille du rejeu, où la lecture se pose. */
  frame: number
  killerXuid: string
  /** Vide = victime non nommée : seul le tueur sera mis en valeur. */
  victimXuid: string
  weapon: KillFocusWeapon
}

/** Une demande de mise en valeur : une cible et le jeton du clic qui l'a posée. */
export interface KillFocusRequest extends KillFocusTarget {
  /** Un clic de plus = un jeton de plus, même sur la même ligne : l'animation repart. */
  token: number
}

/** L'état tenu par le magasin : la demande courante, et le dernier jeton émis. */
export interface KillFocusState {
  current: KillFocusRequest | null
  lastToken: number
}

export const NO_KILL_FOCUS: KillFocusState = { current: null, lastToken: 0 }

/** Ce qui fait bouger l'état : un clic sur une ligne de frag, ou l'effacement. */
export type KillFocusEvent = { type: 'select'; target: KillFocusTarget } | { type: 'clear' }

/**
 * reduceKillFocus applique un événement à l'état. `clear` est ce que font la Lecture et tout saut
 * manuel ; un état déjà vide est rendu tel quel (même objet), pour ne réveiller personne.
 */
export function reduceKillFocus(state: KillFocusState, event: KillFocusEvent): KillFocusState {
  if (event.type === 'clear') {
    return state.current === null ? state : { current: null, lastToken: state.lastToken }
  }
  const token = state.lastToken + 1
  return { current: { ...event.target, token }, lastToken: token }
}

/**
 * frameAtOrAfter rend la première image dont l'instant n'est pas antérieur à `ms`. La lecture se
 * pose dessus : le fil n'affiche que les lignes déjà survenues à l'instant courant (`feedAt`), et
 * une image arrondie vers le bas ferait disparaître du fil la ligne qu'on vient de cliquer.
 */
export function frameAtOrAfter(ms: number, doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>): number {
  let frame = Math.ceil(msToFrames(ms, doc))
  // Garde d'arrondi : le produit image x intervalle peut retomber un ulp sous l'instant visé.
  while (frameToMs(frame, doc) < ms) frame++
  return frame
}

/** focusTargetOf : la cible d'une ligne du fil — `null` pour tout ce qui n'est pas un frag. */
export function focusTargetOf(
  entry: ReplayFeedEntry,
  doc: Pick<ReplayDocumentReady, 'frameIntervalMs'>,
): KillFocusTarget | null {
  const k = entry.kill
  if (!k) return null
  return {
    entryKey: entry.key,
    frame: frameAtOrAfter(k.replayMs, doc),
    killerXuid: k.xuid,
    victimXuid: k.victimXuid,
    weapon: { imageUrl: k.weaponImageUrl, tinted: k.weaponTinted },
  }
}

/** Ce que la résolution lit du film et de la page : toutes les lectures sont INJECTÉES. */
export interface KillFocusResolvers {
  /** Position monde d'un joueur à une image (le bipède, ou son véhicule), `null` si inconnue. */
  posOf: (xuid: string, frame: number) => XY | null
  /** Le joueur occupe-t-il une fiche à cette image ? Sinon il n'a pas de pion (règle du pion). */
  seated: (xuid: string, frame: number) => boolean
  /** Couleur d'équipe du joueur, `null` = pas de pion dessiné. */
  colorOf: (xuid: string) => string | null
  /** Nom écrit sous son pion, `null` = aucun nom connu. */
  nameOf: (xuid: string) => string | null
  /** Slot de la vie qui porte son pion à l'image : sert à retirer le nom normal du pion. */
  slotOf: (xuid: string, frame: number) => number | null
}

/** Un joueur mis en valeur : positionné, coloré, et son nom pour l'étiquette. */
export interface FocusActor {
  xuid: string
  pos: XY
  color: string
  name: string | null
  slot: number | null
}

/** La mise en valeur résolue : ce que le calque peint. */
export interface ResolvedKillFocus {
  token: number
  frame: number
  killer: FocusActor | null
  victim: FocusActor | null
  weapon: KillFocusWeapon
}

/** Un acteur n'est résolu que s'il est positionné, assis à une fiche et coloré : rien n'est inventé. */
function resolveActor(xuid: string, frame: number, r: KillFocusResolvers): FocusActor | null {
  if (!xuid || !r.seated(xuid, frame)) return null
  const pos = r.posOf(xuid, frame)
  const color = r.colorOf(xuid)
  if (!pos || !color) return null
  return { xuid, pos, color, name: r.nameOf(xuid), slot: r.slotOf(xuid, frame) }
}

/**
 * resolveKillFocus transforme la demande en ce qui se peint. `null` quand ni le tueur ni la
 * victime n'est positionné : le clic a bien posé la lecture et sélectionné la ligne, mais il n'y
 * a rien à désigner sur la carte.
 */
export function resolveKillFocus(
  request: KillFocusRequest,
  r: KillFocusResolvers,
): ResolvedKillFocus | null {
  const killer = resolveActor(request.killerXuid, request.frame, r)
  const victim = resolveActor(request.victimXuid, request.frame, r)
  if (!killer && !victim) return null
  return { token: request.token, frame: request.frame, killer, victim, weapon: request.weapon }
}

/**
 * hiddenNameSlots : les slots dont le nom, sous le pion, cède la place à l'étiquette pleine. Un
 * second affichage du même nom sur le même pion est exactement ce que la mise en valeur écarte.
 */
export function hiddenNameSlots(focus: ResolvedKillFocus | null): ReadonlySet<number> {
  const slots = new Set<number>()
  for (const a of [focus?.killer, focus?.victim]) {
    if (a && a.slot !== null) slots.add(a.slot)
  }
  return slots
}
