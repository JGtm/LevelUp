// @vitest-environment node
/**
 * Garde-rail (règle n°6 CLAUDE.md) — le suffixe de DONNÉES ` [bot]` (killsource,
 * schéma 36) n'a qu'UN seul point de retrait à l'affichage : `stripBotSuffix` /
 * `displayPlayerName` dans ce dossier (`lib/players/displayName.ts`).
 *
 * Contexte (lot D, retour user 2026-09-02) : plusieurs surfaces rendaient un
 * gamertag de bot BRUT, suffixe compris (`ReplayKillFeed`, `MatchScoreboard`,
 * `MatchEncountersTable`, les fiches du rejeu…). Toutes ont été routées vers le
 * chokepoint. Ce test interdit qu'une réimplémentation locale du littéral
 * ` [bot]` (sous forme de chaîne JS/TS citée) réapparaisse ailleurs — sans quoi
 * la factorisation re-diverge à la prochaine surface qui rend un nom de bot.
 *
 * Ce que le test NE couvre PAS :
 *   - les fichiers `*.test.ts(x)`, où des fixtures portent légitimement un
 *     gamertag de bot déjà suffixé (donnée d'entrée simulée, pas une
 *     réimplémentation du retrait) — même convention que `appLinks.guard.test.ts` ;
 *   - les mentions en PROSE dans les commentaires (guillemets français « … »,
 *     jamais des guillemets droits collés au littéral) — le marqueur de donnée
 *     est documenté à plusieurs endroits (roster.go côté Go, rosterLogic.ts côté
 *     web), et cette prose n'est pas une chaîne JS exécutée.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const HERE = dirname(fileURLToPath(import.meta.url))
const WEB_SRC = join(HERE, '..', '..')

/** Seul fichier autorisé à porter le littéral cité (chaîne JS/TS, pas de la prose). */
const CANONICAL = 'lib/players/displayName.ts'

/** Le littéral CITÉ comme chaîne JS/TS : un guillemet droit collé à ` [bot]` collé
 * à un guillemet droit — `' [bot]'`, `" [bot]"` ou `` ` [bot]` ``. Une mention en
 * prose (guillemets français, texte libre) ne matche jamais ce motif. */
const QUOTED_LITERAL_RE = /['"`] \[bot\]['"`]/

function collectSourceFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'dist') continue
      out.push(...collectSourceFiles(full))
      continue
    }
    if (!/\.(ts|tsx)$/.test(entry.name)) continue
    if (/\.test\.(ts|tsx)$/.test(entry.name)) continue
    out.push(full)
  }
  return out
}

function toPosix(p: string): string {
  return p.split('\\').join('/')
}

describe('botSuffix — source unique du retrait du suffixe " [bot]"', () => {
  it('aucune réimplémentation du littéral hors de lib/players/displayName.ts', () => {
    const offenders: string[] = []
    for (const file of collectSourceFiles(WEB_SRC)) {
      const rel = toPosix(relative(WEB_SRC, file))
      if (rel === CANONICAL) continue
      const src = readFileSync(file, 'utf8')
      if (QUOTED_LITERAL_RE.test(src)) {
        offenders.push(`${rel} contient le littéral " [bot]" — importer stripBotSuffix / displayPlayerName depuis @/lib/players/displayName`)
      }
    }
    expect(offenders).toEqual([])
  })
})
