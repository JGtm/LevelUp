/**
 * fda.ts — LE FDA D'UN MATCH, ET LE PALIER QUI LE COLORE.
 *
 * LA FORMULE EST UN NET DIVISÉ PAR UN NOMBRE DE MATCHS, PAS UN QUOTIENT PAR LES MORTS. Le
 * FDA canonique du dépôt (CLAUDE.md règle 9, ADR 0006, `internal/analysis/indicators.go`)
 * s'écrit
 *
 *     FDA = (frags + assistances / 3 − morts) / nb_matchs
 *
 * et ce module en sert le cas d'UN match, où le dénominateur vaut 1 :
 *
 *     FDA d'un match = (frags + assistances / 3 − morts) / 1
 *
 * ÉCRIRE LE « / 1 » N'EST PAS UNE COQUETTERIE : c'est ce qui rappelle que le per-match et
 * l'agrégat sont LA MÊME grandeur à des échelles différentes — `AggregateKDA` divise
 * exactement ce même net par N — et que le dénominateur du FDA n'a jamais été le nombre de
 * morts. C'est aussi la valeur que l'API Halo sert nativement
 * (`match_participants.kda`). Le tiers pondère une assistance : trois assistances valent
 * un frag.
 *
 * CE QU'ELLE N'EST PAS : `(frags + assistances/3) / max(1, morts)`. Ce quotient existe en
 * Go sous le nom `CombatEfficiency` — une métrique INTERNE au score de performance — et il
 * est toujours positif, donc incapable de dire ce que le net dit d'un coup d'œil : un
 * joueur qui meurt plus qu'il ne frags passe SOUS ZÉRO.
 *
 * LES TROIS PALIERS suivent cette lecture (décision utilisateur 2026-08-29) : négatif =
 * déficitaire, de 0 à 1 = à l'équilibre, au-delà de 1 = bénéficiaire. Ce sont des paliers
 * de LECTURE, pas des seuils de niveau — d'où des tokens d'état (destructive / info /
 * success) et non une échelle de performance.
 *
 * Module PUR : ni React, ni DOM, ni token résolu (l'appelant fait le rendu).
 */

/** Poids canonique d'une assistance dans le FDA : trois assistances valent un frag. */
export const FDA_ASSIST_WEIGHT = 1 / 3

/**
 * Le dénominateur du FDA : un NOMBRE DE MATCHS, pas un nombre de morts. Il vaut 1 ici — ce
 * module lit UN match — et il est nommé pour que la parenté avec `AggregateKDA` (le même
 * net divisé par N) reste lisible dans le calcul lui-même.
 */
const FDA_MATCHES = 1

/** Les trois compteurs dont le FDA se déduit. `null` / absent = compteur non lu. */
export interface FdaCounters {
  kills?: number | null
  deaths?: number | null
  assists?: number | null
}

/**
 * matchFda rend le FDA net d'un match, ou `null` si l'UN des trois compteurs manque.
 *
 * `null` PLUTÔT QU'UN ZÉRO DE REPLI : un compteur non lu n'est pas un compteur à zéro, et
 * un FDA calculé sur une lacune se lirait comme une mesure (même règle que les compteurs
 * de fiche du rejeu, cf. `lib/replay/scoreTimeline`).
 */
export function matchFda(c: FdaCounters | null | undefined): number | null {
  if (!c) return null
  const { kills, deaths, assists } = c
  if (kills == null || deaths == null || assists == null) return null
  if (!Number.isFinite(kills) || !Number.isFinite(deaths) || !Number.isFinite(assists)) return null
  return (kills + assists * FDA_ASSIST_WEIGHT - deaths) / FDA_MATCHES
}

/** Le palier de lecture d'un FDA, exprimé en token sémantique du dépôt. */
export type FdaTone = 'destructive' | 'info' | 'success'

/** Palier NÉGATIF : le joueur meurt plus qu'il ne produit de frags effectifs. */
export const FDA_BREAK_EVEN = 0
/** Palier au-dessus duquel le bénéfice est net (un frag effectif d'avance au moins). */
export const FDA_POSITIVE = 1

/**
 * fdaTone rend le palier d'un FDA : négatif (destructive), de 0 à 1 inclus (info),
 * strictement au-delà de 1 (success).
 *
 * LES BORNES SONT CELLES DE L'ÉNONCÉ, à la lettre : 0 et 1 appartiennent au palier
 * médian. Un FDA exactement à 1 est à l'équilibre, pas « bénéficiaire ».
 */
export function fdaTone(fda: number): FdaTone {
  if (fda < FDA_BREAK_EVEN) return 'destructive'
  if (fda > FDA_POSITIVE) return 'success'
  return 'info'
}
