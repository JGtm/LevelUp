/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : à la troisième copie, un helper ET un garde-rail) : les libellés des
 * natures des « Outils de destruction » (mêlée, grenades, mécaniques, reliquat, catégories de source)
 * ont UNE source, `toolKindLabels` (`squad/charts/squadFragTools.ts`), lue par l'Escouade, Sessions et
 * la Vue match. Un fichier de production qui construit un `SquadToolKindLabels` en lisant le
 * manifeste `frags` lui-même est une copie : refusé.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

/** La signature d'une copie : un objet `SquadToolKindLabels` ET une lecture du manifeste `frags`. */
function isToolLabelsCopy(source: string): boolean {
  return /:\s*SquadToolKindLabels\s*=/.test(source) && /fragsManifest/.test(source)
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\./.test(entry.name)) out.push(full)
  }
  return out
}

describe('garde-rail toolKindLabels (source unique des libellés des Outils de destruction)', () => {
  it('reconnaît une copie (ancien patron de SquadFragSection et de SessionToolsCard)', () => {
    const ancien = `const role = (r: string) => formatMessage(fragsManifest, \`frags.role.\${r}\` as never, locale)
    const labels: SquadToolKindLabels = { melee: classLabel('melee') }`
    expect(isToolLabelsCopy(ancien)).toBe(true)
    expect(isToolLabelsCopy('const labels = toolKindLabels(locale, t.weaponKills)')).toBe(false)
  })

  it('aucune feature ne recopie les libellés hors du helper', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const allowed = join(srcRoot, 'features', 'squad', 'charts', 'squadFragTools.ts')
    const offenders = walk(join(srcRoot, 'features')).filter((f) => f !== allowed && isToolLabelsCopy(readFileSync(f, 'utf8')))
    expect(offenders, `libellés des outils à prendre de toolKindLabels : ${offenders.join(', ')}`).toEqual([])
  })
})
