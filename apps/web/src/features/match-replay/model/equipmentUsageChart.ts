/**
 * equipmentUsageChart.ts — LA PROJECTION DE LA GRILLE « Usage d'équipements, par joueur », et
 * l'encre des familles de geste (`USAGE_GROUP_TOKENS`) et des issues (`USAGE_OUTCOME_TOKENS`).
 *
 * La grille partagée (`components/charts/ValueGrid`) : une ligne par joueur, une colonne par
 * colonne de mesure, chaque colonne avec SON échelle. La part de chaque équipe se lit dans
 * « Contrôle des ressources, par match » (Vue match), pas ici.
 *
 * LA FAMILLE, PAS LA COLONNE, PORTE LA COULEUR. `usageColumnGroups` décide déjà quelles
 * familles la mesure justifie ; la table des encres est indexée PAR CETTE CLÉ, jamais par le rang
 * de la colonne — une famille absente d'un match ne doit pas repeindre les autres, sans quoi deux
 * matchs voisins se liraient avec deux conventions de couleur. Le typage
 * `Record<UsageGroupKey, …>` rend la table exhaustive.
 *
 * POURQUOI LA FAMILLE `frag-*`. C'est la SEULE famille du dépôt dont la distance perceptuelle
 * toutes-paires est tenue par un garde-rail PALETTE PAR PALETTE (`fragClass.guard.test.ts`) :
 * la couleur ne dit rien d'ordinal, elle identifie une famille.
 *
 * Pur : aucun React, aucun hex, aucune langue — les libellés et les encres d'équipe arrivent
 * par l'appelant.
 */
import type { ValueGridModel, ValueGridRow } from '@/components/charts/valueGridModel'
import { buildValueGrid } from '@/components/charts/valueGridModel'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

import type { UsageColumn, UsageColumnGroup, UsageGroupKey } from './equipmentUsageColumns'
import type { EquipmentUsageTeam } from './equipmentUsageLogic'
import type { ReplayCamp } from '../../../lib/replay/replayCamps'

/**
 * L'ENCRE DE CHAQUE FAMILLE DE GESTE. Indexée par famille, jamais par rang (cf. en-tête).
 * Ordre d'écriture = celui de `usageColumnGroups`, pour que la table se relise contre elle.
 *
 * `deployed` ET `dropped` ONT FUSIONNÉ EN `equipment` (E2, PLAN_EQUIPEMENT_GACHIS_2026-09-09.md,
 * décisions P2/P3) : une seule colonne par famille, empilée sur ses issues — `equipment`
 * reprend le jeton de l'ancien `deployed`, `dropped` n'a plus de jeton DE FAMILLE (le lâché
 * est désormais un SEGMENT d'issue, coloré par `USAGE_OUTCOME_TOKENS`, pas une famille).
 */
export const USAGE_GROUP_TOKENS: Record<UsageGroupKey, SemanticToken> = {
  grapple: 'frag-sidearm', // vert
  equipment: 'frag-shoulder', // cyan — reprend le jeton de l'ancien `deployed`
}

/** L'encre d'une famille, en variable CSS — jamais un hex (garde-rail color-tokens). */
export function usageGroupColor(group: UsageGroupKey): string {
  return tokenCssVar(USAGE_GROUP_TOKENS[group])
}

/**
 * LES TROIS ENCRES D'ISSUE (§3.1 de PLAN_EQUIPEMENT_GACHIS_2026-09-09.md — table NORMATIVE,
 * aucun autre jeton n'est autorisé ici). Elles distinguent COMMENT un geste s'est terminé,
 * jamais QUI l'a fait — à l'inverse de `USAGE_GROUP_TOKENS`, qui distingue la famille. La
 * gamme est `divergent-*` : l'issue EST un jugement (bon / neutre / mauvais), la même gamme
 * que la bande de régularité du même bloc.
 */
export const USAGE_OUTCOME_TOKENS = {
  used: 'divergent-pos',
  kept: 'divergent-neutral',
  dropped: 'divergent-neg',
} satisfies Record<string, SemanticToken>

/** L'encre d'une issue, en variable CSS — jamais un hex (garde-rail color-tokens). */
export function usageOutcomeColor(outcome: keyof typeof USAGE_OUTCOME_TOKENS): string {
  return tokenCssVar(USAGE_OUTCOME_TOKENS[outcome])
}

/** Une colonne de la grille, et la famille dont elle relève (pour son encre). */
export interface UsageLeaf {
  column: UsageColumn
  group: UsageGroupKey
}

/** usageLeaves aplatit les groupes en colonnes, chacune gardant sa famille. */
export function usageLeaves(groups: UsageColumnGroup[]): UsageLeaf[] {
  return groups.flatMap((g) => g.columns.map((column) => ({ column, group: g.key })))
}

/**
 * Ce que l'appelant doit fournir pour habiller un camp DU FILM : son nom (`campLabel`) et son
 * encre allié / adverse. Une ligne de joueur EST un camp au sens de ce type (elle porte le sien).
 */
export interface UsageTeamVisual {
  teamLabel: (camp: ReplayCamp) => string
  teamAccent: (camp: ReplayCamp) => string
}

/** Les entrées de la grille « Nombre de gestes par joueur ». */
export interface UsageGridInput extends UsageTeamVisual {
  teams: EquipmentUsageTeam[]
  groups: UsageColumnGroup[]
  /** xuid du joueur de la page : sa ligne est mise en avant. `null` = aucune. */
  meXUID: string | null
  /** Le texte d'une infobulle de barre : joueur, grandeur, valeur écrite. */
  tipFmt: (player: string, column: string, value: string) => string
}

/**
 * buildUsageGrid — la vue 1. Les lignes gardent l'ordre du roster, camp par camp : un joueur
 * se lit EN LIGNE d'une colonne à l'autre, et un filet sépare deux camps.
 */
export function buildUsageGrid(input: UsageGridInput): ValueGridModel {
  const leaves = usageLeaves(input.groups)
  const players = input.teams.flatMap((team) => team.players)
  const rows: ValueGridRow[] = players.map((p) => ({
    key: `${p.xuid}||${p.name}`,
    label: p.name,
    // LE FILET ENTRE DEUX CAMPS SUIT LE CAMP DU FILM : deux camps sans côté de feuille restent
    // deux groupes.
    group: `camp:${p.team}`,
    accent: input.teamAccent(p),
    emphasis: p.xuid === input.meXUID && input.meXUID != null,
    hint: `${p.name} — ${input.teamLabel(p)}`,
  }))
  return buildValueGrid({
    rows,
    columns: leaves.map((leaf) => ({
      key: `${leaf.group}.${leaf.column.key}`,
      label: leaf.column.label,
      duration: leaf.column.duration,
      showTotal: true,
    })),
    value: (r, c) => leaves[c].column.value(players[r]),
    format: (v, c) => leaves[c].column.format(v),
    color: (_r, c) => usageGroupColor(leaves[c].group),
    // Une colonne qui porte son PROPRE détail (états actifs : utilisations, durée cumulée,
    // frags sous l'effet) l'écrit à la place de la valeur formatée — c'est ce qui lui évite
    // d'ouvrir une colonne par unité (2026-09-13).
    tooltip: (r, c, text) =>
      input.tipFmt(
        players[r].name,
        leaves[c].column.label,
        leaves[c].column.tooltip?.(players[r]) ?? text,
      ),
  })
}
