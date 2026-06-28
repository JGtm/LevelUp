import { describe, expect, it } from 'vitest'
import { classifyFeedback, matchArea } from './classifyFeedback'
import type { ConsoleEntry } from '@/lib/global-capture/buffers'

const noConsole: ConsoleEntry[] = []
const fatalConsole: ConsoleEntry[] = [
  { level: 'error', message: 'TypeError: Cannot read properties of undefined', timestamp: 1 },
]
const warnConsole: ConsoleEntry[] = [
  { level: 'error', message: '500 Internal Server Error', timestamp: 1 },
]

describe('classifyFeedback — type', () => {
  it("respecte le type choisi explicitement (jamais 'auto' override)", () => {
    const c = classifyFeedback(
      { pickedType: 'enhancement', description: 'crash impossible' },
      { pathname: '/', recentConsole: fatalConsole },
    )
    expect(c.type).toBe('enhancement')
  })

  it("infère 'bug' depuis TypeError console", () => {
    const c = classifyFeedback(
      { pickedType: 'auto', description: '' },
      { pathname: '/', recentConsole: fatalConsole },
    )
    expect(c.type).toBe('bug')
  })

  it("infère 'bug' depuis description 'cassé'", () => {
    const c = classifyFeedback(
      { pickedType: 'auto', description: 'le bouton est cassé' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.type).toBe('bug')
  })

  it("infère 'question' depuis '?'", () => {
    const c = classifyFeedback(
      { pickedType: 'auto', description: 'comment ça marche ?' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.type).toBe('question')
  })

  it("default → 'enhancement'", () => {
    const c = classifyFeedback(
      { pickedType: 'auto', description: 'rien de spécial' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.type).toBe('enhancement')
  })
})

describe('classifyFeedback — severity', () => {
  it('TypeError console → critical', () => {
    const c = classifyFeedback(
      { pickedType: 'bug', description: 'foo' },
      { pathname: '/', recentConsole: fatalConsole },
    )
    expect(c.severity).toBe('critical')
  })

  it('description "crash" sans erreur console → high', () => {
    const c = classifyFeedback(
      { pickedType: 'bug', description: 'ça crash quand je clique' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.severity).toBe('high')
  })

  it('description "impossible" → high', () => {
    const c = classifyFeedback(
      { pickedType: 'bug', description: 'impossible de me connecter' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.severity).toBe('high')
  })

  it('console error non-fatale + type=bug → high', () => {
    const c = classifyFeedback(
      { pickedType: 'bug', description: 'foo' },
      { pathname: '/', recentConsole: warnConsole },
    )
    expect(c.severity).toBe('high')
  })

  it('description "bug" sans signal high → medium', () => {
    const c = classifyFeedback(
      { pickedType: 'bug', description: 'petit bug visuel' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.severity).toBe('medium')
  })

  it('default → low', () => {
    const c = classifyFeedback(
      { pickedType: 'enhancement', description: '' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.severity).toBe('low')
  })

  it('question → low', () => {
    const c = classifyFeedback(
      { pickedType: 'auto', description: 'comment fonctionne X ?' },
      { pathname: '/', recentConsole: noConsole },
    )
    expect(c.severity).toBe('low')
  })
})

describe('matchArea — toutes les routes', () => {
  it.each<[string, string]>([
    ['/players/Foo/synthesis', 'synthesis'],
    ['/players/Foo/synthesis?period=7d', 'synthesis'],
    ['/players/Foo/explorer', 'explorer'],
    ['/players/Foo/explorer/extra', 'explorer'],
    ['/players/Foo/squad', 'squad'],
    ['/players/Foo/squad/contributions', 'squad'],
    ['/players/Foo/squad/synergies', 'squad'],
    ['/players/Foo/stats/sessions', 'sessions'],
    ['/players/Foo/stats/timeseries', 'timeseries'],
    ['/players/Foo/matches/abc-123', 'match_view'],
    ['/players/Foo/palmares', 'palmares'],
    ['/players/Foo/palmares/prestige', 'palmares'],
    ['/players/Foo/palmares/relations', 'palmares'],
    ['/players/Foo/palmares/season-pass', 'palmares'],
    ['/players/Foo/community', 'palmares'],
    ['/players/Foo/community/relations', 'palmares'],
    ['/players/Foo/home', 'player_home'],
    ['/players/Foo/media', 'media'],
    ['/players/Foo/career', 'career'],
    ['/players/Foo/notifications', 'notifications'],
    ['/players/Foo/objectifs', 'objectifs'],
    ['/players/Foo/ascension', 'objectifs'],
    ['/players/Foo/ascension/realisations', 'objectifs'],
    ['/players/Foo/citations', 'citations'],
    ['/setup', 'settings'],
    ['/settings', 'settings'],
    ['/changelog', 'meta'],
    ['/help', 'meta'],
    ['/login', 'general'],
    ['/', 'general'],
    ['/random/path', 'general'],
  ])('%s → %s', (path, expected) => {
    expect(matchArea(path)).toBe(expected)
  })
})
