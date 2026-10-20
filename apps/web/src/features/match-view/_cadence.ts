/**
 * _cadence.ts — logique pure de la CADENCE DES FRAGS (`MatchCadenceChart`).
 *
 * POURQUOI CE FICHIER EXISTE (registre 2026-09-05, N1). Le calcul vivait dans un
 * `useCallback` de 162 lignes à l'intérieur du `.tsx`, au-dessus du seuil de 80 lignes par
 * fonction (CLAUDE.md n° 5), et le seul test qui nommait ce composant le REMPLAÇAIT par un
 * stub : zéro ligne exécutée, zéro assertion. Il est extrait sur le patron de
 * `_scoreCurve.ts` : la géométrie ici, l'habillage ECharts là-bas.
 *
 * CE QU'IL CALCULE : les deux empilements par phase, leurs moyennes mobiles, le PIC (la
 * phase la plus meurtrière, camps confondus) et les libellés d'abscisse. Les couleurs, les
 * épaisseurs et l'ordre de superposition restent au composant — ils dépendent de la palette
 * d'accessibilité au moment du rendu.
 *
 * Zéro dépendance React/ECharts : testable en pur (`_cadence.test.ts`).
 */
import type { MatchViewCadence } from '@/lib/api/types'

import { formatBinSeconds } from './_chartSeries'
import type { XuidMeta } from './xuidMeta'

/** Durée d'une phase, en secondes, quand le serveur ne la déclare pas. */
export const CADENCE_PHASE_SECONDS_DEFAULT = 30

/** Largeur de la moyenne mobile : trois phases, fenêtre réduite au démarrage. */
export const CADENCE_MA_WINDOW = 3

/** Le pic : la phase la plus meurtrière du match, camps confondus. */
export interface CadencePeak {
  /** Rang de la phase (index d'abscisse). */
  index: number
  /** Frags des deux camps cumulés sur cette phase. */
  total: number
}

export interface Cadence {
  /** Libellé d'abscisse de chaque phase (l'instant de son DÉBUT, en MmSSs). */
  categories: string[]
  /** Frags du camp du joueur de la page, par phase. */
  ally: number[]
  /** Frags du camp adverse, par phase. */
  enemy: number[]
  /** Moyennes mobiles des deux séries, arrondies au dixième. */
  allyMA: number[]
  enemyMA: number[]
  /** Le pic ; `total` vaut 0 quand aucune phase ne porte de frag. */
  peak: CadencePeak
}

export interface CadenceInput {
  cadence: MatchViewCadence | null | undefined
  meXUID: string | null
  /** La cascade « allié = même camp que moi », résolue une fois par l'appelant. */
  xuidMeta: XuidMeta
}

/**
 * buildCadence projette les phases servies par l'API en deux empilements et leurs moyennes.
 *
 * Rend `null` — donc RIEN À L'ÉCRAN — sans phase du tout : un histogramme sans barre est un
 * cadre vide, et un cadre vide est une promesse non tenue. Un match dont toutes les phases
 * sont à zéro rend en revanche un modèle : c'est au composant de décider s'il l'affiche
 * (`hasKills`), parce que « personne n'a fragué » reste une lecture possible.
 */
export function buildCadence(input: CadenceInput): Cadence | null {
  const { cadence, meXUID, xuidMeta } = input
  if (!cadence || cadence.datapoints.length === 0) return null

  // Le `xuid === meXUID` est conservé pour le cas — anormal — où le joueur de la page n'a
  // pas de ligne au scoreboard : il reste son propre allié.
  const isAlly = (xuid: string): boolean => xuid === meXUID || (xuidMeta.get(xuid)?.ally ?? false)
  const phaseSeconds =
    (cadence.meta?.phase_seconds as number | undefined) ?? CADENCE_PHASE_SECONDS_DEFAULT

  const ally: number[] = []
  const enemy: number[] = []
  for (const dp of cadence.datapoints) {
    let amis = 0
    let adverses = 0
    for (const [xuid, count] of Object.entries(dp.components)) {
      if (isAlly(xuid)) amis += count
      else adverses += count
    }
    ally.push(amis)
    enemy.push(adverses)
  }

  return {
    categories: cadence.datapoints.map((_, i) => formatBinSeconds(i * phaseSeconds)),
    ally,
    enemy,
    allyMA: movingAverage(ally, CADENCE_MA_WINDOW),
    enemyMA: movingAverage(enemy, CADENCE_MA_WINDOW),
    peak: peakOf(ally, enemy),
  }
}

/**
 * movingAverage — moyenne mobile droite à fenêtre EXPANSIVE au démarrage.
 *
 * Les `window − 1` premiers points emploient une fenêtre réduite plutôt que de rendre
 * `null` : la courbe part dès la première phase au lieu de commencer au milieu du graphe.
 * Arrondi au dixième — la moyenne d'un compte entier n'a pas plus de précision que ça.
 */
export function movingAverage(values: readonly number[], window = CADENCE_MA_WINDOW): number[] {
  return values.map((_, i) => {
    const start = Math.max(0, i - (window - 1))
    let sum = 0
    for (let j = start; j <= i; j++) sum += values[j]
    return Math.round((sum / (i - start + 1)) * 10) / 10
  })
}

/**
 * peakOf rend la phase la plus meurtrière, camps confondus.
 *
 * LA PREMIÈRE GAGNE À ÉGALITÉ : sur deux phases aussi meurtrières, l'étiquette se pose sur
 * la plus ancienne — le « pic » du match est celui qu'on a vu arriver en premier.
 */
function peakOf(ally: readonly number[], enemy: readonly number[]): CadencePeak {
  let index = 0
  let total = 0
  for (let i = 0; i < ally.length; i++) {
    const somme = ally[i] + enemy[i]
    if (somme > total) {
      total = somme
      index = i
    }
  }
  return { index, total }
}
