/**
 * usageMetricKinds.ts — LA CLASSIFICATION des grandeurs du bloc « usages d'équipement, armes
 * spéciales et objectifs » : ordre canonique, nature d'une clé du contrat, libellés, encres.
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/` (étape E5.1).
 *
 * Pur : aucun React, aucune lecture de store — les libellés viennent de `usageI18n`.
 */
import type { SemanticToken } from '@/lib/accessibility'
import type { SessionUsageMetric } from '@/lib/api/types'

import { deployedFamilyLabel, equipmentFamilyLabel, type UsageText } from './usageI18n'

// ─── Grandeurs : ordre canonique, nature, libellés, encres ───────────────────────

/** La nature d'une grandeur du contrat — décide libellé, encre et bloc d'accueil. */
export type UsageMetricKind =
  | 'camo'
  | 'overshield'
  | 'wall'
  | 'deployed_other'
  | 'grapple'
  | 'dropped'
  | 'pads'
  | 'equipment'
  | 'other'

const DEPLOYED_PREFIX = 'deployed_'
/**
 * EQUIPMENT_PREFIX — LE BILAN D'ÉQUIPEMENT par famille (étape E4, jumeau web de
 * `sessionusage.MetricEquipmentPrefix`). Sa valeur est un compte d'OBJETS
 * (utilisé + gardé + lâché) et non de gestes ; c'est la seule famille de clés qui
 * porte `SessionUsageMetric.outcomes` — voir `equipmentMetrics` pour la règle de
 * substitution face à `deployed_<famille>` / `camo_episodes` / `overshield_episodes`.
 */
const EQUIPMENT_PREFIX = 'equipment_'

/** metricKind classe une clé du contrat (ensembles ouverts côté `deployed_*` et `equipment_*`). */
export function metricKind(key: string): UsageMetricKind {
  switch (key) {
    case 'camo_episodes':
      return 'camo'
    case 'overshield_episodes':
      return 'overshield'
    case 'deployed_wall':
      return 'wall'
    case 'grapple_pulls':
      return 'grapple'
    case 'dropped_objects':
      return 'dropped'
    case 'pad_pickups':
      return 'pads'
    default:
      if (key.startsWith(EQUIPMENT_PREFIX)) return 'equipment'
      return key.startsWith(DEPLOYED_PREFIX) ? 'deployed_other' : 'other'
  }
}

/** L'ordre d'affichage du bloc équipement (les inconnues ferment la marche). */
const METRIC_RANK: Partial<Record<UsageMetricKind, number>> = {
  camo: 0,
  overshield: 1,
  equipment: 2,
  wall: 3,
  deployed_other: 4,
  grapple: 5,
  other: 7,
}

/** metricLabel — le libellé bilingue d'une grandeur (vocabulaire du handoff). */
export function metricLabel(key: string, t: UsageText): string {
  switch (metricKind(key)) {
    case 'camo':
      return t.metricCamo
    case 'overshield':
      return t.metricOvershield
    case 'wall':
      return t.metricWall
    case 'grapple':
      return t.metricGrapple
    case 'dropped':
      return t.metricDropped
    case 'pads':
      return t.metricPads
    case 'equipment':
      return equipmentFamilyLabel(key.slice(EQUIPMENT_PREFIX.length), t)
    case 'deployed_other':
      return deployedFamilyLabel(key.slice(DEPLOYED_PREFIX.length), t)
    case 'other':
      return key
  }
}

/**
 * equipmentBilanFamilyOf — l'identité de famille (vocabulaire du bilan) que porte
 * une grandeur SUSCEPTIBLE D'ÊTRE SUPPLANTÉE par son équivalent `equipment_<famille>`
 * (voir `equipmentMetrics`). `null` pour toute autre clé (grapple, pads, inconnue…).
 */
function equipmentBilanFamilyOf(key: string): string | null {
  switch (key) {
    case 'camo_episodes':
      return 'powerup_camo'
    case 'overshield_episodes':
      return 'powerup_overshield'
    default:
      return key.startsWith(DEPLOYED_PREFIX) ? key.slice(DEPLOYED_PREFIX.length) : null
  }
}

/**
 * L'ENCRE DE CHAQUE FAMILLE DE GESTE — 2e copie ASSUMÉE de `USAGE_GROUP_TOKENS`
 * (match-replay/equipmentUsageChart.ts, import croisé interdit par le ratchet
 * lint-cross-feature-imports) : mêmes jetons pour que la session se lise avec la
 * même convention de couleur que la vue match. À la 3e copie : centraliser dans
 * `components/` et poser le garde-rail (règle CLAUDE.md n°6).
 */
