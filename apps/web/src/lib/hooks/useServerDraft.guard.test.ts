/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : la copie éditable d'un objet servi passe par `useServerDraft`.
 *
 * Le motif « valeur précédente » recopié à la main a porté trois fois le même défaut (page
 * Paramètres, Admin · Sync, Admin · Sons du rejeu, corrigés le 2026-10-07) : la valeur précédente
 * partait de l'objet en cache, la première copie était sautée, chaque contrôle montrait son
 * défaut. Ce test échoue si un fichier recopie le motif sur des réglages (`prevSettings`,
 * `useState(settings)`).
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const MOTIFS: readonly RegExp[] = [/\bprevSettings\b/, /\bsetPrevSettings\b/, /useState(<[^>]*>)?\(\s*settings\s*\)/]

function sansCommentaires(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')
}

function fautif(source: string): boolean {
  const propre = sansCommentaires(source)
  return MOTIFS.some((re) => re.test(propre))
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) out.push(full)
  }
  return out
}

describe('garde-rail : copie éditable des réglages par useServerDraft', () => {
  const racine = resolve(process.cwd(), 'src')

  it('aucun fichier ne recopie le motif « valeur précédente » sur des réglages', () => {
    const fautifs = walk(racine)
      .filter((f) => fautif(readFileSync(f, 'utf8')))
      .map((f) => relative(racine, f).replace(/\\/g, '/'))
    expect(fautifs, 'copie éditable des réglages : `useServerDraft(settings)` (lib/hooks/useServerDraft.ts)').toEqual([])
  })

  it('le détecteur voit l’ancien motif et ignore les commentaires', () => {
    expect(fautif('const [prevSettings, setPrevSettings] = useState(settings)')).toBe(true)
    expect(fautif('const [p, setP] = useState<SettingsResponse | undefined>(settings)')).toBe(true)
    expect(fautif('// prevSettings = useState(settings)\nconst [local] = useServerDraft(settings)')).toBe(false)
  })
})
