/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : helper canonique + garde-rail le même jour) — LE COUP D'ENVOI
 * INCONNU SE LIT COMME ZÉRO EN UN SEUL ENDROIT : `matchKickoffMs` (`matchClock.ts`).
 *
 * POURQUOI CE GARDE. Le serveur sert `header.t0_ms` (le coup d'envoi du match, ADR 0050) et rien
 * quand il est inconnu (Halo 5, estimation refusée). Trois surfaces du client remplaçaient
 * chacune cette absence par zéro, de trois façons différentes (`?? 0`, `Number.isFinite`, `> 0`) :
 * le jour où l'une dérive, deux surfaces posent le même fait à deux instants. Une seule lecture,
 * donc ; ce test échoue si un fichier de production réécrit le remplacement.
 *
 * LES TESTS NE SONT PAS SCANNÉS (ils posent des fixtures) ; le fichier généré non plus.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

const SRC = resolve(__dirname, '..', '..')
const FOYER = 'lib/replay/matchClock.ts'

/** Le remplacement d'un coup d'envoi absent par zéro : `t0_ms ?? 0`, `t0Ms ?? 0`, `t0Ms > 0 ? t0Ms : 0`. */
const REMPLACEMENT = [/\bt0_?ms\??\s*\?\?\s*0\b/i, /\bt0_?ms\b\s*>\s*0\s*\?/i, /isFinite\([\w.?]*t0_?ms\b/i]

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) && entry.name !== 'generated.ts') {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail : un seul remplacement du coup d’envoi inconnu par zéro', () => {
  it('aucun fichier de production hors de matchClock.ts ne le réécrit', () => {
    const fautifs = walk(SRC)
      .filter((f) => relative(SRC, f).replace(/\\/g, '/') !== FOYER)
      .filter((f) => {
        const src = readFileSync(f, 'utf8')
        return REMPLACEMENT.some((re) => re.test(src))
      })
      .map((f) => relative(SRC, f).replace(/\\/g, '/'))
    expect(fautifs).toEqual([])
  })

  it('et matchClock.ts, lui, le porte — sans quoi ce test ne garderait rien', () => {
    expect(REMPLACEMENT.some((re) => re.test(readFileSync(join(SRC, FOYER), 'utf8')))).toBe(true)
  })
})
