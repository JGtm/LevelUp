#!/usr/bin/env node
/**
 * Triage automatique des issues feedback via Claude Haiku 4.5.
 *
 * Trigger : .github/workflows/triage-feedback.yml (issues.opened, label
 * `feedback`). Affine la sévérité/zone, suggère causes/workarounds,
 * commente l'issue. Fail-safe : sortie 0 sur erreur, label
 * `triage:needs-review` ou `triage:parse-error` selon le cas.
 */
import Anthropic from '@anthropic-ai/sdk'
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const PROMPT_PATH = join(__dirname, 'triage-feedback-prompt.md')

const {
  ANTHROPIC_API_KEY,
  GH_TOKEN,
  ISSUE_NUMBER,
  ISSUE_TITLE = '',
  ISSUE_BODY = '',
  REPO,
} = process.env

if (!ANTHROPIC_API_KEY) fail('ANTHROPIC_API_KEY manquant')
if (!GH_TOKEN) fail('GH_TOKEN manquant')
if (!ISSUE_NUMBER) fail('ISSUE_NUMBER manquant')
if (!REPO) fail('REPO manquant')

const issueNumber = String(ISSUE_NUMBER)
log(`received issue #${issueNumber} "${truncateForLog(ISSUE_TITLE)}"`)

const recentIssues = listRecentIssues()
log(`fetched ${recentIssues.length} recent open issues for context`)

const systemPrompt = readFileSync(PROMPT_PATH, 'utf-8')
const userMessage = renderUserMessage(ISSUE_TITLE, ISSUE_BODY, recentIssues)

const client = new Anthropic({ apiKey: ANTHROPIC_API_KEY })
let response
try {
  log('calling claude-haiku-4-5')
  response = await client.messages.create({
    model: 'claude-haiku-4-5-20251001',
    max_tokens: 1024,
    system: systemPrompt,
    messages: [{ role: 'user', content: userMessage }],
  })
} catch (err) {
  log(`ERROR: claude API failed: ${err?.message ?? err}`)
  applyLabel('triage:needs-review')
  process.exit(0)
}

const usage = response.usage
log(`usage: input_tokens=${usage?.input_tokens} output_tokens=${usage?.output_tokens}`)

const rawText = response.content
  .filter((block) => block.type === 'text')
  .map((block) => block.text)
  .join('')

let parsed
try {
  parsed = JSON.parse(extractJsonObject(rawText))
} catch (err) {
  log(`ERROR: JSON parse failed: ${err?.message ?? err}`)
  applyLabel('triage:parse-error')
  process.exit(0)
}

log(
  `parsed JSON: severity=${parsed.severity_refined} area=${parsed.area_refined} is_actionable=${parsed.is_actionable}`,
)

const labelsToAdd = ['triage:claude-analyzed']
if (parsed.severity_refined) labelsToAdd.push(`severity:${parsed.severity_refined}`)
if (parsed.area_refined) labelsToAdd.push(`area:${parsed.area_refined}`)

if (!parsed.is_actionable) {
  applyLabel('triage:needs-review')
  log('issue marked needs-review (not actionable)')
  process.exit(0)
}

applyLabel(labelsToAdd.join(','))
log(`applied labels: ${labelsToAdd.join(', ')}`)

const comment = renderComment(parsed)
postComment(comment)
log(`posted comment (${comment.length} chars)`)

// ---------------------------------------------------------------------------

export function renderUserMessage(title, body, recentIssues) {
  const recentBlock = recentIssues
    .map((it) => `- #${it.number} ${it.title}`)
    .join('\n')
  return [
    '## Issue à trier',
    `**Titre** : ${title}`,
    '',
    '**Body** :',
    body,
    '',
    '## Issues récentes (contexte pour doublons)',
    recentBlock || '_(aucune)_',
  ].join('\n')
}

export function renderComment(parsed) {
  const lines = ['## 🤖 Triage automatique (Claude Haiku 4.5)']
  if (parsed.summary_one_liner) {
    lines.push('', `**Résumé** : ${parsed.summary_one_liner}`)
  }
  if (parsed.probable_cause) {
    lines.push('', `**Cause probable** : ${parsed.probable_cause}`)
  }
  if (Array.isArray(parsed.suggestions) && parsed.suggestions.length > 0) {
    lines.push('', '**Suggestions** :')
    for (const s of parsed.suggestions) lines.push(`- ${s}`)
  }
  if (
    Array.isArray(parsed.similar_internal_issues) &&
    parsed.similar_internal_issues.length > 0
  ) {
    lines.push('', '**Issues internes potentiellement liées** :')
    for (const it of parsed.similar_internal_issues) {
      lines.push(`- #${it.number} — ${it.reason}`)
    }
  }
  lines.push('', '_(Ce commentaire est généré automatiquement. Edit manuellement les labels si besoin.)_')
  return lines.join('\n')
}

export function extractJsonObject(text) {
  const trimmed = text.trim()
  if (trimmed.startsWith('{') && trimmed.endsWith('}')) return trimmed
  // Fallback : extraire le premier bloc JSON contenant `{...}`
  const start = trimmed.indexOf('{')
  const end = trimmed.lastIndexOf('}')
  if (start !== -1 && end !== -1 && end > start) {
    return trimmed.slice(start, end + 1)
  }
  throw new Error('no JSON object found')
}

function listRecentIssues() {
  try {
    const raw = execFileSync(
      'gh',
      ['issue', 'list', '--state', 'open', '--limit', '50', '--json', 'number,title'],
      { env: process.env, encoding: 'utf-8' },
    )
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr : []
  } catch (err) {
    log(`WARN: could not list recent issues: ${err?.message ?? err}`)
    return []
  }
}

function applyLabel(labels) {
  try {
    execFileSync(
      'gh',
      ['issue', 'edit', issueNumber, '--add-label', labels, '--repo', REPO],
      { env: process.env, stdio: 'inherit' },
    )
  } catch (err) {
    log(`WARN: gh issue edit failed: ${err?.message ?? err}`)
  }
}

function postComment(body) {
  try {
    execFileSync(
      'gh',
      ['issue', 'comment', issueNumber, '--body', body, '--repo', REPO],
      { env: process.env, stdio: 'inherit' },
    )
  } catch (err) {
    log(`WARN: gh issue comment failed: ${err?.message ?? err}`)
  }
}

function log(msg) {
  console.log(`[triage] ${msg}`)
}

function fail(msg) {
  console.error(`[triage] FATAL: ${msg}`)
  process.exit(1)
}

function truncateForLog(s) {
  return s.length > 80 ? s.slice(0, 77) + '...' : s
}
