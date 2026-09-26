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
import { EXPERIENCE_TO_CASCADE } from '@/features/_shared/experienceCascade'
import type {
  FilterContextInput,
  SessionLabelEntry,
  SessionOption,
  TacticalMapCard,
  TeammateOption,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { stripSessionCountSuffix } from '@/lib/sessions/sessionLabels'

import type { TacticalScope } from './tacticalScope'

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
 * contexteFiltre — le `FilterContextInput` que la barre L2 produit, et que le
 * client fait résoudre en `match_ids` par `/filters/match-ids`.
 *
 * C'EST LA MÊME RÉSOLUTION QUE L'OMNIBAR ET QUE L'EXPLORATEUR, et c'est tout
 * l'intérêt : le périmètre de l'onglet est calculé par le pipeline qui sait lire les
 * SESSIONS (base joueur), là où les requêtes shared du lecteur tactique ne le
 * peuvent pas. C'est ce qui fait MARCHER le filtre de session ici.
 *
 * `filter_mode` suit la SÉLECTION, pas un réglage : dès qu'une session est épinglée,
 * le mode est `sessions` et la période cesse de décider — exactement la règle du
 * backend (`splitTemporalFiltered` applique le filtre de session dès qu'il y en a
 * une, quel que soit `filter_mode`). Les deux doivent dire la même chose, sinon
 * l'écran annonce une période que le serveur n'applique pas.
 */
export function contexteFiltre(scope: TacticalScope): FilterContextInput {
  const sessions = scope.sessions.filter((s) => s.trim() !== '')
  const ctx: FilterContextInput = {
    filter_mode: sessions.length > 0 ? 'sessions' : 'period',
    period: {
      start_date: scope.debut || null,
      end_date: scope.fin || null,
    },
    sessions: { picked_sessions: sessions, gap_minutes: 120 },
    cascade: {
      experience_types: EXPERIENCE_TO_CASCADE[scope.experience],
      playlists: scope.playlists,
      modes: scope.modes,
      maps: [],
    },
  }
  if (scope.vue !== 'all') ctx.match_context = scope.vue
  return ctx
}

/**
 * sessionsProposees — les sessions offertes au sélecteur, DANS LE SENS DE LA
 * COMPOSITION (même mécanique que la barre de l'Escouade) : dès qu'un coéquipier
 * est choisi, on propose les sessions d'ESCOUADE ; sinon les sessions SOLO.
 *
 * ÉCART ASSUMÉ AVEC LA PAGE ESCOUADE, et il est dans le sens de la prudence : elle
 * propose les sessions de la COMPOSITION EXACTE, que seule sa requête de page sait
 * calculer (`composition_sessions`). Ici la liste vient de `/filters/resolve`, donc
 * des sessions d'escouade du joueur — et c'est le SERVEUR qui resserre ensuite sur
 * la composition (liste blanche × `coequipiers`). Une session proposée peut donc ne
 * porter aucun match de la composition ; l'inverse — masquer une session que la
 * composition a jouée — serait la faute grave, et il ne peut pas se produire.
 *
 * La forme change aussi : `/filters/resolve` rend des `SessionOption`
 * (`started_at_utc`), `SessionMultiSelect` attend des `SessionLabelEntry`
 * (`started_at`). La projection est ici, pas dans le composant partagé.
 */
export function sessionsProposees(
  options: readonly SessionOption[],
  avecComposition: boolean,
): SessionLabelEntry[] {
  return options
    .filter((o) => o.is_squad === avecComposition)
    .map((o) => ({
      label: o.label,
      started_at: o.started_at_utc,
      ended_at: o.ended_at_utc,
      match_count: o.match_count,
    }))
}

/**
 * sessionsHorsListe — les sessions ÉPINGLÉES que la liste courante ne propose pas.
 *
 * Le cas qui compte n'est pas le zombie de synchronisation (un compte de matchs qui a
 * bougé, remappé par `reconcileSquadSessionLabels`) : c'est le CHANGEMENT DE CONTEXTE.
 * Une session SOLO épinglée, puis un coéquipier ajouté, et la liste proposée devient
 * celle des sessions d'escouade — le label épinglé n'y figure plus, sans avoir rien
 * perdu de sa validité. Le dire est la seule sortie honnête : le retirer élargirait le
 * périmètre d'une soirée à l'historique entier, en silence.
 *
 * L'identité d'un label est sa forme SANS le compte (`stripSessionCountSuffix`), la même
 * que celle de la réconciliation — deux notions d'identité donneraient deux verdicts.
 */
export function sessionsHorsListe(
  epinglees: readonly string[],
  proposees: readonly SessionLabelEntry[],
): string[] {
  if (epinglees.length === 0 || proposees.length === 0) return []
  const cles = new Set(proposees.map((s) => stripSessionCountSuffix(s.label)))
  return epinglees.filter((l) => !cles.has(stripSessionCountSuffix(l)))
}

/** Résultat de la traduction gamertags → xuids de la composition. */
export interface CompositionResolue {
  /** Les xuids à envoyer, dans l'ordre de la composition. */
  xuids: string[]
  /** Les gamertags qu'on n'a pas su traduire — la requête ne doit PAS partir. */
  inconnus: string[]
}

/**
 * resoudreComposition — traduit les gamertags de la composition en xuids.
 *
 * POURQUOI UN NOM NON RÉSOLU ARRÊTE TOUT plutôt que d'être ignoré : l'ignorer
 * ÉLARGIT le périmètre (le serveur ne resserrerait plus sur ce joueur) et rendrait
 * une grille plus fournie que demandé, sans que rien ne le dise. Un filtre qu'on ne
 * sait pas appliquer se signale, il ne se retire pas tout seul.
 *
 * Le cas normal est transitoire : la liste des coéquipiers arrive de façon
 * asynchrone. Le cas durable est une URL bricolée — la page le dit alors en clair.
 */
export function resoudreComposition(
  gamertags: readonly string[],
  options: readonly TeammateOption[],
): CompositionResolue {
  const parGamertag = new Map<string, string>()
  for (const o of options) {
    if (o.xuid) parGamertag.set(o.gamertag.toLowerCase(), o.xuid)
  }
  const xuids: string[] = []
  const inconnus: string[] = []
  for (const gt of gamertags) {
    const xuid = parGamertag.get(gt.trim().toLowerCase())
    if (xuid) xuids.push(xuid)
    else inconnus.push(gt)
  }
  return { xuids, inconnus }
}
