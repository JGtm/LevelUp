/**
 * changeRefine.ts — LA DATATION FINE des lectures de fiche (schémas 24 et 25).
 *
 * LE PROBLÈME QU'IL RÉSOUT, ET IL EST MESURÉ. Les armes portées et la capacité d'armure se
 * lisent aux IMAGES-CLÉS du film, une toutes les ~20 s : sur les 21 899 fiches d'un match,
 * l'âge médian de la lecture d'armes est de 8,4 s, et 7,1 % seulement ont moins d'une seconde
 * (cf. `LoadoutReading`). Entre deux images-clés, la fiche montre donc l'état d'AVANT, estompé
 * pour le dire — c'est honnête, mais c'est tout ce qu'on savait.
 *
 * CE QUE LES DEUX NOUVEAUX CALQUES APPORTENT : le flux delta transmet le composant d'état
 * d'arme (schéma 25) et celui d'équipement (schéma 26) AU CHANGEMENT, datés à la milliseconde.
 * Là où une image-clé dit un ÉTAT échantillonné, ces événements datent la TRANSITION. La fiche
 * peut donc basculer À LA FRAME où le joueur a changé d'arme, au lieu d'attendre le prochain
 * relevé.
 *
 * LA LECTURE D'IMAGE-CLÉ RESTE LA BASE, ET CE N'EST PAS UN DÉTAIL : le canal delta est JUSTE
 * (sur 5 627 tirs de trois films, il ne retire jamais une arme encore utilisée) mais sa
 * COMPLÉTUDE n'est pas prouvée — rien ne dit qu'il voit TOUTES les prises (cf.
 * document_weapon_changes.go). Reconstruire l'état à partir des seuls événements dériverait à
 * la première émission manquée. On part donc du relevé, et on n'applique QUE ce qui s'est passé
 * APRÈS lui : la prochaine image-clé resynchronise tout, quoi qu'il arrive.
 *
 * CE QUE LA VERSION DE CE LOT N'APPLIQUE PAS, ET POURQUOI C'EST ÉCRIT ICI PLUTÔT QUE FAIT :
 *
 *   - LES LÂCHERS (`kind: 'dropped'`, `w` vide) ne retirent pas l'arme de la rangée. Le film ne
 *     donne AUCUN index d'emplacement sur l'événement : retirer une entrée décalerait les
 *     indices que le sélecteur d'emplacement dégainé (`Inventory.d`, lu à une AUTRE image-clé,
 *     avec son propre âge) adresse — la fiche marquerait « en main » l'arme voisine. Le prix de
 *     l'abstention est nommé : une arme lâchée reste affichée, estompée, jusqu'au prochain
 *     relevé. C'est exactement le comportement d'avant ce lot.
 *   - LES PRISES SUR EMPLACEMENT VIDE (`from` vide) n'ajoutent rien à la rangée, pour la même
 *     raison inversée : la rangée lue n'a pas d'emplacement libre déclaré, et en inventer un
 *     ferait apparaître une troisième arme sur une fiche qui n'en porte que deux.
 *
 * NE SONT DONC APPLIQUÉES QUE LES SUBSTITUTIONS D'IDENTITÉ — `from` connue ET présente dans la
 * rangée lue, `w` non vide. Elles ne changent NI la longueur de la rangée NI l'ordre de ses
 * emplacements : c'est la seule transformation qui ne peut pas désaligner le sélecteur.
 *
 * Tout ce fichier est PUR : aucun React, aucun document, donc testable sans monter quoi que ce
 * soit — même partage que `equippedLogic.ts` et `weaponPadTime.ts`.
 */
import {
  REPLAY_NO_ABILITY_RANK,
  type ReplayEquipmentChange,
  type ReplayWeaponChange,
} from '@/lib/api/types'

/**
 * La PROVENANCE d'une lecture de capacité venue du calque des changements.
 *
 * TROISIÈME VALEUR À CÔTÉ DE `kf` et `delta` (les deux canaux d'`abilities`), et elle dit une
 * chose qu'aucune des deux ne dit : ce n'est pas une LECTURE de ce que le joueur porte, c'est un
 * ÉVÉNEMENT daté qui l'a fait changer. Un appelant qui voudrait les distinguer le peut.
 */
export const ABILITY_SRC_CHANGE = 'chg'

/** Une lecture d'armes portées : la rangée et l'âge du relevé qui la fonde. */
export interface WeaponsReading {
  weapons: string[]
  age: number
}

