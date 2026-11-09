import { normalizeModeLabel } from '@/lib/halo/modeLabel'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { Locale } from '@/lib/i18n/locale'

export function buildMatchHeadingStr(
  mapUI: string | null | undefined,
  modeUI: string | null | undefined,
  locale: Locale,
): string {
  const normalizedMode = normalizeModeLabel(modeUI, mapUI)
  const connector = formatMessage(commonManifest, 'common.match_card.heading_connector', locale)
  if (normalizedMode && mapUI) {
    return `${normalizedMode} ${connector} ${mapUI}`
  }
  return normalizedMode ?? mapUI ?? ''
}
