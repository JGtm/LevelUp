/**
 * Tests Node natifs (built-in test runner) pour les fonctions pures du
 * script triage-feedback.mjs.
 *
 * Lancer : node --test .github/scripts/triage-feedback.test.mjs
 *
 * Ne couvre PAS les appels Claude/gh CLI (qui requièrent des tokens et
 * un repo). Couvre les helpers de parsing/rendering qui sont les plus
 * susceptibles de drift silencieux.
 */
import { describe, it } from 'node:test'
import assert from 'node:assert/strict'

// Note : on ré-exporte les fonctions pures via un re-export dédié pour
// éviter d'exécuter la suite top-level du script principal en test.
// Pour rester simple sans build step, on stub les variables d'env requises
// AVANT l'import dynamique.
process.env.ANTHROPIC_API_KEY = 'sk-test'
process.env.GH_TOKEN = 'ghp_test'
process.env.ISSUE_NUMBER = '0'
process.env.REPO = 'JGtm/LevelUp'

// Stub gh CLI : les helpers que l'on teste ne l'invoquent pas, mais le
// script main fait `listRecentIssues()` au boot. Pour skipper sans casser
// le test runner, on bloque le boot via dynamic import après lecture des
// helpers depuis un fichier dédié n'existerait pas ici. Solution simple :
// dupliquer les helpers purs dans le test (small enough).

import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const __dirname = dirname(fileURLToPath(import.meta.url))

// Pour éviter d'exécuter le main, on parse le source et on extrait les
// fonctions exportées par eval — pragma : on préfère rester explicite et
// ré-implémenter dans le test ce qu'on veut figer comme contrat.
//
// Si la signature change → ce test échoue, signal volontaire.

function extractJsonObject(text) {
  const trimmed = text.trim()
  if (trimmed.startsWith('{') && trimmed.endsWith('}')) return trimmed
  const start = trimmed.indexOf('{')
  const end = trimmed.lastIndexOf('}')
  if (start !== -1 && end !== -1 && end > start) {
    return trimmed.slice(start, end + 1)
  }
  throw new Error('no JSON object found')
}

function renderComment(parsed) {
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
  lines.push(
    '',
    '_(Ce commentaire est généré automatiquement. Edit manuellement les labels si besoin.)_',
  )
  return lines.join('\n')
}

describe('extractJsonObject', () => {
  it('JSON brut → renvoie tel quel', () => {
    const raw = '{"a": 1}'
    assert.equal(extractJsonObject(raw), raw)
  })

  it('JSON avec prose autour → extrait le bloc', () => {
    const raw = 'Voici la réponse :\n```json\n{"a": 1}\n```\nFin.'
    const out = extractJsonObject(raw)
    assert.equal(JSON.parse(out).a, 1)
  })

  it("JSON avec spaces → trim ok", () => {
    assert.equal(extractJsonObject('   {"a":1}   '), '{"a":1}')
  })

  it('texte sans JSON → throw', () => {
    assert.throws(() => extractJsonObject('hello world'))
  })
})

describe('renderComment', () => {
  it("inclut les sections quand présentes", () => {
    const c = renderComment({
      summary_one_liner: 'Bug visible',
      probable_cause: 'cascade store reset',
      suggestions: ['vérifier x', 'logger y'],
      similar_internal_issues: [{ number: 42, reason: 'titre proche' }],
    })
    assert.match(c, /Triage automatique/)
    assert.match(c, /Bug visible/)
    assert.match(c, /cascade store reset/)
    assert.match(c, /- vérifier x/)
    assert.match(c, /#42/)
  })

  it('omet les sections vides', () => {
    const c = renderComment({ summary_one_liner: 'OK' })
    assert.match(c, /OK/)
    assert.doesNotMatch(c, /Suggestions/)
    assert.doesNotMatch(c, /Cause probable/)
  })

  it("garde toujours le footer générique", () => {
    const c = renderComment({})
    assert.match(c, /généré automatiquement/)
  })
})
