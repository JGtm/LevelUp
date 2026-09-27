/**
 * production.logic.ts — LES MODÈLES PURS du bloc « Prendre, et s'en servir » de l'onglet Emprise
 * (lot L5 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 ; maquette de l'onglet,
 * `renderProductivite` et `renderRendement`).
 *
 * Tout vient de `squad_emprise.production[]`, calculé côté Go (lot L4) — le web ne recalcule
 * aucun rendement :
 *
 *   - « Frags obtenus avec les ressources » : `kills` (barre épaisse : frags pendant l'effet pour
 *     les bonus, frags obtenus avec pour les armes spéciales, feuille de match), `exposure.value`
 *     (barre fine : temps d'effet, prises) ;
 *   - « Rendement face à l'adversaire » : `yield_us` / `yield_them` / `relative_gap`, lus sur
 *     `exposure.kills` (un rendement se lit sur UN périmètre : les matchs dont l'exposition est
 *     mesurée).
 *
 * Sans film (Halo 5, D10) la production des armes spéciales n'a que `kills` : la barre épaisse
 * seule, aucune ligne de rendement. Pur : aucun React, aucune couleur, aucune chaîne de langue.
 */
import type { SquadEmpriseBlock, SquadEmpriseCount } from '@/lib/api/types'

import { RESOURCE_ORDER } from './emprise.logic'

/** Natures d'exposition (contrat Go `domain.EmpriseExposure*`). */
export const EXPOSURE_EFFECT_MS = 'effect_ms'
export const EXPOSURE_PICKUPS = 'pickups'

const total = (c: SquadEmpriseCount) => c.us + c.them

/** Une ligne de « Frags obtenus avec les ressources ». */
export interface ProductionRow {
  resource: string
  /** Les frags de chaque camp (barre épaisse). */
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
 * pas nulle.
 */
export function buildProductionRows(block: SquadEmpriseBlock): ProductionRow[] {
  return byOrder(block.production ?? []).flatMap((p) => {
    if (total(p.kills) <= 0) return []
    const exposure = p.exposure && total(p.exposure.value) > 0 ? { kind: p.exposure.kind, value: p.exposure.value } : null
    return [{ resource: p.resource, kills: p.kills, exposure }]
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
export function yieldGeometry(gap: number): { left: number; width: number; x: number; clamped: boolean } {
  const pct = gap * 100
  const g = Math.max(-YIELD_AXIS_MAX_PCT, Math.min(YIELD_AXIS_MAX_PCT, pct))
  const x = 50 + (g / YIELD_AXIS_MAX_PCT) * 50
  return { left: Math.min(50, x), width: Math.abs(x - 50), x, clamped: g !== pct }
}
