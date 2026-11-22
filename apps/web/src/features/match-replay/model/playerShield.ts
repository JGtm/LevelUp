/**
 * playerShield.ts — LES BRIS ET LES RECHARGES DE BOUCLIER D'UN JOUEUR, COMPTÉS JUSQU'À L'INSTANT LU. Sans React.
 *
 * LE SERVEUR DATE, LE WEB COMPTE. `doc.shieldCycles` donne, par xuid, les instants de chaque bris et de chaque
 * recharge complète en millisecondes de l'axe du document (image fois pas d'image : l'horloge de la tête de
 * lecture, `frameToMs`). La définition d'un bris et d'une recharge est celle du serveur
 * (`analysis/shieldcycle`) : rien n'est redéfini ici, on ne fait que compter les instants <= tête de lecture et
 * lire la longueur des listes pour le match.
 *
 * UNE VALEUR QUE LE DOCUMENT NE DONNE PAS EST ABSENTE (`null`), jamais un zéro : un joueur que le film ne mesure
 * pas n'a pas d'entrée dans `shieldCycles` (mode sans bouclier, joueur sans mesure, document sans pas d'image).
 * Un joueur mesuré sans événement a deux listes vides : zéro bris est alors une mesure.
 *
 * Module pur : ni DOM, ni texte d'interface, ni couleur.
 */
import { lastIndexAt } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Les instants (ms de l'axe du document) des bris et des recharges d'un joueur, triés. */
export interface ShieldSeries {
  readonly breaks: readonly { t: number }[]
  readonly recharges: readonly { t: number }[]
}

/** Ce que la fiche écrit : la valeur à l'instant et le total du match, pour chacune des trois lignes. */
export interface ShieldReading {
  breaks: number
  recharges: number
  /** Recharges moins bris à l'instant : négatif quand le joueur a plus de bris que de recharges. */
  differential: number
  matchBreaks: number
  matchRecharges: number
  matchDifferential: number
}

const dated = (instants: readonly number[] | null | undefined): { t: number }[] =>
  (instants ?? []).map((t) => ({ t }))

/**
 * La série du joueur : la première clé d'identité que `shieldCycles` connaît. `null` quand aucune clé n'y est
 * (joueur non mesuré, bot sans xuid, document sans le calque).
 */
export function readShieldSeries(
  doc: Pick<ReplayDocumentReady, 'shieldCycles'>,
  keys: readonly string[],
): ShieldSeries | null {
  const cycles = doc.shieldCycles
  if (!cycles) return null
  const key = keys.find((k) => cycles[k] !== undefined)
  if (key === undefined) return null
  return { breaks: dated(cycles[key].breaks), recharges: dated(cycles[key].recharges) }
}

/** Les bris, les recharges et leur différentiel jusqu'à `nowMs` (compris), avec les totaux du match. */
export function shieldReadingAt(series: ShieldSeries, nowMs: number): ShieldReading {
  const breaks = lastIndexAt(series.breaks, nowMs) + 1
  const recharges = lastIndexAt(series.recharges, nowMs) + 1
  const matchBreaks = series.breaks.length
  const matchRecharges = series.recharges.length
  return {
    breaks,
    recharges,
    differential: recharges - breaks,
    matchBreaks,
    matchRecharges,
    matchDifferential: matchRecharges - matchBreaks,
  }
}
