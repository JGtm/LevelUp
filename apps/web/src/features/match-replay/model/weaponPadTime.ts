/**
 * weaponPadTime.ts — CE QUE LE FILM DIT D'UN SOCLE À UN INSTANT : son occupation, son état, et
 * le temps qui reste avant le retour de l'arme.
 *
 * POURQUOI CE FICHIER EXISTE, séparé du calque : cette lecture est du TEMPS PUR, pas du dessin.
 * Elle sert au tracé, au survol et à l'infobulle — trois appelants pour une seule règle — et
 * elle n'a besoin ni d'un canvas, ni d'une encre, ni d'une densité de pixels. Extraite de
 * `weaponPadsLayer.ts` le 2026-08-27, quand le compte à rebours a gagné sa seconde source (D3)
 * et que le calque a dépassé le seuil de taille du dépôt.
 *
 * TROIS ÉTATS, ET LE TROISIÈME EST L'HONNÊTETÉ DE LA MESURE. Une occupation publie trois
 * instants : `t0` l'apparition (mesurée), `tLow` le dernier instant où l'arme est PROUVÉE
 * présente, `tHigh` le premier où son absence est prouvée. Entre les deux, le film ne dit rien —
 * les images-clés sont espacées de ~20 s. Un état qui basculerait pile à un instant affirmerait
 * une datation que la source n'a pas : l'intervalle s'appelle donc INCERTAIN et se lit comme tel
 * à l'écran.
 *
 * LE RAMASSEUR N'EST PAS ICI, et il ne le sera pas : le champ existe au contrat
 * (`padPickups[].xuid`) et est PUBLIÉ depuis le schéma 30 (2026-08-31) — l'événement natif le
 * porte. Ce module ne le lit pas : il ne calcule que des instants.
 * Aucune ligne de ce fichier ne le lit.
 */
import type { ReplayWeaponPadReady } from '../../../lib/replay/replayNormalize'

/**
 * L'état d'un socle à un instant : plein (présence prouvée), incertain (le film ne dit rien),
 * vide (absence prouvée). `empty` couvre aussi l'avant-première-apparition.
 */
export type PadState = 'full' | 'uncertain' | 'empty'

/**
 * padOccupancyIndexAt — le RANG de l'occupation en cours, ou -1 avant la première.
 *
 * LE RANG ET NON L'OCCUPATION, parce que le compte à rebours a besoin de la SUIVANTE : depuis
 * D3 (2026-08-27) il vise la prochaine apparition MESURÉE, et on ne l'atteint qu'en sachant où
 * l'on est dans la liste.
 */
function padOccupancyIndexAt(pad: ReplayWeaponPadReady, frame: number): number {
  let found = -1
  for (let i = 0; i < pad.presence.length; i++) {
    if (pad.presence[i].t0 > frame) break
    found = i
  }
  return found
}

/**
 * padOccupancyAt — l'occupation en cours à cette image, c'est-à-dire la DERNIÈRE dont
 * l'apparition a eu lieu. Null avant la première : le socle n'a alors rien porté du tout.
 */
export function padOccupancyAt(
  pad: ReplayWeaponPadReady,
  frame: number,
): ReplayWeaponPadReady['presence'][number] | null {
  const i = padOccupancyIndexAt(pad, frame)
  return i < 0 ? null : pad.presence[i]
}

/**
 * padStateAt — l'état du socle à cette image.
 *
 * L'ordre des comparaisons EST la règle : plein tant que la présence est prouvée, incertain
 * tant que l'absence ne l'est pas, vide ensuite.
 *
 * LE CAS « JAMAIS VIDÉ » EST À PART, et il est fréquent (8 occupations sur 28 sur un des
 * témoins) : quand l'arme est encore recensée à la DERNIÈRE image-clé, `tHigh` ne dépasse pas
 * `tLow` — aucune absence n'a jamais été prouvée. Le socle reste alors PLEIN jusqu'au bout.
 * L'écrire vide, fût-ce une image, affirmerait un ramassage que rien n'a observé.
 */
export function padStateAt(pad: ReplayWeaponPadReady, frame: number): PadState {
  const occ = padOccupancyAt(pad, frame)
  if (!occ) return 'empty'
  if (frame < occ.tLow || occ.tHigh <= occ.tLow) return 'full'
  return frame < occ.tHigh ? 'uncertain' : 'empty'
}

/**
 * Le compte à rebours, ET D'OÙ IL VIENT — la seconde moitié de l'information (D3, 2026-08-27).
 *
 * `measured` distingue deux chiffres que rien ne séparait à l'écran : la prochaine apparition
 * VUE DANS LE FILM (exacte, le rejeu connaît la suite) et celle que le CYCLE prédit (une
 * moyenne, qui peut tomber à côté). Le calque n'en fait rien — un socle vide reste un socle
 * vide — mais l'infobulle le dit, et c'est là que la réserve doit se lire.
 */
export interface PadRespawn {
  seconds: number
  measured: boolean
}

/**
 * padRespawnAt — les secondes restantes avant la réapparition, et leur provenance, ou null.
 *
 * L'ORDRE DES SOURCES EST LA RÈGLE (D3, retour utilisateur du 2026-08-27 : « compteur pas
 * toujours visible ») : le rejeu connaît LA SUITE DU FILM, donc la prochaine apparition mesurée
 * l'emporte toujours sur la prédiction. Le cycle ne sert plus que pour le DERNIER trou, celui
 * qu'aucune apparition ne ferme. Sans l'un ni l'autre : rien — un tiret suggérerait qu'on sait.
 *
 * CE QUE ÇA CHANGE, ET C'ÉTAIT LE DÉFAUT SIGNALÉ : le compte s'affichait UNIQUEMENT sur les
 * socles à cycle établi (24 sur 57 n'en portent pas), donc jamais sur la moitié de la carte,
 * alors même que le film montrait l'arme revenir. Il s'affiche désormais sur TOUS les trous
 * refermés par une apparition, cycle ou pas.
 *
 * DEUX CONDITIONS INCHANGÉES : le socle est VIDE (avant `tHigh`, rien n'est fini) et le compte
 * n'est pas épuisé. Pour le repli prédictif, le départ reste `tHigh`, la borne HAUTE de la
 * disparition : partir de `tLow` avancerait la prédiction d'un intervalle que la source ne date
 * pas.
 */
export function padRespawnAt(
  pad: ReplayWeaponPadReady,
  frame: number,
  frameMs: number,
): PadRespawn | null {
  if (!(frameMs > 0) || padStateAt(pad, frame) !== 'empty') return null
  const i = padOccupancyIndexAt(pad, frame)
  if (i < 0) return null
  // `next.t0 > frame` PAR CONSTRUCTION (l'occupation courante est la dernière dont l'apparition
  // a eu lieu) : le compte mesuré est toujours positif, il n'y a aucune garde à écrire ici.
  const next = pad.presence[i + 1]
  if (next) return { seconds: ((next.t0 - frame) * frameMs) / 1000, measured: true }
  const cycle = pad.cycle
  if (!cycle || !(cycle.medianS > 0)) return null
  const left = cycle.medianS - ((frame - pad.presence[i].tHigh) * frameMs) / 1000
  return left > 0 ? { seconds: left, measured: false } : null
}
