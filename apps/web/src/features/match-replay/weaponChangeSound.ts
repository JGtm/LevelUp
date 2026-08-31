/**
 * weaponChangeSound.ts — LE RAMASSAGE ET LE LÂCHER D'ARME, datés par le film.
 *
 * ## Pourquoi ce fichier peut exister depuis le 2026-08-30, et pas avant
 *
 * L'ancienne règle (`padSound.ts`, supprimée le même jour) jouait le ramassage au premier TIR
 * d'une famille d'arme de socle — mesures justes, conclusion fausse : la doctrine de cette
 * chaîne est que le rejeu SE TAIT plutôt que de deviner, et déplacer un son sur un autre geste
 * n'est pas se taire. Le chantier ramassage (schéma 25) a changé la donne : `doc.weaponChanges`
 * PUBLIE les prises et les lâchers, datés à la frame, par vie. Il n'y a plus rien à déduire —
 * on joue ce que le calque date, comme partout ailleurs.
 *
 * ## Les sons, et d'où ils viennent
 *
 * Les deux gestes vivent dans la banque de bruitage du Spartan (`sb_006_chm_un_spartan`,
 * `e9a52b26`) et portent la même signature Wwise : une couche, une variante parmi trois,
 * gain -6 dB.
 *
 *  - LE LÂCHER : `play_006_chm_un_spartan_weapondrop` (`6cdd92fd`, 0,31 s) — le seul événement
 *    de ces 230 dont le nom se casse, DÉSIGNÉ à l'oreille par l'utilisateur le 2026-08-30.
 *  - LE RAMASSAGE : `168832f6` (0,34 s) — nom non cassé, désigné le même jour PAR DÉFAUT
 *    (« faute de mieux ») parmi les onze gestes de la banque qui portent la signature du
 *    lâcher. La désignation est plus faible que les autres et le RE le dit (§ 12.5) ; si le
 *    rendu sonne faux en situation, les dix autres candidats restent sur la planche
 *    `3c84fab7-5e36-4777-a2d9-bd1c90b08f65`.
 *
 * ## La règle, et ses deux choix écrits
 *
 *  - UN `swapped` SONNE LE RAMASSAGE, ET RIEN D'AUTRE. Un échange est un lâcher et une prise
 *    au même instant ; jouer les deux fichiers superposés ferait un artefact de mixage, pas
 *    deux gestes. Ce qu'on entend d'un échange, c'est l'arme qui ARRIVE en main.
 *  - TOUTES LES VIES SONNENT, comme les tirs et les grenades : le calque publie l'auteur, le
 *    tiroir de réglages filtre par catégorie, pas par joueur. L'ordre de grandeur mesuré du
 *    canal est ~100-230 changements par match — celui des lancers de grenade.
 */
import type { ReplayDocumentReady } from './replayNormalize'
import { frameToMs } from './replayLogic'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'

/**
 * Les fichiers, par geste. Le `swapped` n'a pas d'entrée : il joue `taken` (cf. l'en-tête).
 */
export const WEAPON_CHANGE_SOUND_STEMS = {
  /** L'arme arrive en main — ramassage, ou moitié audible d'un échange. */
  taken: 'weapon_pickup',
  /** L'emplacement passe à vide : l'arme part au sol. */
  dropped: 'weapon_drop',
} as const

/**
 * weaponChangeSoundEvents — un son par changement d'arme publié, à sa frame.
 *
 * Aucun camp et aucun filtre : le geste appartient à celui qui le fait, et le document
 * n'écarte déjà les ré-annonces de spawn (ce ne sont pas des ramassages, cf.
 * document_weapon_changes.go côté Go).
 */
export function weaponChangeSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  for (const c of doc.weaponChanges) {
    const stem =
      c.kind === 'dropped' ? WEAPON_CHANGE_SOUND_STEMS.dropped : WEAPON_CHANGE_SOUND_STEMS.taken
    out.push(soundEvent(frameToMs(c.t, doc), stem))
  }
  return out
}
