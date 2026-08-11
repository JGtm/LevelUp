/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n6, regle « <= 2 copies » -> helper + garde-rail) : la cascade qui
 * decide QUI EST ALLIE dans un match — « allie = ligne marquee is_me, ou meme team_side que
 * le joueur de la page » — a UNE source, `xuidMeta.ts`.
 *
 * POURQUOI CE GARDE. Elle etait ecrite deux fois a l'identique (MatchTugOfWarChart,
 * MatchKDCumulChart) et le kill feed du rejeu 2D en reclamait une troisieme. Trois copies
 * d'une regle d'appartenance, ce sont trois couleurs possibles pour la meme equipe sur trois
 * surfaces du meme match — et la divergence ne se voit qu'a l'oeil, tard.
 *
 * Ce test echoue si la comparaison de camp reapparait ailleurs que dans le helper.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// La signature de la cascade : la comparaison du team_side d'une ligne au camp de reference.
// Volontairement tolerante aux espaces, pour qu'un reformatage ne la fasse pas passer.
const CASCADE = /team_side\s*===\s*allyTeam/

// Seul le helper (et son test) nomme legitimement cette comparaison.
const ALLOWED = new Set(['xuidMeta.ts', 'xuidMeta.test.ts', 'xuidMeta.guard.test.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail : une seule cascade d’appartenance d’équipe', () => {
  it('ne réapparaît nulle part ailleurs que dans xuidMeta.ts', () => {
    const racine = resolve(__dirname, '..', '..')
    const fautifs = walk(racine).filter((f) => {
      const base = f.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) return false
      return CASCADE.test(readFileSync(f, 'utf8'))
    })
    expect(fautifs).toEqual([])
  })

  it('et le helper, lui, la porte bien — sans quoi ce test ne garderait rien', () => {
    const helper = readFileSync(join(__dirname, 'xuidMeta.ts'), 'utf8')
    expect(CASCADE.test(helper)).toBe(true)
  })
})
