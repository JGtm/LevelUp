/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, règle « ≤ 2 copies ») : le lien de la marque vers l'accueil a une
 * SOURCE UNIQUE, `components/shell/HomeLink.tsx`. Il était recopié dans la barre du haut, la barre
 * latérale dépliée et le rail replié (revue L21a, constat P2). Ce test échoue si le libellé du
 * lien (`common.shell.logo_aria`), signature de la copie, réapparaît hors du composant (la cible vers
 * l'accueil du joueur est, elle, légitime ailleurs : liens de pages, changement de joueur).
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve, sep } from 'node:path'

const FOOTPRINTS: { name: string; re: RegExp }[] = [
  { name: 'libellé du lien de la marque', re: /common\.shell\.logo_aria/ },
]

const HELPER = 'components/shell/HomeLink.tsx'

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

const SRC = resolve(__dirname, '../..')

describe('garde-rail : lien de la marque vers l’accueil', () => {
  it('les empreintes reconnaissent l’ancienne copie', () => {
    const ancienne = `<Link to="/{-$lang}" className={logoClass} aria-label={t('common.shell.logo_aria')}>`
    for (const { re } of FOOTPRINTS) expect(re.test(ancienne)).toBe(true)
  })

  it('aucune copie hors de HomeLink', () => {
    const violations: string[] = []
    for (const file of walk(SRC)) {
      const rel = file.slice(SRC.length + 1).split(sep).join('/')
      if (rel === HELPER) continue
      const source = readFileSync(file, 'utf-8')
      for (const { name, re } of FOOTPRINTS) if (re.test(source)) violations.push(`${rel} : ${name}`)
    }
    expect(violations).toEqual([])
  })
})
