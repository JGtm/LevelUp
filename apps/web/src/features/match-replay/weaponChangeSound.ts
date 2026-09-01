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
 *
 * ## Le canal NATIF vient COMBLER les trous, et surtout pas doubler (schéma 30, 2026-08-31)
 *
 * `doc.pickups` publie l'événement `biped_pickup` que la bobine écrit elle-même. Là où les deux
 * canaux voient la même prise, ils s'accordent (21/21 et 11/12 appariements arme nommée à moins
 * de 500 ms) — jouer les deux ferait sonner DEUX FOIS le même geste. Mais `weaponChanges` a un
 * rappel PARTIEL : sur le film de référence, les images-clés révèlent sept arrivées d'arme
 * qu'il n'explique pas (Gravity Hammer ×3, M41 SPNKr ×3, Stalker Rifle ×1), et le canal natif
 * en nomme cinq. Ces prises-là sont aujourd'hui MUETTES.
 *
 * La règle est donc : un ramassage natif d'ARME sonne SI ET SEULEMENT SI aucun `taken` /
 * `swapped` de `weaponChanges` ne le couvre (même vie, même famille, à moins de cinq frames —
 * la tolérance d'appariement de 500 ms sur une grille de 100 ms). C'est exactement la doctrine
 * appliquée côté Go aux classes d'équipement : combler un trou, jamais doublonner.
 *
 * LES RAMASSAGES NON-ARME NE SONNENT PAS ICI. Le canal natif les publie (classes 2 et 3 : équipement,
 * grenades, consommables) mais leur son est déjà l'affaire d'`equipmentChangeSound`. Leur donner
 * en plus le bruit du ramassage d'arme ferait entendre une arme là où il n'y en a pas.
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
  for (const p of nativePickupsNotAlreadyHeard(doc)) {
    out.push(soundEvent(frameToMs(p.t, doc), WEAPON_CHANGE_SOUND_STEMS.taken))
  }
  return out
}

/**
 * NATIVE_PICKUP_MATCH_FRAMES — la fenêtre qui décide qu'un ramassage natif et un `taken` de
 * `weaponChanges` sont LE MÊME geste. Cinq frames = 500 ms sur la grille de 100 ms du rejeu,
 * c'est-à-dire la tolérance sous laquelle l'accord des deux canaux a été mesuré côté Go.
 *
 * NON EXPORTÉ : rien hors de ce module n'a à connaître la fenêtre. La première version
 * l'exportait « au cas où » — personne ne l'importait, et `knip` l'aurait signalé.
 */
const NATIVE_PICKUP_MATCH_FRAMES = 5

/**
 * nativePickupsNotAlreadyHeard — les ramassages natifs d'ARME que `weaponChanges` ne couvre
 * pas. C'est la seule population qui a le droit de sonner en plus : tout le reste ferait
 * entendre deux fois le même geste (cf. l'en-tête). Non exporté, pour la même raison : son
 * seul consommateur est `weaponChangeSoundEvents`, et les tests passent par lui.
 */
function nativePickupsNotAlreadyHeard(
  doc: ReplayDocumentReady,
): ReplayDocumentReady['pickups'] {
  // Optionnel à dessein : un artefact antérieur au schéma 30 n'a pas de `pickups`, et le
  // rejeu doit se taire sur ce qu'il ne porte pas, pas lever.
  if (!doc.pickups?.length) return []
  const heard = doc.weaponChanges.filter((c) => c.kind !== 'dropped')
  return doc.pickups.filter((p) => {
    if (p.kind !== 'weapon') return false
    return !heard.some(
      (c) =>
        c.slot === p.slot &&
        c.w === p.w &&
        Math.abs(c.t - p.t) <= NATIVE_PICKUP_MATCH_FRAMES,
    )
  })
}
