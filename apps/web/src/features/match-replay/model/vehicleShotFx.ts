/**
 * vehicleShotFx.ts — LE STYLE D'ÉCLAIR D'UNE ARME DE VÉHICULE : sa FORME et sa TEINTE.
 *
 * # LE DÉFAUT QUE CE FICHIER FERME (mesuré au lot 5.2a.5, 2026-09-20)
 *
 * `buildShotFx` lit la forme et la teinte d'un tir dans `weaponLabels[Shot.w]` — le registre
 * des armes DE JOUEUR. Les armes DE VÉHICULE n'y sont PAS : leur `Shot.w` vaut
 * `0x<weap>00000000` et le registre ne les nomme jamais. **68 % des tirs de véhicule de
 * `4f77afc1`** tombaient donc sur la famille `plain` et la teinte `neutral` — un halo gris pâle
 * centré sur un sprite, qui ne se lit pas comme un tir. C'est la moitié du constat utilisateur
 * du 2026-09-19 (« toujours pas d'effets de tir pour les véhicules ») que 5.2a.5 n'a pas pu
 * traiter : il a rendu la DIRECTION, pas le style.
 *
 * # POURQUOI LA TABLE EST ICI ET NON DANS `replay_labels.toml` — ET C'EST UNE MESURE, PAS UN GOÛT
 *
 * Le brief du lot 5.8 la voulait dans le manifeste du titre (`[shot_effects]` / `[shot_tints]`).
 * VÉRIFICATION SUR PIÈCES : ces deux tables sont keyées par `weapon_key` (`hinf_br75`…) et le
 * seul chemin qui les porte au client est `weaponLabels`, composé À LA REQUÊTE par
 * `service/replay_weapon_labels.go` pour les seules armes que le REGISTRE CANONIQUE nomme
 * (`FamilyOfWeaponID` -> `cat.Keys[family]`). Une arme de véhicule n'y a pas de famille : rien
 * ne l'y ferait entrer sans un champ de document neuf, c'est-à-dire une MONTÉE DE SCHÉMA que le
 * lot s'interdit.
 *
 * ET SURTOUT, LA FAIRE ENTRER DANS `weaponLabels` CASSERAIT DEUX RÈGLES MESURÉES : la présence
 * d'une clé dans `weaponLabels` EST le discriminateur « arme de véhicule » de
 * `shotFx.vehicleShotSourceOf` (5.2a.5) **et** la garde du repli sonore de
 * `replaySound.shotSoundStem` (lot sons de tir, 2026-09-04). Publier ces armes côté serveur
 * rendrait donc muets les tirs de véhicule et leur reprendrait la direction que 5.2a.5 vient de
 * leur donner. La table vit donc CÔTÉ CLIENT, au même endroit et sous la même forme que sa
 * jumelle sonore (`sound/vehicleShotSound.ts`) — même clé, même gabarit `vehicleWeapTag`, même
 * régime d'absence.
 *
 * # CE QUE LA TABLE DIT, ET CE QU'ELLE NE DIT PAS
 *
 * Elle nomme une FORME (`shot_effects` du dépôt : la mécanique du projectile) et une NATURE de
 * décharge (`shot_tints` : ce qui sort du canon). JAMAIS une couleur : la couleur reste un token
 * du thème, résolu par `fxInk`. Les valeurs sont celles des deux listes fermées du dépôt, pas
 * des noms neufs.
 *
 * LE PARTAGE SUIT LA DÉCISION DU LOT : énergie pour le Ghost, la Banshee et le Wraith (armes à
 * plasma Banished), cinétique pour le Warthog, le Scorpion, le Wasp, le Chopper, le Gungoose et
 * la tourelle du Falcon. Il ne cherche pas la finesse d'un catalogue d'armes de joueur : deux
 * styles franchement distincts valent mieux à l'écran qu'une taxonomie que rien ne mesure.
 *
 * LE SHADE N'Y EST PAS, ET IL NE PEUT PAS Y ÊTRE : aucun rapport de RE ne documente son tag
 * `weap` (même trou que pour son montage et son son). Pas de tag = pas d'entrée, jamais un tag
 * voisin réemployé en devinant.
 *
 * LE WARTHOG GARDE SA RÉSERVE ÉCRITE : son unique tag (`c7d50912`) ne départage pas LAAG / Gauss
 * / roquettes. Les trois sont cinétiques, la ligne vaut donc pour les trois — c'est le SON qui
 * souffre de l'ambiguïté, pas le style.
 */
import type { FxTint } from '../layers/fxInk'
import type { ShotFamily } from '../layers/shotEffects'
import { vehicleWeapTag } from './vehicleWeaponMounts'

/** Le style d'un tir : la FORME de l'éclair et la NATURE de la décharge. */
export interface VehicleShotStyle {
  fx: ShotFamily
  tint: FxTint
}

/**
 * Les deux styles du lot, nommés une fois — onze entrées les partagent, et les répéter en clair
 * aurait invité à les faire diverger une ligne à la fois.
 */
const PLASMA: VehicleShotStyle = { fx: 'plasma', tint: 'plasma_cool' }
const BALISTIQUE: VehicleShotStyle = { fx: 'ballistic', tint: 'kinetic' }

/**
 * VEHICLE_SHOT_FX — `Shot.w` (gabarit `weap`) -> style. Un tag absent garde le rendu neutre :
 * c'est le comportement d'avant ce fichier, jamais le style d'une arme voisine.
 */
const VEHICLE_SHOT_FX: ReadonlyMap<string, VehicleShotStyle> = new Map([
  // ÉNERGIE (plasma Banished).
  [vehicleWeapTag('00015435'), PLASMA], // Ghost — canons à plasma jumeaux.
  [vehicleWeapTag('0000aa68'), PLASMA], // Banshee M1 — canons à plasma.
  [vehicleWeapTag('0000aa69'), PLASMA], // Banshee M2 — bombe à combustible.
  [vehicleWeapTag('121b4009'), PLASMA], // Wraith — mortier à plasma.
  // CINÉTIQUE.
  [vehicleWeapTag('c7d50912'), BALISTIQUE], // Warthog — LAAG / Gauss / roquettes (tag non départagé).
  [vehicleWeapTag('00015cfa'), BALISTIQUE], // Scorpion — canon principal.
  [vehicleWeapTag('11725dc4'), BALISTIQUE], // Wasp M1 — autocanon de menton.
  [vehicleWeapTag('d3c407ed'), BALISTIQUE], // Wasp M2 — missiles (muets, mais ils se VOIENT).
  [vehicleWeapTag('b40e9618'), BALISTIQUE], // Chopper — canons jumeaux avant.
  [vehicleWeapTag('0042678e'), BALISTIQUE], // Gungoose — mitrailleuses avant.
  [vehicleWeapTag('00015cd3'), BALISTIQUE], // Falcon — tourelle LMG.
])

/**
 * vehicleShotStyleOf — le style d'une arme de véhicule, ou `null`.
 *
 * APPELÉ EN SECOND, exactement comme `vehicleShotSoundStem` : une arme de JOUEUR garde le style
 * de son registre. C'est l'appelant (`buildShotFx`) qui tient l'ordre — le mettre ici obligerait
 * cette table à connaître le document.
 */
export function vehicleShotStyleOf(tag: string | undefined): VehicleShotStyle | null {
  if (!tag) return null
  return VEHICLE_SHOT_FX.get(tag) ?? null
}
