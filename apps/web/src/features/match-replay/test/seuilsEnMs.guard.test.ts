/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) — UN SEUIL TEMPOREL DU REJEU S'ÉCRIT EN MILLISECONDES.
 *
 * POURQUOI. Une constante « 12 images » vaut 1,2 s à 100 ms par image et 0,4 s à 33 ms : changer la
 * cadence du rejeu raccourcirait en silence chaque effet, chaque tolérance de son et chaque maintien
 * de visée. Une durée en millisecondes, convertie avec le pas du document (`msToFrames`, `frameMs`),
 * fait de la cadence un réglage. Le test `model/seuilsEnMs.test.ts` tient la valeur de chaque site ;
 * ce garde-rail tient qu'aucun ne revient en images.
 *
 * RÈGLE. Aucune constante dont le nom finit par `_FRAMES` ou `Frames` ne reçoit un littéral numérique
 * dans le code de production de `features/match-replay` (`const X_FRAMES = 12`, `const demiFrames = 1 / 64`).
 * L'allowlist est vide : ne jamais y ajouter sans justification datée.
 *
 * LIMITE. Règle lexicale sur le NOM : une durée en images nommée autrement (`HOLD`, `tolerance`) passe,
 * et les tests ne sont pas balayés (ils fixent leurs propres images d'essai). Le garde-rail ne regarde
 * ni `lib/replay/` ni `libs/film` : le décodeur tient sa cadence de son côté.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, sourcesDeLaFeature } from './featureFiles'

/** Allowlist : « chemin:NOM » toléré. VIDE. */
const ALLOWLIST = new Set<string>()

/** Une déclaration `const NOM_FRAMES = <littéral numérique…>` (ou `let`, `var`, avec un type facultatif). */
const SEUIL_EN_IMAGES =
  /\b(?:const|let|var)\s+((?:[A-Za-z0-9]+_)*FRAMES|[A-Za-z][A-Za-z0-9]*[a-z0-9]Frames)\b\s*(?::\s*number\s*)?=\s*-?\(?\s*\d/g

/** seuilsEnImages rend les noms de constantes en images initialisées par un littéral numérique. */
function seuilsEnImages(src: string): string[] {
  return [...src.matchAll(SEUIL_EN_IMAGES)].map((m) => m[1])
}

/** Les durées des seuils convertis à l'étape B (`*_MS` de la page), qui se convertissent au pas historique. */
const DUREES_DES_SEUILS = [
  'BOMB_BLAST_HOLD_MS', 'FLAG_CAPTURE_HOLD_MS', 'PROJECTILE_TAIL_MS', 'TOLERANCE_MS', 'RETOMBEE_MS', 'NATIVE_PICKUP_MATCH_MS',
]
const PAR_L_HORLOGE = new RegExp(String.raw`\bmsToFrames\(\s*(${DUREES_DES_SEUILS.join('|')})\b`, 'g')

/** conversionsParLHorloge rend les durées de seuil converties par `msToFrames` (au repli de lecture). */
function conversionsParLHorloge(src: string): string[] {
  return [...src.matchAll(PAR_L_HORLOGE)].map((m) => m[1])
}

describe('garde-rail : aucun seuil temporel du rejeu en nombre d images', () => {
  it('aucune constante *_FRAMES numérique dans le code de production', () => {
    const fautifs: string[] = []
    for (const f of sourcesDeLaFeature()) {
      for (const nom of seuilsEnImages(lire(f))) {
        const cle = `${cheminCourt(f)}:${nom}`
        if (!ALLOWLIST.has(cle)) fautifs.push(cle)
      }
    }
    expect(
      fautifs,
      `ces constantes sont des durées en images : les écrire en millisecondes (suffixe _MS) et les convertir ` +
        `avec le pas du document (msToFrames, frameMs) — [${fautifs.join(', ')}]`,
    ).toEqual([])
  })

  it('les durées des seuils convertis se passent par durationToFrames / durationFrameMs, jamais par l horloge', () => {
    // Un document sans pas se lit, pour ses durées, au pas historique : msToFrames et frameToMs(1, doc) y
    // donneraient six fois plus d'images (repli de lecture à 60 images par seconde).
    const fautifs: string[] = []
    for (const f of sourcesDeLaFeature()) {
      for (const nom of conversionsParLHorloge(lire(f))) fautifs.push(`${cheminCourt(f)}:${nom}`)
    }
    expect(
      fautifs,
      `convertir ces durées avec durationToFrames / durationFrameMs (pas historique sans frameIntervalMs) — [${fautifs.join(', ')}]`,
    ).toEqual([])
  })

  it('le détecteur de conversions voit les formes visées', () => {
    const src = [
      'msToFrames(BOMB_BLAST_HOLD_MS, doc)',
      'const t = msToFrames(RETOMBEE_MS, doc)',
      'durationToFrames(RETOMBEE_MS, doc)',
      'msToFrames(k.replayMs, doc)',
    ].join(String.fromCharCode(10))
    expect(conversionsParLHorloge(src)).toEqual(['BOMB_BLAST_HOLD_MS', 'RETOMBEE_MS'])
  })

  it('le détecteur voit les formes visées et laisse passer les autres', () => {
    const src = [
      'export const HOLD_FRAMES = 12',
      'const BLINK_PERIOD_FRAMES = 10',
      'const PAD_BETWEEN_FRAMES = 1 / 64',
      'const TOLERANCE_FRAMES: number = 5',
      'let demiFrames = 3',
      'const HOLD_MS = 1200',
      'const holdFrames = msToFrames(HOLD_MS, doc)',
      'const HOLD_FRAMES = HOLD_MS / frameMs',
      'let frames = 0',
    ].join('\n')
    expect(seuilsEnImages(src)).toEqual(['HOLD_FRAMES', 'BLINK_PERIOD_FRAMES', 'PAD_BETWEEN_FRAMES', 'TOLERANCE_FRAMES', 'demiFrames'])
  })

  it('et la feature compte bien des sources — sans quoi ce garde ne garderait rien', () => {
    expect(sourcesDeLaFeature().length).toBeGreaterThan(100)
  })
})
