/**
 * weaponSoundPlayer.test.ts — LE CHEMIN DU SIGNAL, TESTÉ SANS NAVIGATEUR.
 *
 * Même principe que `canvasRecording.test.ts` : le lecteur n'INTERROGE jamais WebAudio, il
 * y branche des nœuds. Un contexte ENREGISTREUR — qui note les nœuds créés et les
 * connexions faites — suffit donc à observer exactement ce qui sera dans le chemin du
 * signal. Aucune dépendance, aucun navigateur.
 *
 * CE QUE CES TESTS PROTÈGENT. L'exigence « à 0 % de distance, AUCUN nœud » n'est vérifiable
 * qu'ici : à l'oreille, un GainNode à 1 de trop est inaudible — jusqu'au jour où l'on
 * compare le rendu de l'app au fichier extrait et où l'on cherche pendant des heures d'où
 * vient l'écart.
 */
import { describe, expect, it } from 'vitest'

import {
  DEFAULT_WEAPON_SOUND_SETTINGS,
  fetchWeaponSoundManifest,
  manifestURL,
  soundURL,
  WeaponSoundPlayer,
} from './weaponSoundPlayer'
import type { WeaponSoundManifest } from './weaponSoundLogic'

interface FauxNoeud {
  kind: string
  connect: (cible: FauxNoeud) => void
  [k: string]: unknown
}

/** Trace : les nœuds créés, et les connexions dans l'ordre où elles ont été faites. */
interface Trace {
  crees: string[]
  liens: string[]
  demarrages: number
  ctx: AudioContext
}

function recordingAudioContext(): Trace {
  const trace: Trace = { crees: [], liens: [], demarrages: 0, ctx: null as unknown as AudioContext }
  const destination: FauxNoeud = { kind: 'destination', connect: () => {} }
  const noeud = (kind: string, extra: Record<string, unknown>): FauxNoeud => {
    trace.crees.push(kind)
    return {
      kind,
      ...extra,
      connect: (cible: FauxNoeud) => trace.liens.push(`${kind}->${cible.kind}`),
    }
  }
  trace.ctx = {
    destination,
    createBufferSource: () =>
      noeud('source', {
        buffer: null,
        playbackRate: { value: 1 },
        start: () => {
          trace.demarrages++
        },
      }),
    createGain: () => noeud('gain', { gain: { value: 1 } }),
    createBiquadFilter: () => noeud('lowpass', { type: '', frequency: { value: 0 } }),
    decodeAudioData: async () => ({ duration: 1 }) as unknown as AudioBuffer,
  } as unknown as AudioContext
  return trace
}

const MANIFESTE: WeaponSoundManifest = {
  source: 'test',
  sons: [
    { arme: 'nue', fichier: 'nue.wav' },
    {
      arme: 'variable',
      fichier: 'variable.wav',
      variation: { volume_db: { bas: -6, haut: -6 }, pitch_cents: { bas: 1200, haut: 1200 } },
    },
  ],
}

/** fetch factice : rend le manifeste en JSON et des octets pour n'importe quel `.wav`. */
function fauxFetch(ok = true): typeof fetch {
  return (async () => ({
    ok,
    json: async () => MANIFESTE,
    arrayBuffer: async () => new ArrayBuffer(8),
  })) as unknown as typeof fetch
}

async function lecteurPret(trace: Trace, settings = DEFAULT_WEAPON_SOUND_SETTINGS, random = () => 0.5) {
  const player = new WeaponSoundPlayer(trace.ctx, settings, random)
  player.setManifest(MANIFESTE)
  await player.preload(['nue', 'variable'], fauxFetch())
  return player
}

describe('URLs', () => {
  it('passent par le helper d’assets, jamais par un /static/ en dur', () => {
    expect(manifestURL()).toBe('/static/weapons-assets/halo_infinite/sons/index.json')
    expect(soundURL('MA40_RAFALE_3p.wav')).toBe(
      '/static/weapons-assets/halo_infinite/sons/MA40_RAFALE_3p.wav',
    )
  })
})

