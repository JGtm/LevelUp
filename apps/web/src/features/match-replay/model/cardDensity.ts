/**
 * cardDensity.ts — QUEL GABARIT POUR CE MATCH : la densité se lit sur le TYPE DE MATCH.
 *
 * DÉCISION D1 (utilisateur, 2026-09-06 : « les matchs de type 4v4 on touche pas ») : la tuile
 * compacte est celle des Grandes équipes, c'est-à-dire de la catégorie de mode `BTB` que
 * l'en-tête de la vue match porte déjà (`MatchViewHeader.mode_category`, posé par
 * `applyMatchHeaderModeCategory` depuis la taxonomie du titre — `BTB` et `BTB Heavies`). La
 * même table `Record<string, …>` que `session-detail/SessionParamPills.tsx` : une convention du
 * dépôt, pas une troisième.
 *
 * CE QUE CETTE FONCTION NE REGARDE PAS, et c'est le point : ni le nombre de sièges, ni le
 * nombre de joueurs, ni les lignes du tableau. Un 4v4 reste un 4v4 quel que soit le nombre de
 * relais ; un 12v12 dont la catégorie manque reste en gabarit normal. Elle ne reçoit donc que
 * l'en-tête — un effectif n'a aucun moyen d'entrer ici.
 *
 * TITLE-AGNOSTIC PAR CONSTRUCTION : un titre sans taxonomie de modes (Halo 5) laisse la
 * catégorie vide → gabarit normal, sans branche sur le slug. Vue match indisponible → normal.
 */
import type { PresenceHeader } from './presenceFeed'
import { GABARIT_COMPACT, GABARIT_NORMAL, type CardGabarit } from './cardGabarit'

/** Catégorie de mode (backend) → gabarit. Toute catégorie absente de la table = normal. */
const GABARIT_PAR_CATEGORIE: Record<string, CardGabarit> = {
  BTB: GABARIT_COMPACT,
}

export function cardDensity(header: PresenceHeader | null | undefined): CardGabarit {
  const categorie = header?.mode_category
  // `Object.hasOwn`, pas un accès nu : une catégorie qui serait un nom de la chaîne de
  // prototypes (« constructor ») rendrait une fonction à la place d'un gabarit.
  if (categorie && Object.hasOwn(GABARIT_PAR_CATEGORIE, categorie)) {
    return GABARIT_PAR_CATEGORIE[categorie]
  }
  return GABARIT_NORMAL
}
