/**
 * equipmentUsageChart.ts — LA PROJECTION DES DEUX VUES DU BILAN D'ÉQUIPEMENT, et l'encre des
 * quatre familles de geste (E2, 2026-09-09 : `deployed` et `dropped` ont fusionné en
 * `equipment` — cf. `USAGE_GROUP_TOKENS`).
 *
 * LE TABLEAU EST DEVENU UN GRAPHE (2026-09-03, retours utilisateur sur l'onglet Chronologie).
 * Deux vues empilées remplacent le tableau à deux niveaux d'en-tête :
 *   1. « Nombre de gestes par joueur » — la grille partagée (`components/charts/ValueGrid`),
 *      une colonne par colonne de mesure, chaque colonne avec SON échelle ;
 *   2. « Part de chaque équipe » — depuis le 2026-09-21 (D20, proposition 5.A), une barre
 *      ÉPAISSE par famille, toutes sur LA MÊME échelle d'usages (et non plus une barre 100 %
 *      par groupe de colonnes, qui donnait deux barres de même longueur pour 24 et 38 gestes).
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
 * LA VUE 2 LIT LA MÊME MESURE QUE LA CELLULE DE LA GRILLE : la `value()` de la colonne,
 * appliquée au compteur d'un CAMP au lieu de celui d'un joueur. `usageGestureCount`, qui
 * recomptait les gestes d'un GROUPE entier pour l'ancien découpage, est mort avec lui (lot K,
 * 2026-09-21) : un second calcul du même nombre finit toujours par diverger de celui que la
 * grille écrit juste à côté (CLAUDE.md n°6).
 *
 * Pur : aucun React, aucun hex, aucune langue — les libellés et les encres d'équipe arrivent
 * par l'appelant.
 */
import type { ValueGridModel, ValueGridRow } from '@/components/charts/valueGridModel'
import { buildValueGrid } from '@/components/charts/valueGridModel'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

import type { UsageColumn, UsageColumnGroup, UsageGroupKey } from './equipmentUsageColumns'
import type { EquipmentUsageTeam } from './equipmentUsageLogic'

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

/** Un camp dans la barre d'une famille : son nom, son encre, son compte et ses deux parts. */
export interface UsageFamilyBarSegment {
  side: string | null
  label: string
  accent: string
  count: number
  /** Part du camp DANS SA FAMILLE (0..100, arrondi) — l'infobulle, jamais la longueur. */
  percent: number
  /** Longueur du segment, en % de la BORNE COMMUNE à toutes les lignes (0..100). */
  widthPct: number
}

/** Une famille de geste et la barre de ses deux camps. */
export interface UsageFamilyBarRow {
  key: string
  label: string
  /** La réserve de mesure de la famille (`UsageColumnGroup.hint`), portée par son nom. */
  hint: string
  total: number
  segments: UsageFamilyBarSegment[]
}

/** Un camp en légende de la vue 2 : dans l'ordre des segments, mon camp d'abord. */
export interface UsageFamilyBarTeam {
  side: string | null
  label: string
  accent: string
}

/**
 * Les barres de la vue 2. LA BORNE COMMUNE N'EST PAS PUBLIÉE : elle est déjà DÉPENSÉE dans le
 * `widthPct` de chaque segment — la republier donnerait à l'appelant de quoi refaire la
 * division, donc de quoi la refaire autrement.
 */
export interface UsageFamilyBars {
  rows: UsageFamilyBarRow[]
  /** La légende des camps, dans l'ORDRE DES SEGMENTS — sans quoi elle se lit à l'envers. */
  legend: UsageFamilyBarTeam[]
}

/** Ce que `buildUsageFamilyBars` demande en plus de l'habillage des camps. */
export interface UsageFamilyBarsInput extends UsageTeamVisual {
  teams: EquipmentUsageTeam[]
  groups: UsageColumnGroup[]
  /** Le camp du joueur de la page : son segment ouvre chaque barre. `null` = ordre du film. */
  allySide: string | null
}

/**
 * orderedTeams — MON CAMP D'ABORD (D20, 2026-09-21). Le segment de gauche est toujours le
 * mien : une barre qui changerait de main d'une famille à l'autre ne se compare pas d'un
 * coup d'œil. Sans camp connu (aucun `is_me` au tableau des scores), l'ordre du film reste.
 */
function orderedTeams(teams: EquipmentUsageTeam[], allySide: string | null): EquipmentUsageTeam[] {
  if (allySide == null) return teams
  return [...teams].sort((a, b) => Number(b.side === allySide) - Number(a.side === allySide))
}

/**
 * buildUsageFamilyBars — la vue 2 depuis le 2026-09-21 (D20, proposition 5.A de la maquette).
 *
 * UNE BARRE PAR FAMILLE, PAS PAR GROUPE, ET SUR UNE SEULE ÉCHELLE. La vue rendait jusque-là
 * une barre 100 % par GROUPE de colonnes : deux barres de même longueur, l'une valant 24
 * gestes et l'autre 38, dont la seconde mêlait murs, capteurs, propulseurs, surbouclier et
 * camouflage — aucune des colonnes de la grille voisine ne s'y retrouvait. Les lignes sont
 * maintenant LES COLONNES DE LA GRILLE (`usageLeaves`), même liste et même ordre, et leur
 * longueur est le nombre de gestes RÉEL rapporté à la borne commune : la barre du grappin
 * (24) fait quatre fois celle du capteur (6).
 *
 * LA FAMILLE PORTE SA PROPRE MESURE : le compte d'un camp est la `value()` de la colonne sur
 * le compteur du camp — la même plume que la cellule de la grille, jamais un second calcul
 * (CLAUDE.md n°6). Une famille dont aucun camp n'a fait le moindre geste n'a pas de ligne.
 *
 * Le pourcentage reste calculé (infobulle) ; le COMPTE BRUT fait foi et s'écrit dans le
 * segment quand il tient.
 */
export function buildUsageFamilyBars(input: UsageFamilyBarsInput): UsageFamilyBars {
  const teams = orderedTeams(input.teams, input.allySide)
  const mesures = usageLeaves(input.groups)
    .map((leaf) => {
      const counts = teams.map((team) => leaf.column.value(team.total) ?? 0)
      return { leaf, counts, total: counts.reduce((a, b) => a + b, 0) }
    })
    .filter((m) => m.total > 0)
  const bound = Math.max(1, ...mesures.map((m) => m.total))
  return {
    legend: teams.map((team) => ({
      side: team.side,
      label: input.teamLabel(team.side),
      accent: input.teamAccent(team.side),
    })),
    rows: mesures.map(({ leaf, counts, total }) => ({
      // LA MÊME CLÉ QUE LA COLONNE DE LA GRILLE (`buildUsageGrid`) : les deux vues nomment la
      // même liste, elles doivent la nommer pareil.
      key: `${leaf.group}.${leaf.column.key}`,
      label: leaf.column.label,
      hint: input.groups.find((g) => g.key === leaf.group)?.hint ?? '',
      total,
      segments: teams
        .map((team, i) => ({
          side: team.side,
          label: input.teamLabel(team.side),
          accent: input.teamAccent(team.side),
          count: counts[i],
          percent: Math.round((counts[i] / total) * 100),
          widthPct: (counts[i] / bound) * 100,
        }))
        .filter((s) => s.count > 0),
    })),
  }
}
