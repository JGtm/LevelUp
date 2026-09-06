/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : helper canonique + garde-rail le même jour) — LA VUE MATCH NE
 * CONVERTIT PLUS UNE IMAGE DE FILM EN MILLISECONDES PAR ELLE-MÊME.
 *
 * POURQUOI CE GARDE. `frame × frameIntervalMs` rend des millisecondes depuis le PREMIER
 * PAQUET DE POSITION du film, pas depuis le coup d'envoi — 3,6 à 50,8 s d'écart selon le
 * match. Écrite dans un fichier de la vue match, cette multiplication pose une seconde
 * horloge sous « Frags cumulés », qui compte, lui, depuis le coup d'envoi (`event_time_ms`).
 * C'est exactement le défaut relevé au registre du 2026-09-05 (P0-7), et il est revenu deux
 * fois : d'abord dans `_scoreCurve.ts`, puis dans `_scoreEvents.ts` écrit sur son patron.
 *
 * La conversion a désormais UN foyer, `matchClock.ts`, qui applique la soustraction. Ce test
 * échoue si un fichier de `features/match-view/` la réécrit.
 *
 * PÉRIMÈTRE. `features/match-replay/` en est exclu à dessein : le rejeu 2D vit sur l'axe du
 * FILM (ses images sont son unité de travail) et convertit légitimement par `frameToMs`.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

/**
 * La signature de la conversion : une multiplication ou une division dont `frameIntervalMs`
 * est un opérande. Les deux sens sont couverts ; une étoile de bloc de commentaire (` * `
 * en tête de ligne) ne matche pas, faute d'opérande à sa gauche.
 *
 * LA LECTURE QUALIFIÉE COMPTE AUTANT QUE LA LECTURE NUE (2026-09-06, revue R1, constat C6).
 * Le premier motif exigeait que `frameIntervalMs` SUIVE immédiatement l'opérateur : il
 * attrapait la forme historique (`p.t * frameIntervalMs`, champ destructuré) et laissait
 * passer `paliers[0].t * clock.frameIntervalMs` — la forme que le nouveau code rend
 * naturelle, l'horloge étant devenue un objet. Le préfixe optionnel `<objet>.` (ou `?.`)
 * ferme cet angle mort des deux côtés de l'opérateur.
 */
const QUALIFIE = String.raw`(?:[A-Za-z0-9_$)\]]+\s*\??\.\s*)?`
const CONVERSION = [
  new RegExp(String.raw`[A-Za-z0-9_)\]]\s*[*/]\s*` + QUALIFIE + 'frameIntervalMs'),
  new RegExp('frameIntervalMs' + String.raw`\s*[*/]\s*` + QUALIFIE + String.raw`[A-Za-z0-9_([]`),
]

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

function porteLaConversion(fichier: string): boolean {
  const src = readFileSync(fichier, 'utf8')
  return CONVERSION.some((re) => re.test(src))
}

describe('garde-rail : une seule conversion image -> horloge du gameplay', () => {
  it('aucun fichier de features/match-view/ ne convertit par frameIntervalMs', () => {
    const matchView = resolve(__dirname, '..', '..', 'features', 'match-view')
    expect(walk(matchView).filter(porteLaConversion)).toEqual([])
  })

  it('et matchClock.ts, lui, la porte bien — sans quoi ce test ne garderait rien', () => {
    expect(porteLaConversion(join(__dirname, 'matchClock.ts'))).toBe(true)
  })
})
