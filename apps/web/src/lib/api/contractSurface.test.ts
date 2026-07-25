// @vitest-environment node
/**
 * Tests unitaires du parseur de surface de contrat (contractSurface.ts).
 *
 * Le garde-rail `contract-surface.guard.test.ts` compare une extraction à un snapshot :
 * si le PARSEUR est aveugle à une forme, l'enum correspondant n'entre jamais au
 * snapshot et sa régression ne peut plus être détectée (échec silencieux du filet).
 * Ces cas figent donc les formes que le parseur DOIT reconnaître — au premier rang
 * desquelles les enums d'ITEMS de tableau, rendus parenthésés par openapi-typescript
 * et invisibles avant la contre-revue V72.
 */
import { describe, it, expect } from 'vitest'
import { extractContractSurface } from './contractSurface'

/** Enveloppe minimale : le parseur exige paths/components/operations. */
function generatedTs(schemaBody: string, operationsBody = '        _placeholder?: never;\n'): string {
  return [
    'export interface paths {',
    '    "/probe": {',
    '        get: operations["probe"];',
    '    };',
    '}',
    'export interface components {',
    '    schemas: {',
    schemaBody,
    '    };',
    '    responses: {',
    '    };',
    '    parameters: {',
    '    };',
    '}',
    'export interface operations {',
    '    probe: {',
    operationsBody,
    '    };',
    '}',
    '',
  ].join('\n')
}

describe('extractContractSurface — unions de littéraux', () => {
  it('reconnaît une union simple', () => {
    const src = generatedTs(
      ['        Probe: {', '            status: "open" | "closed";', '        };'].join('\n'),
    )
    expect(extractContractSurface(src).enums['Probe.status']).toEqual(['closed', 'open'])
  })

  it('reconnaît une union nullable (marqueurs null/undefined non figés)', () => {
    const src = generatedTs(
      ['        Probe: {', '            status: "open" | "closed" | null;', '        };'].join('\n'),
    )
    expect(extractContractSurface(src).enums['Probe.status']).toEqual(['closed', 'open'])
  })

  it('reconnaît un enum d’ITEMS de tableau (forme parenthésée) — régression V72', () => {
    // Forme réellement rendue par openapi-typescript pour `items.enum` :
    // avant durcissement, le split naïf sur « | » voyait `("game` et `"other")[]`
    // → aucun membre reconnu → enum ABSENT de la surface.
    const src = generatedTs(
      [
        '        Probe: {',
        '            track_roles: ("game" | "voice" | "other")[];',
        '        };',
      ].join('\n'),
    )
    expect(extractContractSurface(src).enums['Probe.track_roles']).toEqual([
      'game',
      'other',
      'voice',
    ])
  })

  it('reconnaît un tableau d’enum NULLABLE (type: [array, "null"])', () => {
    const src = generatedTs(
      [
        '        Probe: {',
        '            track_roles?: ("game" | "voice" | "other")[] | null;',
        '        };',
      ].join('\n'),
    )
    expect(extractContractSurface(src).enums['Probe.track_roles']).toEqual([
      'game',
      'other',
      'voice',
    ])
  })

  it('reconnaît un tableau d’enum readonly', () => {
    const src = generatedTs(
      ['        Probe: {', '            kinds: readonly ("a" | "b")[];', '        };'].join('\n'),
    )
    expect(extractContractSurface(src).enums['Probe.kinds']).toEqual(['a', 'b'])
  })

  it('n’invente PAS d’enum sur un tableau non littéral (string[] / (string|null)[])', () => {
    const src = generatedTs(
      [
        '        Probe: {',
        '            names: string[];',
        '            others: (string | null)[];',
        '        };',
      ].join('\n'),
    )
    const { enums } = extractContractSurface(src)
    expect(enums['Probe.names']).toBeUndefined()
    expect(enums['Probe.others']).toBeUndefined()
  })

  it('couvre aussi le bloc operations (paramètres de query énumérés en tableau)', () => {
    const src = generatedTs(
      ['        Probe: {', '            id: string;', '        };'].join('\n'),
      ['        kinds: ("a" | "b")[];'].join('\n'),
    )
    expect(extractContractSurface(src).enums['probe.kinds']).toEqual(['a', 'b'])
  })
})
