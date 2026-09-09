/**
 * equipmentUsageChart.ts — LA PROJECTION DES DEUX VUES DU BILAN D'ÉQUIPEMENT, et l'encre des
 * quatre familles de geste (E2, 2026-09-09 : `deployed` et `dropped` ont fusionné en
 * `equipment` — cf. `USAGE_GROUP_TOKENS`).
 *
 * LE TABLEAU EST DEVENU UN GRAPHE (2026-09-03, retours utilisateur sur l'onglet Chronologie).
 * Deux vues empilées remplacent le tableau à deux niveaux d'en-tête :
 *   1. « Nombre de gestes par joueur » — la grille partagée (`components/charts/ValueGrid`),
 *      une colonne par colonne de mesure, chaque colonne avec SON échelle ;
 *   2. « Part de chaque équipe, geste par geste » — une barre 100 % par FAMILLE de geste.
 *
 * LA FAMILLE, PAS LA COLONNE, PORTE LA COULEUR. `usageColumnGroups` décide déjà quelles
 * familles la mesure justifie (grappin, états actifs, poses, lâchés, lancers) ; la table des
 * encres est indexée PAR CETTE CLÉ, jamais par le rang de la colonne — une famille absente d'un
 * match ne doit pas repeindre les autres, sans quoi deux matchs voisins se liraient avec deux
 * conventions de couleur. Le typage `Record<UsageGroupKey, …>` rend la table exhaustive.
 *
 * POURQUOI LA FAMILLE `frag-*` PLUTÔT QUE CINQ JETONS DE GAMMES DIFFÉRENTES. Les cinq teintes
 * validées par la maquette (ambre, cyan, rose, violet, vert) sont, à une nuance de vert près,
 * exactement celles de la famille `frag-*` de la palette. Et c'est la SEULE famille du dépôt
 * dont la distance perceptuelle toutes-paires est tenue par un garde-rail PALETTE PAR PALETTE
 * (`fragClass.guard.test.ts`) : emprunter cinq jetons à cinq gammes ordinales ou de statut
 * (`perf-tier-*`, `warning`, `narrative-*`) donnerait cinq teintes distinctes sur la palette
 * défaut et deux teintes confondues sur Okabe-Ito. Ici la couleur ne dit rien d'ordinal — elle
 * ne fait qu'identifier une famille — donc la gamme est un vocabulaire, pas un jugement.
 *
 * LA PART D'ÉQUIPE SE COMPTE EN GESTES, PAS EN COLONNES. Une famille peut porter plusieurs
 * colonnes d'unités différentes (les états actifs en comptent trois : épisodes, durée, frags) :
 * les additionner n'aurait aucun sens. La vue 2 compte donc le NOMBRE DE GESTES de la famille —
 * la même grandeur que `tallyTotal`, ventilée famille par famille.
 *
 * Pur : aucun React, aucun hex, aucune langue — les libellés et les encres d'équipe arrivent
 * par l'appelant.
 */
import type { ValueGridModel, ValueGridRow } from '@/components/charts/valueGridModel'
import { buildValueGrid } from '@/components/charts/valueGridModel'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

import { EQUIP_FAMILY_CAMO, EQUIP_FAMILY_OVERSHIELD } from './equipmentFx'
import type { UsageColumn, UsageColumnGroup, UsageGroupKey } from './equipmentUsageColumns'
import type { EquipmentUsageTally, EquipmentUsageTeam } from './equipmentUsageLogic'

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
  episodes: 'frag-heavy', // violet
  equipment: 'frag-shoulder', // cyan — reprend le jeton de l'ancien `deployed`
  grenades: 'frag-grenade', // ambre
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
 * usageGestureCount — le NOMBRE DE GESTES d'une famille dans un compteur.
 *
 * Un épisode d'état actif est UN geste (sa durée et ses frags le décrivent, ils ne s'ajoutent
 * pas à lui). Les frags sous effet actif n'en sont pas un : ce sont des conséquences.
 */
export function usageGestureCount(tally: EquipmentUsageTally, group: UsageGroupKey): number {
  const sum = (m: Record<string, number> | Record<number, number>): number =>
    Object.values(m).reduce((a: number, b: number) => a + b, 0)
  switch (group) {
    case 'grapple':
      return tally.grapplePulls
    case 'episodes':
      return Object.values(tally.episodes).reduce((a, e) => a + e.count, 0)
    case 'equipment':
      // FUSION (E2) : les poses déployées, les objets lâchés, ET les activations des deux
      // power-ups (leur côté « utilisé » vient des épisodes, pas d'une pose — P2). Le compte
      // d'épisode y figure DEUX FOIS au total du bloc (aussi dans la ligne `episodes`,
      // décision documentée de garder les deux vues) : ce n'est pas une double mesure, c'est
      // la même mesure lue sous deux questions différentes.
      return (
        sum(tally.deployed) +
        sum(tally.dropped) +
        (tally.episodes[EQUIP_FAMILY_CAMO]?.count ?? 0) +
        (tally.episodes[EQUIP_FAMILY_OVERSHIELD]?.count ?? 0)
      )
    case 'grenades':
      return sum(tally.grenades)
  }
}

/** Ce que l'appelant doit fournir pour habiller un camp : son nom et son encre. */
export interface UsageTeamVisual {
  teamLabel: (side: string | null) => string
  teamAccent: (side: string | null) => string
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
    group: p.side ?? '',
    accent: input.teamAccent(p.side),
    emphasis: p.xuid === input.meXUID && input.meXUID != null,
    hint: `${p.name} — ${input.teamLabel(p.side)}`,
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
    tooltip: (r, c, text) => input.tipFmt(players[r].name, leaves[c].column.label, text),
  })
}

/** Un camp dans une barre de part : son nom, son encre, son compte et son pourcentage. */
export interface UsageShareSegment {
  side: string | null
  label: string
  accent: string
  count: number
  percent: number
}

/** Une famille de geste et la part de chaque camp dedans. */
export interface UsageShareRow {
  key: UsageGroupKey
  label: string
  /** La réserve de mesure de la famille (`UsageColumnGroup.hint`), portée par son nom. */
  hint: string
  color: string
  total: number
  segments: UsageShareSegment[]
}

/**
 * buildUsageShares — la vue 2. Une ligne par famille MESURÉE ; une famille dont aucun camp n'a
 * fait le moindre geste n'a pas de ligne (une barre vide n'est pas une part).
 *
 * Le pourcentage est arrondi à l'entier pour l'affichage ; le COMPTE BRUT l'accompagne toujours,
 * c'est lui qui fait foi — deux segments à 50 % ne disent pas s'ils valent 1 ou 40.
 */
export function buildUsageShares(
  input: { teams: EquipmentUsageTeam[]; groups: UsageColumnGroup[] } & UsageTeamVisual,
): UsageShareRow[] {
  const rows: UsageShareRow[] = []
  for (const group of input.groups) {
    const counts = input.teams.map((team) => usageGestureCount(team.total, group.key))
    const total = counts.reduce((a, b) => a + b, 0)
    if (total === 0) continue
    rows.push({
      key: group.key,
      label: group.label,
      hint: group.hint,
      color: usageGroupColor(group.key),
      total,
      segments: input.teams
        .map((team, i) => ({
          side: team.side,
          label: input.teamLabel(team.side),
          accent: input.teamAccent(team.side),
          count: counts[i],
          percent: Math.round((counts[i] / total) * 100),
        }))
        .filter((s) => s.count > 0),
    })
  }
  return rows
}
