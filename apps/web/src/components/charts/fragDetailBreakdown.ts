/**
 * buildFragDetailBreakdown — source UNIQUE du « Détails des frags » (breakdown recoloré par
 * classe) partagé par Match view / Synthesis / Sessions.
 *
 * = armes (guns, per-arme, depuis la liste per-arme de la surface) + détail des classes
 * NON-arme (mêlée / grenade / capacités spartanes) tiré de la `FragDistribution` (compteurs
 * natifs autoritatifs), SANS double-comptage : la mêlée/grenade/capacité ne vient JAMAIS de
 * la liste per-arme (on écarte ces classes des guns), le résidu « Non attribué » est exclu.
 *
 * Garde-rail : `fragDetailBreakdown.guard.test.ts` interdit de réinliner ce littéral ailleurs
 * (le set GUN_CLASSES + le filtre) — cf. règle ≤2 copies (3e usage = centraliser).
 */
import type { FragDistribution, SynthesisWeaponKillEntry } from '@/lib/api/types'
import { fragRoleDisplayLabel } from './fragRoleLabel'

/** Classes servies par des ARMES (registre) — les seules gardées depuis la liste per-arme. */
export const GUN_CLASSES = new Set(['shoulder', 'sidearm', 'heavy'])

/**
 * Classes qu'AUCUNE ligne per-arme ne porte : leurs entrées de registre n'ont pas d'id
 * numérique, elles n'entrent donc jamais dans `weapon_kills`. Miroir EXACT du set Go
 * `nonCombatFragClasses` (apps/go-api/internal/domain/frag_distribution.go) — source unique
 * du littéral côté web, dont `NON_COMBAT_WEAPON_ROLES` (features/synthesis/weaponRoleInsight)
 * DÉRIVE en y ajoutant véhicule et tourelle.
 *
 * CE SET NE FILTRE PAS À LUI SEUL LE « DÉTAILS DES FRAGS », et c'est voulu : depuis le lot
 * « kills hors arme à feu » (2026-08-29), `equipment` et `environmental` ont un niveau 2 PAR
 * OBJET (répulseur, bobines, chute) servi par la source de dégât du film — ce sont des lignes
 * nommables, elles ont leur place dans le tableau. Seul le résidu `unattributed` en est exclu,
 * parce qu'il n'a par définition aucun objet à nommer. Aucun garde-rail ne teste le miroir
 * Go/web : il n'y a pas de contrat de sérialisation entre les deux, seulement une intention
 * commune, re-vérifiée à la main.
 */
export const NON_WEAPON_FRAG_CLASSES: ReadonlySet<string> = new Set([
  'unattributed',
  'environmental',
  'other',
  'equipment',
])

export interface FragDetailLabels {
  roleLabel: (role: string) => string
  classLabel: (className: string) => string
}

/**
 * Construit le « Détails des frags » trié par valeur décroissante.
 * @param distribution FragDistribution de la surface (classes → rôles ; source du détail non-arme).
 * @param weapons liste per-arme NORMALISÉE (label / kills / class) de la surface.
 * @param labels résolveurs i18n (rôle/classe) — injectés (helper pur, testable).
 */
export function buildFragDetailBreakdown(
  distribution: FragDistribution | null | undefined,
  weapons: SynthesisWeaponKillEntry[],
  labels: FragDetailLabels,
): SynthesisWeaponKillEntry[] {
  // Armes (guns) : on écarte d'éventuelles lignes non-arme pour ne pas doubler avec le détail.
  const guns = weapons.filter((w) => !w.class || GUN_CLASSES.has(w.class))

  // Détail NON-arme depuis la distribution (mêlée par rôle Assassinat/Corps-à-corps, grenade,
  // capacités spartanes) — par rôle si niveau 2, sinon en feuille. Skip guns + « Non attribué ».
  const details: SynthesisWeaponKillEntry[] = []
  for (const c of distribution?.classes ?? []) {
    if (GUN_CLASSES.has(c.class) || c.class === 'unattributed') continue
    const roles = c.roles ?? []
    if (roles.length > 0) {
      // Nom d'engin servi par l'API (véhicule/tourelle) ou clé canonique traduite.
      for (const r of roles) {
        details.push({ label: fragRoleDisplayLabel(r, labels.roleLabel), kills: r.kills, class: c.class })
      }
    } else {
      details.push({ label: labels.classLabel(c.class), kills: c.kills, class: c.class })
    }
  }

  return [...guns, ...details].sort((a, b) => b.kills - a.kills)
}
