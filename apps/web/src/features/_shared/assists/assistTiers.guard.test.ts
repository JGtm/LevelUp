/// <reference types="node" />
// @vitest-environment node
/**
 * assistTiers.guard.test.ts — ratchet : les bornes des tranches de part d'assistance
 * (25 / 50 %) n'ont que DEUX foyers côté web, tous deux dans ce dossier :
 *   - assistsI18n.ts   : les libellés d'infobulle (« moins de 25 % », « 25 à 50 % »…) ;
 *   - assistExchange.ts : les calculs de segments (qui ne portent aucun littéral de borne
 *     — les tranches arrivent déjà découpées du Go, `domain.AssistTier*MaxPct`).
 *
 * Le 2026-09-18 (lot 3, tuile de match), la barre à trois tons a été extraite en
 * `AssistTierBar.tsx` et posée sur la tuile (`components/ui/match-card*.tsx`). Deux
 * surfaces de plus pour recopier « 25 » ou « 50 » dans un libellé, une infobulle ou un
 * seuil : ce test l'interdit (règle CLAUDE.md n° 6 — un helper canonique sans garde-rail
 * re-diverge). Un libellé de tranche se prend dans `ASSISTS_TEXT`, jamais en dur.
 *
 * PÉRIMÈTRE : les fichiers listés ci-dessous, hors tests (les tests posent des fixtures
 * chiffrées librement). Le motif vise un « 25 » ou « 50 » isolé (pas « 250 », pas
 * « 0.25 » — une opacité ou un ratio n'est pas une borne de tranche).
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'

const SRC = join(process.cwd(), 'src')

/** Fichiers (chemins relatifs à src/, ou préfixe de dossier) où la borne est INTERDITE. */
const GUARDED: RegExp[] = [/^components\/ui\/match-card[^/]*\.tsx$/, /^features\/_shared\/assists\/AssistTierBar\.tsx$/]

/** Le seul foyer web qui ÉCRIT la borne (assistExchange.ts n'en porte aucune : témoin du motif). */
const ALLOWED = 'features/_shared/assists/assistsI18n.ts'

/** « 25 » ou « 50 » comme nombre isolé : ni chiffre, point, « / » (opacité Tailwind `/50`)
 *  ni « - » (classe utilitaire) collé avant, ni chiffre après. */
const TIER_BOUND = /(?<![\d./-])(?:25|50)(?!\d)/

function walk(dir: string, out: string[]): string[] {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) walk(p, out)
    else if (/\.(ts|tsx)$/.test(name) && !/\.test\.tsx?$/.test(name)) out.push(p)
  }
  return out
}

function guardedFiles(): string[] {
  return walk(SRC, [])
    .map((f) => relative(SRC, f).replace(/\\/g, '/'))
    .filter((rel) => GUARDED.some((re) => re.test(rel)))
    .sort()
}

describe('bornes des tranches d’assistance — deux foyers, jamais recopiées', () => {
  it('témoin : le balayage voit bien la tuile et la barre à trois tons', () => {
    expect(guardedFiles()).toEqual(
      expect.arrayContaining(['components/ui/match-card.tsx', 'features/_shared/assists/AssistTierBar.tsx']),
    )
  })

  it('aucun littéral 25 / 50 de tranche dans la tuile de match ni dans AssistTierBar', () => {
    const hits: string[] = []
    for (const rel of guardedFiles()) {
      const lines = readFileSync(join(SRC, rel), 'utf8').split('\n')
      lines.forEach((line, i) => {
        if (TIER_BOUND.test(line)) hits.push(`${rel}:${i + 1}: ${line.trim()}`)
      })
    }
    expect(hits, 'prendre le libellé dans ASSISTS_TEXT (assistsI18n.ts), la borne vit côté Go').toEqual([])
  })

  it('témoin positif : les foyers autorisés portent bien la borne (le motif fonctionne)', () => {
    expect(TIER_BOUND.test(readFileSync(join(SRC, ALLOWED), 'utf8'))).toBe(true)
  })
})
