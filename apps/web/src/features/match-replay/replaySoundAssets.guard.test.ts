/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le manifeste sonore du rejeu (replaySound.ts) et le
 * dossier d'assets `static/sounds/halo_infinite/` sont la MÊME liste, rejouée ici.
 *
 * POURQUOI. Le client ne sonde jamais le serveur pour savoir si un son existe (pas de
 * 404 exploratoire) : il croit le manifeste. Un stem sans fichier serait un silence
 * inexpliqué en production ; un fichier sans stem, un asset mort que rien ne joue. Les
 * deux dérives cassent CE test, pas l'écoute.
 *
 * LE SECOND GARDE-RAIL traverse la frontière Go/TS, et il le doit : le son d'un kill à la
 * grenade ou à la mêlée se joint par la VIGNETTE de la source de dégât (killfeed-NN), dont
 * la table d'autorité est `killicon/data/rules.tsv` côté Go. Cette table GRANDIT à chaque
 * saison. Si un index d'atlas bougeait, ou si une cinquième grenade apparaissait, la
 * jointure du son deviendrait fausse EN SILENCE — c'est-à-dire qu'on entendrait l'explosion
 * d'une autre grenade. Rejouer les cinq clés contre le fichier source rend cette dérive
 * rouge au lieu de la rendre inaudible.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import {
  EQUIPMENT_SOUND_STEMS,
  KILL_SPRITE_SOUND_STEMS,
  WEAPON_SOUND_STEMS,
  THROW_SOUND_STEMS,
} from './replaySound'

const REPO_ROOT = resolve(__dirname, '..', '..', '..', '..', '..')
const SOUNDS_DIR = resolve(REPO_ROOT, 'static', 'sounds', 'halo_infinite')
const KILLICON_RULES = resolve(
  REPO_ROOT,
  'apps/go-api/internal/games/halo_infinite/film/killicon/data/rules.tsv',
)

describe('garde-rail : manifeste sonore = dossier d assets', () => {
  const shipped = new Set(
    readdirSync(SOUNDS_DIR)
      .filter((f) => f.endsWith('.wav'))
      .map((f) => f.slice(0, -'.wav'.length)),
  )
  const referenced = new Set([
    ...Object.values(WEAPON_SOUND_STEMS),
    ...THROW_SOUND_STEMS,
    ...Object.values(KILL_SPRITE_SOUND_STEMS),
    // Les équipements (lot du 2026-08-16) : deux stems par famille mesurée.
    ...Object.values(EQUIPMENT_SOUND_STEMS).flatMap((s) => [s.activate, s.deactivate]),
  ])

  it('chaque stem du manifeste a son fichier .wav', () => {
    const missing = [...referenced].filter((s) => !shipped.has(s))
    expect(missing).toEqual([])
  })

  it('chaque fichier .wav livré est joué par un stem (0 asset mort)', () => {
    const orphans = [...shipped].filter((s) => !referenced.has(s))
    expect(orphans).toEqual([])
  })
})

/** Les lignes utiles de rules.tsv : genre, clé, vignette (colonnes 0, 1 et 2). */
function killiconRules(): { genre: string; key: string; sprite: string }[] {
  return readFileSync(KILLICON_RULES, 'utf8')
    .split('\n')
    .map((l) => l.trimEnd())
    .filter((l) => l !== '' && !l.startsWith('#') && !l.startsWith('genre\t'))
    .map((l) => l.split('\t'))
    .map(([genre, key, sprite]) => ({ genre, key, sprite }))
}

describe('garde-rail : vignettes du son de kill = table killicon (Go)', () => {
  const rules = killiconRules()

  /**
   * CE QUE CHAQUE VIGNETTE DOIT DÉSIGNER, verbatim depuis rules.tsv. GGGL N = l'entrée N de
   * la liste des grenades du jeu (0 Frag, 1 Plasma, 2 Dynamo, 3 Spike) ; CLASSE MELEE = le
   * geste partagé par tout l'arsenal, qu'aucune arme ne peut nommer.
   */
  const attendu: Record<string, { genre: string; key: string }> = {
    'killfeed-46': { genre: 'GGGL', key: '0' },
    'killfeed-47': { genre: 'GGGL', key: '1' },
    'killfeed-48': { genre: 'GGGL', key: '2' },
    'killfeed-49': { genre: 'GGGL', key: '3' },
    'killfeed-65': { genre: 'CLASSE', key: 'MELEE' },
  }

  it('la table sonore couvre exactement les vignettes attendues', () => {
    expect(Object.keys(KILL_SPRITE_SOUND_STEMS).sort()).toEqual(Object.keys(attendu).sort())
  })

  it('chaque vignette est portée par UNE SEULE règle, et par celle qu on croit', () => {
    for (const [sprite, veut] of Object.entries(attendu)) {
      const trouvees = rules.filter((r) => r.sprite === sprite)
      expect(trouvees, `vignette ${sprite}`).toHaveLength(1)
      expect({ genre: trouvees[0].genre, key: trouvees[0].key }, `vignette ${sprite}`).toEqual(veut)
    }
  })

  it('les quatre entrées gggl du jeu sont toutes servies (aucune grenade sans son)', () => {
    const gggl = rules.filter((r) => r.genre === 'GGGL').map((r) => r.key).sort()
    expect(gggl).toEqual(['0', '1', '2', '3'])
    for (const r of rules.filter((x) => x.genre === 'GGGL')) {
      expect(KILL_SPRITE_SOUND_STEMS[r.sprite], `gggl ${r.key}`).toBeTruthy()
    }
  })
})
