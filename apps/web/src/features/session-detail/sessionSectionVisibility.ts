/**
 * sessionSectionVisibility — CE QUE LA SECTION « Frags et usages » VA RENDRE, décidé UNE
 * fois, pour elle ET pour les cartes qu'elle coiffe.
 *
 * POURQUOI CE FICHIER EXISTE. Depuis le 2026-09-22 la page Sessions coiffe ses blocs de
 * quatre titres de section, et un titre ne doit JAMAIS se poser au-dessus de rien. La
 * section « Frags et usages » réunit la carte des frags et les cartes d'usage, qui se
 * masquent toutes seules selon la donnée : il faut donc savoir AVANT de poser le titre si
 * au moins une carte va se dessiner.
 *
 * ET CE N'EST PAS UNE COPIE DES GARDES DES CARTES : ce sont les cartes qui LISENT ce
 * prédicat (`EquipmentCards`, `PadControlCard`, `ObjectivesCard` appellent
 * `sessionUsageCardsShown`). Écrire la condition deux fois — une pour le titre, une pour la
 * carte — aurait fabriqué exactement la divergence que la règle n°6 interdit : un titre
 * au-dessus du vide le jour où l'une des deux bouge.
 *
 * Pur : aucun React, aucun dictionnaire (la règle des deux portes se lit par
 * `usageAvailabilityKind`, qui n'a pas besoin de langue).
 */
import { buildFragDetailBreakdown, type FragDetailLabels } from '@/components/charts/fragDetailBreakdown'
import type { SessionCompareEntry, SessionUsageBlock } from '@/lib/api/types'

import { usageAvailabilityKind } from '@/features/_shared/usage/usageAvailability'
import { equipmentMetrics, padMetric } from '@/features/_shared/usage/usageMetricKinds'
import { padTierLines } from '@/features/_shared/usage/usagePadTiersModel'

/** Quelles cartes du bloc vont se dessiner, une par carte de `SessionUsageSection`. */
export interface SessionUsageCardsShown {
  /** Les trois cartes d'équipement (cadences, parts, régularité) — elles vont ensemble. */
  equipment: boolean
  /** « Contrôle des armes spéciales » — familles, total, bonus OU niveaux d'arme. */
  padControl: boolean
  /** « Objectifs par rôle et par famille » — scope indépendant des films. */
  objectives: boolean
}

/**
 * sessionUsageCardsShown — la porte de CHAQUE carte, au même endroit.
 *
 * `padControl` compte les NIVEAUX (2026-09-14) : ils viennent d'une autre passe, sur
 * d'autres matchs, et une session dont seuls les niveaux sont mesurés doit rendre sa carte.
 * Les bonus (`powerup_pickups`) n'y comptent PAS : ils ne sont qu'un pied de carte, jamais
 * de quoi en ouvrir une.
 *
 * `objectives` s'ouvre dès qu'un match de la sélection porte un objectif, MÊME SANS RÔLE
 * MESURÉ (D8) : la carte dit alors pourquoi elle est vide — une sélection sans mode à
 * objectif est une réponse, pas une absence de mesure.
 */
export function sessionUsageCardsShown(usage: SessionUsageBlock): SessionUsageCardsShown {
  const objectives = usage.objectives
  return {
    equipment: equipmentMetrics(usage.metrics).length > 0,
    padControl:
      padMetric(usage.metrics) != null ||
      (usage.pad_families ?? []).length > 0 ||
      padTierLines(usage.pad_tiers).length > 0,
    objectives: objectives != null && objectives.matches_with_objectives > 0,
  }
}

/**
 * sessionUsageShowsSomething — le bloc va-t-il dessiner QUELQUE CHOSE de visible ?
 *
 * L'ÉTAT « aucun film » COMPTE POUR UN BLOC VISIBLE : c'est une carte qui porte une phrase
 * et un dénominateur, pas une absence. Masquer le titre au-dessus d'elle la laisserait
 * orpheline entre deux sections titrées.
 */
export function sessionUsageShowsSomething(usage: SessionUsageBlock | null | undefined): boolean {
  const kind = usageAvailabilityKind(usage)
  if (kind === 'hidden' || usage == null) return false
  if (kind === 'empty') return true
  const shown = sessionUsageCardsShown(usage)
  return shown.equipment || shown.padControl || shown.objectives
}

/**
 * Libellés NEUTRES pour COMPTER, jamais pour afficher : `buildFragDetailBreakdown` produit
 * exactement le même NOMBRE de lignes quels que soient les résolveurs (ils ne font que
 * nommer). Le prédicat reste donc pur — ni store de langue, ni hook.
 */
const COUNT_ONLY_LABELS: FragDetailLabels = {
  roleLabel: (role) => role,
  classLabel: (className) => className,
  locale: 'fr',
}

/**
 * sessionFragCardHasContent — la carte des frags a-t-elle quelque chose à dessiner ?
 *
 * C'est LA porte de `SessionFragCard` (le composant l'appelle), pas une copie : le titre de
 * section et la carte lisent la même condition.
 */
export function sessionFragCardHasContent(entry: SessionCompareEntry | null | undefined): boolean {
  const distribution = entry?.frag_distribution ?? null
  if ((distribution?.total_kills ?? 0) > 0) return true
  return (
    buildFragDetailBreakdown(distribution, entry?.top_weapon_kills ?? [], COUNT_ONLY_LABELS)
      .length > 0
  )
}
