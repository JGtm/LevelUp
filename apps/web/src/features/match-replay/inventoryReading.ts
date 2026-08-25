/**
 * inventoryReading.ts — QUELLE lecture d'inventaire la fiche affiche, et ce qu'elle en dit.
 *
 * POURQUOI CE FICHIER EXISTE (2026-08-25, lot 1 « lecture vide »). La règle « la lecture la
 * plus proche <= T » ne suffisait plus. L'artefact publie des lectures VIDES — 17,4 % du total
 * mesuré — et, la plus récente gagnant, une lecture vide EFFAÇAIT la fiche du joueur pendant
 * ~20 s alors qu'une lecture pleine la précédait. Le correctif n'est pas « ignorer les lectures
 * vides » : une lecture vide DIT quelque chose (le schéma 19 dit même quoi, cf. `empty`). C'est
 * « ne pas laisser une lecture vide effacer le dernier état connu, et afficher l'état vide À
 * CÔTÉ ». Les deux informations coexistent, aucune ne remplace l'autre.
 *
 * Extrait de `rosterLogic.ts` (seuil de taille du dépôt) : le report de lecture canonique
 * (`nearestReading`) y reste, ce fichier ne fait que le composer.
 *
 * Tout ce fichier est PUR : aucun React, aucun canvas, donc testable.
 */
import { catalogText, type CatalogLabel } from './catalogLabel'
import type { REPLAY_TEXT, ReplayLocale } from './i18n'
import { formatSeconds, frameToMs } from './replayLogic'
import type { ReplayDocumentReady, ReplayInventoryReady } from './replayNormalize'
import { nearestReading } from './rosterLogic'

/**
 * InventoryEmptyKind — POURQUOI la lecture qui couvre cette image ne rend rien.
 *
 * Les deux valeurs viennent telles quelles de l'artefact (`inventory[].empty`, schéma 19) :
 *   - 'dead'    : le FIL DES MORTS corrobore — le porteur du slot est mort dans les 8 s ;
 *   - 'unknown' : la lecture est vide et rien ne l'explique.
 * Une valeur inconnue d'un artefact futur est ramenée à 'unknown' : afficher « mort » sur une
 * étiquette qu'on ne comprend pas serait pire que de dire « indisponible ».
 */
export type InventoryEmptyKind = 'dead' | 'unknown'

/**
 * InventoryEmptyState — l'état vide qui couvre l'image courante, et l'âge de CETTE lecture.
 *
 * L'ÂGE EST CELUI DE LA LECTURE VIDE, pas celui de l'état affiché à côté : les deux ne datent
 * pas du même instant, et les confondre ferait passer un état de vingt secondes pour frais.
 */
export interface InventoryEmptyState {
  kind: InventoryEmptyKind
  age: number
}

/** InventoryReading — l'inventaire d'un slot et l'ÂGE de cette lecture, en frames. */
export interface InventoryReading {
  state: ReplayInventoryReady
  age: number
  /**
   * Présent = la lecture qui couvre l'image est VIDE. `state` porte alors la dernière lecture
   * PLEINE du même slot quand il en existe une ; à défaut, la lecture vide elle-même — et la
   * fiche n'a qu'un état à montrer, celui-ci.
   */
  empty?: InventoryEmptyState
  /**
   * true = `state` est une lecture PLEINE substituée à la lecture vide qui couvre l'image.
   *
   * CE BOOLÉEN EXISTE POUR QUE L'INFOBULLE NE MENTE PAS. Sans lui, la fiche disait « l'équipement
   * affiché est la dernière lecture pleine, lue il y a X » y compris quand AUCUNE lecture pleine
   * n'existait avant cet instant — `state` était alors la lecture vide elle-même, et le X était
   * l'âge de CETTE lecture vide. Deux situations, une seule phrase, dont une fausse.
   */
  substituted: boolean
}

