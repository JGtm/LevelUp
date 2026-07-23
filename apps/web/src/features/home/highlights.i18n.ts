/**
 * highlights.i18n.ts — adapter mince entre `home.toml` et la résolution des
 * clés `title_key` / `label_key` / `detail_key` envoyées par le backend dans
 * `HighlightItem` / `HighlightSlide`.
 *
 * Phase 3 P3.C : source de vérité = `lib/i18n/manifests/home.toml` sections
 * `home.highlights.*`. Le mapping clé backend → clé manifest est défini ici
 * (les clés backend `highlight.title.*` / `highlight.slide.*` sont mappées
 * vers `home.highlights.title.*` / `home.highlights.label.*` etc.).
 *
 * Les détails paramétrés (win_streak, favorite_map, volume_*) utilisent ICU
 * (pluriels + interpolation) pour reproduire les anciennes lambdas.
 */
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import type { Locale } from '@/lib/i18n/locale'

type Params = Record<string, string | number>

interface HighlightTextDict {
  section: {
    title: string
    tooltipIntro: string
  }
}

export function normalizeHighlightLocale(locale?: string | null): Locale {
  return locale === 'en' ? 'en' : 'fr'
}

function t(
  loc: Locale,
  key: HomeManifestKey,
  values?: Record<string, string | number>,
): string {
  return formatMessage(homeManifest, key, loc, values)
}

export function getHighlightText(locale?: string | null): HighlightTextDict {
  const loc = normalizeHighlightLocale(locale)
  return {
    section: {
      title: t(loc, 'home.highlights.section_title'),
      tooltipIntro: t(loc, 'home.highlights.section_tooltip'),
    },
  }
}

// ─── Mapping backend keys → manifest keys ─────────────────────────────────

const TITLE_KEY_MAP: Record<string, HomeManifestKey> = {
  'highlight.title.perf_avg': 'home.highlights.title.perf_avg',
  'highlight.title.skill_delta_lusr': 'home.highlights.title.skill_delta_lusr',
  'highlight.title.skill_delta_csr': 'home.highlights.title.skill_delta_csr',
  'highlight.title.best_underdog_win': 'home.highlights.title.best_underdog_win',
  'highlight.title.kda_peak': 'home.highlights.title.kda_peak',
  'highlight.title.mastery': 'home.highlights.title.mastery',
  'highlight.title.per_minute': 'home.highlights.title.per_minute',
  'highlight.title.volume': 'home.highlights.title.volume',
  'highlight.title.serie': 'home.highlights.title.serie',
}

const LABEL_KEY_MAP: Record<string, HomeManifestKey> = {
  'highlight.slide.headshots': 'home.highlights.label.headshots',
  'highlight.slide.perfect_kills': 'home.highlights.label.perfect_kills',
  'highlight.slide.accuracy': 'home.highlights.label.accuracy',
  'highlight.slide.kills': 'home.highlights.label.kills',
  'highlight.slide.deaths': 'home.highlights.label.deaths',
  'highlight.slide.assists': 'home.highlights.label.assists',
  'highlight.slide.killing_spree_max': 'home.highlights.label.killing_spree_max',
  'highlight.slide.win_streak': 'home.highlights.label.win_streak',
  'highlight.slide.favorite_map': 'home.highlights.label.favorite_map',
}

const DETAIL_KEY_MAP: Record<string, HomeManifestKey> = {
  'highlight.detail.win_streak': 'home.highlights.detail.win_streak',
  'highlight.detail.favorite_map': 'home.highlights.detail.favorite_map',
  'highlight.detail.volume_wr': 'home.highlights.detail.volume_wr',
  'highlight.detail.volume_kda_wr': 'home.highlights.detail.volume_kda_wr',
}

/** Résout un titre. Fallback : la clé brute pour repérer les manques. */
export function resolveTitle(locale: string | null | undefined, key: string | undefined): string {
  if (!key) return ''
  const mapped = TITLE_KEY_MAP[key]
  if (!mapped) return key
  return t(normalizeHighlightLocale(locale), mapped)
}

/**
 * Mapping highlight.slide.* → FieldKey canonique. Si le backend
 * `useFieldMappings` fournit le label, il prime ; sinon fallback manifest.
 */
const SLIDE_TO_FIELD_KEY: Record<string, string> = {
  'highlight.slide.headshots': 'headshot_kills',
  'highlight.slide.accuracy': 'accuracy',
  'highlight.slide.kills': 'kills',
  'highlight.slide.deaths': 'deaths',
  'highlight.slide.assists': 'assists',
}

export function resolveLabel(
  locale: string | null | undefined,
  key: string | undefined,
  fieldMappings?: { fields: Record<string, { label: string }> },
): string {
  if (!key) return ''
  const fieldKey = SLIDE_TO_FIELD_KEY[key]
  if (fieldKey && fieldMappings?.fields[fieldKey]) {
    return fieldMappings.fields[fieldKey].label
  }
  const mapped = LABEL_KEY_MAP[key]
  if (!mapped) return key
  return t(normalizeHighlightLocale(locale), mapped)
}

/**
 * Largeur en sous-unités de la grille fine (20 cols) par tuile.
 * Conservé inline car c'est de la config de layout, pas un libellé i18n.
 */
const COL_SPANS: Record<string, number> = {
  'highlight.title.perf_avg': 2,
  'highlight.title.skill_delta_lusr': 2,
  'highlight.title.skill_delta_csr': 2,
  'highlight.title.best_underdog_win': 3,
  'highlight.title.kda_peak': 3,
  'highlight.title.mastery': 2,
  'highlight.title.per_minute': 2,
  'highlight.title.volume': 3,
  'highlight.title.serie': 3,
}

export function resolveColSpan(titleKey: string | undefined): number {
  if (!titleKey) return 1
  return COL_SPANS[titleKey] ?? 1
}

const UNIT_KEY_MAP: Record<string, HomeManifestKey> = {
  'highlight.title.volume': 'home.highlights.unit.matches',
}

export function resolveUnit(locale: string | null | undefined, titleKey: string | undefined): string {
  if (!titleKey) return ''
  const mapped = UNIT_KEY_MAP[titleKey]
  if (!mapped) return ''
  return t(normalizeHighlightLocale(locale), mapped)
}

export function resolveDetail(
  locale: string | null | undefined,
  key: string | undefined,
  params?: Params,
): string {
  if (!key) return ''
  const mapped = DETAIL_KEY_MAP[key]
  if (!mapped) return key
  return t(normalizeHighlightLocale(locale), mapped, params)
}
