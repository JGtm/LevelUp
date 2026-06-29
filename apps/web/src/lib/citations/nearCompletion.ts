/**
 * nearCompletion — sélection « citations bientôt terminées » pour l'accueil.
 *
 * Pur, title-agnostic : opère sur le view-model partagé `CitationDisplayItem`
 * (Infinite = citations dérivées, H5 = commendations natives), donc les deux
 * titres sont couverts sans logique spécifique. `pct` y est la progression
 * INTRA-palier (proximité au prochain palier, 0..100), exactement le signal
 * recherché — voir analysis.computeTierProgress côté Go.
 *
 * Définition retenue de « bientôt terminée » : une citation TIERÉE, ENTAMÉE,
 * NON maîtrisée, dont le prochain palier est proche en PROPORTION (`pct` élevé).
 * Le critère proportionnel est robuste quelle que soit l'échelle des paliers
 * (un palier 0→10 et un palier 0→5000 sont comparables). Le franchissement du
 * DERNIER palier maîtrise la citation → on le signale (`isFinalTier`).
 */
import type { CitationDisplayItem } from './types'

/**
 * Seuil de proximité (progression intra-palier, %) en deçà duquel une citation
 * n'est pas considérée « bientôt terminée ». Tunable : plus haut = liste plus
 * courte mais plus « imminente ».
 */
export const NEAR_COMPLETION_MIN_PCT = 70

/** Nombre de tuiles affichées par défaut dans la section accueil (une seule ligne). */
export const NEAR_COMPLETION_DEFAULT_LIMIT = 5

export interface NearCompletionItem {
  item: CitationDisplayItem
  /** Unités restantes avant le prochain palier (`nextTierTarget - total`), ≥ 1. */
  remaining: number
  /** Le prochain palier est le DERNIER → le franchir maîtrise la citation. */
  isFinalTier: boolean
}

/**
 * selectNearCompletion retourne les citations les plus proches de leur prochain
 * palier, triées de la plus imminente à la moins imminente.
 *
 * Tri : (1) proximité proportionnelle `pct` desc — la plus proche d'abord ;
 * (2) à proximité égale, le dernier palier d'abord (récompense = maîtrise) ;
 * (3) moins d'unités restantes ; (4) nom (stable).
 */
export function selectNearCompletion(
  items: CitationDisplayItem[],
  limit = NEAR_COMPLETION_DEFAULT_LIMIT,
): NearCompletionItem[] {
  const candidates: NearCompletionItem[] = items
    .filter(
      (i) =>
        i.tierCount > 0 && // tierée → `pct` = proximité au prochain palier
        !i.isMastered && // pas déjà maîtrisée
        i.total > 0 && // entamée
        i.nextTierTarget > i.total && // un prochain palier concret devant soi
        i.pct >= NEAR_COMPLETION_MIN_PCT,
    )
    .map((i) => ({
      item: i,
      remaining: i.nextTierTarget - i.total,
      isFinalTier: i.tierIndex >= i.tierCount - 1,
    }))

  candidates.sort((a, b) => {
    if (b.item.pct !== a.item.pct) return b.item.pct - a.item.pct
    if (a.isFinalTier !== b.isFinalTier) return a.isFinalTier ? -1 : 1
    if (a.remaining !== b.remaining) return a.remaining - b.remaining
    return a.item.name.localeCompare(b.item.name)
  })

  return candidates.slice(0, Math.max(0, limit))
}

/**
 * allCitationsMastered — vrai si le joueur a TOUT maîtrisé : au moins une citation
 * tiérée existe et toutes les citations tiérées sont maîtrisées. Pilote le message
 * « tout complété » de l'accueil quand `selectNearCompletion` ne retourne rien parce
 * qu'il ne reste plus aucun palier à franchir (et non parce que rien n'est proche).
 *
 * Les citations non tiérées (`tierCount === 0` → pas d'anneau de progression) sont
 * ignorées : elles n'ont pas de notion de maîtrise par palier.
 */
export function allCitationsMastered(items: CitationDisplayItem[]): boolean {
  const tiered = items.filter((i) => i.tierCount > 0)
  return tiered.length > 0 && tiered.every((i) => i.isMastered)
}
