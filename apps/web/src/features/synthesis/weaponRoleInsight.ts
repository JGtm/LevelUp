/**
 * weaponRoleInsight — dérive un insight coach « data-driven » de la répartition
 * des frags par rôle d'arme. Fonction PURE.
 *
 * Depuis P7 (retrait du DTO `kills_by_role`), les kills par rôle sont dérivés de la
 * FragDistribution — les rôles des GUN CLASSES (shoulder/sidearm/heavy) — au lieu
 * d'un bloc dédié. Même vocabulaire registre (row.Role), source unique. La logique
 * coach est INCHANGÉE (priorité dans l'ordre) :
 *  1. blind_spot_power : le rôle `power` représente < 3% des frags de COMBAT →
 *     angle mort à travailler (contrôle des power weapons = avantage match).
 *  2. over_reliance : un seul rôle dépasse 70% des frags de combat → arsenal trop
 *     prévisible.
 *
 * Garde-fou : minimum de frags (MIN_KILLS) pour éviter le bruit sur les petits
 * échantillons. null = aucun insight pertinent.
 */
import { GUN_CLASSES, NON_WEAPON_FRAG_CLASSES } from '@/components/charts/fragDetailBreakdown'
import type { FragDistribution } from '@/lib/api/types'

export type RoleInsight =
  | { kind: 'blind_spot_power' }
  | { kind: 'over_reliance'; role: string; pct: number }
  | null

const MIN_KILLS = 50
const POWER_BLIND_THRESHOLD = 0.03
const OVER_RELIANCE_THRESHOLD = 0.7

// Classes « gun » (ventilation par arme, registre) dont les rôles nourrissent l'insight
// coach : le SET partagé GUN_CLASSES (source unique buildFragDetailBreakdown). Les classes
// API (melee/grenade/spartan_ability) et le résidu (unattributed) sont exclus — l'insight
// raisonne sur l'ARSENAL réel.

/**
 * Rôles NON-COMBAT (frags hors-arsenal H5 : véhicules, tourelles, environnement,
 * équipement, bucket d'attribution « Spartan », UGC). Ils DOIVENT être exclus de tout
 * calcul coach : sans ça, un gros bucket non-combat gonfle le dénominateur (fausse
 * `blind_spot_power`) ou devient une fausse `over_reliance`. Conservé comme garde
 * défensif — la dérivation depuis les gun classes les écarte déjà à la source, mais
 * l'invariant reste explicite et testé.
 *
 * DÉRIVÉ de `NON_WEAPON_FRAG_CLASSES` (le miroir du set Go, source unique du littéral)
 * PLUS véhicule et tourelle. La divergence est VOULUE et ne se refermera pas : le
 * breakdown par-arme nomme légitimement le Warthog ou la tourelle Gauss (V73-3.2, un
 * engin EST un outil identifiable), alors que l'insight coach juge un STYLE DE JEU —
 * « tu ne prends jamais les armes lourdes » n'a aucun sens si le dénominateur compte
 * les frags au Warthog. Un seul littéral, deux usages.
 */
export const NON_COMBAT_WEAPON_ROLES: ReadonlySet<string> = new Set([
  // équipement / environnement / UGC / résidu — cf. NON_WEAPON_FRAG_CLASSES.
  ...NON_WEAPON_FRAG_CLASSES,
  'vehicle',
  'turret',
])

/** Paire {rôle, kills} — forme d'entrée du cœur de calcul (agnostique de la source). */
export type RoleKills = { role: string; kills: number }

/**
 * rolesFromDistribution aplatit les rôles des gun classes d'une FragDistribution en
 * une liste {role, kills} FUSIONNÉE par rôle (un rôle trans-classes comme `shotgun`,
 * présent sous Épaule ET Lourde, est sommé). Une gun class FEUILLE (ex. Poing/sidearm,
 * sans niveau 2) contribue au rôle portant son nom de classe — byte-équivalent à
 * l'ancien `kills_by_role` (row.Role='sidearm'). Classes API + résidu ignorés.
 */
export function rolesFromDistribution(distribution: FragDistribution | null | undefined): RoleKills[] {
  if (!distribution?.classes) return []
  const byRole = new Map<string, number>()
  for (const c of distribution.classes) {
    if (!GUN_CLASSES.has(c.class)) continue
    if (c.roles && c.roles.length > 0) {
      for (const r of c.roles) byRole.set(r.role, (byRole.get(r.role) ?? 0) + r.kills)
    } else {
      byRole.set(c.class, (byRole.get(c.class) ?? 0) + c.kills)
    }
  }
  return [...byRole.entries()].map(([role, kills]) => ({ role, kills }))
}

/**
 * insightFromRoles — cœur de la logique coach (INCHANGÉ depuis kills_by_role) :
 * exclut les rôles non-combat, exige MIN_KILLS de combat, puis applique les deux
 * règles (blind_spot_power prioritaire sur over_reliance).
 */
export function insightFromRoles(allRoles: RoleKills[]): RoleInsight {
  if (allRoles.length === 0) return null
  // Ne raisonner QUE sur les rôles de combat (arsenal réel).
  const roles = allRoles.filter((r) => !NON_COMBAT_WEAPON_ROLES.has(r.role))
  if (roles.length === 0) return null
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

/** Insight coach dérivé directement de la FragDistribution (gun classes). */
export function weaponRoleInsight(distribution: FragDistribution | null | undefined): RoleInsight {
  return insightFromRoles(rolesFromDistribution(distribution))
}
