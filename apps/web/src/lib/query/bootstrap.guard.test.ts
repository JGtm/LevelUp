/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.10) : la lecture de `GET /bootstrap` est écrite UNE fois,
 * dans `lib/query/bootstrap.ts` (`fetchBootstrap`, `bootstrapQueryOptions`, `useBootstrap`).
 *
 * Le chemin `/bootstrap` ne se lit comme littéral nulle part ailleurs. `lib/api/client.ts` le cite pour
 * exempter cette lecture de l'événement « authentification requise » : c'est la seule exception, car le
 * client ne lit pas le bootstrap, il le reconnaît.
 *
 * Plafond 0 hors de ces deux fichiers.
 */
import { describe, expect, it } from 'vitest'
import ts from 'typescript'
import { formatHits, lineOf, parseSource, SCAN_TIMEOUT_MS, scanSources, visitAll, type Hit, type ScannedSource } from '../archScan.testutil'

const OWNER = 'lib/query/bootstrap.ts'
const RECOGNISED_BY = 'lib/api/client.ts'
// Aides de test (gestionnaires MSW) : elles imitent le serveur, elles ne l'appellent pas.
const TEST_SUPPORT = 'test/'
const BOOTSTRAP_PATH = /^\/bootstrap(?:[/?]|$)/

function bootstrapPathLiterals(source: ScannedSource): Hit[] {
  const hits: Hit[] = []
  visitAll(source, (node) => {
    if (
      (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node) || ts.isTemplateHead(node)) &&
      BOOTSTRAP_PATH.test(node.text)
    ) {
      hits.push({ file: source.rel, line: lineOf(source, node), detail: `littéral « ${node.text} »` })
    }
  })
  return hits
}

describe('garde-rail : une seule lecture de /bootstrap', () => {
  const sources = scanSources().filter((s) => !s.rel.startsWith(TEST_SUPPORT))

  it('le chemin /bootstrap ne s’écrit que dans lib/query/bootstrap.ts', () => {
    const hits = sources
      .filter((s) => s.rel !== OWNER && s.rel !== RECOGNISED_BY)
      .flatMap(bootstrapPathLiterals)
    expect(
      hits.length,
      formatHits(hits, "Passer par `fetchBootstrap`, `bootstrapQueryOptions` ou `useBootstrap` de '@/lib/query/bootstrap'."),
    ).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit le littéral et le gabarit, pas un autre chemin', () => {
    const n = (code: string): number => bootstrapPathLiterals(parseSource('features/x/a.ts', code)).length
    expect(n("api.get('/bootstrap')")).toBe(1)
    expect(n('api.get(`/bootstrap`)')).toBe(1)
    expect(n("api.get('/bootstrap?x=1')")).toBe(1)
    expect(n("api.get('/bootstrap-status')")).toBe(0)
    expect(n("// api.get('/bootstrap')\nconst a = 1")).toBe(0)
  })

  it('le garde n’est pas fantôme : le module propriétaire écrit bien le chemin', () => {
    const owner = sources.find((s) => s.rel === OWNER)
    expect(owner).toBeDefined()
    expect(bootstrapPathLiterals(owner!).length).toBe(1)
  })
})
