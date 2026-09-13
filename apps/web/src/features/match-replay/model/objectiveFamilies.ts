/**
 * objectiveFamilies.ts — LA FAMILLE D'OBJECTIF D'UNE STATISTIQUE NOMMÉE. Logique pure, pas de
 * React, aucune chaîne d'interface.
 *
 * CE QUE `doc.objectives` PORTE, ET QUI N'EST PAS UN OBJECTIF. Le contrat de transport
 * (`ObjectiveAction`) n'a qu'un champ `stat` — aucun `kind`, aucune `family` : c'est le NOM de
 * la statistique qui dit la famille, et il est posé côté Go par
 * `internal/analysis/objectiveevents/named.go`. Or les tables de ce fichier ne contiennent pas
 * que des objectifs : `kills` (`comp 2 A`) y est l'ANCRE D'IDENTITÉ du balayage et `assists`
 * (`comp 3 A`) son voisin de contrôle croisé. Les deux sont publiés comme les autres — et sur
 * `8bc6074f` ils font 99 % du calque (15 648 actions sur 15 808, audit du 2026-09-10 §12-1).
 *
 * LA RÈGLE DE NOMMAGE EST LE CONTRAT, et elle se vérifie : toute statistique d'objectif est
 * préfixée par sa famille (`flag_captures`, `zone_secures`, `vip_selected`, `bomb_detonations`,
 * `skull_grabs`...), et les familles sont exactement les `ObjectiveType*` de
 * `objectiveevents/extract.go`. Les deux statistiques hors objectif, elles, n'ont AUCUN préfixe.
 * La liste ci-dessous n'est donc pas une énumération devinée des stats connues — qui périmerait
 * au premier emplacement nommé — mais la liste des FAMILLES, stable et fermée côté serveur
 * (garde-rail Go : `objectiveevents.TestStatsNommeesPortentLeurFamille`, qui tient la règle de
 * nommage sur toutes les tables du décodeur).
 *
 * LA PARITÉ AVEC LA LISTE GO EST TENUE PAR UN TEST, depuis le 2026-09-13 :
 * `objectiveevents.TestFamillesObjectifPariteGoTS` LIT ce fichier et compare le tableau
 * ci-dessous ET le type union `ObjectiveFamily` à la liste Go, ordre compris. Jusque-là ce
 * commentaire affirmait la garantie sans qu'aucun test ne l'assure : une 7e famille ajoutée au
 * décodeur rendait le Go rouge et laissait ce fichier VERT, et le calque cessait sans un mot de
 * dessiner les pulses de cette famille (constat R6 de la revue du 2026-09-13). Toute famille
 * ajoutée ici s'ajoute dans `objectiveevents/families.go` — et réciproquement.
 *
 * POURQUOI UNE LISTE BLANCHE ET NON UNE LISTE NOIRE `kills`/`assists` : une statistique future
 * qui ne serait pas un objectif (le balayage en ajoute au fil des corpus) passerait une liste
 * noire sans que rien ne le signale. Une famille d'objectif nouvelle, elle, est un ajout
 * DÉLIBÉRÉ au décodeur, et l'ajouter ici est la même ligne que côté Go.
 */

/** Les familles d'objectif, telles que `objectiveevents/extract.go` les nomme. */
export type ObjectiveFamily = 'flag' | 'zone' | 'hill' | 'skull' | 'vip' | 'bomb'

/**
 * L'ordre est celui des constantes Go (`ObjectiveTypeFlag`, `Zone`, `Hill`, `Skull`, `Vip`,
 * `Bomb`) : deux listes qui se répondent se lisent côte à côte.
 */
export const OBJECTIVE_FAMILIES: readonly ObjectiveFamily[] = [
  'flag',
  'zone',
  'hill',
  'skull',
  'vip',
  'bomb',
]

/**
 * objectiveFamilyOf rend la famille d'objectif d'une statistique nommée, ou `null` quand la
 * statistique n'en est pas une (`kills`, `assists`, et tout nom sans préfixe de famille).
 *
 * Le séparateur est exigé : `flag_grabs` est de la famille `flag`, `flagrant` ne l'est pas.
 */
export function objectiveFamilyOf(stat: string): ObjectiveFamily | null {
  for (const family of OBJECTIVE_FAMILIES) {
    if (stat.startsWith(`${family}_`)) return family
  }
  return null
}

/** isObjectiveFamilyStat — la même question en booléen, pour les filtres et les dénominateurs. */
export function isObjectiveFamilyStat(stat: string): boolean {
  return objectiveFamilyOf(stat) !== null
}
