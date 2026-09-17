/**
 * compareWeapons_logic — LES DÉCISIONS DE LECTURE DU PROFIL D'ARMES, hors JSX.
 *
 * Tout ce que la section doit DÉCIDER (quelles classes aligner, quels rôles, dans quel ordre,
 * comment se nomme un rôle) vit ici, pur et testable. Le composant ne fait que poser des
 * nœuds ; l'option ECharts vient de `features/synthesis/_weaponRangeChart`.
 *
 * # CE QUI DIFFÈRE DE LA SYNTHÈSE, ET POURQUOI CE FICHIER EXISTE
 *
 * La Synthèse décrit UN joueur : ses lignes viennent d'un seul bloc, l'ordre du backend fait
 * foi. Le Face-à-face en superpose DEUX — et TROIS en mode miroir — dont les blocs n'ont ni les
 * mêmes classes ni les mêmes rôles : l'un a fragué au fusil de précision, l'autre jamais. Il
 * faut donc UNIR les listes sans en privilégier une, et publier une ligne vide là où un côté
 * n'a rien : une ligne absente d'une colonne se lirait « pas de donnée » alors que la vérité
 * est « il ne l'a jamais fait ».
 */
import type {
  CompareFragClass,
  CompareWeaponSide,
  WeaponRangeRow,
  WeaponRangeSide,
} from '@/lib/api/types'
import type { WeaponRangeLine } from '@/features/synthesis/_weaponRangeChart'

/**
 * Un côté du profil, ou son absence. Les trois joueurs de la page miroir passent par les mêmes
 * fonctions : `null`/`undefined` est un côté légitime, pas un cas d'erreur.
 */
type Side = CompareWeaponSide | null | undefined

/** Une classe alignée sur les joueurs de la section — un côté sans cette classe vaut zéro. */
export interface FragClassRow {
  classKey: string
  /** Une entrée par joueur, DANS L'ORDRE DES CÔTÉS REÇUS. */
  parts: { kills: number; sharePct: number }[]
}

/** La classe vide, pour le côté qui n'a pas joué cette classe. */
const CLASSE_ABSENTE: CompareFragClass = { class: '', kills: 0, share_pct: 0 }

/**
 * fragClassRows — l'union des classes de TOUS les joueurs de la section, alignée ligne à ligne.
 *
 * # POURQUOI UN NOMBRE VARIABLE DE CÔTÉS (2026-09-17, gate visuel)
 *
 * En mode miroir la page compare TROIS joueurs. Une union à deux, appliquée deux fois, donnait
 * deux listes de classes différentes et deux colonnes décalées dès qu'un seul des trois portait
 * une classe que les autres n'ont pas — cas réel : des frags « Environnement » chez un joueur
 * seulement. L'union se calcule donc sur tous les côtés à la fois, et la ligne existe pour tout
 * le monde.
 *
 * # L'ORDRE EST CELUI DU PREMIER CÔTÉ, PUIS CE QUE LES SUIVANTS AJOUTENT
 *
 * Les blocs arrivent déjà dans l'ordre canonique du backend (épaule, poing, lourde, mêlée,
 * grenade, capacités, véhicule, tourelle, équipement, environnement, non attribué). Prendre le
 * premier côté conserve cet ordre pour la colonne de référence, et ce que les autres ajoutent
 * se range à la suite dans LEUR ordre canonique — pas de tri maison, qui serait une seconde
 * doctrine divergeant de celle du backend au premier ajout de classe.
 *
 * UNE CLASSE ABSENTE D'UN CÔTÉ VAUT ZÉRO, JAMAIS « absent ». Le backend omet déjà les classes
 * à zéro frag (elles n'apprennent rien seules) ; mais mises FACE à une colonne qui en porte,
 * elles disent quelque chose — « lui n'a jamais tué à la grenade » est une information de
 * style, exactement ce que la section compare.
 */