/**
 * inventoryAt rend l'inventaire à afficher pour un SLOT à `frame`.
 *
 * LA RECHERCHE PORTE SUR LE SLOT, PAS SUR LE JOUEUR, et c'est ce qui rend le report SÛR : un
 * slot est réattribué à chaque réapparition, donc une dotation ne peut jamais franchir une
 * mort. Chercher par joueur ferait survivre un inventaire à son porteur.
 *
 * AVANT LA PREMIÈRE IMAGE-CLÉ D'UNE VIE, la lecture rendue est la plus proche À VENIR du
 * même slot — même repli que loadoutAt, et même honnêteté : âge NÉGATIF publié tel quel,
 * estompé sur sa valeur absolue, dit « à venir » en infobulle. DÉCISION UTILISATEUR
 * (2026-08-12) : elle va AU-DELÀ du POC, qui refusait la lecture future pour les compteurs
 * (grenades, munitions) au motif qu'ils sont volatils — l'arbitrage produit est qu'une
 * dotation de spawn affichée avec son âge « à venir » informe mieux que vingt secondes de
 * vide. Le repli ne franchit jamais une mort : un slot est une vie.
 *
 * QUAND CETTE LECTURE EST VIDE, on ne la jette pas et on ne s'y arrête pas : on rend l'état
 * vide dans `empty` ET la dernière lecture PLEINE du même slot dans `state`. C'est la seule
 * façon de dire les deux choses vraies à la fois — « il portait ceci » et « il est mort ».
 */
export function inventoryAt(
  doc: ReplayDocumentReady,
  slot: number,
  frame: number,
): InventoryReading | null {
  const read = nearestReading(doc.inventory ?? [], slot, frame)
  if (!read) return null
  if (!read.value.empty) return { state: read.value, age: read.age, substituted: false }
  // LECTURE VIDE À VENIR (âge négatif) : ne rien affirmer. La mort qu'elle rapporte n'est pas
  // encore survenue à l'image courante — le badge dirait « Mort » sur un joueur vivant, jusqu'à
  // ~11 s d'avance mesurées (8 vies sur 90 du film de référence, revue du 2026-08-25). Et le
  // report `lastFullBefore` chercherait avant `read.t`, donc parmi des lectures FUTURES elles
  // aussi : il servirait un équipement pas encore ramassé. On rend la lecture comme une lecture
  // ordinaire « à venir » — même repli que loadoutAt : la fiche n'en tire rien de plein, et
  // l'infobulle de la ligne dit déjà « dans X s ».
  if (read.age < 0) return { state: read.value, age: read.age, substituted: false }
  const empty: InventoryEmptyState = {
    kind: read.value.empty === 'dead' ? 'dead' : 'unknown',
    age: read.age,
  }
  // La dernière lecture PLEINE strictement antérieure à la lecture vide. On ne remonte PAS
  // au-delà de la vie : le slot est réattribué à chaque réapparition, donc toute lecture de
  // ce slot appartient à la même vie que celle qui vient de finir.
  const full = lastFullBefore(doc.inventory ?? [], slot, read.value.t)
  if (!full) return { state: read.value, age: read.age, empty, substituted: false }
  return { state: full, age: frame - full.t, empty, substituted: true }
}

/** lastFullBefore — la lecture PORTEUSE la plus récente d'un slot, strictement avant `t`. */
function lastFullBefore(
  samples: readonly ReplayInventoryReady[],
  slot: number,
  t: number,
): ReplayInventoryReady | null {
  let best: ReplayInventoryReady | null = null
  for (const s of samples) {
    if (s.slot !== slot || s.empty || s.t >= t) continue
    if (!best || s.t > best.t) best = s
  }
  return best
}

/**
 * inventoryEmptyHint — l'infobulle de l'état vide, en DEUX moitiés qui ne datent pas du même
 * instant. Elle vit ICI et non dans le composant : c'est une composition de texte pure, donc
 * testable sans rendu (même règle que le reste de ce fichier).
 *
 * PREMIÈRE MOITIÉ : pourquoi la lecture est vide, et l'âge de CETTE lecture (`empty.age`) — pas
 * celui de l'équipement affiché à côté. Un joueur mort il y a deux secondes dont l'équipement
 * date de vingt n'est pas la même chose qu'un joueur mort il y a vingt secondes, et un seul âge
 * ne pouvait pas dire les deux.
 *
 * SECONDE MOITIÉ : ce que l'écran montre à côté. Elle ne promet « la dernière lecture pleine »
 * que lorsqu'une lecture pleine est RÉELLEMENT substituée (`InventoryReading.substituted`). Sans
 * lecture pleine antérieure, la fiche n'affiche aucun équipement — l'infobulle le dit au lieu
 * d'annoncer un report qui n'a pas eu lieu, avec l'âge de la lecture vide en guise d'âge
 * d'équipement.
 */
