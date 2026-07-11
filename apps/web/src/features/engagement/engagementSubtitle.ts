/**
 * engagementSubtitle — logique de composition du sous-titre du graphe engagement
 * (Match View). Extrait de EngagementMatchSection.tsx (logique hors composant,
 * convention *_logic ; évite aussi le warning react-refresh d'export mixte).
 *
 * Compose : « forme (percentile) — base de l'attendu » + mention discrète de
 * calibration provisoire (chantier F7 E5, DE-8) quand le titre est en calibration
 * provisoire (Halo 5 degraded), masquée quand validée.
 */
import type { EngagementScoreResultAPI } from '@/lib/api/types'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { engagementManifest } from '@/lib/i18n/generated/engagement'

// buildSubtitle compose « forme (percentile) — base de l'attendu » + mention de
// calibration provisoire le cas échéant. undefined si data absente.
export function buildSubtitle(
  data: EngagementScoreResultAPI | undefined,
  locale: ManifestLocale,
): string | undefined {
  if (!data) return undefined
  const base = buildBaseSubtitle(data, locale)
  return withProvisionalMention(base, data, locale)
}

function buildBaseSubtitle(
  data: EngagementScoreResultAPI,
  locale: ManifestLocale,
): string | undefined {
  if (data.expected_basis === 'cold_start') {
    return formatMessage(engagementManifest, 'engagement.narrative.insufficient', locale)
  }
  const basis =
    data.expected_basis === 'bin'
      ? formatMessage(engagementManifest, binSubtitleKey(data.intensity_bin), locale)
      : formatMessage(engagementManifest, 'engagement.expected.global', locale)
  const form = formNarrative(data, locale)
  return form ? `${form} — ${basis}` : basis
}

// withProvisionalMention appose une mention discrète « calibration provisoire »
// (F7 DE-8) quand calibration === 'provisional'. Masquée sinon (validated/absente).
function withProvisionalMention(
  base: string | undefined,
  data: EngagementScoreResultAPI,
  locale: ManifestLocale,
): string | undefined {
  if (data.calibration !== 'provisional') return base
  const mention = formatMessage(engagementManifest, 'engagement.calibration.provisional', locale)
  return base ? `${base} · ${mention}` : mention
}

// binSubtitleKey mappe le libellé de bin (API : calme/standard/chaotique) sur la
// clé manifest de la phrase d'attendu.
type BinSubtitleKey =
  | 'engagement.expected.bin_calme'
  | 'engagement.expected.bin_standard'
  | 'engagement.expected.bin_chaotique'

function binSubtitleKey(bin: string): BinSubtitleKey {
  switch (bin) {
    case 'calme':
      return 'engagement.expected.bin_calme'
    case 'chaotique':
      return 'engagement.expected.bin_chaotique'
    default:
      return 'engagement.expected.bin_standard'
  }
}

// formNarrative rend la phrase de forme (percentile vs habitude). undefined si
// pas de score (historique partiel sans percentile calculable).
function formNarrative(
  data: EngagementScoreResultAPI,
  locale: ManifestLocale,
): string | undefined {
  if (data.engagement_score == null) return undefined
  const percentile = Math.round(data.engagement_score)
  if (percentile > 60) {
    return formatMessage(engagementManifest, 'engagement.narrative.above', locale, { percentile })
  }
  if (percentile < 40) {
    return formatMessage(engagementManifest, 'engagement.narrative.below', locale, { percentile })
  }
  return formatMessage(engagementManifest, 'engagement.narrative.normal', locale, { percentile })
}