export function fragClassRows(...sides: Side[]): FragClassRow[] {
  const cartes = sides.map((s) => new Map((s?.frag_classes ?? []).map((c) => [c.class, c])))
  const ordre: string[] = []
  const vus = new Set<string>()
  for (const s of sides) {
    for (const c of s?.frag_classes ?? []) {
      if (!vus.has(c.class)) {
        vus.add(c.class)
        ordre.push(c.class)
      }
    }
  }
  return ordre.map((classKey) => ({
    classKey,
    parts: cartes.map((m) => {
      const c = m.get(classKey) ?? CLASSE_ABSENTE
      return { kills: c.kills, sharePct: c.share_pct }
    }),
  }))
}

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
 * roleAxis — L'AXE DES RÔLES DE LA SECTION, calculé UNE FOIS pour TOUS ses graphes.
 *
 * # LE PROBLÈME QU'IL RÈGLE (gate visuel 2026-09-17)
 *
 * En miroir, les deux paires de graphes (A vs B, A vs C) étaient construites indépendamment :
 * un rôle présent chez A et C mais pas chez B donnait un axe plus court à gauche qu'à droite,
 * et les graphes ne se lisaient plus ligne à ligne. Constaté en vrai — un joueur avait des
 * frags « Environnement » que l'autre n'avait pas, et tout était décalé d'un cran.
 *
 * La règle est donc : L'AXE EST L'UNION DES RÔLES DE TOUS LES JOUEURS DE LA SECTION, frags ET
 * morts confondus. Un joueur sans mesure sur un rôle garde sa ligne, vide — c'est une
 * information (« lui n'a jamais fragué au corps à corps »), pas un trou à refermer.
 *
 * # L'ORDRE, ÉCRIT ICI ET NULLE PART AILLEURS
 *
 * Médiane de la RÉFÉRENCE (le premier côté, côté FRAGS) croissante : le graphe se lit du
 * contact à la longue portée, comme la Synthèse (D6). Les rôles que la référence n'a pas
 * mesurés ne peuvent pas se ranger dans ce continuum — ils vont À LA FIN, triés par libellé,
 * plutôt que de s'intercaler à une place qu'aucune mesure ne justifie. À médianes égales, la
 * clé tranche : deux chargements des mêmes données rendent le même ordre.
 */
export function roleAxis(sides: Side[], roleName: (key: string) => string): RoleAxisEntry[] {
  const reference = new Map((sides[0]?.range?.weapons ?? []).map((w) => [w.weapon_key, w]))
  const cles = new Set<string>()
  for (const s of sides) for (const w of s?.range?.weapons ?? []) cles.add(w.weapon_key)

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
 * # DEUX GRAPHES, PAS QUATRE BÂTONS
 *
 * Empiler « A frague », « A meurt », « B frague », « B meurt » sur une même bande donnerait
 * quatre bâtons illisibles. La section produit donc DEUX graphes — « Où ils fraguent », « Où
 * ils meurent » — et chacun superpose les deux JOUEURS : `top` = A, `bottom` = B. C'est ce que
 * le renommage `top`/`bottom` du module de rendu rend possible sans mensonge de nommage.
 *
 * # L'AXE VIENT DU DEHORS, ET RIEN N'EST FILTRÉ
 *
 * Toutes les lignes de l'axe sont rendues, DANS SON ORDRE, y compris celles où ni l'un ni
 * l'autre n'a de mesure de ce côté (`top` et `bottom` à `null`, l'infobulle dit « aucune
 * mesure »). C'est ce qui garde les quatre graphes de la page alignés ligne à ligne : filtrer
 * ici les lignes vides ferait réapparaître le décalage que `roleAxis` vient de supprimer.
 */
export function roleRangeLines(
  axis: readonly RoleAxisEntry[],
  sideA: Side,
  sideB: Side,
  side: RangeSideKey,
): WeaponRangeLine[] {
  const parA = new Map((sideA?.range?.weapons ?? []).map((w) => [w.weapon_key, w]))
  const parB = new Map((sideB?.range?.weapons ?? []).map((w) => [w.weapon_key, w]))
  return axis.map(({ weaponKey, label }) => ({
    weaponKey,
    label,
    top: sideOf(parA.get(weaponKey), side),
    bottom: sideOf(parB.get(weaponKey), side),
  }))
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
 * Les trois blocs sont indépendants côté contrat : il suffit qu'UN des joueurs porte une
 * classe, une ligne de portée ou une arme pour que la section ait un sens. Tout vide = pas de
 * section, jamais une section vide.
 */
export function hasWeaponProfile(...sides: Side[]): boolean {
  const porte = (s: Side) =>
    (s?.frag_classes?.length ?? 0) > 0 ||
    (s?.top_weapons?.length ?? 0) > 0 ||
    (s?.range?.weapons?.length ?? 0) > 0
  return sides.some(porte)
}
