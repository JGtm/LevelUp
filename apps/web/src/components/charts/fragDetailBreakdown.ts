/**
 * buildFragDetailBreakdown — source UNIQUE du « Détails des frags » (breakdown recoloré par
 * classe) partagé par Match view / Synthesis / Sessions.
 *
 * = armes (guns, per-arme, depuis la liste per-arme de la surface) + détail des classes
 * NON-arme (mêlée / grenade / capacités spartanes / engins) tiré de la `FragDistribution`
 * (compteurs natifs autoritatifs), SANS double-comptage : la mêlée/grenade/capacité ne vient
 * JAMAIS de la liste per-arme (on écarte ces classes des guns). Les classes sans outil de
 * destruction identifiable (`NON_WEAPON_FRAG_CLASSES` : équipement, environnement, UGC,
 * résidu « Non attribué ») sont exclues — elles restent au sunburst, pas ici.
 *
 * Garde-rail : `fragDetailBreakdown.guard.test.ts` interdit de réinliner ce littéral ailleurs
 * (le set GUN_CLASSES + le filtre) — cf. règle ≤2 copies (3e usage = centraliser).
 */
import type { FragDistribution, SynthesisWeaponKillEntry } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { fragRoleDisplayLabel } from './fragRoleLabel'

/** Classes servies par des ARMES (registre) — les seules gardées depuis la liste per-arme. */
export const GUN_CLASSES = new Set(['shoulder', 'sidearm', 'heavy'])

/**
 * Classes SANS outil de destruction identifiable — exclues du breakdown par-arme.
 * MIROIR EXACT de `domain.nonCombatFragClasses` (Go, internal/domain/frag_distribution.go),
 * dont le contrat est précisément « exclues du breakdown par-ARME » : équipement
 * (répulseur, bobines), environnement (chute, explosifs de map), UGC (`other`) et le
 * résidu non attribué. Elles restent PLEINEMENT visibles au sunburst (niveau 1 + 2), qui
 * les sert par une provenance à part (source de dégât du film) — c'est le tableau des
 * armes qu'elles polluaient : une bobine ou une chute n'est pas un outil du joueur.
 *
 * Véhicule et tourelle n'en font PAS partie (V73-3.2, même choix que côté Go) : ce sont
 * des outils réels, identifiés PAR ENGIN, avec un breakdown légitime.
 *
 * NE PAS confondre avec `NON_COMBAT_WEAPON_ROLES`
 * (features/synthesis/weaponRoleInsight.ts), qui DÉRIVE de cet ensemble en y ajoutant
 * véhicule et tourelle : l'insight coach raisonne sur le STYLE DE JEU du joueur, où un
 * frag au Warthog ne dit rien de son arsenal, alors que le tableau des armes, lui, a
 * toute raison de nommer le Warthog. Deux usages, deux ensembles, un seul littéral.
 */
export const NON_WEAPON_FRAG_CLASSES: ReadonlySet<string> = new Set([
  'unattributed',
  'environmental',
  'equipment',
  'other',
])

export interface FragDetailLabels {
  roleLabel: (role: string) => string
  classLabel: (className: string) => string
  /** Locale d'affichage courante — choisit label/label_en pour les rôles OBJET (D2). */
  locale: Locale
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
  // capacités spartanes, engins véhicule/tourelle) — par rôle si niveau 2, sinon en feuille.
  // Skip guns (déjà servis par la liste per-arme) + classes sans outil identifiable.
  const details: SynthesisWeaponKillEntry[] = []
  for (const c of distribution?.classes ?? []) {
    if (GUN_CLASSES.has(c.class) || NON_WEAPON_FRAG_CLASSES.has(c.class)) continue
    const roles = c.roles ?? []
    if (roles.length > 0) {
      // Nom d'engin servi par l'API (véhicule/tourelle) ou clé canonique traduite.
      for (const r of roles) {
        details.push({ label: fragRoleDisplayLabel(r, labels.locale, labels.roleLabel), kills: r.kills, class: c.class })
      }
    } else {
      details.push({ label: labels.classLabel(c.class), kills: c.kills, class: c.class })
    }
  }

  return [...guns, ...details].sort((a, b) => b.kills - a.kills)
}