export function inventoryEmptyHint(
  t: (typeof REPLAY_TEXT)[ReplayLocale],
  read: InventoryReading,
  empty: InventoryEmptyState,
  doc: ReplayDocumentReady,
): string {
  const why = empty.kind === 'dead' ? t.inventoryDeadHint : t.inventoryEmptyHint
  const emptyAge = formatSeconds(frameToMs(Math.abs(empty.age), doc))
  if (!read.substituted) return `${why} ${emptyAge} · ${t.inventoryNoPriorHint}`
  const gearAge = formatSeconds(frameToMs(Math.abs(read.age), doc))
  return `${why} ${emptyAge} · ${t.inventoryFallbackHint} ${gearAge}`
}

/**
 * grenadesCarried rend les types de grenade PORTÉS, avec leur nom et leur compteur.
 *
 * LES COMPTEURS À ZÉRO SONT ÉCARTÉS DE L'AFFICHAGE mais pas de la lecture : le tableau publié
 * est complet, et un zéro y signifie « ce type, aucune en réserve ». Montrer quatre types dont
 * trois à zéro noierait celui qui compte.
 */
export function grenadesCarried(
  state: ReplayInventoryReady,
  labels: CatalogLabel[] | undefined,
  locale: ReplayLocale,
): { rank: number; name: string; count: number }[] {
  if (!state.g) return []
  const out: { rank: number; name: string; count: number }[] = []
  state.g.forEach((count, rank) => {
    if (count <= 0) return
    // Sans table, le RANG s'affiche tel quel : c'est ce que le document dit, et c'est
    // vrai. Inventer un nom serait pire (cf. catalogLabel.ts).
    out.push({ rank, name: catalogText(labels?.[rank], locale) ?? `rang ${rank}`, count })
  })
  return out
}

/**
 * GrenadeSelection — le type ÉQUIPÉ, celui qui partira au prochain lancer, avec sa
 * PROVENANCE. Les trois formes ne se confondent jamais :
 *   - { rank, read: true }  : LU dans le film (sélecteur i47 de l'image-clé) ;
 *   - { rank, read: false } : DÉDUIT — un seul type porté, donc c'est lui ;
 *   - 'indeterminate'       : plusieurs types portés et sélecteur non lu — l'écran doit le
 *     DIRE (« sél. ? »), pas choisir ;
 *   - null                  : rien à désigner (compteurs non lus, ou aucun type porté).
 */
export type GrenadeSelection = { rank: number; read: boolean } | 'indeterminate' | null

/**
 * selectedGrenade désigne le type équipé.
 *
 * LA LECTURE PRIME LA DÉDUCTION : le sélecteur du film (`gs`) est publié sous garde de
 * cohérence (masque == compteurs, unanimité) — quand il est là, c'est lui. À défaut, la
 * déduction ne vaut que quand elle ne peut pas être autre chose : un seul type porté.
 * Dès qu'un joueur porte deux types sans sélecteur lu, on rend 'indeterminate' : deviner
 * reviendrait à afficher une certitude qu'on n'a pas.
 */
export function selectedGrenade(state: ReplayInventoryReady): GrenadeSelection {
  const carried = (state.g ?? []).map((c, r) => ({ c, r })).filter((x) => x.c > 0)
  if (carried.length === 0) return null
  const gs = state.gs
  if (gs !== undefined && carried.some((x) => x.r === gs)) {
    return { rank: gs, read: true }
  }
  if (carried.length === 1) return { rank: carried[0].r, read: false }
  return 'indeterminate'
}
