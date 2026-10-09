/// <reference types="node" />
/**
 * endMatchVoiceCap.witness.test.tsx — LA FIN DE PARTIE SUR UN MATCH RÉEL, JOUÉE JUSQU'À LA BORNE.
 *
 * # CE QUE CE TEST PROUVE (item 11 du backlog, lot A1 du plan du 2026-09-26)
 *
 * L'utilisateur entendait la voix de l'annonceur en fin de rejeu, voyait le résultat, mais
 * n'entendait plus la fanfare. Mécanisme établi ici, de façon déterministe, sur le témoin
 * `000d5950` (Slayer, fin dense) joué à `SOUND_MAX_SPEED` : les derniers tirs occupent sept des
 * huit voix du lecteur à l'arrivée sur la borne ; la voix de l'annonceur, jouée en premier,
 * prend la huitième ; la musique, jouée en second, trouvait le plafond plein et était refusée EN
 * SILENCE. À 1×, la même fin laisse trois ou quatre voix occupées (relevé du 2026-09-26, dix
 * graines) : la conclusion y jouait déjà entière.
 *
 * # COMMENT
 *
 * - La piste est la VRAIE : le document produit par Go (chargé par `goFixtures`, jamais par un
 *   nom de fichier), passé par la frontière (`testReplayDoc`, seule porte admise par
 *   `testDoc.guard.test.ts`), puis par `useReplaySound` tel que la page le monte.
 * - Les durées sont les VRAIES : le `fetch` rend les octets des WAV livrés et le décodage lit
 *   leur en-tête RIFF. Une voix tient donc exactement ce qu'elle tient dans un navigateur.
 * - L'horloge AVANCE : le contexte audio de ce test porte un `currentTime` que la simulation
 *   pousse à 60 images par seconde d'horloge murale (le film avance de `vitesse` fois autant),
 *   et il déclenche la fin (`onended`) de chaque source à l'heure d'arrêt qu'on lui a
 *   programmée — c'est ce qui libère une voix en vrai.
 * - Le hasard est SEMÉ (variantes, variation d'arme, prise de la voix de fin) : deux passages
 *   rendent les mêmes chiffres. À 2×, le nombre de voix occupées à la borne dépend du tirage :
 *   7 sur 10 graines relevées tiennent sept voix (le défaut), 3 en tiennent six (tout passait).
 *   `GRAINE` est l'une des sept.
 * - La borne de fin est celle que la page calcule (`replayWindow`) avec l'en-tête MESURÉ du
 *   match (t0 = 18 465 ms, 478 s jouables, relevé du 2026-08-26, cf. `replayWindow.test.ts`).
 *
 * LIMITE ÉCRITE : la fixture ne porte pas les kills (ils viennent de la vue match, pas du
 * film). Leurs sons s'ajouteraient à ceux des tirs : les voix occupées relevées ici sont un
 * MINIMUM.
 */
import { readFileSync } from 'node:fs'
import { basename, resolve } from 'node:path'

import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { frameToMs } from '@/lib/replay/replayLogic'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { replayWindow } from '../model/replayWindow'
import { FakeContext, FakeSource, flushAudio } from '../test/fakeAudio'
import { racineDuDepot } from '../test/featureFiles'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { testReplayDoc } from '../test/testDoc'
import type { EndMatchSoundSpec } from './endMatchSound'
import { SOUND_MAX_VOICES } from './replayAudio'
import { seededRandom } from './replayAudioMix'
import { SOUND_MAX_SPEED } from './replaySoundCursor'
import { useReplaySound } from './useReplaySound'

/** Le témoin dense : Slayer sur Cliffhanger, JGtm équipe 0. */
const TEMOIN = '000d5950'
/** L'en-tête mesuré du témoin (cf. `replayWindow.test.ts`). */
const TEMOIN_HEADER = { t0_ms: 18_465, playable_duration_seconds: 478 }
const SOUNDS_DIR = resolve(racineDuDepot(), 'static', 'sounds', 'halo_infinite')
const VICTOIRE_FR: EndMatchSoundSpec = { outcome: 'win', ffa: false, locale: 'fr' }
/** Pas de la boucle d'animation simulée : 60 images par seconde. */
const PAS_MS = 1000 / 60
/** Graine du hasard : l'une de celles où la fin à 2× tient sept voix (cf. l'en-tête). */
const GRAINE = 7919

/** Un tampon « décodé » : sa durée réelle, et le fichier dont il vient (pour nommer). */
interface TamponTemoin {
  duration: number
  fichier: string
}

/** Le contexte à HORLOGE QUI AVANCE : `ended` tombe à l'heure d'arrêt programmée. */
class HorlogeContext extends FakeContext {
  finies = new Set<FakeSource>()
  private octetsVersFichier = new WeakMap<ArrayBuffer, string>()

  noter(raw: ArrayBuffer, fichier: string) { this.octetsVersFichier.set(raw, fichier) }

  override decodeAudioData(raw: ArrayBuffer) {
    const bytes = Buffer.from(raw)
    return Promise.resolve({ duration: dureeWav(bytes), fichier: this.octetsVersFichier.get(raw) ?? '?' } as unknown as AudioBuffer)
  }

  /** Avance l'horloge et termine chaque source dont l'arrêt programmé est atteint. */
  avancer(t: number) {
    this.currentTime = t
    for (const s of this.sources) {
      if (this.finies.has(s) || s.stopped === null || s.stopped > t) continue
      this.finies.add(s)
      s.end()
    }
  }

