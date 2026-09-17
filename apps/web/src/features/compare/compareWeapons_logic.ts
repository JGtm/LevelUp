/**
 * compareWeapons_logic — LES DÉCISIONS DE LECTURE DU PROFIL D'ARMES, hors JSX.
 *
 * Tout ce que la section doit DÉCIDER (quelles classes aligner, dans quel ordre, comment se
 * nomme un rôle, ce que dit la ligne « sous le seuil ») vit ici, pur et testable. Le composant
 * ne fait que poser des nœuds ; l'option ECharts vient de `features/synthesis/_weaponRangeChart`.
 *
 * # CE QUI DIFFÈRE DE LA SYNTHÈSE, ET POURQUOI CE FICHIER EXISTE
 *
 * La Synthèse décrit UN joueur : ses lignes viennent d'un seul bloc, l'ordre du backend fait
 * foi. Le Face-à-face en superpose DEUX, dont les blocs n'ont ni les mêmes classes ni les mêmes
 * rôles — l'un a fragué au fusil de précision, l'autre jamais. Il faut donc UNIR les deux
 * listes sans en privilégier une, et publier un zéro là où un côté n'a rien : une ligne absente
 * d'une colonne se lirait « pas de donnée » alors que la vérité est « il ne l'a jamais fait ».
 */
import type {
  CompareFragClass,
  CompareWeaponSide,
  WeaponRangeRow,
  WeaponRangeSide,
} from '@/lib/api/types'
import type { WeaponRangeLine } from '@/features/synthesis/_weaponRangeChart'

/** Une classe alignée sur les deux joueurs — l'un des deux côtés peut être à zéro. */
export interface FragClassRow {
  classKey: string
  killsA: number
  killsB: number
  sharePctA: number
  sharePctB: number
}

/** La classe vide, pour le côté qui n'a pas joué cette classe. */
const CLASSE_ABSENTE: CompareFragClass = { class: '', kills: 0, share_pct: 0 }

/**
 * fragClassRows — l'union des classes des deux joueurs, alignée ligne à ligne.
 *
 * # L'ORDRE EST CELUI DE A, PUIS CE QUE B AJOUTE
 *
 * Les deux blocs arrivent déjà dans l'ordre canonique du backend (épaule, poing, lourde,
 * mêlée, grenade, capacités, véhicule, tourelle, équipement, environnement, non attribué).
 * Prendre A d'abord conserve cet ordre pour la colonne de référence, et les classes que seul B
 * porte se rangent à la suite dans LEUR ordre canonique — pas de tri maison, qui serait une
 * seconde doctrine divergeant de celle du backend au premier ajout de classe.
 *
 * UNE CLASSE ABSENTE D'UN CÔTÉ VAUT ZÉRO, JAMAIS « absent ». Le backend omet déjà les classes
 * à zéro frag (elles n'apprennent rien seules) ; mais mises FACE à une colonne qui en porte,
 * elles disent quelque chose — « lui n'a jamais tué à la grenade » est une information de style,
 * exactement ce que la section compare.
 */
export function fragClassRows(
  sideA: CompareWeaponSide | null | undefined,
  sideB: CompareWeaponSide | null | undefined,
): FragClassRow[] {
  const parA = new Map((sideA?.frag_classes ?? []).map((c) => [c.class, c]))
  const parB = new Map((sideB?.frag_classes ?? []).map((c) => [c.class, c]))
  const ordre: string[] = []
  for (const c of sideA?.frag_classes ?? []) ordre.push(c.class)
  for (const c of sideB?.frag_classes ?? []) if (!parA.has(c.class)) ordre.push(c.class)

  return ordre.map((classKey) => {
    const a = parA.get(classKey) ?? CLASSE_ABSENTE
    const b = parB.get(classKey) ?? CLASSE_ABSENTE
    return {
      classKey,
      killsA: a.kills,
      killsB: b.kills,
      sharePctA: a.share_pct,
      sharePctB: b.share_pct,
    }
  })
}

