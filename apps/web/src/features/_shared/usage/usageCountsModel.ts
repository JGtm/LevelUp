/**
 * usageCountsModel.ts — LA VARIANTE COMPTES du bloc « servi ou gâché » (décision P9,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8/E6.2) : Solo (Synthèse) et Escouade.
 *
 * Sur ces deux pages l'AXE EST EN OBJETS PRIS, pas en pourcentage d'équipe (contrairement
 * à Sessions, P8) : aucun trait de parité (il n'a pas de position sur un axe de comptes),
 * et les lignes sont triées du plus pris au moins pris.
 *
 * RÉUTILISE la cellule `UsageGauge` de `UsageForms.tsx` À L'IDENTIQUE : seul le SENS de
 * `valuePct` change (longueur relative au maximum de l'axe, pas une part d'équipe) —
 * `parityPct` reste toujours `null`, la pile des trois issues et les deux repères de taux
 * (P7) sont la MÊME fonction `buildOutcomeSegments` que la page Sessions (CLAUDE.md n°6) :
 * `EquipmentUsageFamilyLine`/`EquipmentUsagePlayerLine` portent structurellement les mêmes
 * champs que `SessionUsageOutcomes` (used/kept/dropped + les deux taux de référence).
 *
 * Pur : aucun React, aucune couleur en dur.
 */
import type { Locale } from '@/lib/i18n/locale'

import { formatUsageCount } from './usageFormat'
import { buildOutcomeSegments, type UsageGaugeModel } from './usageGaugeModel'
import type { UsageText } from './usageI18n'

/** Les trois issues + les deux repères de taux — mêmes champs que `SessionUsageOutcomes`,
 *  structurellement compatibles avec `EquipmentUsageFamilyLine`/`EquipmentUsagePlayerLine`
 *  (le contrat les EMBED directement, cf. internal/domain/equipment_usage.go). */
export interface UsageCountsOutcomesLike {
  used: number
  kept: number
  dropped: number
  teammates_used_rate_pct?: number
  opponents_used_rate_pct?: number
}

export interface UsageCountsRowInput {
  key: string
  label: string
  /** Le dénominateur d'honnêteté ET la longueur de la barre (P9). */
  taken: number
  /** Absent = pas de troisième issue mesurée à ce grain (armes spéciales, P5/E6.1) :
   *  la barre rend un aplat simple, jamais un zéro inventé. */
  outcomes?: UsageCountsOutcomesLike | null
}

export interface UsageCountsRowModel {
  key: string
  label: string
  taken: number
  gauge: UsageGaugeModel
}

export interface UsageCountsGridModel {
  rows: UsageCountsRowModel[]
  /** Le libellé complet de la graduation de fin d'axe (« 140 objets pris »). */
  axisMaxText: string
}

export interface BuildCountsGridOptions {
  t: UsageText
  locale: Locale
  /** Équipement (« pris ») ou armes spéciales (« prises ») — vocabulaire des textes. */
  unit: 'equipment' | 'weapon'
  /** Trier du plus pris au moins pris (P9). Défaut `true`. `false` quand l'appelant a
   *  déjà l'ordre voulu (`families[]` est déjà trié côté Go). */
  sort?: boolean
}

/**
 * niceAxisMax — un maximum d'axe ROND, toujours ≥ `max` (jamais un bord de barre pile sur
 * le bord du graphe). Algorithme standard « nice number » (1/2/5/10 × 10^n) : pas de
 * prétention à reproduire une maquette au pixel, juste un axe lisible.
 */
function niceAxisMax(max: number): number {
  if (max <= 0) return 1
  const magnitude = Math.pow(10, Math.floor(Math.log10(max)))
  const normalized = max / magnitude
  let niceNorm: number
  if (normalized <= 1) niceNorm = 1
  else if (normalized <= 2) niceNorm = 2
  else if (normalized <= 5) niceNorm = 5
  else niceNorm = 10
  return niceNorm * magnitude
}

function clampPct(v: number): number {
  return Math.max(0, Math.min(100, v))
}

function unitValueFmt(unit: 'equipment' | 'weapon', n: string, t: UsageText): string {
  return unit === 'equipment' ? t.countTakenFmt(n) : t.countPickupsFmt(n)
}

function buildCountsRow(
  input: UsageCountsRowInput,
  axisMax: number,
  opts: BuildCountsGridOptions,
): UsageCountsRowModel {
  const { t, locale, unit } = opts
  const count = (v: number) => formatUsageCount(v, locale)
  const valueText = unitValueFmt(unit, count(input.taken), t)
  const valuePct = axisMax > 0 ? clampPct((input.taken / axisMax) * 100) : 0

  // `buildOutcomeSegments` ne lit jamais `taken` (seulement used/kept/dropped) — le
  // champ n'existe dans le type que parce que `SessionUsageOutcomes` (le contrat Go)
  // le porte. On lui donne la valeur de la ligne, sans effet sur le résultat.
  const segments = input.outcomes
    ? buildOutcomeSegments({ ...input.outcomes, taken: input.taken }, t)
    : undefined
  const teammatesRatePct = segments != null ? (input.outcomes?.teammates_used_rate_pct ?? null) : null
  const opponentsRatePct = segments != null ? (input.outcomes?.opponents_used_rate_pct ?? null) : null

  let tooltip = t.countsTipFmt(input.label, valueText)
  if (segments != null && input.outcomes != null) {
    tooltip = t.gaugeOutcomeTipFmt(
      tooltip,
      count(input.outcomes.used),
      count(input.outcomes.kept),
      count(input.outcomes.dropped),
    )
  }
  if (teammatesRatePct != null || opponentsRatePct != null) {
    tooltip = t.gaugeReferenceTipFmt(
      tooltip,
      teammatesRatePct != null ? `${count(teammatesRatePct)} %` : '—',
      opponentsRatePct != null ? `${count(opponentsRatePct)} %` : '—',
    )
  }

  const gauge: UsageGaugeModel = {
    key: input.key,
    valuePct,
    // P9 : un axe de comptes n'a pas de position de parité.
    parityPct: null,
    valueText,
    honestyText: valueText,
    tooltip,
    segments,
    teammatesRatePct,
    opponentsRatePct,
  }
  return { key: input.key, label: input.label, taken: input.taken, gauge }
}

/**
 * buildCountsGrid — la grille complète : les lignes (triées par défaut, P9) et la
 * graduation de fin d'axe. `inputs` vide ⇒ grille vide, jamais un axe fantôme.
 */
export function buildCountsGrid(
  inputs: UsageCountsRowInput[],
  opts: BuildCountsGridOptions,
): UsageCountsGridModel {
  const ordered = opts.sort === false ? inputs : [...inputs].sort((a, b) => b.taken - a.taken)
  const axisMax = niceAxisMax(Math.max(0, ...ordered.map((r) => r.taken)))
  const rows = ordered.map((input) => buildCountsRow(input, axisMax, opts))
  const axisMaxText =
    opts.unit === 'equipment'
      ? opts.t.axisEquipmentTakenFmt(formatUsageCount(axisMax, opts.locale))
      : opts.t.countPickupsFmt(formatUsageCount(axisMax, opts.locale))
  return { rows, axisMaxText }
}
