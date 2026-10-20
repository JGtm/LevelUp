/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, règle « ≤ 2 copies ») : le gamertag de référence d'une vue
 * médias (joueur courant, sinon le profil de la page) se lit par `usePlayerGamertag`
 * (`features/media/usePlayerGamertag.ts`). La galerie, l'onglet Médias d'un match et le rail
 * de l'accueil le calculaient chacun ; une copie locale qui diverge rouvrirait l'écart entre
 * l'interface et la règle serveur de réassociation. Pas d'allowlist.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve, sep } from 'node:path'

const FOOTPRINT = /currentPlayer\?\.gamertag\s*\?\?\s*\w*\.?availablePlayers\.find\(/
const HELPER = 'features/media/usePlayerGamertag.ts'

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail usePlayerGamertag (source unique)', () => {
  it('reconnaît la copie migrée de la galerie', () => {
    const copy =
      '(s) => s.currentPlayer?.gamertag ?? s.availablePlayers.find((p) => p.player_slug === playerSlug)?.gamertag,'
    expect(FOOTPRINT.test(copy)).toBe(true)
    expect(FOOTPRINT.test('const gamertag = usePlayerGamertag(playerSlug)')).toBe(false)
  })

  it('aucune copie hors du hook', () => {
    const src = resolve(__dirname, '../..')
    const offenders = walk(src)
      .map((file) => file.slice(src.length + 1).split(sep).join('/'))
      .filter((rel) => rel !== HELPER)
      .filter((rel) => FOOTPRINT.test(readFileSync(join(src, rel), 'utf8')))
    expect(offenders).toEqual([])
  })
})
