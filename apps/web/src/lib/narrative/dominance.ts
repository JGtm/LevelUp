/**
 * dominance — source UNIQUE des correspondances `dominance_flag` → i18n / token.
 *
 * Le drapeau (canonical.DominanceFlag côté Go) vaut 0 (aucun badge) ou 1..5.
 * Trois surfaces l'affichent : la colonne Dominance de l'Explorateur, les badges
 * narratifs Career/Match view, et le marqueur de la bande de résultats
 * (Timeseries + Escouade). Les libellés et les couleurs DOIVENT être identiques
 * partout : une seule table ici, pas de copie par feature.
 */
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import type { SemanticToken } from '@/lib/accessibility'
import { formatMessage } from '@/lib/i18n/format'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
import type { Locale } from '@/lib/i18n/locale'

/** Drapeau → clé i18n (manifeste match_view, FR + EN déjà présents). */
export const DOMINANCE_LABEL_KEYS: Record<DominanceValue, MatchViewManifestKey> = {
  1: 'narrative.dominance.domination',
  2: 'narrative.dominance.humiliation',
  3: 'narrative.dominance.remontada',
  4: 'narrative.dominance.debandade',
  5: 'narrative.dominance.contre_remontada',
}

/** Drapeau → token sémantique de couleur (cf. semantic-tokens.ts). */
export const DOMINANCE_COLOR_TOKENS: Record<DominanceValue, SemanticToken> = {
  1: 'narrative-dominant',
  2: 'narrative-humiliation',
  3: 'narrative-remontada',
  4: 'narrative-debacle',
  5: 'narrative-contre-remontada',
}

/**
 * dominanceLabels — libellés localisés des 5 drapeaux, prêts pour le tooltip de
 * la bande de résultats (`OutcomeSequenceTape.dominanceLabels`).
 */
export function dominanceLabels(locale: Locale): Record<DominanceValue, string> {
  return {
    1: formatMessage(matchViewManifest, DOMINANCE_LABEL_KEYS[1], locale),
    2: formatMessage(matchViewManifest, DOMINANCE_LABEL_KEYS[2], locale),
    3: formatMessage(matchViewManifest, DOMINANCE_LABEL_KEYS[3], locale),
    4: formatMessage(matchViewManifest, DOMINANCE_LABEL_KEYS[4], locale),
    5: formatMessage(matchViewManifest, DOMINANCE_LABEL_KEYS[5], locale),
  }
}
