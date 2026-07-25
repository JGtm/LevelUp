/// <reference types="node" />
// @vitest-environment node
/**
 * Garde SÉMANTIQUE du contrat généré (V72-01 / H7).
 *
 * Contexte : depuis H6, `apps/go-api/api/openapi.yaml` est GÉNÉRÉ (`make openapi-gen`) et
 * `generated.ts` en dérive (`make generate-types`). Une régénération qui PERDRAIT un
 * schéma, une opération, un chemin ou un membre d'enum passerait silencieusement : `tsc`
 * ne voit qu'un type devenu plus permissif (`string` au lieu d'une union), et les
 * consommateurs qui ne lisent pas le champ perdu continuent de compiler. Ce test fige la
 * SURFACE du contrat (noms + membres d'enums) et échoue sur toute DISPARITION.
 *
 * Contrat du garde-rail :
 *   - AJOUT (nouveau schéma, nouvelle opération, nouveau membre d'enum) = TOLÉRÉ, le
 *     snapshot ne borne pas la croissance du contrat.
 *   - DISPARITION (nom absent, enum redevenu `string`, membre d'enum absent) = ROUGE.
 *
 * Mise à jour du snapshot (retrait ASSUMÉ d'un schéma / renommage côté Go) :
 *   cd apps/web && UPDATE_CONTRACT_SURFACE=1 npx vitest run src/lib/api/contract-surface.guard.test.ts
 * puis JUSTIFIER la disparition (message de commit + plan/journal du chantier en cours).
 */
import { describe, it, expect } from 'vitest'
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { extractContractSurface, type ContractSurface } from './contractSurface'

const GENERATED_TS = resolve(process.cwd(), 'src/lib/api/generated.ts')
const SNAPSHOT_JSON = resolve(process.cwd(), 'src/lib/api/contract-surface.snapshot.json')

const current = extractContractSurface(readFileSync(GENERATED_TS, 'utf8'))

if (process.env.UPDATE_CONTRACT_SURFACE === '1') {
  writeFileSync(SNAPSHOT_JSON, `${JSON.stringify(current, null, 2)}\n`, 'utf8')
}

const snapshot = JSON.parse(readFileSync(SNAPSHOT_JSON, 'utf8')) as ContractSurface

/** Noms du snapshot absents de l'extraction courante. */
function lost(before: string[], after: string[]): string[] {
  const present = new Set(after)
  return before.filter((n) => !present.has(n)).sort()
}

const UPDATE_HINT =
  `  cd apps/web && UPDATE_CONTRACT_SURFACE=1 npx vitest run src/lib/api/contract-surface.guard.test.ts`

describe('garde sémantique du contrat généré (V72-01 H7 — surface de generated.ts)', () => {
  it("l'extraction est non vacuante (sinon le garde-rail ne mordrait plus)", () => {
    expect(current.schemas.length, 'schémas extraits').toBeGreaterThan(400)
    expect(current.operations.length, 'opérations extraites').toBeGreaterThan(100)
    expect(current.paths.length, 'chemins extraits').toBeGreaterThan(100)
    expect(Object.keys(current.enums).length, 'enums extraits').toBeGreaterThan(10)
    expect(snapshot.schemas.length, 'schémas du snapshot').toBeGreaterThan(400)
  })

  const buckets: Array<[keyof ContractSurface, string]> = [
    ['schemas', 'schéma(s) du contrat'],
    ['operations', 'opération(s)'],
    ['paths', 'chemin(s)'],
    ['responses', 'réponse(s) partagée(s)'],
    ['parameters', 'paramètre(s) partagé(s)'],
  ]
  for (const [bucket, label] of buckets) {
    it(`aucun ${label} perdu depuis le snapshot`, () => {
      const missing = lost(snapshot[bucket] as string[], current[bucket] as string[])
      expect(
        missing,
        `${missing.length} ${label} présent(s) au snapshot mais ABSENT(S) de generated.ts : ${missing.join(', ')}.\n` +
          `Une régénération ne doit rien perdre. Corriger le DTO/handler Go (puis make openapi-gen && make generate-types),\n` +
          `ou — si le retrait est assumé — régénérer le snapshot :\n${UPDATE_HINT}`,
      ).toEqual([])
    })
  }

  it("aucun enum ni membre d'enum perdu depuis le snapshot", () => {
    const regressions: string[] = []
    for (const [key, members] of Object.entries(snapshot.enums)) {
      const now = current.enums[key]
      if (!now) {
        regressions.push(`${key} : enum DISPARU (le champ n'est plus une union de littéraux)`)
        continue
      }
      const missing = members.filter((m) => !now.includes(m))
      if (missing.length > 0) {
        regressions.push(`${key} : membre(s) perdu(s) ${missing.map((m) => `"${m}"`).join(', ')}`)
      }
    }
    expect(
      regressions,
      `Régression d'enum dans le contrat généré :\n  ${regressions.join('\n  ')}\n` +
        `Un enum qui redevient \`string\` (ou perd une valeur) = perte de sécurité de type SILENCIEUSE pour tsc.\n` +
        `Corriger le tag \`enum:\` du DTO Go, ou régénérer le snapshot si le retrait est assumé :\n${UPDATE_HINT}`,
    ).toEqual([])
  })
})
