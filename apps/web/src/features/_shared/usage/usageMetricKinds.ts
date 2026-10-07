/**
 * usageMetricKinds.ts — L'ENCRE DES RÔLES D'OBJECTIF (prendre, défendre, tenir), lue par les cartes
 * d'objectif de l'Escouade (`squad/objectif/*`).
 *
 * Famille dédiée `objective-role-*` (D9, plan Emprise, 2026-09-26) : la couleur IDENTIFIE un rôle,
 * elle ne juge rien — une gamme ordinale (perf-tier) ou de statut mentirait.
 *
 * Pur : aucun React, aucune lecture de store.
 */
import type { SemanticToken } from '@/lib/accessibility'

const ROLE_TOKENS: Record<string, SemanticToken> = {
  take: 'objective-role-take',
  defend: 'objective-role-defend',
  hold: 'objective-role-hold',
}

/** Le jeton d'un rôle, avec repli neutre pour un rôle non catalogué. */
export function roleToken(role: string): SemanticToken {
  return ROLE_TOKENS[role] ?? 'chart-series-4'
}
