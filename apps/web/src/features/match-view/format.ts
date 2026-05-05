function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function normalizeModeLabel(
  modeLabel: string | null | undefined,
  mapLabel: string | null | undefined,
): string | null {
  if (!modeLabel) return null
  let normalized = modeLabel.trim()
  if (!normalized) return null

  const spacedSeparatorIndex = normalized.indexOf(' : ')
  if (spacedSeparatorIndex > 0) {
    normalized = normalized.slice(0, spacedSeparatorIndex).trim()
  } else {
    const separatorIndex = normalized.lastIndexOf(':')
    if (separatorIndex >= 0 && separatorIndex < normalized.length - 1) {
      normalized = normalized.slice(separatorIndex + 1).trim()
    }
  }

  if (mapLabel?.trim()) {
    const escapedMap = escapeRegExp(mapLabel.trim())
    normalized = normalized.replace(new RegExp(`\\s+(?:on|sur)\\s+${escapedMap}$`, 'i'), '')
  } else {
    normalized = normalized.replace(/\s+(?:on|sur)\s+.+$/i, '')
  }

  normalized = normalized.replace(/\s*-\s*(?:Forge|Ranked)\b/gi, '').trim()
  return normalized || modeLabel.trim()
}

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
