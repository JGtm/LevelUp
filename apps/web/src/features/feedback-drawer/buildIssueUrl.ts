/**
 * Construit l'URL GitHub Issues préremplie à partir du contexte feedback.
 *
 * Pratique : URLs GitHub Issues fonctionnent jusqu'à ~8000 chars en pratique.
 * On vise un body ≤ 7000 chars avec **troncature progressive** :
 *   1. Erreurs console (la plus volatile)
 *   2. Filtres actifs
 *   3. Description (en dernier recours)
 * On annote `…[truncated]` à chaque section tronquée.
 */
import type { Classification, FeedbackType } from './classifyFeedback'
import type { FeedbackContext } from './collectContext'
import { GITHUB_ISSUES_URL, GITHUB_REPO } from '@/lib/appLinks'
import { formatMessage } from '@/lib/i18n/format'
import { feedbackDrawerManifest, type FeedbackDrawerManifestKey } from '@/lib/i18n/generated/feedback_drawer'
import type { Locale } from '@/lib/i18n/locale'

// Slug du dépôt et URL des issues : source unique dans lib/appLinks.
const BASE_URL = `${GITHUB_ISSUES_URL}/new`
const MAX_BODY_LENGTH = 7000
const TRUNCATED_MARKER = '…[truncated]'

// Labels GitHub (métadonnées machine, pas de contenu utilisateur) : restent en
// anglais quelle que soit la locale — un label GitHub n'est pas traduit.
const TYPE_LABEL_MAP: Record<FeedbackType, string> = {
  bug: 'bug',
  enhancement: 'enhancement',
  question: 'question',
}

const TYPE_PREFIX_KEY_MAP: Record<FeedbackType, FeedbackDrawerManifestKey> = {
  bug: 'feedback_drawer.issue_title.bug_prefix',
  enhancement: 'feedback_drawer.issue_title.enhancement_prefix',
  question: 'feedback_drawer.issue_title.question_prefix',
}

export interface BuildIssueUrlInput {
  title: string
  description: string
  context: FeedbackContext
  classification: Classification
  /** Locale du corps de l'issue GitHub généré — bilan fork 2026-09-11 pt 6. */
  locale: Locale
}

export interface BuildIssueUrlResult {
  url: string
  body: string
  wasTruncated: boolean
}

function t(locale: Locale, key: FeedbackDrawerManifestKey): string {
  return formatMessage(feedbackDrawerManifest, key, locale)
}

export function buildIssueUrl(input: BuildIssueUrlInput): BuildIssueUrlResult {
  const fullTitle = t(input.locale, TYPE_PREFIX_KEY_MAP[input.classification.type]) + input.title
  const labels = [
    'feedback',
    TYPE_LABEL_MAP[input.classification.type],
    `severity:${input.classification.severity}`,
    `area:${input.classification.area}`,
  ].join(',')

  const { body, wasTruncated } = buildBody(input)
  const params = new URLSearchParams({
    labels,
    title: fullTitle,
    body,
  })
  return { url: `${BASE_URL}?${params.toString()}`, body, wasTruncated }
}

interface BodySections {
  description: string
  context: string
  environment: string
  filters: string
  classification: string
  consoleErrors: string
  failedRequests: string
  footer: string
}

function buildBody(input: BuildIssueUrlInput): { body: string; wasTruncated: boolean } {
  const sections = renderSections(input)
  let body = composeBody(input.locale, sections)

  if (body.length <= MAX_BODY_LENGTH) return { body, wasTruncated: false }

  // Troncature progressive : erreurs console → filtres → description
  sections.consoleErrors = TRUNCATED_MARKER
  sections.failedRequests = TRUNCATED_MARKER
  body = composeBody(input.locale, sections)

  if (body.length > MAX_BODY_LENGTH) {
    sections.filters = TRUNCATED_MARKER
    body = composeBody(input.locale, sections)
  }
  if (body.length > MAX_BODY_LENGTH) {
    const remaining = MAX_BODY_LENGTH - (body.length - sections.description.length)
    const safeBudget = Math.max(200, remaining - TRUNCATED_MARKER.length - 4)
    sections.description = sections.description.slice(0, safeBudget) + '\n' + TRUNCATED_MARKER
    body = composeBody(input.locale, sections)
  }
  return { body, wasTruncated: true }
}

function composeBody(locale: Locale, s: BodySections): string {
  return [
    t(locale, 'feedback_drawer.issue_body.heading_description'),
    s.description,
    '',
    '---',
    '',
    t(locale, 'feedback_drawer.issue_body.heading_context'),
    s.context,
    '',
    t(locale, 'feedback_drawer.issue_body.heading_environment'),
    s.environment,
    '',
    t(locale, 'feedback_drawer.issue_body.heading_filters'),
    s.filters,
    '',
    t(locale, 'feedback_drawer.issue_body.heading_classification'),
    s.classification,
    '',
    t(locale, 'feedback_drawer.issue_body.heading_console_errors'),
    s.consoleErrors,
    '',
    t(locale, 'feedback_drawer.issue_body.heading_failed_requests'),
    s.failedRequests,
    '',
    '---',
    s.footer,
  ].join('\n')
}

