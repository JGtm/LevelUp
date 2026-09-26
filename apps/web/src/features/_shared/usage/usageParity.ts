/**
 * usageParity.ts — LA PARITÉ du dénominateur « mon camp / lobby » du bloc « usages
 * d'équipement, armes spéciales et objectifs ».
 *
 * Extrait de `session-detail/usageLogic.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/` (étape E5.1).
 */

/**
 * La parité du dénominateur « mon camp / lobby » : la part attendue d'un camp moyen,
 * soit 100 × effectif d'équipe / effectif du lobby. Non publiée telle quelle par le
 * contrat (qui publie les deux parités de JOUEUR) — dérivée des deux effectifs moyens.
 */
export function teamOfLobbyParityPct(
  teamSizeAvg: number | null | undefined,
  lobbySizeAvg: number | null | undefined,
): number | null {
  if (teamSizeAvg == null || lobbySizeAvg == null || lobbySizeAvg <= 0) return null
  return (teamSizeAvg / lobbySizeAvg) * 100
}
