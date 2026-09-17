/**
 * compareWeapons_logic — LES DÉCISIONS DE LECTURE DU PROFIL D'ARMES, hors JSX.
 *
 * Tout ce que la section doit DÉCIDER de PROPRE À ELLE (quelles classes aligner, a-t-elle
 * quelque chose à montrer) vit ici, pur et testable. Le composant ne fait que poser des nœuds.
 *
 * # CE QUI DIFFÈRE DE L'ONGLET RÉSUMÉ, ET POURQUOI CE FICHIER EXISTE
 *
 * Le Résumé décrit UN joueur : ses lignes viennent d'un seul bloc, l'ordre du backend fait
 * foi. Le Face-à-face en superpose DEUX — et TROIS en mode miroir — dont les blocs n'ont ni les
 * mêmes classes ni les mêmes rôles : l'un a fragué au fusil de précision, l'autre jamais. Il
 * faut donc UNIR les listes sans en privilégier une, et publier une ligne vide là où un côté
 * n'a rien : une ligne absente d'une colonne se lirait « pas de donnée » alors que la vérité
 * est « il ne l'a jamais fait ».
 *
 * # L'AXE DES RÔLES N'EST PLUS ICI (2026-09-17)
 *
 * `roleAxis`, `roleRangeLines` et `roleLabel` sont partis dans
 * `@/components/charts/weaponRangeRoles` quand le bloc « Portée des frags » de l'Explorer a
 * demandé les mêmes décisions : même axe, même ordre, même nommage. Les recopier aurait donné
 * deux ordres de lignes pour la même mesure.
 */
import type { CompareFragClass, CompareWeaponSide } from '@/lib/api/types'

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