function renderSections(input: BuildIssueUrlInput): BodySections {
  const { context, classification, locale } = input
  const description = input.description.trim() || t(locale, 'feedback_drawer.issue_body.no_description')
  const na = t(locale, 'feedback_drawer.issue_body.not_applicable')

  return {
    description,
    context: [
      `- ${t(locale, 'feedback_drawer.issue_body.label_url')} : ${context.browser.url}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_title')} : ${context.shell.titleSlug}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_player')} : ${context.shell.playerSlug ?? na}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_locale')} : ${context.browser.locale}  ·  ${t(locale, 'feedback_drawer.issue_body.label_theme')} : ${context.browser.theme}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_timestamp')} : ${context.browser.timestampIso}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_focused_element')} : ${context.browser.focusedElement ?? na}`,
    ].join('\n'),
    environment: [
      `- ${t(locale, 'feedback_drawer.issue_body.label_app_version')} : ${context.shell.appVersion ?? 'unknown'}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_user_agent')} : ${context.browser.userAgent}`,
      `- ${t(locale, 'feedback_drawer.issue_body.label_viewport')} : ${context.browser.viewportWidth} × ${context.browser.viewportHeight}`,
    ].join('\n'),
    filters: renderFilters(locale, context.filters),
    classification: `- ${t(locale, 'feedback_drawer.issue_body.label_classification_type')} : ${classification.type}  ·  ${t(locale, 'feedback_drawer.issue_body.label_classification_severity')} : ${classification.severity}  ·  ${t(locale, 'feedback_drawer.issue_body.label_classification_area')} : ${classification.area}`,
    consoleErrors: renderConsoleErrors(locale, context.console),
    failedRequests: renderFailedRequests(locale, context.failedRequests),
    footer:
      t(locale, 'feedback_drawer.issue_body.footer_generated') + '\n' +
      t(locale, 'feedback_drawer.issue_body.footer_auto_analysis'),
  }
}

function renderFilters(locale: Locale, filters: FeedbackContext['filters']): string {
  if (!filters) return t(locale, 'feedback_drawer.issue_body.no_active_filters')
  const lines = [`- ${t(locale, 'feedback_drawer.issue_body.label_filter_mode')} : ${filters.filter_mode}`]
  if (filters.period?.start_date || filters.period?.end_date) {
    lines.push(
      `- ${t(locale, 'feedback_drawer.issue_body.label_period')} : ${filters.period.start_date ?? '?'} → ${filters.period.end_date ?? '?'}`,
    )
  }
  if (filters.cascade?.modes?.length) {
    lines.push(`- ${t(locale, 'feedback_drawer.issue_body.label_modes')} : ${filters.cascade.modes.join(', ')}`)
  }
  if (filters.cascade?.maps?.length) {
    lines.push(`- ${t(locale, 'feedback_drawer.issue_body.label_maps')} : ${filters.cascade.maps.join(', ')}`)
  }
  if (filters.cascade?.playlists?.length) {
    lines.push(`- ${t(locale, 'feedback_drawer.issue_body.label_playlists')} : ${filters.cascade.playlists.join(', ')}`)
  }
  if (filters.sessions?.picked_sessions?.length) {
    lines.push(
      `- ${t(locale, 'feedback_drawer.issue_body.label_sessions')} : ${filters.sessions.picked_sessions.length} ${t(locale, 'feedback_drawer.issue_body.sessions_selected')}`,
    )
  }
  return lines.length === 1 ? t(locale, 'feedback_drawer.issue_body.default_filters') : lines.join('\n')
}

function renderConsoleErrors(locale: Locale, entries: FeedbackContext['console']): string {
  if (!entries.length) return t(locale, 'feedback_drawer.issue_body.no_console_errors')
  const lines = entries.map((e) => {
    const ts = formatTime(e.timestamp)
    const head = `[${e.level.toUpperCase()} ${ts}] ${e.message}`
    return e.stack ? `${head}\n${e.stack.split('\n').slice(0, 3).join('\n')}` : head
  })
  return '```js\n' + lines.join('\n') + '\n```'
}

function renderFailedRequests(locale: Locale, reqs: FeedbackContext['failedRequests']): string {
  if (!reqs.length) return t(locale, 'feedback_drawer.issue_body.no_failed_requests')
  const lines = reqs.map((r) => `${r.method} ${r.url} → ${r.status} (${formatTime(r.timestamp)})`)
  return '```\n' + lines.join('\n') + '\n```'
}

function formatTime(ts: number): string {
  const d = new Date(ts)
  return d.toISOString().slice(11, 19)
}

// ---------------------------------------------------------------------------
// GitHub search helper (pour useSimilarIssues)
// ---------------------------------------------------------------------------

const SEARCH_RESERVED = /[:+"()/\\]/g

/**
 * Sanitize un titre user pour l'injecter dans `?q=` de la GitHub Search API.
 * Supprime les opérateurs réservés (`:`, `+`, `"`, `(`, `)`, `/`) qui casseraient
 * la query, puis collapse les espaces multiples.
 */
export function escapeSearchQuery(title: string): string {
  return title.replace(SEARCH_RESERVED, ' ').replace(/\s+/g, ' ').trim()
}

/**
 * Construit l'URL de la GitHub Search API pour les issues similaires.
 * Repo public → pas de token nécessaire. Limite 60 req/h/IP.
 */
export function buildSearchIssuesUrl(title: string): string {
  const sanitized = escapeSearchQuery(title)
  const q = `${sanitized} is:issue repo:${GITHUB_REPO}`
  return `https://api.github.com/search/issues?q=${encodeURIComponent(q)}&per_page=3`
}
