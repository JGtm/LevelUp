/**
 * weaponRangeRoles — LES DÉCISIONS DE LECTURE D'UN GRAPHE DE PORTÉE PAR RÔLE, hors JSX.
 *
 * Quels rôles aligner, dans quel ordre, comment se nomme un rôle, et quelles lignes passer au
 * module de rendu (`weaponRangeChart`). Pur et testable ; aucun composant, aucun manifeste —
 * la résolution des libellés est INJECTÉE.
 *
 * # POURQUOI CE FICHIER EST PARTAGÉ (2026-09-17)
 *
 * Deux surfaces superposent deux JOUEURS sur la même bande : le profil d'armes du Face-à-face
 * (« Où ils fraguent », « Où ils meurent ») et le bloc « Portée des frags » de l'encart cible
 * de l'Explorer. Elles ont besoin du même axe, du même ordre et du même nommage. Ces
 * fonctions ont vécu dans `features/compare/` jusqu'à ce que l'Explorer les demande : les
 * recopier aurait donné deux ordres de lignes pour la même mesure, et les laisser sous
 * `features/compare/` aurait demandé une dérogation à l'anti-import inter-features.
 *
 * # L'ENTRÉE EST UN BLOC DE PORTÉE, PAS UN CÔTÉ DE PAGE
 *
 * `SynthesisWeaponRange | null | undefined` — le contrat commun que servent les deux pages.
 * Un joueur sans bloc est une entrée légitime (« il n'a rien de mesuré »), jamais une erreur.
 */
import type { SynthesisWeaponRange, WeaponRangeRow, WeaponRangeSide } from '@/lib/api/types'

import type { WeaponRangeLine } from './weaponRangeChart'

/** Le bloc de portée d'un joueur, ou son absence. */
export type WeaponRangeBlock = SynthesisWeaponRange | null | undefined

/** Le côté du contrat lu pour un graphe : celui des frags, ou celui des morts. */
export type RangeSideKey = 'kills' | 'deaths'

/** Une ligne de l'axe partagé : sa clé de rôle et son libellé déjà résolu. */
export interface RoleAxisEntry {
  weaponKey: string
  label: string
}

/** Le côté demandé d'une ligne du contrat, normalisé en `null` s'il n'est pas mesuré. */
function sideOf(row: WeaponRangeRow | undefined, side: RangeSideKey): WeaponRangeSide | null {
  if (!row) return null
  return (side === 'kills' ? row.kills : row.deaths) ?? null
}

/**
 * roleAxis — L'AXE DES RÔLES, calculé UNE FOIS pour TOUS les graphes d'une surface.
 *
 * # LE PROBLÈME QU'IL RÈGLE (gate visuel 2026-09-17)
 *
 * En miroir, les deux paires de graphes (A vs B, A vs C) étaient construites indépendamment :
 * un rôle présent chez A et C mais pas chez B donnait un axe plus court à gauche qu'à droite,
 * et les graphes ne se lisaient plus ligne à ligne. Constaté en vrai — un joueur avait des
 * frags « Environnement » que l'autre n'avait pas, et tout était décalé d'un cran.
 *
 * La règle est donc : L'AXE EST L'UNION DES RÔLES DE TOUS LES BLOCS REÇUS, frags ET morts
 * confondus. Un joueur sans mesure sur un rôle garde sa ligne, vide — c'est une information
 * (« lui n'a jamais fragué au corps à corps »), pas un trou à refermer.
 *
 * # L'ORDRE, ÉCRIT ICI ET NULLE PART AILLEURS
 *
 * Médiane de la RÉFÉRENCE (le premier bloc, côté FRAGS) croissante : le graphe se lit du
 * contact à la longue portée. Les rôles que la référence n'a pas mesurés ne peuvent pas se
 * ranger dans ce continuum — ils vont À LA FIN, triés par libellé, plutôt que de s'intercaler
 * à une place qu'aucune mesure ne justifie. À médianes égales, la clé tranche : deux
 * chargements des mêmes données rendent le même ordre.
 */
