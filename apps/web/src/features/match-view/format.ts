import { normalizeModeLabel } from '@/lib/halo/modeLabel'
import type { Locale } from '@/lib/i18n/locale'

export function buildMatchHeadingStr(
  mapUI: string | null | undefined,
  modeUI: string | null | undefined,
  locale: Locale,
): string {
  const normalizedMode = normalizeModeLabel(modeUI, mapUI)
  const connector = locale === 'en' ? 'on' : 'sur'
  if (normalizedMode && mapUI) {
    return `${normalizedMode} ${connector} ${mapUI}`
  }
  return normalizedMode ?? mapUI ?? ''
}
