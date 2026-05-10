import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { SemanticToken } from '@/lib/accessibility'

export const LUSR_GROUP_TOKENS: Record<string, SemanticToken> = {
  arena_slayer:   'compare-a',
  arena_objectif: 'compare-b',
  btb:            'divergent-pos',
  chaos:          'narrative-humiliation',
}

export function lusrChainLabel(group: string, locale: ManifestLocale): string {
  const key = `career.lusr.chain.${group}` as keyof typeof careerManifest
  return careerManifest[key]?.[locale] ?? group
}