export const USAGE_METRIC_TOKENS: Record<UsageMetricKind, SemanticToken> = {
  grapple: 'frag-sidearm', // vert — comme la famille grappin de la vue match
  camo: 'frag-heavy', // violet — états actifs
  overshield: 'frag-heavy', // violet — états actifs (même famille que camo)
  wall: 'frag-shoulder', // cyan — poses
  deployed_other: 'frag-shoulder', // cyan — poses
  // Même jeton que le groupe fusionné `equipment` de la vue match
  // (match-replay/equipmentUsageChart.ts, USAGE_GROUP_TOKENS, étape E2) : le bilan
  // d'équipement se lit avec la même convention de couleur sur les deux écrans.
  equipment: 'frag-shoulder',
  dropped: 'frag-melee', // rose — lâchés
  pads: 'frag-grenade', // ambre — libre ici (les grenades sont hors contrat)
  other: 'frag-unattributed', // gris — grandeur non cataloguée
}

/**
 * L'encre des trois rôles d'objectif (colonnes des grilles du bloc 3). Gamme
 * `chart-series-*` : la couleur IDENTIFIE un rôle, elle ne juge rien — une gamme
 * ordinale (perf-tier) ou de statut mentirait.
 */
export const ROLE_TOKENS: Record<string, SemanticToken> = {
  take: 'chart-series-1',
  defend: 'chart-series-2',
  hold: 'chart-series-3',
}

/** Le jeton d'un rôle, avec repli neutre pour un rôle non catalogué. */
export function roleToken(role: string): SemanticToken {
  return ROLE_TOKENS[role] ?? 'chart-series-4'
}

/**
 * Les grandeurs du bloc ÉQUIPEMENT, dans l'ordre canonique.
 *
 * TROIS EXCLUSIONS (étape E4) :
 *   - `pad_pickups` (armes spéciales, sa propre carte) — inchangé ;
 *   - `dropped_objects` (E4.5) — « une mort n'est pas un geste, elle est devenue un
 *     segment » : le total des lâchers vit désormais DANS la pile de chaque famille
 *     du bilan (`equipment_<famille>.outcomes.dropped`), une ligne à part le
 *     compterait deux fois ;
 *   - toute grandeur SUPPLANTÉE par son équivalent `equipment_<famille>` de LA MÊME
 *     SESSION : `deployed_<famille>`, `camo_episodes`, `overshield_episodes` comptent
 *     des GESTES (poses, épisodes), quand `equipment_<famille>` compte des OBJETS
 *     (les trois issues, décision P1) — deux lignes pour une même famille
 *     dupliqueraient la grandeur. Sans équivalent (grappin, propulseur — la table
 *     `equipmentOutcomeStems` côté Go ne les nomme pas, cf. §6 du plan), la grandeur
 *     GESTE reste seule, rendu STRICTEMENT INCHANGÉ (E1/E4.1 : absent de bilan =
 *     comme avant).
 */
export function equipmentMetrics(
  metrics: SessionUsageMetric[] | null | undefined,
): SessionUsageMetric[] {
  const all = metrics ?? []
  const bilanFamilies = new Set(
    all
      .filter((m) => metricKind(m.key) === 'equipment')
      .map((m) => m.key.slice(EQUIPMENT_PREFIX.length)),
  )
  return all
    .filter((m) => metricKind(m.key) !== 'pads')
    .filter((m) => metricKind(m.key) !== 'dropped')
    .filter((m) => {
      const family = equipmentBilanFamilyOf(m.key)
      return family == null || !bilanFamilies.has(family)
    })
    .sort((a, b) => {
      const ra = METRIC_RANK[metricKind(a.key)] ?? 9
      const rb = METRIC_RANK[metricKind(b.key)] ?? 9
      return ra !== rb ? ra - rb : a.key.localeCompare(b.key)
    })
}

/** Le TOTAL des ramassages d'armes spéciales (clé `pad_pickups` du contrat). */
export function padMetric(
  metrics: SessionUsageMetric[] | null | undefined,
): SessionUsageMetric | null {
  return (metrics ?? []).find((m) => metricKind(m.key) === 'pads') ?? null
}
