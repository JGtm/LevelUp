import { describe, expect, it } from 'vitest'
import {
  buildIssueUrl,
  buildSearchIssuesUrl,
  escapeSearchQuery,
} from './buildIssueUrl'
import type { Classification } from './classifyFeedback'
import type { FeedbackContext } from './collectContext'

const baseContext: FeedbackContext = {
  browser: {
    url: '/players/Foo/synthesis',
    pathname: '/players/Foo/synthesis',
    userAgent: 'Mozilla/5.0',
    viewportWidth: 1920,
    viewportHeight: 1080,
    locale: 'fr',
    theme: 'dark',
    timestampIso: '2026-05-05T12:00:00Z',
    focusedElement: 'button.foo',
  },
  shell: {
    titleSlug: 'halo_infinite',
    playerSlug: 'Foo',
    appVersion: '7.0.0',
  },
  filters: { filter_mode: 'period', period: { start_date: '2026-04-01', end_date: '2026-05-01' } },
  console: [],
  failedRequests: [],
}

const bugClassif: Classification = { type: 'bug', severity: 'high', area: 'synthesis' }
const ideaClassif: Classification = { type: 'enhancement', severity: 'low', area: 'general' }

describe('buildIssueUrl — base', () => {
  it("préfixe le titre selon le type", () => {
    const r = buildIssueUrl({
      title: 'truc cassé',
      description: 'foo',
      context: baseContext,
      classification: bugClassif,
    })
    const url = new URL(r.url)
    expect(url.searchParams.get('title')).toBe('[Bug] truc cassé')
  })

  it("encode les labels avec virgule", () => {
    const r = buildIssueUrl({
      title: 't',
      description: 'd',
      context: baseContext,
      classification: bugClassif,
    })
    const url = new URL(r.url)
    expect(url.searchParams.get('labels')).toBe(
      'feedback,bug,severity:high,area:synthesis',
    )
  })

  it("URL pointe vers le repo public JGtm/LevelUp", () => {
    const r = buildIssueUrl({
      title: 't',
      description: 'd',
      context: baseContext,
      classification: ideaClassif,
    })
    expect(r.url).toContain('https://github.com/JGtm/LevelUp/issues/new?')
  })

  it("body contient toutes les sections principales", () => {
    const r = buildIssueUrl({
      title: 't',
      description: 'description user',
      context: baseContext,
      classification: bugClassif,
    })
    expect(r.body).toContain('## Description')
    expect(r.body).toContain('description user')
    expect(r.body).toContain('## Contexte')
    expect(r.body).toContain('## Environnement client')
    expect(r.body).toContain('## Filtres actifs')
    expect(r.body).toContain('## Classification heuristique')
    expect(r.body).toContain('Auto-généré par le drawer feedback')
  })

  it("description vide → fallback explicite", () => {
    const r = buildIssueUrl({
      title: 't',
      description: '',
      context: baseContext,
      classification: bugClassif,
    })
    expect(r.body).toContain('_(aucune description fournie)_')
  })
})

describe('buildIssueUrl — troncature progressive', () => {
  it("body sous 7000 chars en cas standard", () => {
    const r = buildIssueUrl({
      title: 't',
      description: 'foo',
      context: baseContext,
      classification: bugClassif,
    })
    expect(r.body.length).toBeLessThan(7000)
    expect(r.wasTruncated).toBe(false)
  })

  it("description 8000 chars → tronquée avec marker", () => {
    const huge = 'x'.repeat(8000)
    const r = buildIssueUrl({
      title: 't',
      description: huge,
      context: baseContext,
      classification: bugClassif,
    })
    expect(r.body.length).toBeLessThanOrEqual(7000)
    expect(r.wasTruncated).toBe(true)
    expect(r.body).toContain('…[truncated]')
  })

  it("nombreuses erreurs console + petit description → erreurs tronquées en premier", () => {
    const manyConsole = Array.from({ length: 50 }, (_, i) => ({
      level: 'error' as const,
      message: 'x'.repeat(200) + ` err-${i}`,
      timestamp: i,
      stack: 'a'.repeat(500),
    }))
    const r = buildIssueUrl({
      title: 't',
      description: 'desc courte',
      context: { ...baseContext, console: manyConsole },
      classification: bugClassif,
    })
    expect(r.body.length).toBeLessThanOrEqual(7000)
    expect(r.wasTruncated).toBe(true)
    expect(r.body).toContain('## Erreurs console récentes')
    // La description courte doit être préservée intacte
    expect(r.body).toContain('desc courte')
  })
})

describe('escapeSearchQuery', () => {
  it.each<[string, string]>([
    ['foo: bar', 'foo bar'],
    ['feature+', 'feature'],
    ['(crash) test', 'crash test'],
    ['"quoted"', 'quoted'],
    ['a/b/c', 'a b c'],
    [':::+++', ''],
    ['  spaces   here  ', 'spaces here'],
    ['normal title', 'normal title'],
  ])('%s → %s', (input, expected) => {
    expect(escapeSearchQuery(input)).toBe(expected)
  })
})

describe('buildSearchIssuesUrl', () => {
  it("encode-URI la query complète", () => {
    const url = buildSearchIssuesUrl('foo bar')
    expect(url).toMatch(/^https:\/\/api\.github\.com\/search\/issues\?q=/)
    expect(url).toContain('per_page=3')
    // décodé doit contenir le scope repo
    const q = new URL(url).searchParams.get('q')
    expect(q).toBe('foo bar is:issue repo:JGtm/LevelUp')
  })

  it("titre avec opérateurs réservés est sanitize", () => {
    const url = buildSearchIssuesUrl('crash: TypeError "boom"')
    const q = new URL(url).searchParams.get('q')
    // Le titre user ne doit plus contenir d'opérateurs réservés
    const userPart = q?.replace(/ is:issue repo:.*/, '')
    expect(userPart).not.toContain(':')
    expect(userPart).not.toContain('"')
    // Le scope `is:issue repo:` est ajouté APRÈS escape, donc présent
    expect(q).toContain('is:issue')
    expect(q).toContain('repo:JGtm/LevelUp')
  })
})
