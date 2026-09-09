/**
 * usageObjectives.ts — L'ORDRE CANONIQUE des rôles d'objectif (prendre / défendre / tenir) du
 * bloc « usages d'équipement, armes spéciales et objectifs ».
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/` (étape E5.1).
 *
 * Les libellés de rôle, de famille de mode et de bonus vivent avec le dictionnaire
 * (`usageI18n.ts` : roleLabel, familyLabel, powerupLabel) — ce fichier ne garde que
 * l'ordre canonique, dont le tri a besoin.
 */
import type { SessionObjectiveRoleMetric } from '@/lib/api/types'

export const ROLE_ORDER = ['take', 'defend', 'hold'] as const

/** Tri des rôles dans l'ordre canonique prendre / défendre / tenir. */
export function sortRoles(roles: SessionObjectiveRoleMetric[] | null | undefined) {
  const rank = (r: string) => {
    const i = (ROLE_ORDER as readonly string[]).indexOf(r)
    return i === -1 ? ROLE_ORDER.length : i
  }
  return [...(roles ?? [])].sort((a, b) => rank(a.role) - rank(b.role))
}
