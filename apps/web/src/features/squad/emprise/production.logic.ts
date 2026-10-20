/**
 * production.logic.ts — LES MODÈLES PURS du bloc « Prendre, et s'en servir » de l'onglet Emprise
 * (lot L5 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 ; maquette de l'onglet,
 * `renderProductivite` et `renderRendement`).
 *
 * Tout vient de `squad_emprise.production[]`, calculé côté Go (lot L4) — le web ne recalcule
 * aucun rendement :
 *
 *   - « Frags obtenus avec les ressources » : barre épaisse = les frags, barre fine = ce qui les
 *     a permis (`exposure.value` : temps d'effet, prises). LES DEUX BARRES PORTENT SUR LA MÊME
 *     POPULATION que le rendement : quand l'exposition est tracée (barre fine), la barre épaisse lit
 *     `exposure.kills` (frags des matchs où l'exposition est mesurée), jamais `kills` (feuille de
 *     match, toute la soirée) — sinon « la coupure de l'épaisse à gauche de celle de la fine »
 *     ne voudrait plus dire « moins produit qu'on n'a eu » (constat R3 de la revue L6.1). Les
 *     frags de toute la soirée restent lisibles match par match (grille, ligne « frags obtenus
 *     avec ») ;
 *   - « Rendement face à l'adversaire » : `yield_us` / `yield_them` / `relative_gap`, lus sur
 *     `exposure.kills`.
 *
 * Sans film (Halo 5, D10) la production des armes spéciales n'a que `kills` : la barre épaisse
 * seule (toute la feuille), aucune ligne de rendement. Pur : aucun React, aucune couleur, aucune
 * chaîne de langue.
 */
import type { SquadEmpriseBlock, SquadEmpriseCount } from '@/lib/api/types'

import { RESOURCE_ORDER } from './emprise.logic'

const total = (c: SquadEmpriseCount) => c.us + c.them

/** Une ligne de « Frags obtenus avec les ressources ». */
export interface ProductionRow {
  resource: string
  /** Les frags de chaque camp (barre épaisse), sur la population de l'exposition quand elle existe. */
  kills: SquadEmpriseCount
  /** L'exposition de chaque camp (barre fine) ; null sans mesure du film. */
  exposure: { kind: string; value: SquadEmpriseCount } | null
}

function byOrder<T extends { resource: string }>(rows: T[]): T[] {
  return rows
    .filter((r) => RESOURCE_ORDER.includes(r.resource))
    .sort((a, b) => RESOURCE_ORDER.indexOf(a.resource) - RESOURCE_ORDER.indexOf(b.resource))
}

/**
 * buildProductionRows — une ligne par ressource produite qui a au moins un frag (une barre de
 * frags vide n'a rien à partager) ; la barre fine seulement quand l'exposition existe et n'est
 * pas nulle, et alors la barre épaisse lit les frags de SA population (`exposure.kills`).
 */
export function buildProductionRows(block: SquadEmpriseBlock): ProductionRow[] {
  return byOrder(block.production ?? []).flatMap((p) => {
    const exposure = p.exposure && total(p.exposure.value) > 0 ? { kind: p.exposure.kind, value: p.exposure.value } : null
    const kills = exposure && p.exposure ? p.exposure.kills : p.kills
    if (total(kills) <= 0) return []
    return [{ resource: p.resource, kills, exposure }]
  })
}

/** Une ligne de « Rendement face à l'adversaire ». */
export interface YieldRow {
  resource: string
  /** Notre rendement / le sien − 1 (0 = autant que l'adversaire). */
  gap: number
  us: number
  them: number
}

/** Une ligne par ressource dont les deux rendements et l'écart sont publiés. */
export function buildYieldRows(block: SquadEmpriseBlock): YieldRow[] {
  return byOrder(block.production ?? []).flatMap((p) =>
    p.relative_gap != null && p.yield_us != null && p.yield_them != null
      ? [{ resource: p.resource, gap: p.relative_gap, us: p.yield_us, them: p.yield_them }]
      : [],
  )
}

/** L'axe du rendement : −50 % à +50 % (maquette, `RMAX`). */
export const YIELD_AXIS_MAX_PCT = 50

/**
 * La géométrie d'une barre de rendement, en pourcentage de la piste : le zéro au milieu, l'écart
 * borné à l'axe (±50 %) ; `x` = le bout de la barre.
 */
export function yieldGeometry(gap: number): { left: number; width: number; x: number } {
  const g = Math.max(-YIELD_AXIS_MAX_PCT, Math.min(YIELD_AXIS_MAX_PCT, gap * 100))
  const x = 50 + (g / YIELD_AXIS_MAX_PCT) * 50
  return { left: Math.min(50, x), width: Math.abs(x - 50), x }
}
