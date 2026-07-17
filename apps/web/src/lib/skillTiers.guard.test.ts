/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6, regle « <= 2 copies ») : le mapping des NOMS de tiers
 * de skill (CSR / LUSR) EN<->FR a une SOURCE UNIQUE : `lib/skillTiers.ts`
 * (`localizeTierName` / `localizeTierLabel`, derives de `LUSR_TIER_GRID`). Ce test
 * echoue si un composant de `src/features/**` re-declare un mapping FR local
 * (ex. l'ancien `CSR_TIER_FR` de ExplorerTargetSeasonCSR) en litteralisant les noms
 * de tiers FR distinctifs.
 *
 * Contexte : 3e copie du mapping (grille + CSR_TIER_FR + localizeTierLabel) ->
 * centralisation dans skillTiers.ts + ce garde-rail. Sans lui, le mapping re-diverge
 * (les libelles FR baken sous UI EN, symptome documente skillTiers.ts).
 *
 * Detection : les noms de tiers FR qui DIFFERENT de l'EN (« Argent », « Platine »,
 * « Diamant ») en tant que litteraux chaine. Bronze/Or/Onyx/Champion sont ambigus
 * (mot courant, invariant de locale) donc hors detection. La couche i18n legitime
 * (`lib/i18n/manifests/*.toml` + `lib/i18n/generated/*.ts`, labels de filtre) vit
 * sous `lib/` et n'est PAS scannee — seul `src/features/**` l'est.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Noms de tiers FR distinctifs (differents de l'EN) comme litteral chaine.
const FR_TIER_LITERAL = /['"](Argent|Platine|Diamant)['"]/

// Sequence distinctive du mapping des sous-paliers en chiffres romains
// (['', 'I', 'II', 'III', 'IV', 'V', 'VI']). Source unique : lib/skillTiers.ts
// (subTierRoman). Interdit toute re-declaration locale sous src/features/**.
const SUBTIER_ROMAN_LITERAL = /['"]II['"]\s*,\s*['"]III['"]\s*,\s*['"]IV['"]/

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

describe('garde-rail mapping tiers CSR/LUSR (source unique lib/skillTiers.ts)', () => {
  const featuresRoot = resolve(process.cwd(), 'src', 'features')
  const files = walk(featuresRoot)

  it('aucun litteral de nom de tier FR sous src/features/** (utiliser localizeTierName)', () => {
    const offenders = files
      .filter((f) => FR_TIER_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(featuresRoot, 'src/features'))
    expect(
      offenders,
      `Noms de tiers FR a localiser via localizeTierName/localizeTierLabel (lib/skillTiers.ts) : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  it('aucun mapping romain de sous-palier re-declare sous src/features/** (utiliser subTierRoman)', () => {
    const offenders = files
      .filter((f) => SUBTIER_ROMAN_LITERAL.test(readFileSync(f, 'utf8')))
      .map((f) => f.replace(featuresRoot, 'src/features'))
    expect(
      offenders,
      `Mapping romain des sous-paliers a centraliser via subTierRoman (lib/skillTiers.ts) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
