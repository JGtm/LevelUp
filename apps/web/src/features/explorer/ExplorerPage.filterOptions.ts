/**
 * ExplorerPage — helpers de construction des options MultiSelectFilter.
 *
 * Découpé depuis ExplorerPage.tsx (audit #6 god-file split).
 * Fonction unique buildFilterOptions() qui produit les 7 listes d'options +
 * la map de counts squad — toutes dérivées du summary backend.
 */
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { SKILL_TIER_VALUES } from '@/lib/skillTiers'
import type { ExplorerMatchesQueryResponse, LabelValue } from '@/lib/api/types'
import type { MultiSelectOption } from './MultiSelectFilter'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

type Translator = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

// Mappe les counts backend (LabelValue[]) sur les options frontend (qui portent
// labels i18n + swatch). Si pas de backend data, count reste undefined (pas de
// grayout). Match par `value` exact.
function withCounts(opts: MultiSelectOption[], backend?: LabelValue[]): MultiSelectOption[] {
  if (!backend) return opts
  const map = new Map(backend.map((b) => [b.value, b.count]))
  return opts.map((o) => ({ ...o, count: map.get(o.value) ?? 0 }))
}

export interface ExplorerFilterOptions {
  expTypeOptions: MultiSelectOption[]
  playlistOptions: MultiSelectOption[]
  modeOptions: MultiSelectOption[]
  mapOptions: MultiSelectOption[]
  perfTierOptions: MultiSelectOption[]
  outcomeOptions: MultiSelectOption[]
  skillTierOptions: MultiSelectOption[]
  squadCountByValue: Map<string, number>
}

export function buildExplorerFilterOptions(
  summary: ExplorerMatchesQueryResponse['summary'] | undefined,
  t: Translator,
): ExplorerFilterOptions {
  return {
    // available_experience_types est désormais LabelValue[] : Label localisé backend
    // (EN sous UI EN), Value FR canonique = clé de filtre intacte (renvoyée telle
    // quelle dans experience_types + cascade rankedContext FR-hardcodée). GH6-1.
    expTypeOptions: (summary?.available_experience_types ?? []).map(
      (o) => ({ value: o.value, label: o.label }),
    ),
    playlistOptions: (summary?.available_playlists ?? []).map((v) => ({
      value: v,
      label: v,
    })),
    modeOptions: (summary?.available_modes ?? []).map((v) => ({
      value: v,
      label: v,
    })),
    mapOptions: (summary?.available_maps ?? []).map((v) => ({
      value: v,
      label: v,
    })),

    perfTierOptions: withCounts(
      [
        { value: '1', label: t('explorer.filters.perf_tier_excellent'), swatch: tokenCssVar('perf-tier-1' as SemanticToken) },
        { value: '2', label: t('explorer.filters.perf_tier_bon'), swatch: tokenCssVar('perf-tier-2' as SemanticToken) },
        { value: '3', label: t('explorer.filters.perf_tier_correct'), swatch: tokenCssVar('perf-tier-3' as SemanticToken) },
        { value: '4', label: t('explorer.filters.perf_tier_faible'), swatch: tokenCssVar('perf-tier-4' as SemanticToken) },
        { value: '5', label: t('explorer.filters.perf_tier_mauvais'), swatch: tokenCssVar('perf-tier-5' as SemanticToken) },
      ],
      summary?.available_perf_tiers ?? undefined,
    ),

    outcomeOptions: withCounts(
      [
        { value: '2', label: t('explorer.filters.outcome_win'), swatch: tokenCssVar('outcome-win' as SemanticToken) },
        { value: '3', label: t('explorer.filters.outcome_loss'), swatch: tokenCssVar('outcome-loss' as SemanticToken) },
        { value: '1', label: t('explorer.filters.outcome_tie'), swatch: tokenCssVar('outcome-draw' as SemanticToken) },
      ],
      summary?.available_outcomes ?? undefined,
    ),

    skillTierOptions: withCounts(
      [
        { value: SKILL_TIER_VALUES[0], label: t('explorer.filters.skill_tier_bronze') },
        { value: SKILL_TIER_VALUES[1], label: t('explorer.filters.skill_tier_silver') },
        { value: SKILL_TIER_VALUES[2], label: t('explorer.filters.skill_tier_gold') },
        { value: SKILL_TIER_VALUES[3], label: t('explorer.filters.skill_tier_platinum') },
        { value: SKILL_TIER_VALUES[4], label: t('explorer.filters.skill_tier_diamond') },
        { value: SKILL_TIER_VALUES[5], label: t('explorer.filters.skill_tier_onyx') },
      ],
      summary?.available_skill_tiers ?? undefined,
    ),

    // Count pour le single-select squad scope — interpolé dans les <option>
    // labels et désactive celles à count=0.
    squadCountByValue: new Map(
      (summary?.available_squad_scopes ?? []).map((b) => [b.value, b.count]),
    ),
  }
}
