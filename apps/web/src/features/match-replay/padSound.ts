/**
 * padSound.ts — LE RAMASSAGE D'UNE ARME SUR SOCLE, daté à la frame.
 *
 * ## Pourquoi ce fichier existe, et ce qu'il contourne
 *
 * Le son est identifié depuis le 2026-08-26 (`play_007_abl_shared_pickup`, confirmé à l'oreille
 * par l'utilisateur : « c'est bien pour le ramassage des armes sur socles de power up »). Il
 * n'était pas branché parce que le canal évident ne sait pas DATER.
 *
 * **CE QUI EST MESURÉ, sur les 39 artefacts locaux :**
 *
 *   padPickups   1 234 ramassages, et l'écart `tHigh - tLow` a pour MÉDIANE 20,00 s.
 *                3,2 % seulement tombent sous 2 s. Le socle s'est vidé « quelque part » dans
 *                un intervalle de vingt secondes — poser un son là-dedans le ferait partir
 *                pendant que le joueur fait tout autre chose.
 *   loadouts     597 changements d'arme, et le même verdict : 0 sur 597 sont datés à moins de
 *                5 s. Les deux canaux publiés vivent sur la MÊME grille d'images-clés (~20 s).
 *
 * **CE QUI DATE, LUI, À LA FRAME : le TIR.** `doc.shots` porte `t`, `slot` et l'identifiant
 * d'arme complet, sur le même axe que les positions. Un joueur qui TIRE avec une arme la
 * TIENT — c'est une preuve de possession, pas une estimation.
 *
 * ## La règle, et ce qu'elle ne prétend pas
 *
 * Le son part au PREMIER TIR d'une famille d'arme qui appartient à un SOCLE de ce match
 * (`doc.weaponPads`), une fois par couple (joueur, famille). Rendement mesuré : 464 événements
 * sur 34 artefacts, soit ~14 par match — l'ordre de grandeur d'un ramassage d'arme lourde.
 *
 * CE N'EST PAS L'INSTANT DU RAMASSAGE, c'est celui de sa PREMIÈRE PREUVE, et l'écart est le
 * temps que le joueur met à tirer après avoir pris l'arme — court, par nature, sur une arme de
 * puissance. Deux conséquences assumées, écrites plutôt que masquées :
 *
 *   - une arme ramassée et JAMAIS tirée ne sonne pas. Le rejeu se tait au lieu d'inventer un
 *     instant, comme partout ailleurs dans cette chaîne ;
 *   - une arme de la même famille ramassée AILLEURS que sur un socle (au sol, sur un mort)
 *     sonne aussi. Distinguer les deux demanderait le ramasseur, que le calque REFUSE de
 *     publier (`PadPickup.XUID` vaut `null` partout : l'oracle indépendant donne 88,1 %, sous
 *     le seuil de 90 % que le lot s'était donné).
 *
 * UNE FOIS PAR COUPLE (joueur, famille) ET PAR MATCH : un joueur qui garde son fusil de
 * précision trois minutes ne doit pas ré-annoncer son ramassage à chaque tir.
 */
import type { ReplayDocumentReady } from './replayNormalize'
import { frameToMs } from './replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/**
 * Le son du ramassage sur socle. Banque `sb_007_abl_shared`, événement `c73036e4` =
 * `play_007_abl_shared_pickup` — nom cassé par hachage ET confirmé à l'oreille.
 */
export const PAD_PICKUP_SOUND_STEM = 'objective_pad_pickup'

/**
 * familleDe extrait la FAMILLE d'une arme de son identifiant complet : les 32 bits de poids
 * fort, écrits en hexadécimal minuscule. C'est la même clé que `WeaponPad.weapon` et que
 * `weaponLabels` — la jointure ne tient que si les deux côtés la calculent pareil, d'où cette
 * fonction unique plutôt qu'un `slice` recopié.
 */
export function familleDe(weaponID: string): string {
  const s = String(weaponID)
  return s.length >= 10 ? ('0x' + s.slice(2, 10)).toLowerCase() : s.toLowerCase()
}

/** Un socle, réduit à ce dont ce fichier a besoin (typage structurel). */
interface PadLike {
  weapon?: string | null
}

/** Un tir, réduit de même. */
interface ShotLike {
  t: number
  slot: number
  w?: string | null
}

/**
 * padPickupSoundEvents — un son par première preuve de possession d'une arme de socle.
 *
 * Rend une liste vide quand le match n'a aucun socle mesuré : sans catalogue de familles, la
 * règle n'a rien sur quoi joindre, et tous les tirs sonneraient.
 */
export function padPickupSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const pads = (doc.weaponPads ?? []) as readonly PadLike[]
  if (pads.length === 0) return []
  const familles = new Set<string>()
  for (const p of pads) {
    if (p.weapon) familles.add(familleDe(p.weapon))
  }
  if (familles.size === 0) return []

  const shots = ((doc.shots ?? []) as readonly ShotLike[])
    .filter((s) => typeof s.t === 'number' && s.w)
    .slice()
    .sort((a, b) => a.t - b.t)

  const vus = new Set<string>()
  const out: ReplaySoundEvent[] = []
  for (const s of shots) {
    const fam = familleDe(s.w as string)
    if (!familles.has(fam)) continue
    const cle = `${s.slot}|${fam}`
    if (vus.has(cle)) continue
    vus.add(cle)
    out.push(soundEvent(frameToMs(s.t, doc), PAD_PICKUP_SOUND_STEM))
  }
  return out
}
