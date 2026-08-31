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

// Slug du dépôt et URL des issues : source unique dans lib/appLinks.
const BASE_URL = `${GITHUB_ISSUES_URL}/new`
const MAX_BODY_LENGTH = 7000
const TRUNCATED_MARKER = '…[truncated]'

const TYPE_LABEL_MAP: Record<FeedbackType, string> = {
  bug: 'bug',
  enhancement: 'enhancement',
  question: 'question',
}

const TYPE_PREFIX_MAP: Record<FeedbackType, string> = {
  bug: '[Bug] ',
  enhancement: '[Idée] ',
  question: '[?] ',
}

export interface BuildIssueUrlInput {
  title: string
  description: string
  context: FeedbackContext
  classification: Classification
}

export interface BuildIssueUrlResult {
  url: string
  body: string
  wasTruncated: boolean
}

export function buildIssueUrl(input: BuildIssueUrlInput): BuildIssueUrlResult {
  const fullTitle = TYPE_PREFIX_MAP[input.classification.type] + input.title
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
  let body = composeBody(sections)

  if (body.length <= MAX_BODY_LENGTH) return { body, wasTruncated: false }

  // Troncature progressive : erreurs console → filtres → description
  sections.consoleErrors = TRUNCATED_MARKER
  sections.failedRequests = TRUNCATED_MARKER
  body = composeBody(sections)

  if (body.length > MAX_BODY_LENGTH) {
    sections.filters = TRUNCATED_MARKER
    body = composeBody(sections)
  }
  if (body.length > MAX_BODY_LENGTH) {
    const remaining = MAX_BODY_LENGTH - (body.length - sections.description.length)
    const safeBudget = Math.max(200, remaining - TRUNCATED_MARKER.length - 4)
    sections.description = sections.description.slice(0, safeBudget) + '\n' + TRUNCATED_MARKER
    body = composeBody(sections)
  }
  return { body, wasTruncated: true }
}

function composeBody(s: BodySections): string {
  return [
    '## Description',
    s.description,
    '',
    '---',
    '',
    '## Contexte',
    s.context,
    '',
    '## Environnement client',
    s.environment,
    '',
    '## Filtres actifs',
    s.filters,
    '',
    '## Classification heuristique (front)',
    s.classification,
    '',
    '## Erreurs console récentes',
    s.consoleErrors,
    '',
    '## Requêtes échouées récentes',
    s.failedRequests,
    '',
    '---',
    s.footer,
  ].join('\n')
}

function renderSections(input: BuildIssueUrlInput): BodySections {
  const { context, classification } = input
  const description = input.description.trim() || '_(aucune description fournie)_'

  return {
    description,
    context: [
      `- **URL** : ${context.browser.url}`,
      `- **Titre** : ${context.shell.titleSlug}`,
      `- **Joueur** : ${context.shell.playerSlug ?? '_n/a_'}`,
      `- **Locale** : ${context.browser.locale}  ·  **Thème** : ${context.browser.theme}`,
      `- **Timestamp** : ${context.browser.timestampIso}`,
      `- **Élément focus** : ${context.browser.focusedElement ?? '_n/a_'}`,
    ].join('\n'),
    environment: [
      `- **Version app** : ${context.shell.appVersion ?? 'unknown'}`,
      `- **User-Agent** : ${context.browser.userAgent}`,
      `- **Viewport** : ${context.browser.viewportWidth} × ${context.browser.viewportHeight}`,
    ].join('\n'),
    filters: renderFilters(context.filters),
    classification: `- **Type** : ${classification.type}  ·  **Sévérité** : ${classification.severity}  ·  **Zone** : ${classification.area}`,
    consoleErrors: renderConsoleErrors(context.console),
    failedRequests: renderFailedRequests(context.failedRequests),
    footer:
      '*Auto-généré par le drawer feedback — LevelUp web*\n' +
      "*Une analyse automatique sera ajoutée en commentaire dans quelques secondes.*",
  }
}

function renderFilters(filters: FeedbackContext['filters']): string {
  if (!filters) return '_(aucun filtre actif)_'
  const lines = [`- **Mode filtre** : ${filters.filter_mode}`]
  if (filters.period?.start_date || filters.period?.end_date) {
    lines.push(
      `- **Période** : ${filters.period.start_date ?? '?'} → ${filters.period.end_date ?? '?'}`,
    )
  }
  if (filters.cascade?.modes?.length) {
    lines.push(`- **Modes** : ${filters.cascade.modes.join(', ')}`)
  }
  if (filters.cascade?.maps?.length) {
    lines.push(`- **Maps** : ${filters.cascade.maps.join(', ')}`)
  }
  if (filters.cascade?.playlists?.length) {
    lines.push(`- **Playlists** : ${filters.cascade.playlists.join(', ')}`)
  }
  if (filters.sessions?.picked_sessions?.length) {
    lines.push(`- **Sessions** : ${filters.sessions.picked_sessions.length} sélectionnée(s)`)
  }
  return lines.length === 1 ? '_(filtres par défaut)_' : lines.join('\n')
}

function renderConsoleErrors(entries: FeedbackContext['console']): string {
  if (!entries.length) return '_(aucune erreur console capturée)_'
  const lines = entries.map((e) => {
    const ts = formatTime(e.timestamp)
    const head = `[${e.level.toUpperCase()} ${ts}] ${e.message}`
    return e.stack ? `${head}\n${e.stack.split('\n').slice(0, 3).join('\n')}` : head
  })
  return '```js\n' + lines.join('\n') + '\n```'
}

function renderFailedRequests(reqs: FeedbackContext['failedRequests']): string {
  if (!reqs.length) return '_(aucune requête échouée capturée)_'
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
