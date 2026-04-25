/**
 * Traductions des libellés de la section « Faits marquants » (Home).
 *
 * Les clés sont émises par le backend via les champs `title_key`, `label_key`,
 * `detail_key` de `HighlightItem` / `HighlightSlide`. Le front résout ici.
 *
 * Les détails localisables acceptent des paramètres via `detail_params`
 * (p.ex. `{count: 3}` pour le pluriel).
 */

export type HighlightLocale = 'fr' | 'en'

type Params = Record<string, string | number>

interface DetailEntry {
  (params: Params): string
}

interface HighlightTextDict {
  titles: Record<string, string>
  labels: Record<string, string>
  details: Record<string, DetailEntry>
  section: {
    title: string
    tooltipIntro: string
  }
}

const FR: HighlightTextDict = {
  titles: {
    'highlight.title.perf_avg': 'Performance',
    'highlight.title.skill_delta_lusr': 'LUSR',
    'highlight.title.skill_delta_csr': 'CSR',
    'highlight.title.best_underdog_win': 'Plus belle victoire',
    'highlight.title.kda_peak': 'Pic FDA récent',
    'highlight.title.mastery': 'Maîtrise',
    'highlight.title.per_minute': 'Stats par min.',
    'highlight.title.volume': 'Volume',
    'highlight.title.serie': 'Séries',
  },
  labels: {
    'highlight.slide.headshots': 'Tirs à la tête',
    'highlight.slide.perfect_kills': 'Frags parfaits',
    'highlight.slide.accuracy': 'Précision',
    'highlight.slide.kills': 'Frags',
    'highlight.slide.deaths': 'Morts',
    'highlight.slide.assists': 'Assistances',
    'highlight.slide.killing_spree_max': 'Folie meurtrière (max)',
    'highlight.slide.win_streak': 'Victoires consécutives',
    'highlight.slide.favorite_map': 'Carte fétiche',
  },
  details: {
    'highlight.detail.win_streak': ({ count }) =>
      `${count} ${Number(count) === 1 ? 'victoire' : 'victoires'} d'affilée`,
    'highlight.detail.favorite_map': ({ wins, losses, wr }) =>
      `${wins}V/${losses}D · ${wr}% victoires`,
    'highlight.detail.volume_wr': ({ wr }) => `Taux de victoire ${wr}%`,
    'highlight.detail.volume_kda_wr': ({ kda, wr }) => `FDA ${kda} · ${wr}% victoires`,
  },
  section: {
    title: 'Faits marquants',
    tooltipIntro:
      'Chiffres calculés sur ta dernière session et jusqu’à 4 sessions récentes avec la même composition (solo/escouade) et la même playlist dominante.',
  },
}

const EN: HighlightTextDict = {
  titles: {
    'highlight.title.perf_avg': 'Avg. performance',
    'highlight.title.skill_delta_lusr': 'LUSR',
    'highlight.title.skill_delta_csr': 'CSR',
    'highlight.title.best_underdog_win': 'Best underdog win',
    'highlight.title.kda_peak': 'Recent KDA peak',
    'highlight.title.mastery': 'Mastery',
    'highlight.title.per_minute': 'Per minute',
    'highlight.title.volume': 'Volume',
    'highlight.title.serie': 'Streak',
  },
  labels: {
    'highlight.slide.headshots': 'Headshots',
    'highlight.slide.perfect_kills': 'Perfect kills',
    'highlight.slide.accuracy': 'Accuracy',
    'highlight.slide.kills': 'Kills',
    'highlight.slide.deaths': 'Deaths',
    'highlight.slide.assists': 'Assists',
    'highlight.slide.killing_spree_max': 'Killing spree (max)',
    'highlight.slide.win_streak': 'Win streak',
    'highlight.slide.favorite_map': 'Favorite map',
  },
  details: {
    'highlight.detail.win_streak': ({ count }) =>
      `${count} ${Number(count) === 1 ? 'win' : 'wins'} in a row`,
    'highlight.detail.favorite_map': ({ wins, losses, wr }) =>
      `${wins}W/${losses}L · ${wr}% win rate`,
    'highlight.detail.volume_wr': ({ wr }) => `Win rate ${wr}%`,
    'highlight.detail.volume_kda_wr': ({ kda, wr }) => `KDA ${kda} · ${wr}% win rate`,
  },
  section: {
    title: 'Highlights',
    tooltipIntro:
      'Figures computed from your last session and up to 4 recent sessions with the same party (solo/squad) and dominant playlist.',
  },
}

const DICTS: Record<HighlightLocale, HighlightTextDict> = { fr: FR, en: EN }

export function normalizeHighlightLocale(locale?: string | null): HighlightLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getHighlightText(locale?: string | null): HighlightTextDict {
  return DICTS[normalizeHighlightLocale(locale)]
}

/** Résout un titre. Fallback : la clé elle-même, pour repérer les manques. */
export function resolveTitle(locale: string | null | undefined, key: string | undefined): string {
  if (!key) return ''
  return getHighlightText(locale).titles[key] ?? key
}

/**
 * Mapping des highlight.slide.* vers les FieldKey canoniques (Phase D plan
 * multi-titres). Une clé absente de cette table reste résolue uniquement par
 * le dict local (pas de FieldKey équivalent).
 */
const SLIDE_TO_FIELD_KEY: Record<string, string> = {
  'highlight.slide.headshots': 'headshot_kills',
  'highlight.slide.accuracy': 'accuracy',
  'highlight.slide.kills': 'kills',
  'highlight.slide.deaths': 'deaths',
  'highlight.slide.assists': 'assists',
}

/**
 * Résout un label de slide.
 *
 * Si fieldMappings est fourni et que la clé highlight a un FieldKey canonique
 * équivalent, le libellé du backend TOML prime. Sinon fallback sur le dict
 * local (rétrocompatibilité quand l'endpoint /field-mappings est absent).
 */
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
  return getHighlightText(locale).labels[key] ?? key
}

/**
 * Largeur (en sous-unités d'une grille fine de 20 colonnes) allouée à chaque tuile.
 * Total des spans = 20 (8 tuiles). Granularité fine pour permettre des ajustements
 * de ±1 sous-unité sans toucher aux autres tuiles.
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

/** Résout un détail paramétré. Retourne '' si aucune clé. */
export function resolveDetail(
  locale: string | null | undefined,
  key: string | undefined,
  params?: Params,
): string {
  if (!key) return ''
  const entry = getHighlightText(locale).details[key]
  if (!entry) return key
  return entry(params ?? {})
}
