/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.6) : le client HTTP du front est unique.
 *
 *  1. Aucun `fetch` hors de `lib/api/` et `lib/desktop/`, sauf les quatre appels de l'allowlist
 *     datée ci-dessous : chaque entrée dit pourquoi le client ne convient pas.
 *  2. Une seule source de la base d'URL de l'API : `VITE_API_BASE_URL` ne se lit que dans
 *     `lib/api/client.ts` (`API_BASE_URL` pour qui compose une URL de navigation), et aucun
 *     littéral `/api/v1` n'est écrit ailleurs.
 *
 * Ce que le client apporte et qu'un `fetch` direct oublie : en-têtes de titre et de locale,
 * cookies, mapping des erreurs, événement `levelup:auth-required`, garde anti-fuite entre titres.
 *
 * Plafond 0 hors allowlist ; une entrée d'allowlist qu'aucun `fetch` ne sert plus fait échouer.
 */
import { describe, expect, it } from 'vitest'
import ts from 'typescript'
import {
  callsNamed,
  formatHits,
  isUnder,
  lineOf,
  parseSource,
  SCAN_TIMEOUT_MS,
  scanSources,
  visitAll,
  type Hit,
  type ScannedSource,
} from '../archScan.testutil'

const CLIENT_DIRS = ['lib/api/', 'lib/desktop/']
const CLIENT_FILE = 'lib/api/client.ts'
// Aides de test (gestionnaires MSW de `src/test/`) : elles imitent le serveur, elles ne l'appellent pas.
const TEST_SUPPORT_DIRS = ['test/']
const FETCH: ReadonlySet<string> = new Set(['fetch'])

// `fetch` hors client autorisés (2026-11-13). Liste décroissante : une entrée tombe dès que son appel
// passe par le client ou disparaît.
const ALLOWED_RAW_FETCH: Readonly<Record<string, string>> = {
  'features/feedback-drawer/queries.ts':
    "api.github.com : le client y enverrait les en-têtes de titre et de locale de LevelUp à un tiers (credentials omit)",
  'features/match-replay/sound/replayAudio.ts':
    "sons du rejeu (fichiers statiques binaires, décodés par WebAudio, silence propre sur absence)",
  'features/match-replay/sound/replayAudioMix.ts':
    "sons du rejeu à l'export (mêmes fichiers statiques binaires que replayAudio)",
  'features/match-replay/layers/useReplayVehicles.ts':
    "manifeste statique des véhicules du rejeu (absence = aucune famille dimensionnée, avertissement)",
}

const API_PREFIX_LITERAL = /^\/api\/v1(?:\/|$)/

function apiBaseUrlReads(source: ScannedSource): Hit[] {
  const hits: Hit[] = []
  visitAll(source, (node) => {
    if (ts.isPropertyAccessExpression(node) && node.name.text === 'VITE_API_BASE_URL') {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: 'lit VITE_API_BASE_URL' })
    } else if (
      (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node) || ts.isTemplateHead(node)) &&
      API_PREFIX_LITERAL.test(node.text)
    ) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: `littéral « ${node.text} »` })
    }
  })
  return hits
}

describe('garde-rail : client HTTP unique', () => {
  const sources = scanSources().filter((s) => !isUnder(s.rel, TEST_SUPPORT_DIRS))

  it('aucun fetch hors du client, hors allowlist datée', () => {
    const hits = sources
      .filter((s) => !isUnder(s.rel, CLIENT_DIRS))
      .flatMap((s) => callsNamed(s, FETCH))
      .filter((h) => !(h.file in ALLOWED_RAW_FETCH))
    expect(
      hits.length,
      formatHits(hits, "Passer par `api` (lib/api/client) : api.get, api.getText, api.getStatic, api.getBlob… ou, si le client ne convient pas, l'ajouter à ALLOWED_RAW_FETCH avec la cause."),
    ).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('ALLOWED_RAW_FETCH décroissante : chaque entrée garde un fetch', () => {
    const withFetch = new Set(
      sources.filter((s) => !isUnder(s.rel, CLIENT_DIRS) && callsNamed(s, FETCH).length > 0).map((s) => s.rel),
    )
    const stale = Object.keys(ALLOWED_RAW_FETCH).filter((f) => !withFetch.has(f))
    expect(stale, `Entrée(s) sans fetch : ${stale.join(', ')}. Retirer la ligne.`).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('la base d’URL de l’API n’a qu’une source : lib/api/client.ts', () => {
    const hits = sources.filter((s) => s.rel !== CLIENT_FILE).flatMap(apiBaseUrlReads)
    expect(
      hits.length,
      formatHits(hits, "Importer `API_BASE_URL` de '@/lib/api/client' (navigation pleine page, lien de téléchargement) ; ne jamais relire l'environnement ni écrire /api/v1."),
    ).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit fetch, window.fetch, la variable d’environnement et le littéral', () => {
    const calls = (code: string): number => callsNamed(parseSource('features/x/a.ts', code), FETCH).length
    expect(calls("const r = await fetch('/x')")).toBe(1)
    expect(calls("const r = await window.fetch('/x')")).toBe(1)
    expect(calls("const r = await globalThis.fetch('/x')")).toBe(1)
    expect(calls("queryClient.fetchQuery({})")).toBe(0)
    expect(calls("// fetch('/x')\nconst a = refetch()")).toBe(0)
    const base = (code: string): number => apiBaseUrlReads(parseSource('features/x/a.ts', code)).length
    expect(base("const b = import.meta.env.VITE_API_BASE_URL ?? '/api/v1'")).toBe(2)
    expect(base('const u = `/api/v1/players/${slug}`')).toBe(1)
    expect(base("const u = '/api/v10'")).toBe(0)
    expect(base("const u = `${API_BASE_URL}/players`")).toBe(0)
  })

  it('le garde n’est pas fantôme : le client lui-même appelle fetch et lit la variable', () => {
    const client = sources.find((s) => s.rel === CLIENT_FILE)
    expect(client).toBeDefined()
    expect(callsNamed(client!, FETCH).length).toBeGreaterThan(0)
    expect(apiBaseUrlReads(client!).length).toBeGreaterThan(0)
  })
})
