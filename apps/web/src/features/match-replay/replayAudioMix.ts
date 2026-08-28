/**
 * replayAudioMix.ts — LA PISTE SONORE DE L'EXPORT, mixée HORS DU TEMPS RÉEL.
 *
 * # LA MÊME PISTE, JOUÉE AUTREMENT
 *
 * Le rejeu sonore de la page est un LECTEUR : un curseur avance avec la lecture et déclenche
 * les sons au passage (`replaySoundCursor.ts`), un par un, à l'instant réel. L'export n'a pas
 * de lecture à suivre — il connaît la piste entière d'avance et peut donc la POSER d'un coup :
 * chaque événement à son instant absolu dans un `OfflineAudioContext`, qui rend le mixage
 * complet bien plus vite que sa durée.
 *
 * LA PISTE N'EST PAS RECONSTRUITE ICI. Elle vient de `buildSoundTimeline()`, celle-là même que
 * la page joue — le mixage exporté ne peut donc pas contenir un son que le rejeu n'aurait pas
 * fait entendre, ni en manquer un.
 *
 * # CE QUI EST REPRIS À L'IDENTIQUE DU LECTEUR, ET POURQUOI
 *
 * - L'ENVELOPPE (`soundEnvelope`) : tenue pleine puis fondu de sortie. Un `stop()` sec au
 *   milieu d'une onde claque ; c'est vrai hors ligne comme en direct.
 * - LE PLAFOND DE VOIX (`SOUND_MAX_VOICES`) : sur un échange nourri, empiler vingt sources ne
 *   raconte rien de plus qu'un mur de bruit. Sans ce plafond, l'export sonnerait DIFFÉREMMENT
 *   de la page — plus fort et plus confus — alors qu'il doit en être la trace fidèle.
 * - LE TIRAGE DE VARIANTE ET LA VARIATION D'ARME : mêmes fonctions, `pickVariantStem` et
 *   `drawVariation`, qui acceptent toutes deux un générateur injectable.
 *
 * # UNE SEULE CHOSE CHANGE : LE HASARD DEVIENT REPRODUCTIBLE
 *
 * En direct, tirer une variante au hasard est ce qui empêche un geste répété de sonner comme
 * une boucle. Sur un FICHIER, le même hasard rendrait deux exports du même match subtilement
 * différents — impossible à comparer, impossible à re-livrer à l'identique. La graine est donc
 * dérivée de l'ÉVÉNEMENT lui-même (son rang et son stem) : deux exports du même match sonnent
 * pareil, et deux occurrences du même geste sonnent quand même différemment.
 */
import { drawVariation, gainFromDb, type SoundDraw } from './weaponSoundLogic'
import { SOUND_MAX_VOICES, soundEnvelope } from './replayAudio'
import { WEAPON_SOUND_VARIATIONS } from './weaponSoundVariations'
import { pickVariantStem, type ReplaySoundEvent } from './replaySoundVariants'

/** Fréquence d'échantillonnage du mixage. 48 kHz : la cadence native des assets livrés. */
export const MIX_SAMPLE_RATE = 48_000
/** Deux canaux : les sources sont mono, le conteneur et les lecteurs attendent de la stéréo. */
export const MIX_CHANNELS = 2

/** Un son retenu par le mixage : quoi jouer, quand, et avec quelle variation. */
export interface MixedSound {
  /** Instant sur l'axe du CLIP (0 = première image exportée), en millisecondes. */
  atMs: number
  /** Le fichier effectivement joué — variante déjà tirée. */
  stem: string
  draw: SoundDraw
}

/** La plage exportée, sur l'axe du rejeu. */
export interface MixBounds {
  startMs: number
  endMs: number
}

export interface MixOptions {
  /** Réglage d'instance de la variation d'arme (page admin), en pourcentage. */
  variationPercent: number
  /**
   * Les prises de FIN DE PARTIE (voix d'annonceur + fanfare). Elles n'ont pas d'instant sur la
   * piste — elles se posent sur la borne de fin, et seulement si elle est dans la plage.
   */
  endMatchStems?: readonly string[]
}

/**
 * mixSeed — une graine stable dérivée du rang et du stem de l'événement.
 *
 * DÉRIVÉE DE L'ÉVÉNEMENT, et pas d'un compteur global : un compteur ferait dépendre le tirage
 * d'un événement de TOUS ceux qui le précèdent, donc changerait tout le mixage dès qu'on borne
 * la plage autrement. Ici, exporter une manche seule fait sonner ses tirs exactement comme
 * dans l'export du match entier.
 */
