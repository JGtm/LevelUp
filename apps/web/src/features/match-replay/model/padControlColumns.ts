/**
 * padControlColumns.ts — LES COLONNES DU CONTRÔLE DES ARMES SPÉCIALES (une colonne par socle).
 *
 * POURQUOI LE RAIL 100 % EST MORT (2026-09-21, décision D18). Chaque socle avait un rail de la
 * MÊME longueur, découpé en parts de joueur : un socle pris 3 fois occupait autant de largeur
 * qu'un socle pris 12, et l'œil — qui lit des longueurs — concluait qu'ils comptaient pareil.
 * Le total n'était écrit qu'en petit à gauche, seule chose qui détrompait. La forme retenue est
 * la 4.A de la maquette : UNE COLONNE PAR SOCLE, hauteur = les prises NOMMÉES réelles, sur une
 * ÉCHELLE COMMUNE, empilée par joueur. « Le fusil de précision a changé de mains douze fois,
 * l'Empaleur quatre » se lit alors sans lire un chiffre.
 *
 * CE MODULE PROJETTE, IL NE MESURE PAS. Les prises, les joueurs, leur camp et leur
 * éclaircissement viennent tels quels de `padControlChart` (`PadBarRow`/`PadBarSegment`) :
 * aucune règle de couleur, aucune règle de rang n'est rejouée ici. Le module se contente de
 * pivoter des LIGNES en COLONNES et de préparer ce qu'un graphe empilé attend.
 *
 * L'ORDRE DES GROUPES EST CELUI DE L'APPELANT — puissance puis terrain (D2) —, et l'ordre
 * interne reste celui de `padControlLogic` (du socle le plus disputé au moins disputé).
 *
 * L'ORDRE DES JOUEURS est celui de leur PREMIÈRE apparition dans les lignes. Comme chaque
 * ligne est déjà rangée camp du joueur de la page d'abord, l'ordre obtenu ouvre sur ce camp —
 * et il est STABLE d'une colonne à l'autre, ce qui est la seule façon de suivre un joueur à
 * l'œil sur dix colonnes.
 *
 * Pur : aucun React, aucune couleur calculée, aucune langue — les libellés arrivent tout faits.
 */
import type { CategoryGroup } from '@/components/charts/barStackedGroups'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'

import type { PadBarRow } from './padControlChart'

/** Un groupe de colonnes en entrée : son titre (déjà localisé, vide = anonyme) et ses lignes. */
export interface PadColumnGroupInput {
  label: string
  rows: readonly PadBarRow[]
}

/** Un joueur du graphe : sa sous-clé empilée, et de quoi l'encrer côté appelant. */
export interface PadColumnPlayer {
  /** Sous-clé de la pile — le nom d'affichage, désambiguïsé si deux joueurs le partagent. */
  key: string
  xuid: string
  side: string | null
  /** Part de l'encre du camp (100 = l'encre pure), telle que `padControlChart` l'a calculée. */
  tint: number
  /** L'encre DOM déjà prête (`color-mix`) — pour la légende, jamais pour le canvas. */
  cssColor: string
}

/** Le graphe en colonnes : ses points, ses joueurs, ses totaux, ses notes et ses groupes. */
export interface PadColumnsModel {
  datapoints: ChartPointStacked[]
  componentOrder: string[]
  players: PadColumnPlayer[]
  /** Total nommé de chaque colonne, dans l'ordre des colonnes. */
  totals: number[]
  /** Étiquette de colonne -> « + N sans nom ». Une colonne sans manque n'a pas d'entrée. */
  notes: Record<string, string>
  groups: CategoryGroup[]
}

/** Au-delà, le nom du socle déborde sous sa colonne : il est coupé, l'infobulle garde tout. */
const COLUMN_LABEL_MAX = 15

/**
 * Deux étiquettes identiques seraient UNE SEULE catégorie ECharts (les deux colonnes
 * fusionneraient). L'espace fin est invisible et rend la clé unique sans mentir sur le nom.
 */
const THIN_SPACE = ' '

/** Le nom d'un socle sous sa colonne : coupé s'il est long, rendu unique s'il est déjà pris. */
function columnLabel(label: string, used: Set<string>): string {
  const court =
    label.length > COLUMN_LABEL_MAX ? `${label.slice(0, COLUMN_LABEL_MAX - 1).trimEnd()}…` : label
  let clef = court
  while (used.has(clef)) clef += THIN_SPACE
  used.add(clef)
  return clef
}

/** Ce que l'appelant fournit en plus des groupes : la seule langue du module. */
export interface PadColumnsInput {
  groups: readonly PadColumnGroupInput[]
  /** « + N sans nom », déjà localisé. */
  unnamedFmt: (count: number) => string
}

/**
 * buildPadColumns — le pivot lignes -> colonnes.
 *
 * Un groupe sans ligne ne produit AUCUNE colonne et aucun titre : un groupe vide ne se sépare
 * de rien. Le modèle peut donc revenir sans colonne — l'appelant y lit l'état vide de la carte.
 */
export function buildPadColumns(input: PadColumnsInput): PadColumnsModel {
  const joueurs = new Map<string, PadColumnPlayer>()
  const cles = new Map<string, string>()
  const etiquettes = new Set<string>()
  const datapoints: ChartPointStacked[] = []
  const totals: number[] = []
  const notes: Record<string, string> = {}
  const groups: CategoryGroup[] = []

  for (const groupe of input.groups) {
    if (groupe.rows.length === 0) continue
    groups.push({ label: groupe.label, span: groupe.rows.length })
    for (const row of groupe.rows) {
      const categorie = columnLabel(row.label, etiquettes)
      const components: Record<string, number> = {}
      for (const seg of row.segments) {
        const clef = clefJoueur(seg.xuid, seg.name, joueurs, cles)
        if (!joueurs.has(clef)) {
          joueurs.set(clef, {
            key: clef,
            xuid: seg.xuid,
            side: seg.side,
            tint: seg.tint,
            cssColor: seg.color,
          })
        }
        components[clef] = (components[clef] ?? 0) + seg.count
      }
      datapoints.push({ category: categorie, components })
      totals.push(row.total)
      if (row.unnamed > 0) notes[categorie] = input.unnamedFmt(row.unnamed)
    }
  }

  return {
    datapoints,
    componentOrder: [...joueurs.keys()],
    players: [...joueurs.values()],
    totals,
    notes,
    groups,
  }
}

/**
 * La sous-clé d'un joueur : son nom d'affichage, et l'espace fin en secours quand DEUX xuids
 * portent le même nom (un homonyme, un bot dupliqué). Le xuid reste la clé d'identité — le nom
 * n'est que ce qui s'écrit.
 */
function clefJoueur(
  xuid: string,
  name: string,
  joueurs: Map<string, PadColumnPlayer>,
  cles: Map<string, string>,
): string {
  const connue = cles.get(xuid)
  if (connue) return connue
  let clef = name
  while (joueurs.has(clef) && joueurs.get(clef)?.xuid !== xuid) clef += THIN_SPACE
  cles.set(xuid, clef)
  return clef
}
