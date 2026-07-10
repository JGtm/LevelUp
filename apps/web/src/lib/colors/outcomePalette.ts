/**
 * outcomePalette — couleurs sémantiques d'un indicateur signé/ratio.
 *
 * Source unique des helpers « couleur d'un K/D-ratio / taux de victoire / KDA net »
 * qui avaient été recodés à l'identique dans CareerRivalsSection, PalmaresRelationsPage,
 * ExplorerEncounterBriefing, MatchEncountersTable, RelationsTable, RelationsRivalryCards
 * (H7, 2026-07-04). TOKENS sémantiques uniquement (règle couleurs n°12) — via tokenCssVar.
 *
 * Convention de retour : `undefined` = pas de couleur (valeur absente/neutre au sens
 * « ne pas teinter »), sinon la CSS var d'un token outcome-win/loss/draw.
 */
import { tokenCssVar } from '@/lib/accessibility'

/** Couleur d'un RATIO déjà calculé (K/D, etc.) — seuil 1 : >1 gagnant, <1 perdant, =1 nul. */
export function ratioColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  if (v > 1) return tokenCssVar('outcome-win')
  if (v < 1) return tokenCssVar('outcome-loss')
  return tokenCssVar('outcome-draw')
}

/** Couleur d'un TAUX DE VICTOIRE 0..1 — seuil 0.5 : ≥0.5 gagnant, sinon perdant. */
export function winRateColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  return v >= 0.5 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
}

/** Couleur d'un NET SIGNÉ (ex: KDA net) — seuil 0 : >0 gagnant, <0 perdant, =0 nul. */
export function kdaNetColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  if (v > 0) return tokenCssVar('outcome-win')
  if (v < 0) return tokenCssVar('outcome-loss')
  return tokenCssVar('outcome-draw')
}

/**
 * Couleur d'un K/D à partir des kills/deaths bruts. Garde deaths=0 : si kills>0 →
 * gagnant (K/D « infini »), sinon undefined (0/0 = neutre). Sinon délègue à ratioColor.
 */
export function kdRatioColor(
  kills: number | null | undefined,
  deaths: number | null | undefined,
): string | undefined {
  if (kills == null || deaths == null) return undefined
  if (deaths === 0) return kills > 0 ? tokenCssVar('outcome-win') : undefined
  return ratioColor(kills / deaths)
}

/**
 * Variante de kdRatioColor quand le ratio est DÉJÀ calculé mais qu'on veut la garde
 * deaths=0 (ex: butterfly rivals). deaths=0 & ratio>0 → gagnant ; sinon ratioColor(ratio).
 */
export function ratioColorGuarded(deaths: number, ratio: number): string | undefined {
  if (deaths === 0) return ratio > 0 ? tokenCssVar('outcome-win') : undefined
  return ratioColor(ratio)
}

/**
 * Classe Tailwind SÉMANTIQUE d'un taux 0..1 — seuil 0.5. Retourne '' si absent.
 * (variante className de winRateColor, pour les cellules déjà en classes utilitaires.)
 */
export function winRateClass(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return ''
  return v >= 0.5 ? 'text-success font-bold' : 'text-warning font-bold'
}