export function mixSeed(index: number, stem: string): number {
  // FNV-1a 32 bits : court, sans dépendance, et bien assez dispersé pour choisir une variante.
  let h = 0x811c9dc5 ^ index
  for (let i = 0; i < stem.length; i++) {
    h ^= stem.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return h >>> 0
}

/**
 * seededRandom — un générateur uniforme reproductible (mulberry32).
 *
 * `Math.random` ne se sème pas : c'est la seule raison pour laquelle ce générateur existe. Il
 * n'a aucune prétention cryptographique, et n'en a pas besoin — il choisit entre trois prises
 * d'un même tir.
 */
export function seededRandom(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

/**
 * planAudioMix retient les événements de la plage et TIRE leur variante, de façon reproductible.
 *
 * LE RANG UTILISÉ POUR LA GRAINE EST CELUI DANS LA PISTE ENTIÈRE, pas dans la plage : c'est ce
 * qui rend le tirage indépendant du bornage (cf. `mixSeed`).
 */
export function planAudioMix(
  timeline: readonly ReplaySoundEvent[],
  bounds: MixBounds,
  options: MixOptions,
): MixedSound[] {
  const out: MixedSound[] = []
  for (let i = 0; i < timeline.length; i++) {
    const e = timeline[i]
    if (e.ms < bounds.startMs || e.ms > bounds.endMs) continue
    const rnd = seededRandom(mixSeed(i, e.stem))
    const stem = pickVariantStem(e, rnd)
    // La variation RANGED ne concerne que les ARMES : la table est indexée par stem d'arme, et
    // `drawVariation` rend le neutre exact pour tout autre stem.
    const draw = drawVariation(WEAPON_SOUND_VARIATIONS[stem], options.variationPercent, rnd)
    out.push({ atMs: e.ms - bounds.startMs, stem, draw })
  }
  // LA CONCLUSION SE POSE SUR LA BORNE DE FIN, et seulement si la plage l'atteint : un extrait
  // de milieu de match ne se termine pas sur une fanfare de victoire.
  for (const stem of options.endMatchStems ?? []) {
    // AUCUNE VARIATION ICI, comme dans le lecteur : les fourchettes sont celles des armes, une
    // réplique d'annonceur et une fanfare se jouent telles quelles.
    out.push({ atMs: bounds.endMs - bounds.startMs, stem, draw: { gainDb: 0, playbackRate: 1 } })
  }
  return out.sort((a, b) => a.atMs - b.atMs)
}

/**
 * applyVoiceCap rejoue LA MÊME comptabilité de voix que le lecteur temps réel.
 *
 * Le lecteur compte les sources vivantes et refuse celles qui dépassent `SOUND_MAX_VOICES`.
 * Hors ligne, la même règle se calcule d'avance : une source occupe une voix de son instant
 * jusqu'à la fin de son enveloppe. Sans cela, l'export empilerait tout et sonnerait plus fort
 * et plus confus que la page — mesuré sur le corpus local : 28,7 % des sources sont refusées
 * en direct sur un échange nourri.
 *
 * `durationOf` rend la durée du fichier en secondes, ou `null` s'il est absent — un asset
 * manquant est un silence, exactement comme en direct, et il n'occupe alors aucune voix.
 */
export function applyVoiceCap(
  sounds: readonly MixedSound[],
  durationOf: (stem: string) => number | null,
): MixedSound[] {
  const kept: MixedSound[] = []
  /** Instants de libération des voix occupées, en ms sur l'axe du clip. */
  let busy: number[] = []
  for (const s of sounds) {
    const seconds = durationOf(s.stem)
    if (seconds === null) continue
    busy = busy.filter((endsAt) => endsAt > s.atMs)
    if (busy.length >= SOUND_MAX_VOICES) continue
    const { stopS } = soundEnvelope(seconds)
    busy.push(s.atMs + stopS * 1000)
    kept.push(s)
  }
  return kept
}

/** Ce que le rendu hors ligne consomme : les tampons déjà décodés, par stem. */
export type MixBuffers = ReadonlyMap<string, AudioBuffer | null>

/**
 * decodeMixSources charge et décode les fichiers dont le mixage a besoin.
 *
 * UN ASSET ABSENT EST MÉMORISÉ COMME TEL (`null`), jamais retenté et jamais remonté en erreur :
 * c'est la politique du lecteur (`replayAudio.ts`), et un export ne doit pas échouer en entier
 * parce qu'un fichier manque. Le silence est propre, et il est dit une fois dans la console.
 */
export async function decodeMixSources(
  stems: Iterable<string>,
  urlOf: (stem: string) => string | undefined,
  ctx: BaseAudioContext,
): Promise<Map<string, AudioBuffer | null>> {
  const out = new Map<string, AudioBuffer | null>()
  for (const stem of stems) {
    if (out.has(stem)) continue
    const url = urlOf(stem)
    if (!url) {
      out.set(stem, null)
      continue
    }
    try {
      const res = await fetch(url)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      out.set(stem, await ctx.decodeAudioData(await res.arrayBuffer()))
    } catch (err) {
      console.warn('[replay-export] son indisponible, silence :', url, err)
      out.set(stem, null)
    }
  }
  return out
}

/**
 * scheduleMix pose toutes les sources dans un contexte hors ligne, chacune avec son enveloppe.
 *
 * C'est la transposition EXACTE de `ReplayAudioPlayer.play()` : gain de tenue, palier, fondu
 * linéaire jusqu'à zéro, et `playbackRate` quand la variation en demande un. La seule
 * différence est l'instant — absolu ici, « maintenant » là-bas.
 */
export function scheduleMix(
  ctx: BaseAudioContext,
  destination: AudioNode,
  sounds: readonly MixedSound[],
  buffers: MixBuffers,
): void {
  for (const s of sounds) {
    const buf = buffers.get(s.stem)
    if (!buf) continue
    const t0 = s.atMs / 1000
    const { fadeStartS, stopS } = soundEnvelope(buf.duration)
    const tenue = gainFromDb(s.draw.gainDb)
    const gain = ctx.createGain()
    gain.gain.setValueAtTime(tenue, t0)
    gain.gain.setValueAtTime(tenue, t0 + fadeStartS)
    gain.gain.linearRampToValueAtTime(0, t0 + stopS)
    const src = ctx.createBufferSource()
    src.buffer = buf
    if (s.draw.playbackRate !== 1) src.playbackRate.value = s.draw.playbackRate
    src.connect(gain)
    gain.connect(destination)
    src.start(t0)
    src.stop(t0 + stopS)
  }
}

/**
 * renderAudioMix rend la piste complète en un tampon.
 *
 * LA DURÉE EST CELLE DU CLIP, plus la QUEUE du dernier son : couper net à la borne trancherait
 * une fanfare de fin de partie qui dure onze secondes et qui commence, par définition, à la
 * dernière image.
 */
export async function renderAudioMix(
  sounds: readonly MixedSound[],
  buffers: MixBuffers,
  durationMs: number,
  masterGain: number,
): Promise<AudioBuffer> {
  const tailS = tailSeconds(sounds, buffers, durationMs)
  const frames = Math.max(1, Math.ceil(((durationMs / 1000 + tailS) * MIX_SAMPLE_RATE)))
  const ctx = new OfflineAudioContext(MIX_CHANNELS, frames, MIX_SAMPLE_RATE)
  const master = ctx.createGain()
  master.gain.value = Math.min(Math.max(masterGain, 0), 1)
  master.connect(ctx.destination)
  scheduleMix(ctx, master, sounds, buffers)
  return ctx.startRendering()
}

/**
 * tailSeconds — ce qui DÉPASSE la borne, et rien d'autre.
 *
 * Le calcul porte sur le dépassement et non sur la fin absolue du dernier son : additionner
 * cette dernière à la durée du clip doublerait presque la piste pour un match entier.
 */
export function tailSeconds(
  sounds: readonly MixedSound[],
  buffers: MixBuffers,
  durationMs: number,
): number {
  let lastEnd = 0
  for (const s of sounds) {
    const buf = buffers.get(s.stem)
    if (!buf) continue
    lastEnd = Math.max(lastEnd, s.atMs / 1000 + soundEnvelope(buf.duration).stopS)
  }
  return Math.max(0, lastEnd - durationMs / 1000)
}

/**
 * mixReplayAudio — LE POINT D'ENTRÉE UNIQUE : de la piste du rejeu au tampon prêt à encoder.
 *
 * Il enchaîne les quatre étapes dans le seul ordre possible : retenir et tirer (pur), décoder
 * (les durées ne se connaissent qu'après), plafonner les voix (elle en a besoin), rendre.
 *
 * `null` quand il n'y a rien à mixer — piste vide, plage sans le moindre son, ou navigateur
 * sans `OfflineAudioContext`. L'appelant sort alors un clip MUET plutôt que pas de clip.
 */
export async function mixReplayAudio(
  timeline: readonly ReplaySoundEvent[],
  bounds: MixBounds,
  options: MixOptions & { volume: number; urlOf: (stem: string) => string | undefined },
): Promise<AudioBuffer | null> {
  if (typeof OfflineAudioContext !== 'function') return null
  const planned = planAudioMix(timeline, bounds, options)
  if (planned.length === 0) return null
  const durationMs = bounds.endMs - bounds.startMs
  // UN CONTEXTE JETABLE POUR LE SEUL DÉCODAGE : `decodeAudioData` a besoin d'un contexte, mais
  // pas de celui qui rendra le mixage — dont la longueur dépend justement des durées décodées.
  const probe = new OfflineAudioContext(MIX_CHANNELS, 1, MIX_SAMPLE_RATE)
  const buffers = await decodeMixSources(
    planned.map((s) => s.stem),
    options.urlOf,
    probe,
  )
  const kept = applyVoiceCap(planned, (stem) => buffers.get(stem)?.duration ?? null)
  if (kept.length === 0) return null
  return renderAudioMix(kept, buffers, durationMs, options.volume)
}
