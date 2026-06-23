/**
 * weaponRoleInsight — dérive un insight coach « data-driven » de la répartition
 * des frags par rôle d'arme (donnée du registre). Fonction PURE.
 *
 * Règles (priorité dans l'ordre) :
 *  1. blind_spot_power : les armes lourdes (rôle `power`) représentent < 3% des
 *     frags → angle mort à travailler (contrôle des power weapons = avantage match).
 *  2. over_reliance : un seul rôle dépasse 70% des frags → arsenal trop prévisible.
 *
 * Garde-fou : il faut un minimum de frags (MIN_KILLS) pour éviter le bruit sur
 * les petits échantillons. null = aucun insight pertinent.
 */
import type { SynthesisRoleKillEntry } from '@/lib/api/types'

export type RoleInsight =
  | { kind: 'blind_spot_power' }
  | { kind: 'over_reliance'; role: string; pct: number }
  | null

const MIN_KILLS = 50
const POWER_BLIND_THRESHOLD = 0.03
const OVER_RELIANCE_THRESHOLD = 0.7

export function weaponRoleInsight(roles: SynthesisRoleKillEntry[] | undefined): RoleInsight {
  if (!roles || roles.length === 0) return null
  const total = roles.reduce((s, r) => s + r.kills, 0)
  if (total < MIN_KILLS) return null

  const power = roles.find((r) => r.role === 'power')?.kills ?? 0
  if (power / total < POWER_BLIND_THRESHOLD) {
    return { kind: 'blind_spot_power' }
  }

  const top = roles.reduce((a, b) => (b.kills > a.kills ? b : a), roles[0])
  if (top.kills / total > OVER_RELIANCE_THRESHOLD) {
    return { kind: 'over_reliance', role: top.role, pct: Math.round((top.kills / total) * 100) }
  }
  return null
}