/**
 * refineWeaponsReading rejoue, sur une rangée lue à l'image-clé, les CHANGEMENTS D'ARME datés
 * du même slot survenus depuis — et rend la rangée à jour, avec l'âge de la dernière preuve.
 *
 * `readFrame` est l'image du relevé (`frame - age`). Les événements qui la précèdent sont déjà
 * DANS le relevé : les rejouer reviendrait à appliquer deux fois la même transition.
 *
 * UNE LECTURE À VENIR NE SE RAFFINE PAS. Avant la première image-clé d'une vie, le relevé rendu
 * est le plus proche À VENIR et son âge est NÉGATIF (cf. `nearestReading`) : lui appliquer des
 * événements PASSÉS ferait remonter le temps. L'appelant reçoit alors sa lecture inchangée.
 */
export function refineWeaponsReading(
  base: WeaponsReading,
  changes: readonly ReplayWeaponChange[],
  slot: number,
  frame: number,
): WeaponsReading {
  if (base.age < 0 || changes.length === 0) return base
  const readFrame = frame - base.age
  const weapons = [...base.weapons]
  let applied = -1
  // L'ORDRE CHRONOLOGIQUE EST OBLIGATOIRE, et la liste du document ne le garantit pas : deux
  // substitutions sur le même emplacement doivent s'enchaîner dans l'ordre où elles ont eu
  // lieu, sans quoi la fiche montrerait l'avant-dernière arme.
  const utiles = changes
    .filter(
      (c): c is ReplayWeaponChange & { w: string; from: string } =>
        c.slot === slot && c.t > readFrame && c.t <= frame && !!c.w && !!c.from,
    )
    .sort((a, b) => a.t - b.t)
  for (const c of utiles) {
    const at = weapons.indexOf(c.from)
    // LA RANGÉE DOIT NOMMER L'ARME QU'ON REMPLACE. Sinon les deux lectures sont désappariées
    // (le relevé ne portait pas cette arme) et on s'abstient — même honnêteté que la mise en
    // valeur « en main », qui refuse de désigner un emplacement que le loadout ne porte pas.
    if (at < 0) continue
    weapons[at] = c.w
    applied = c.t
  }
  if (applied < 0) return base
  return { weapons, age: frame - applied }
}

/** Une lecture de capacité : le rang de palette, l'âge du relevé, et son canal. */
export interface AbilityRankReading {
  rank: number
  age: number
  src: string
}

/**
 * refineAbilityReading départage la lecture d'`abilities` et le dernier CHANGEMENT D'ÉQUIPEMENT
 * du même slot : la plus RÉCENTE des deux gagne.
 *
 * MÊME DOCTRINE QUE LES DEUX CANAUX D'`abilities` (« la lecture la plus récente gagne, quel que
 * soit son canal ») : les deux disent la même grandeur — le rang de palette porté —, l'une par
 * échantillonnage, l'autre par événement daté.
 *
 * UNE CONSOMMATION REND `null`, ET C'EST UNE MESURE. Le document publie `r` à
 * `REPLAY_NO_ABILITY_RANK` sur un `spent` : le joueur ne porte plus rien. La mesure est solide —
 * sur les 17 émissions à porte ouverte du corpus, aucune ne tombe dans la dernière seconde de la
 * vie, ce n'est donc jamais la mort qui vide l'emplacement. Rendre `null` fait DISPARAÎTRE la
 * vignette de la fiche, ce qui est exactement ce qu'il faut montrer ; la garder jusqu'au
 * prochain relevé affichait un équipement déjà dépensé.
 *
 * UN ÉVÉNEMENT À VENIR N'EST JAMAIS LU (`c.t <= frame`) : le rejeu connaît la suite, la fiche
 * n'a pas le droit de s'en servir.
 */
export function refineAbilityReading(
  base: AbilityRankReading | null,
  changes: readonly ReplayEquipmentChange[],
  slot: number,
  frame: number,
): AbilityRankReading | null {
  let last: ReplayEquipmentChange | null = null
  for (const c of changes) {
    if (c.slot !== slot || c.t > frame) continue
    if (!last || c.t > last.t) last = c
  }
  if (!last) return base
  const age = frame - last.t
  // LE RELEVÉ GAGNE À ÉGALITÉ ET AU-DELÀ : une lecture d'image-clé postérieure à l'événement a
  // déjà vu son effet, et une lecture À VENIR (âge négatif) n'a jamais le droit de primer une
  // information passée — d'où la comparaison sur l'âge et non sur l'instant.
  if (base && base.age >= 0 && base.age <= age) return base
  if (last.r === REPLAY_NO_ABILITY_RANK) return null
  return { rank: last.r, age, src: ABILITY_SRC_CHANGE }
}
