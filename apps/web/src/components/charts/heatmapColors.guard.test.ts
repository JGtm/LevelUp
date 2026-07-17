/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n6, regle « <= 2 copies » -> helper + garde-rail) : les tokens
 * de RAMPE de heatmap continue (heatmap-cold, heatmap-hot, heatmap-freq-low,
 * heatmap-freq-high) ont une SOURCE UNIQUE de composition — le helper
 * heatmapRampTokens() de components/charts/heatmapColors.ts. Les charts/features
 * resolvent leur rampe via ce helper — heatmapRampTokens(mode).map(resolveToken) —
 * jamais en listant les tokens a la main : c'est LUI qui decide, selon la palette
 * d'accessibilite active, de basculer une rampe cold->hot (iso-luminante donc illisible
 * en daltonisme) vers la rampe de frequence mono-teinte CVD-safe. Un usage inline
 * court-circuiterait cette bascule.
 *
 * Ce test echoue si un token de rampe apparait en LITTERAL QUOTE (quote collee au token)
 * sous src/features/** ou src/components/** hors des fichiers autorises ci-dessous.
 *
 * Perimetre / exclusions (legitimes, hors du champ « inline ») :
 *  - heatmapColors.ts : le helper lui-meme — il DEFINIT les rampes (source unique).
 *  - heatmapColors.test.ts : verifie que le helper renvoie ces tokens.
 * NON matches (donc aucune exclusion requise) : les mentions en COMMENTAIRE (non
 * quotees), les formes resolues des tests ou un prefixe separe la quote du token
 * (var(...) / color:...), et les tokens DIVERGENTS (heatmap-divergent-*), hors perimetre
 * de cette regle — leur rampe passe deja par le helper en mode divergent.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Token de rampe heatmap en litteral quote : la quote est COLLEE au token (usage code
// reel, ex. resolveToken(...) ou tableau de tokens). Le ':' / '(' des formes resolues
// des tests (color:heatmap-... / var(heatmap-...)) s'intercale entre la quote et le
// token -> non matche. Les commentaires (tokens non quotes) non plus.
const INLINE_RAMP_TOKEN = /['"]heatmap-(?:cold|hot|freq-[a-z]+)['"]/

// Seuls le helper et son test nomment legitimement ces tokens en litteral.
const ALLOWED = new Set(['heatmapColors.ts', 'heatmapColors.test.ts'])

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

describe('garde-rail rampe heatmap (heatmapRampTokens source unique)', () => {
  it('aucun token de rampe (heatmap-cold/hot/freq-*) inline hors du helper', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const roots = [join(srcRoot, 'features'), join(srcRoot, 'components')]
    const offenders: string[] = []
    for (const root of roots) {
      for (const file of walk(root)) {
        const base = file.split(/[\\/]/).pop() ?? ''
        if (ALLOWED.has(base)) continue
        if (INLINE_RAMP_TOKEN.test(readFileSync(file, 'utf8'))) {
          offenders.push(file.replace(srcRoot, 'src'))
        }
      }
    }
    expect(
      offenders,
      `Rampe heatmap a router via heatmapRampTokens() (components/charts/heatmapColors) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
