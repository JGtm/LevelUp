/**
 * chart-review — manifeste de la tournée de revue visuelle.
 *
 * Une entrée = un graphe identifié par une clé stable + le type de verdict
 * attendu de l'utilisateur + une note courte qui pose la question à trancher.
 *
 * Cycle de vie d'une entrée :
 *   1. le chantier qui touche/ajoute/suspecte un graphe l'inscrit ici ;
 *   2. l'utilisateur balaye les pages et statue à l'écran ;
 *   3. l'entrée est RETIRÉE (commit de clôture de la tournée).
 *
 * MÉCANISME INERTE : `CHART_REVIEW` vide (ou clé absente) → `chartReview()`
 * renvoie `undefined` → `ReviewBadge` rend `null` → AUCUN impact visuel, aucun
 * nœud DOM ajouté. L'outillage peut donc rester en place entre deux tournées
 * sans polluer l'interface (DEC-8).
 *
 * Les clés sont des identifiants de GRAPHE (surface + graphe), pas des libellés :
 * elles ne sont jamais affichées.
 */
import type { Locale } from '@/lib/i18n/locale'

/**
 * Verdict attendu de l'utilisateur :
 *   - `verify`  : graphe corrigé/suspect → « est-ce juste et lisible ? »
 *   - `new`     : graphe (ou marqueur) ajouté → « à garder ? »
 *   - `removal` : candidat à la suppression → « on le retire ? »
 */
export type ChartReviewStatus = 'verify' | 'new' | 'removal'

export interface ChartReview {
  status: ChartReviewStatus
  /** Note courte affichée au survol. FR + EN obligatoires (parité par typage). */
  note: Record<Locale, string>
}

/**
 * Entrées de la tournée en cours. Vider ce dictionnaire suffit à désarmer
 * complètement l'outillage.
 *
 * AUCUNE TOURNÉE EN COURS. La tournée « revue analytique Timeseries & Escouade »
 * (ouverte le 2026-07-25) a été CLOSE LE 2026-09-14 sur décision de l'utilisateur :
 * ses douze entrées ont été retirées (étape 3 du cycle de vie ci-dessus), donc plus
 * aucune pastille ne s'affiche. Le mécanisme reste en place, inerte par construction
 * (DEC-8) : la prochaine tournée réinscrit ses graphes ici.
 */
export const CHART_REVIEW: Record<string, ChartReview> = {}

/**
 * chartReview — entrée de revue d'un graphe, ou `undefined` si le graphe n'est
 * pas dans la tournée en cours (cas nominal hors revue).
 */
export function chartReview(key: string | undefined): ChartReview | undefined {
  if (!key) return undefined
  return CHART_REVIEW[key]
}
