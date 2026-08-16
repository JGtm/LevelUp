/// <reference types="node" />
// @vitest-environment node
/**
 * teamLabel : cascade du libellé d'équipe (tests unitaires) + garde-rail (CLAUDE.md n°6) :
 * `labelHasTeamWord(` — la brique qui trahit une cascade recopiée — ne s'appelle QUE dans
 * `lib/halo/`. Les deux copies historiques (`MatchScoreboard.tsx`, `MatchObjectivesSection.tsx`)
 * ont été migrées vers `resolveTeamLabel` le 2026-08-16, quand le rejeu 2D en a eu besoin.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { resolveTeamLabel, type TeamLabelText } from './teamLabel'

const FR: TeamLabelText = {
  teamLabelFmt: (name) => `Équipe ${name}`,
  teamNumberedFmt: (n) => `Équipe ${n}`,
  teamUnknown: 'Équipe inconnue',
}
const EN: TeamLabelText = {
  teamLabelFmt: (name) => `Team ${name}`,
  teamNumberedFmt: (n) => `Team ${n}`,
  teamUnknown: 'Unknown team',
}

describe('resolveTeamLabel — la cascade', () => {
  it('t0 -> nom officiel préfixé, dans la langue demandée', () => {
    expect(resolveTeamLabel([], 't0', FR)).toBe('Équipe Eagle')
    expect(resolveTeamLabel([], 't1', EN)).toBe('Team Cobra')
  })
  it('un libellé backend nu est préfixé ; un libellé backend déjà complet ne l’est PAS', () => {
    expect(resolveTeamLabel([{ team_name: 'Rouge' }], 't0', FR)).toBe('Équipe Rouge')
    expect(resolveTeamLabel([{ team_name: 'Équipe Cobra' }], 't1', FR)).toBe('Équipe Cobra')
    expect(resolveTeamLabel([{ team_name: 'Team Cobra' }], 't1', EN)).toBe('Team Cobra')
  })
  it('le backend prime sur le référentiel officiel', () => {
    expect(resolveTeamLabel([{ team_name: null }, { team_name: 'Bleu' }], 't1', FR)).toBe('Équipe Bleu')
  })
  it('id connu hors référentiel -> « Équipe N » ; côté absent ou malformé -> inconnue', () => {
    expect(resolveTeamLabel([], 't12', FR)).toBe('Équipe 12')
    expect(resolveTeamLabel([], null, FR)).toBe('Équipe inconnue')
    expect(resolveTeamLabel([], 'rouge', EN)).toBe('Unknown team')
  })
})

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

describe('garde-rail teamLabel (lib/halo/teamLabel.ts source unique)', () => {
  it('labelHasTeamWord( n’est appelé nulle part hors lib/halo/', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const rel = file.replace(srcRoot, 'src').replace(/\\/g, '/')
      if (rel.startsWith('src/lib/halo/')) continue
      if (/labelHasTeamWord\(/.test(readFileSync(file, 'utf8'))) offenders.push(rel)
    }
    expect(
      offenders,
      `Cascade de libellé d'équipe à migrer vers @/lib/halo/teamLabel (resolveTeamLabel) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
