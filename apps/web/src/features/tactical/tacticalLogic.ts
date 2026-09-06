/**
 * tacticalLogic — la logique PURE de la grille des cartes de l'onglet Tactique.
 *
 * Aucun composant, aucun hook, aucune I/O : le tri, l'état sous plancher, la barre
 * victoires / défaites et la ligne de couverture se testent seuls. Les composants ne font
 * que rendre ce que ces fonctions décident (règle du dépôt : pas de logique métier dans un
 * composant React).
 *
 * LE PLANCHER N'EST PAS RECALCULÉ ICI. Le serveur publie `sous_plancher` par carte et
 * `plancher_matchs` pour la page : refaire la comparaison côté client donnerait deux
 * vérités sur « cette carte est-elle lisible ? », qui divergeraient au premier ajustement
 * du seuil. Le client LIT le verdict et NOMME le seuil, il n'en juge pas.
 */
import type { FilterContextInput, TacticalMapCard } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import { filterSpecToQueryString, type MatchFilterSpec } from '@/lib/match-nav/navContext'

/** Parts de la barre de résultats, en fraction de 0 à 1. Leur somme vaut au plus 1. */
export interface BarreResultats {
  victoires: number
  defaites: number
  /** Le reste : nuls, abandons, résultat inconnu. Peint en neutre, jamais en V ni en D. */
  autres: number
}

/**
 * trierCartes — les cartes JOUÉES, de la plus jouée à la moins jouée (décision produit
 * §1 du plan). À nombre de matchs égal, l'ordre est celui du NOM, pour que deux affichages
 * du même jeu de données ne se présentent jamais dans deux ordres différents.
 *
 * Ne filtre RIEN : une carte sous le plancher reste affichée (le joueur doit voir qu'il y
 * a joué), simplement désaturée et non ouvrable.
 */
export function trierCartes(cartes: readonly TacticalMapCard[]): TacticalMapCard[] {
  return [...cartes].sort((a, b) => {
    if (b.matchs !== a.matchs) return b.matchs - a.matchs
    return nomCarteBrut(a).localeCompare(nomCarteBrut(b))
  })
}

/** estOuvrable — une carte s'ouvre si le SERVEUR ne l'a pas classée sous le plancher. */
export function estOuvrable(carte: TacticalMapCard): boolean {
  return !carte.sous_plancher
}

/**
 * barreResultats — les trois parts de la barre victoires / défaites / reste.
 *
 * DEUX PROTECTIONS, chacune contre un affichage qui mentirait :
 *   - `matchs <= 0` rend une barre VIDE plutôt qu'une division par zéro ;
 *   - `victoires + défaites > matchs` (contrat incohérent) normalise sur leur SOMME au
 *     lieu de laisser la barre déborder de son cadre : mieux vaut une proportion juste
 *     entre les deux camps qu'une barre qui sort de la vignette.
 * Le cas nominal est l'inverse : la somme est INFÉRIEURE aux matchs (nuls, abandons,
 * résultat inconnu), et c'est `autres` qui porte l'écart.
 */
export function barreResultats(carte: TacticalMapCard): BarreResultats {
  const victoires = Math.max(0, carte.victoires)
  const defaites = Math.max(0, carte.defaites)
  const matchs = Math.max(0, carte.matchs)
  if (matchs === 0) return { victoires: 0, defaites: 0, autres: 0 }
  const denominateur = Math.max(matchs, victoires + defaites)
  const partV = victoires / denominateur
  const partD = defaites / denominateur
  return { victoires: partV, defaites: partD, autres: Math.max(0, 1 - partV - partD) }
}

/** Couverture de la grille : combien de cartes, et combien de matchs au total. */
export function couvertureGrille(cartes: readonly TacticalMapCard[]): {
  cartes: number
  matchs: number
} {
  return {
    cartes: cartes.length,
    matchs: cartes.reduce((somme, c) => somme + Math.max(0, c.matchs), 0),
  }
}

/**
 * nomCarte — le nom AFFICHÉ d'une carte. Le français vient du contrat (`map_name_fr`,
 * résolu côté Go depuis `asset_translations`) ; à défaut, et en anglais, le nom canonique.
 */
export function nomCarte(carte: TacticalMapCard, locale: Locale): string {
  if (locale === 'fr' && carte.map_name_fr.trim() !== '') return carte.map_name_fr
  return nomCarteBrut(carte)
}

function nomCarteBrut(carte: TacticalMapCard): string {
  return carte.map_name.trim() !== '' ? carte.map_name : carte.map_id
}

/**
 * tacticalFilterQuery — le filtre GLOBAL de l'omnibar, traduit dans le vocabulaire de
 * requête de l'Explorateur (playlist, mode, from, to, outcome, with_player).
 *
 * `session` EST RETIRÉ, et ce n'est pas un oubli : `analysis.BuildNeighborsWhereClause` le
 * range dans ses filtres IGNORÉS — les sessions vivent dans la base JOUEUR, que les
 * requêtes shared de cet onglet ne joignent pas. Le contrat de l'onglet ne l'accepte donc
 * pas (retrait T11, 2026-09-06). L'envoyer quand même ferait croire, côté clé de cache
 * comme côté serveur, à un filtre appliqué qui ne l'est pas.
 *
 * La chaîne rendue sert AUSSI d'empreinte de cache : deux filtres différents produisent
 * deux chaînes différentes, et c'est exactement ce que la clé de requête doit distinguer.
 */
export function tacticalFilterQuery(ctx: FilterContextInput | null | undefined): string {
  const spec = filterContextToMatchFilterSpec(ctx)
  if (!spec) return ''
  const sansSession: MatchFilterSpec = { ...spec }
  delete sansSession.session_id
  return filterSpecToQueryString(sansSession)
}
