/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : la RECETTE DE TEINTE d'un habillage d'équipe ne s'écrit que
 * dans `features/match-view/teamColor.ts` (`teamTintStyles`).
 *
 * Contexte : le dosage vivait en toutes lettres dans l'en-tête d'équipe de
 * `MatchScoreboard.tsx` ; l'écran de victoire du rejeu 2D en aurait été la deuxième copie
 * (2026-08-26). Une factorisation sans garde-rail re-diverge — et deux dosages pour la même
 * couleur d'équipe, ce sont deux identités visuelles sur deux pages du même match.
 *
 * CE QUE CE TEST SURVEILLE : le littéral du FOND (22 %), qui est la signature de la recette.
 * Le trait à 55 % n'est PAS surveillé : ce dosage est un motif d'habillage général du dépôt
 * (badges de revue, pastilles de l'Explorer, modules de briefing) qui n'a rien à voir avec une
 * couleur d'équipe — l'interdire ferait rougir des surfaces innocentes. Le 22 %, lui,
 * n'appartient qu'à cet habillage-là.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// La signature du fond teinté, quel que soit l'espace de mélange et la couleur source.
const TEAM_TINT_BACKGROUND = /22%,\s*transparent/

// Seul teamColor.ts déclare légitimement la recette.
const ALLOWED = new Set(['teamColor.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail teinte d’équipe (teamColor.ts source unique)', () => {
  it('aucune recette de teinte recopiée dans features/ hors teamColor.ts', () => {
    const featuresRoot = resolve(process.cwd(), 'src', 'features')
    const offenders: string[] = []
    for (const file of walk(featuresRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      if (TEAM_TINT_BACKGROUND.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(featuresRoot, 'src/features'))
      }
    }
    expect(
      offenders,
      `Recette de teinte à migrer vers teamTintStyles (features/match-view/teamColor.ts) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