export function roleAxis(blocks: WeaponRangeBlock[], roleName: (key: string) => string): RoleAxisEntry[] {
  const reference = new Map((blocks[0]?.weapons ?? []).map((w) => [w.weapon_key, w]))
  const cles = new Set<string>()
  for (const b of blocks) for (const w of b?.weapons ?? []) cles.add(w.weapon_key)

  return [...cles]
    .map((weaponKey) => ({
      weaponKey,
      label: roleName(weaponKey),
      // La médiane des FRAGS de la référence, ou `null` si elle n'a rien mesuré là.
      rang: sideOf(reference.get(weaponKey), 'kills')?.median ?? null,
    }))
    .sort((x, y) => {
      // `localeCompare` et non `<` : une comparaison brute classe par point de code, donc
      // toutes les majuscules avant toutes les minuscules et les accents en dernier — un
      // ordre qui ne ressemble à rien pour un lecteur français.
      if (x.rang === null && y.rang === null) return x.label.localeCompare(y.label, 'fr')
      if (x.rang === null) return 1
      if (y.rang === null) return -1
      if (x.rang !== y.rang) return x.rang - y.rang
      return x.weaponKey < y.weaponKey ? -1 : 1
    })
    .map(({ weaponKey, label }) => ({ weaponKey, label }))
}

/**
 * roleRangeLines — les lignes d'UN graphe : un côté de mesure, deux joueurs superposés, SUR
 * L'AXE IMPOSÉ.
 *
 * # DEUX JOUEURS PAR GRAPHE, PAS QUATRE BÂTONS
 *
 * Empiler « A frague », « A meurt », « B frague », « B meurt » sur une même bande donnerait
 * quatre bâtons illisibles. Un graphe ne montre donc qu'UN côté de mesure, et y superpose les
 * deux JOUEURS : `top` = A, `bottom` = B. C'est ce que le nommage `top`/`bottom` du module de
 * rendu rend possible sans mensonge.
 *
 * # L'AXE VIENT DU DEHORS, ET RIEN N'EST FILTRÉ
 *
 * Toutes les lignes de l'axe sont rendues, DANS SON ORDRE, y compris celles où ni l'un ni
 * l'autre n'a de mesure de ce côté (`top` et `bottom` à `null`, l'infobulle dit « aucune
 * mesure »). C'est ce qui garde les graphes d'une même surface alignés ligne à ligne :
 * filtrer ici les lignes vides ferait réapparaître le décalage que `roleAxis` supprime.
 */
export function roleRangeLines(
  axis: readonly RoleAxisEntry[],
  top: WeaponRangeBlock,
  bottom: WeaponRangeBlock,
  side: RangeSideKey,
): WeaponRangeLine[] {
  const parTop = new Map((top?.weapons ?? []).map((w) => [w.weapon_key, w]))
  const parBottom = new Map((bottom?.weapons ?? []).map((w) => [w.weapon_key, w]))
  return axis.map(({ weaponKey, label }) => ({
    weaponKey,
    label,
    top: sideOf(parTop.get(weaponKey), side),
    bottom: sideOf(parBottom.get(weaponKey), side),
  }))
}

/**
 * roleLabel — le nom affiché d'une clé de RÔLE.
 *
 * # TROIS ESSAIS, ET JAMAIS UNE CLÉ BRUTE
 *
 * Le back publie des clés de rôle (`precision`, `automatic`, …). Certaines sont leur propre
 * rôle parce que le registre y pose `class == role` : `grenade`, `melee`, `sidearm`,
 * `equipment`, `vehicle`, `turret`, `environmental` — le manifeste les déclare alors sous
 * `frags.class.*` et non `frags.role.*`. D'où l'ordre : rôle, puis classe, puis la clé.
 *
 * `resolve` suit le contrat documenté de `formatMessage` : il rend la CLÉ ELLE-MÊME quand elle
 * est absente du manifeste. C'est ce signal qui permet d'enchaîner les essais, et c'est
 * pourquoi le repli final rend la clé NUE (`precision`) et jamais `frags.role.precision` —
 * même exigence que `fragRoleDisplayLabel` : aucun chemin n'affiche une clé `frags.` brute.
 */
export function roleLabel(key: string, resolve: (manifestKey: string) => string): string {
  const parRole = resolve(`frags.role.${key}`)
  if (parRole !== `frags.role.${key}`) return parRole
  const parClasse = resolve(`frags.class.${key}`)
  if (parClasse !== `frags.class.${key}`) return parClasse
  return key
}