/** Le côté du contrat lu pour un graphe : celui des frags, ou celui des morts. */
export type RangeSideKey = 'kills' | 'deaths'

/** Le côté demandé d'une ligne du contrat, normalisé en `null` s'il n'est pas mesuré. */
function sideOf(row: WeaponRangeRow | undefined, side: RangeSideKey): WeaponRangeSide | null {
  if (!row) return null
  return (side === 'kills' ? row.kills : row.deaths) ?? null
}

/**
 * roleRangeLines — l'union des RÔLES des deux joueurs, sur un seul côté de mesure.
 *
 * # DEUX GRAPHES, PAS QUATRE BÂTONS
 *
 * Empiler « A frague », « A meurt », « B frague », « B meurt » sur une même bande donnerait
 * quatre bâtons illisibles. La section produit donc DEUX graphes — « Où ils fraguent », « Où ils
 * meurent » — et chacun superpose les deux JOUEURS : `top` = A, `bottom` = B. C'est ce que le
 * renommage `top`/`bottom` du module de rendu rend possible sans mensonge de nommage.
 *
 * # LE TRI EST CELUI DE LA MÉDIANE DE A, PUIS DE CELLE DE B
 *
 * Même doctrine que la Synthèse (D6) : le graphe se lit du contact à la longue portée. La
 * médiane de A prime, celle de B départage les rôles que A n'a pas — sans quoi ces rôles-là
 * se rangeraient arbitrairement en tête ou en queue. À médianes égales, la clé de rôle tranche :
 * deux chargements des mêmes données rendent le même ordre.
 *
 * LE LIBELLÉ EST RÉSOLU ICI, par la fonction injectée : ce module ne connaît aucun manifeste.
 */
export function roleRangeLines(
  sideA: CompareWeaponSide | null | undefined,
  sideB: CompareWeaponSide | null | undefined,
  side: RangeSideKey,
  roleLabel: (key: string) => string,
): WeaponRangeLine[] {
  const parA = new Map((sideA?.range?.weapons ?? []).map((w) => [w.weapon_key, w]))
  const parB = new Map((sideB?.range?.weapons ?? []).map((w) => [w.weapon_key, w]))
  const cles = new Set<string>([...parA.keys(), ...parB.keys()])

  const lignes = [...cles]
    .map((key) => ({
      weaponKey: key,
      label: roleLabel(key),
      top: sideOf(parA.get(key), side),
      bottom: sideOf(parB.get(key), side),
    }))
    // Un rôle qu'AUCUN des deux n'a mesuré de ce côté n'a pas de bande à occuper : il
    // viendrait d'une ligne publiée pour l'autre côté seulement.
    .filter((l) => l.top !== null || l.bottom !== null)

  lignes.sort((x, y) => {
    const mx = x.top?.median ?? x.bottom?.median ?? 0
    const my = y.top?.median ?? y.bottom?.median ?? 0
    if (mx !== my) return mx - my
    return x.weaponKey < y.weaponKey ? -1 : 1
  })
  return lignes
}

/**
 * roleLabel — le nom affiché d'une clé de RÔLE.
 *
 * # TROIS ESSAIS, ET JAMAIS UNE CLÉ BRUTE (D8)
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

/**
 * hasWeaponProfile — la section a-t-elle quoi que ce soit à montrer ?
 *
 * Les trois blocs sont indépendants côté contrat : il suffit qu'UN des deux joueurs porte une
 * classe, une ligne de portée ou une arme pour que la section ait un sens. Tout vide = pas de
 * section, jamais une section vide.
 */
export function hasWeaponProfile(
  sideA: CompareWeaponSide | null | undefined,
  sideB: CompareWeaponSide | null | undefined,
): boolean {
  const porte = (s: CompareWeaponSide | null | undefined) =>
    (s?.frag_classes?.length ?? 0) > 0 ||
    (s?.top_weapons?.length ?? 0) > 0 ||
    (s?.range?.weapons?.length ?? 0) > 0
  return porte(sideA) || porte(sideB)
}
