/**
 * teamNames — résolution team_id → nom officiel d'équipe Halo Infinite.
 *
 * Port 1:1 de `src/config.py::TEAM_MAP` (branche main Python). Halo Infinite
 * définit 9 équipes nommées (Eagle / Cobra / ...) attribuées aux team_id 0..8
 * de manière fixe par 343 Industries — ce mapping ne dépend pas du match.
 *
 * Le backend expose `team_side` au format "t{N}" pour MatchScoreboard
 * (cf. buildTeamTabFull qui fait `fmt.Sprintf("t%d", *s.TeamID)`). D'autres
 * surfaces (picker média) exposent l'entier directement — utiliser alors
 * `resolveTeamNameFromID`.
 *
 * Module placé dans `lib/halo/` (et non `features/match-view/`) pour pouvoir
 * être consommé par plusieurs features sans couplage cross-feature.
 */

const TEAM_NAMES_HALO_INFINITE: Record<number, string> = {
  0: 'Eagle',
  1: 'Cobra',
  2: 'Hades',
  3: 'Valkyrie',
  4: 'Rampart',
  5: 'Cutlass',
  6: 'Valor',
  7: 'Hazard',
  8: 'Observer',
}

/**
 * Parse "t{N}" → N. Retourne null si le format n'est pas reconnu (équipe vide
 * ou ancienne convention). Idempotent face à null/undefined.
 */
export function parseTeamSideID(teamSide: string | null | undefined): number | null {
  if (!teamSide) return null
  const m = /^t(\d+)$/.exec(teamSide)
  if (!m) return null
  return Number(m[1])
}

/**
 * Résout team_side "t{N}" → nom officiel Halo Infinite. Retourne null si
 * team_id absent de la map (équipe non standard ou nouvelle non encore
 * référencée) ou si le format est invalide.
 */
export function resolveTeamName(teamSide: string | null | undefined): string | null {
  const id = parseTeamSideID(teamSide)
  if (id == null) return null
  return TEAM_NAMES_HALO_INFINITE[id] ?? null
}

/**
 * Variante directe : team_id (entier) → nom officiel, sans passer par le
 * format "t{N}". Utile pour les surfaces qui exposent déjà l'entier.
 */
export function resolveTeamNameFromID(teamID: number | null | undefined): string | null {
  if (teamID == null) return null
  return TEAM_NAMES_HALO_INFINITE[teamID] ?? null
}
