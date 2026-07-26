/**
 * overtime — source UNIQUE du badge « Prolongation » (overtime).
 *
 * Le backend expose `is_overtime` + `overtime_seconds` sur DEUX surfaces : le
 * header de la Match View (MatchViewHeader) et les lignes de l'Explorateur
 * (ExplorerMatchesRow). Le libellé, la couleur et le tooltip DOIVENT être
 * identiques des deux côtés : une seule table ici, pas de copie par feature
 * (même contrat que lib/narrative/dominance.ts, qu'on ne touche pas).
 *
 * Sémantique du badge : marqueur NEUTRE-INFORMATIF (un match en prolongation
 * n'est ni bon ni mauvais) → token `info`, jamais un token de dominance ni un
 * token à connotation (success/warning/destructive).
 */
import type { SemanticToken } from '@/lib/accessibility'
import { formatDurationMMSS } from '@/lib/formatters/duration'
import { formatMessage } from '@/lib/i18n/format'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
import type { Locale } from '@/lib/i18n/locale'

/** Clé i18n du libellé court du badge (manifeste match_view, FR + EN). */
export const OVERTIME_LABEL_KEY: MatchViewManifestKey = 'narrative.overtime.label'

/** Clé i18n du tooltip « Prolongation : +M:SS ». */
export const OVERTIME_TOOLTIP_KEY: MatchViewManifestKey = 'narrative.overtime.tooltip'

/** Token sémantique du badge (cf. semantic-tokens.ts) — état neutre-informatif. */
export const OVERTIME_COLOR_TOKEN: SemanticToken = 'info'

/** Libellé localisé du badge (« Prolongation » / « Overtime »). */
export function overtimeLabel(locale: Locale): string {
  return formatMessage(matchViewManifest, OVERTIME_LABEL_KEY, locale)
}

/**
 * Tooltip localisé : « Prolongation : +M:SS » quand le dépassement est connu,
 * sinon le libellé seul (le backend peut flaguer sans seconde exploitable).
 */
export function overtimeTooltip(locale: Locale, overtimeSeconds?: number | null): string {
  if (overtimeSeconds == null || overtimeSeconds <= 0) {
    return overtimeLabel(locale)
  }
  return formatMessage(matchViewManifest, OVERTIME_TOOLTIP_KEY, locale, {
    duration: formatDurationMMSS(overtimeSeconds),
  })
}