  vivantes(): FakeSource[] {
    return this.sources.filter((s) => s.started !== null && !this.finies.has(s))
  }
}

/** Durée d'un WAV RIFF : taille du chunk `data` / débit du chunk `fmt `. */
function dureeWav(bytes: Buffer): number {
  let debit = 0
  let i = 12
  while (i + 8 <= bytes.length) {
    const id = bytes.subarray(i, i + 4).toString('ascii')
    const taille = bytes.readUInt32LE(i + 4)
    if (id === 'fmt ') debit = bytes.readUInt32LE(i + 16)
    if (id === 'data') return debit > 0 ? taille / debit : 0
    i += 8 + taille + (taille % 2)
  }
  return 0
}

function fichierDe(s: FakeSource): string {
  return (s.buffer as unknown as TamponTemoin | null)?.fichier ?? '?'
}

let ctx: HorlogeContext

beforeEach(() => {
  ctx = new HorlogeContext()
  vi.stubGlobal('AudioContext', function AudioContextTemoin() { return ctx })
  vi.stubGlobal('fetch', vi.fn((url: string) => {
    const nom = basename(String(url))
    try {
      const b = readFileSync(resolve(SOUNDS_DIR, nom))
      const raw = b.buffer.slice(b.byteOffset, b.byteOffset + b.byteLength) as ArrayBuffer
      ctx.noter(raw, nom)
      return Promise.resolve({ ok: true, arrayBuffer: () => Promise.resolve(raw) })
    } catch {
      return Promise.resolve({ ok: false, status: 404 })
    }
  }))
  vi.spyOn(Math, 'random').mockImplementation(seededRandom(GRAINE))
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function docTemoin(): ReplayDocumentReady {
  const entry = goFixtureEntries().find((e) => e.film === TEMOIN)
  if (!entry) throw new Error(`fixture ${TEMOIN} absente du manifeste`)
  return testReplayDoc(loadGoFixture(entry).doc)
}

/** Ce que la borne a vu : les voix occupées juste avant la conclusion, et ce qui en a joué. */
interface Borne {
  occupees: number
  conclusion: string[]
  releve: string
}

/**
 * Joue le témoin du coup d'envoi à la borne, à `speed`, comme la boucle d'animation le bat
 * (60 images par seconde d'horloge murale), puis appelle la conclusion.
 */
async function jouerJusquALaBorne(speed: number): Promise<Borne> {
  const doc = docTemoin()
  const fenetre = replayWindow(doc, TEMOIN_HEADER)
  if (!fenetre) throw new Error('fenêtre de jeu illisible sur le témoin')
  const { result } = renderHook(() =>
    useReplaySound(doc, [], speed, { allegiance: NO_ALLEGIANCE, endMatch: VICTOIRE_FR, locale: 'fr' }),
  )
  act(() => result.current.toggle())
  await act(async () => { await flushAudio() })
  const debutMs = frameToMs(fenetre.startFrame, doc)
  const finMs = frameToMs(fenetre.endFrame, doc)
  const t0 = ctx.currentTime
  for (let ms = debutMs; ; ms = Math.min(ms + PAS_MS * speed, finMs)) {
    ctx.avancer(t0 + (ms - debutMs) / 1000 / speed)
    result.current.tick(ms)
    if (ms >= finMs) break
  }
  const occupees = ctx.vivantes().length
  const avant = ctx.sources.length
  act(() => result.current.endMatch())
  const conclusion = ctx.sources.slice(avant).map(fichierDe)
  const releve = `vitesse ${speed}x — voix occupées à la borne : ${occupees}/${SOUND_MAX_VOICES}, ` +
    `sources tirées : ${avant}, conclusion jouée : [${conclusion.join(', ')}]`
  return { occupees, conclusion, releve }
}

describe(`fin de partie sur le témoin réel ${TEMOIN} (item 11)`, () => {
  /**
   * LE CAS DU DÉFAUT. À la vitesse la plus haute où le son joue encore (`SOUND_MAX_SPEED`), les
   * derniers tirs tiennent sept voix à la borne : la réplique prend la huitième, et la fanfare
   * était refusée par le plafond. Relevé du 2026-09-26 avant correctif : 7/8 occupées, conclusion
   * jouée = la voix seule.
   */
  it(`à ${SOUND_MAX_SPEED}x, plafond presque plein à la borne : la voix ET la fanfare partent`, async () => {
    const b = await jouerJusquALaBorne(SOUND_MAX_SPEED)
    expect(b.occupees, b.releve).toBe(SOUND_MAX_VOICES - 1)
    expect(b.conclusion.some((f) => f.startsWith('end_victory_voice_fr_')), b.releve).toBe(true)
    expect(b.conclusion, b.releve).toContain('end_victory_music_01.wav')
  })

  /** À 1×, la fin du témoin laisse de la place : la conclusion jouait déjà entière, et le reste. */
  it('à 1x, la conclusion joue entière, comme avant le correctif', async () => {
    const b = await jouerJusquALaBorne(1)
    expect(b.occupees, b.releve).toBeLessThan(SOUND_MAX_VOICES - 1)
    expect(b.conclusion, b.releve).toHaveLength(2)
    expect(b.conclusion, b.releve).toContain('end_victory_music_01.wav')
  })
})
