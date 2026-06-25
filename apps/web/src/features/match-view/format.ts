import { normalizeModeLabel } from '@/lib/halo/modeLabel'

export function buildMatchHeadingStr(
  mapUI: string | null | undefined,
  modeUI: string | null | undefined,
  locale: 'fr' | 'en',
): string {
  const normalizedMode = normalizeModeLabel(modeUI, mapUI)
  const connector = locale === 'en' ? 'on' : 'sur'
  if (normalizedMode && mapUI) {
    return `${normalizedMode} ${connector} ${mapUI}`
  }
  return normalizedMode ?? mapUI ?? ''
}
