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
 * LE PARTAGE, DEPUIS LES RETOURS DU 2026-09-23 (lot L1.5, décisions utilisateur Q6, Q7, Q8) :
 * le plasma Banished est ROUGE — `plasma_hot`, la teinte du Ravageur — pour le Ghost, les canons
 * de la Banshee, le mortier du Wraith et les canons du Chopper (forme plasma) ; cinétique pour le
 * Warthog, le Wasp, le Gungoose et les tourelles du Falcon ; OBUS (forme `explosive`, explosion à
 * l'impact) pour le canon du Scorpion et le lance-grenades du Falcon. La bombe de la Banshee
 * (`0000aa69`) garde le plasma froid : sa teinte n'est pas décidée.
 *
 * LES CLÉS SONT DES TAGS OBSERVÉS, et un garde-rail le tient (`vehicleWeaponTags.guard.test.ts`,
 * fixture datée du parc) : le Scorpion est publié sous `49E40D17` (et non `00015cfa`, le tag du
 * module jamais vu dans un document) ; les armes à TIR CONTINU (Ghost, canons de la Banshee,
 * Chopper, LMG du Falcon, second mode du Wasp) gardent leur entrée — elles serviront dès que le
 * décodeur lira le tir continu (lot M4b) — et sont inscrites comme ATTENDUES NON OBSERVÉES.
 *
 * LE SHADE ET LA TOURELLE DU WRAITH N'Y SONT PAS, ET ILS NE PEUVENT PAS Y ÊTRE : aucun tag `weap`
 * n'est documenté pour eux (décision Q6 : ils seront `plasma_hot` le jour où leur tag sera lu).
 * Pas de tag = pas d'entrée, jamais un tag voisin réemployé en devinant.
 *
 * `c7d50912` N'EST PAS AMBIGU (erratum du 2026-09-23 sur le lot 5.8.4) : c'est le LANCE-ROQUETTES
 * du Rockethog (`WARTHOG_FINAL_2026-09-02.md` §1 : `vehi bcfb852f -> weap c7d50912`, banque
 * `veh_un_rockethog`) ; la LAAG est `0c6fd911` et le canon Gauss `8647925a`, jamais observés.
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
const PLASMA_ROUGE: VehicleShotStyle = { fx: 'plasma', tint: 'plasma_hot' }
const BALISTIQUE: VehicleShotStyle = { fx: 'ballistic', tint: 'kinetic' }
const OBUS: VehicleShotStyle = { fx: 'explosive', tint: 'kinetic' }

/**
 * VEHICLE_SHOT_FX — `Shot.w` (gabarit `weap`) -> style. Un tag absent garde le rendu neutre :
 * c'est le comportement d'avant ce fichier, jamais le style d'une arme voisine. EXPORTÉE pour le
 * garde-rail des tags observés (retrait avec les trois tables client, lot M4a).
 */
export const VEHICLE_SHOT_FX: ReadonlyMap<string, VehicleShotStyle> = new Map([
  // ÉNERGIE (plasma Banished) — ROUGE depuis le 2026-09-23 (Q6, Q7, Q8).
  [vehicleWeapTag('00015435'), PLASMA_ROUGE], // Ghost — canons à plasma jumeaux (tir continu, M4b).
  [vehicleWeapTag('0000aa68'), PLASMA_ROUGE], // Banshee M1 — canons à plasma (tir continu, M4b).
  [vehicleWeapTag('0000aa69'), PLASMA], // Banshee M2 — bombe à combustible (teinte non décidée).
  [vehicleWeapTag('121b4009'), PLASMA_ROUGE], // Wraith — mortier à plasma (172 tirs, 142 en véhicule).
  [vehicleWeapTag('b40e9618'), PLASMA_ROUGE], // Chopper — canons avant (tir continu, M4b).
  // CINÉTIQUE.
  [vehicleWeapTag('c7d50912'), BALISTIQUE], // Rockethog — lance-roquettes (132 tirs, 127 en véhicule).
  [vehicleWeapTag('11725dc4'), BALISTIQUE], // Wasp M1 — son au coup, 450/min (4 tirs, 4 en véhicule).
  [vehicleWeapTag('d3c407ed'), BALISTIQUE], // Wasp M2 — son en boucle, 600/min (jamais vu).
  [vehicleWeapTag('0042678e'), BALISTIQUE], // Gungoose — mitrailleuses avant (40 tirs, 15 en véhicule).
  [vehicleWeapTag('00015cd3'), BALISTIQUE], // Falcon — tourelle LMG (tir continu, M4b).
  // OBUS : explosion à l'impact.
  [vehicleWeapTag('49e40d17'), OBUS], // Scorpion — canon principal (tag PUBLIÉ, 13 tirs, 13 en véhicule).
  [vehicleWeapTag('0bb6976b'), OBUS], // Falcon — lance-grenades de porte (Q10, 52 tirs, 51 en véhicule).
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
