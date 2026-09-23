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
 * LES SUBSTITUTIONS D'IDENTITÉ s'appliquent toujours — `from` connue ET présente dans la rangée
 * lue, `w` non vide. Elles ne changent NI la longueur de la rangée NI l'ordre de ses
 * emplacements : c'est la seule transformation qui ne peut pas désaligner le sélecteur
 * d'emplacement dégainé (`Inventory.d`, lu à une AUTRE image-clé, avec son propre âge).
 *
 * LES FAMILLES SE COMPARENT PAR LEUR ÉCRITURE CANONIQUE (schéma 69, lot M3.3). La rangée vient de
 * `loadouts[].w` (`0x%08X`), les changements de `weaponChanges[]` (`%08x`) : comparées telles
 * quelles, elles ne s'égalaient JAMAIS, et aucune substitution ne s'appliquait sur un artefact
 * réel — les tests de ce module écrivaient les deux côtés dans la même casse.
 *
 * LES LÂCHERS ET LES PRISES SUR EMPLACEMENT VIDE ne s'appliquent QUE sur une rangée SITUÉE — une
 * dotation de naissance, qui porte l'emplacement de chaque arme (`k`) — et par un changement qui
 * porte le sien (`weaponChanges[].k`, schéma 69). Même alors, seulement aux bords, pour que la
 * rangée reste contiguë et donc alignée sur le sélecteur : une prise sur l'emplacement qui SUIT
 * le dernier occupé, un lâcher du DERNIER. Ailleurs — et sur un relevé d'image-clé, qui ne situe
 * pas ses familles —, l'abstention d'avant reste la règle : une arme lâchée reste affichée,
 * estompée, jusqu'au prochain relevé, et aucune troisième arme n'est inventée.
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
  /** Provenance du relevé : `birth` = la dotation de naissance (schéma 69) ; absent = image-clé. */
  src?: string
  /** Emplacement de chaque arme de `weapons`, quand le relevé le situe (dotation de naissance). */
  k?: number[]
}

/**
 * familyKey — l'écriture CANONIQUE d'une famille d'arme (`0x%08X`), celle de `loadouts[].w` et
 * des clés de `weaponLabels`. SECONDE COPIE de la règle de `weaponLabelKeyOf`
 * (`features/match-replay/layers/useReplayWeaponPads.ts`) : `lib/` ne dépend jamais de
 * `features/`. Une troisième copie se centralise, avec son garde-rail (règle n°6 du dépôt).
 */
function familyKey(w: string): string {
  const hex = w.startsWith('0x') || w.startsWith('0X') ? w.slice(2) : w
  if (!/^[0-9a-fA-F]{1,8}$/.test(hex)) return w
  return `0x${hex.toUpperCase().padStart(8, '0')}`
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
  const rangee: Rangee = { weapons: [...base.weapons], k: base.k ? [...base.k] : undefined }
  let applied = -1
  // L'ORDRE CHRONOLOGIQUE EST OBLIGATOIRE, et la liste du document ne le garantit pas : deux
  // substitutions sur le même emplacement doivent s'enchaîner dans l'ordre où elles ont eu
  // lieu, sans quoi la fiche montrerait l'avant-dernière arme.
  const utiles = changes
    .filter((c) => c.slot === slot && c.t > readFrame && c.t <= frame)
    .sort((a, b) => a.t - b.t)
  for (const c of utiles) {
    if (appliquerChangement(rangee, c)) applied = c.t
  }
  if (applied < 0) return base
  return { ...base, weapons: rangee.weapons, k: rangee.k, age: frame - applied }
}

/** La rangée en cours de raffinement : ses armes et, quand elle est située, leurs emplacements. */
interface Rangee {
  weapons: string[]
  k: number[] | undefined
}

/**
 * appliquerChangement applique UN changement à la rangée et dit s'il l'a fait (cf. l'en-tête :
 * substitutions partout, prises et lâchers aux seuls bords d'une rangée située).
 */
function appliquerChangement(r: Rangee, c: ReplayWeaponChange): boolean {
  const { weapons, k } = r
  if (c.w && c.from) {
    // LA RANGÉE DOIT NOMMER L'ARME QU'ON REMPLACE. Sinon les deux lectures sont désappariées
    // (le relevé ne portait pas cette arme) et on s'abstient — même honnêteté que la mise en
    // valeur « en main », qui refuse de désigner un emplacement que le loadout ne porte pas.
    // L'emplacement du changement, quand la rangée est située, départage deux armes égales.
    const from = familyKey(c.from)
    const situe = k && c.k !== undefined ? k.indexOf(c.k) : -1
    const at = situe >= 0 && familyKey(weapons[situe]) === from
      ? situe
      : weapons.findIndex((w) => familyKey(w) === from)
    if (at < 0) return false
    weapons[at] = familyKey(c.w)
    return true
  }
  if (!k || c.k === undefined) return false
  if (c.w) {
    // Prise sur emplacement vide : seulement celui qui SUIT le dernier occupé.
    if (k.includes(c.k) || c.k !== weapons.length || k.some((e, i) => e !== i)) return false
    weapons.push(familyKey(c.w))
    k.push(c.k)
    return true
  }
  // Lâcher : seulement le DERNIER emplacement, et seulement l'arme que le changement nomme.
  const at = k.indexOf(c.k)
  if (at < 0 || at !== weapons.length - 1) return false
  if (c.from && familyKey(weapons[at]) !== familyKey(c.from)) return false
  weapons.pop()
  k.pop()
  return true
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
 *
 * `lifeStart` BORNE AUSSI LE PASSÉ (correctif P0-2, 2026-09-06,
 * `.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md`) — SITE FRÈRE DU MÊME DÉFAUT QUE
 * `nearestReading` (`rosterLogic.ts`) : avant ce correctif, seul `c.t <= frame` bornait la
 * recherche, et un `spent` de la vie PRÉCÉDENTE du même slot (recyclé) restait « le dernier
 * changement », faisant DISPARAÎTRE la vignette d'une vie neuve qui n'avait pourtant rien
 * consommé. `lifeStart` (le début de la vie couvrant `frame`) écarte tout changement antérieur
 * — une lecture hors de cette vie n'est jamais candidate.
 */
export function refineAbilityReading(
  base: AbilityRankReading | null,
  changes: readonly ReplayEquipmentChange[],
  slot: number,
  frame: number,
  lifeStart: number,
): AbilityRankReading | null {
  let last: ReplayEquipmentChange | null = null
  for (const c of changes) {
    if (c.slot !== slot || c.t > frame || c.t < lifeStart) continue
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