describe('chargement du manifeste', () => {
  it('un manifeste absent rend null, sans jeter — c’est l’état courant du chantier', async () => {
    const absent = (async () => ({ ok: false })) as unknown as typeof fetch
    expect(await fetchWeaponSoundManifest(absent)).toBeNull()
    const casse = (async () => {
      throw new Error('réseau')
    }) as unknown as typeof fetch
    expect(await fetchWeaponSoundManifest(casse)).toBeNull()
  })

  it('un JSON sans tableau `sons` est refusé plutôt qu’à moitié accepté', async () => {
    const bancal = (async () => ({ ok: true, json: async () => ({ source: 'x' }) })) as unknown as typeof fetch
    expect(await fetchWeaponSoundManifest(bancal)).toBeNull()
  })

  it('un manifeste valide est indexé par arme', async () => {
    const manifeste = await fetchWeaponSoundManifest(fauxFetch())
    const player = new WeaponSoundPlayer(recordingAudioContext().ctx)
    player.setManifest(manifeste)
    expect(player.armes()).toEqual(['nue', 'variable'])
  })
})

describe('chemin du signal', () => {
  it('réglages par défaut et son sans fourchette : source -> destination, AUCUN nœud de plus', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace)
    expect(player.play('nue')).toBe(true)
    expect(trace.crees).toEqual(['source'])
    expect(trace.liens).toEqual(['source->destination'])
    expect(trace.demarrages).toBe(1)
  })

  it('variation active : un gain, et toujours pas de filtre à distance nulle', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace)
    expect(player.play('variable')).toBe(true)
    expect(trace.crees).toEqual(['source', 'gain'])
    expect(trace.liens).toEqual(['source->gain', 'gain->destination'])
  })

  it('variation à 0 % : le son à fourchette redevient parfaitement neutre', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace, { variationPercent: 0, distancePercent: 0 })
    expect(player.play('variable')).toBe(true)
    expect(trace.crees).toEqual(['source'])
    expect(trace.liens).toEqual(['source->destination'])
  })

  it('distance > 0 : gain PUIS passe-bas, dans cet ordre', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace, { variationPercent: 0, distancePercent: 50 })
    expect(player.play('nue')).toBe(true)
    expect(trace.crees).toEqual(['source', 'gain', 'lowpass'])
    expect(trace.liens).toEqual(['source->gain', 'gain->lowpass', 'lowpass->destination'])
  })

  it('les deux gains sont additionnés en dB : un seul GainNode, jamais deux en série', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace, { variationPercent: 100, distancePercent: 100 })
    expect(player.play('variable')).toBe(true)
    expect(trace.crees.filter((n) => n === 'gain')).toHaveLength(1)
  })
})

describe('absence de son', () => {
  it('une arme hors manifeste ne joue rien et ne crée aucun nœud', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace)
    expect(player.play('inconnue')).toBe(false)
    expect(trace.crees).toEqual([])
  })

  it('une arme non préchargée ne joue rien : un son en retard est pire qu’un son absent', () => {
    const trace = recordingAudioContext()
    const player = new WeaponSoundPlayer(trace.ctx)
    player.setManifest(MANIFESTE)
    expect(player.play('nue')).toBe(false)
    expect(trace.crees).toEqual([])
  })

  it('un fichier illisible est ignoré, le rejeu continue', async () => {
    const trace = recordingAudioContext()
    const player = new WeaponSoundPlayer(trace.ctx)
    player.setManifest(MANIFESTE)
    expect(await player.preload(['nue'], fauxFetch(false))).toBe(0)
    expect(player.play('nue')).toBe(false)
  })

  it('sans manifeste du tout, le lecteur reste muet et sain', async () => {
    const trace = recordingAudioContext()
    const player = new WeaponSoundPlayer(trace.ctx)
    player.setManifest(null)
    expect(player.armes()).toEqual([])
    expect(await player.preload(['nue'], fauxFetch())).toBe(0)
    expect(player.play('nue')).toBe(false)
  })
})

describe('réglages', () => {
  it('setSettings s’applique à la lecture suivante', async () => {
    const trace = recordingAudioContext()
    const player = await lecteurPret(trace, { variationPercent: 0, distancePercent: 0 })
    player.play('variable')
    expect(trace.crees).toEqual(['source'])
    player.setSettings({ variationPercent: 100, distancePercent: 0 })
    player.play('variable')
    expect(trace.crees).toEqual(['source', 'source', 'gain'])
  })

  it('les valeurs d’usine sont « variation du jeu telle quelle, aucune distance »', () => {
    expect(DEFAULT_WEAPON_SOUND_SETTINGS).toEqual({ variationPercent: 100, distancePercent: 0 })
  })
})
