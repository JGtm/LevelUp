/**
 * replaySound.ts — CE QUI SONNE, QUAND, ET COMMENT LE CURSEUR NE REJOUE RIEN DEUX FOIS.
 *
 * LA SOURCE (lot 5, plan parité) : un pack de WAV fourni par l'utilisateur (2026-08-13),
 * rangé sous `static/sounds/halo_infinite/` et nommé par weapon_key — la clé canonique du
 * registre d'armes (weapon_names.toml), PAS le nom de fichier FR des images (piège
 * Crémateur/Cindershot). Le manifeste ci-dessous est la liste EXACTE des fichiers livrés ;
 * le garde-rail `replaySoundAssets.guard.test.ts` le rejoue contre le dossier : un stem
 * sans fichier ou un fichier sans stem casse le test, jamais l'écoute.
 *
 * CE QUI DÉCLENCHE UN SON, ET RIEN D'AUTRE :
 *  - les KILLS du fil (weapon_key présent ET dans le manifeste) — l'horloge est celle du
 *    fil (`alignFeedToTracks`), la même qui date le flash des fiches et l'effet de mort :
 *    un son qui partirait sur l'horloge brute sonnerait à côté de son image ;
 *  - les LANCERS de grenade (doc.grenades, l'auteur est écrit dans le film), par TYPE —
 *    le pack porte les quatre lancers (frag/plasma/dynamo/spike, item 5.3) ;
 *  - un kill À LA grenade sonne l'explosion (c'est elle qui a tué, pas le geste du
 *    lancer) — la Spike n'a pas de weapon_key (mesure killicon) : son kill reste muet.
 *
 * UN KILL SANS weapon_key (mêlée générique, objets) OU UNE ARME SANS FICHIER (Bandit,
 * MA5K, SPNKr à combustible, Vestige — absentes du pack) = SILENCE PROPRE : jamais le son
 * d'une arme voisine, même règle que les effets de rendu (replay_labels.toml).
 *
 * Pas de React, pas de Web Audio ici : logique pure, testée (replaySound.test.ts).
 * La lecture (AudioContext, enveloppe de gain) vit dans replayAudio.ts.
 */
import type { KillEvent } from '@/features/match-view/_momentum'

import { alignFeedToTracks } from './killFeedLogic'
import { frameToMs } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/**
 * Son de KILL par weapon_key -> stem de fichier sous static/sounds/{slug}/.
 *
 * Les armes portent leur propre stem (fichier nommé par la clé) ; les trois grenades à
 * weapon_key pointent l'explosion PARTAGÉE — un seul fichier, pas trois copies (règle
 * « ≤ 2 copies »). Une clé absente de cette table = silence propre.
 */
export const KILL_SOUND_STEMS: Readonly<Record<string, string>> = {
  hinf_ma40_ar: 'hinf_ma40_ar',
  hinf_br75: 'hinf_br75',
  hinf_cqs48_bulldog: 'hinf_cqs48_bulldog',
  hinf_cindershot: 'hinf_cindershot',
  hinf_vk78_commando: 'hinf_vk78_commando',
  hinf_disruptor: 'hinf_disruptor',
  hinf_energy_sword: 'hinf_energy_sword',
  hinf_gravity_hammer: 'hinf_gravity_hammer',
  hinf_heatwave: 'hinf_heatwave',
  hinf_hydra: 'hinf_hydra',
  hinf_mangler: 'hinf_mangler',
  hinf_needler: 'hinf_needler',
  hinf_plasma_pistol: 'hinf_plasma_pistol',
  hinf_pulse_carbine: 'hinf_pulse_carbine',
  hinf_ravager: 'hinf_ravager',
  hinf_s7_sniper: 'hinf_s7_sniper',
  hinf_sentinel_beam: 'hinf_sentinel_beam',
  hinf_shock_rifle: 'hinf_shock_rifle',
  hinf_sidekick: 'hinf_sidekick',
  hinf_skewer: 'hinf_skewer',
  hinf_m41_spnkr: 'hinf_m41_spnkr',
  hinf_stalker_rifle: 'hinf_stalker_rifle',
  hinf_frag_grenade: 'explosion',
  hinf_plasma_grenade: 'explosion',
  hinf_dynamo_grenade: 'explosion',
}

/**
 * Son de LANCER par rang de grenade (l'index EST le rang, même règle que grenadeLabels :
 * 0 Frag, 1 Plasma, 2 Dynamo, 3 Spike — replay_labels.toml). Un rang hors table = silence.
 */
export const THROW_SOUND_STEMS: readonly string[] = [
  'throw_frag',
  'throw_plasma',
  'throw_dynamo',
  'throw_spike',
]

/** Un événement sonore posé sur l'horloge du rejeu. */
export interface ReplaySoundEvent {
  /** Instant en ms sur l'horloge du rejeu (celle du fil et des fiches). */
  ms: number
  /** Stem du fichier sous static/sounds/{titleSlug}/. */
  stem: string
}

/**
 * buildSoundTimeline précalcule la piste sonore du document : kills recalés sur les
 * pistes (même règle que le fil et l'effet de mort — une seule horloge) + lancers de
 * grenade datés par leur frame de film. Triée chronologiquement, construite une fois.
 */
export function buildSoundTimeline(
  doc: ReplayDocumentReady,
  kills: KillEvent[],
  t0Ms: number,
): ReplaySoundEvent[] {
  const out: ReplaySoundEvent[] = []
  if (kills.length > 0 && doc.tracks.length > 0) {
    for (const k of alignFeedToTracks(kills, t0Ms, doc).kills) {
      const stem = k.weaponKey ? KILL_SOUND_STEMS[k.weaponKey] : undefined
      if (stem) out.push({ ms: k.replayMs, stem })
    }
  }
  for (const g of doc.grenades) {
    const stem = THROW_SOUND_STEMS[g.rank ?? -1]
    if (stem) out.push({ ms: frameToMs(g.t, doc), stem })
  }
  return out.sort((a, b) => a.ms - b.ms)
}

/**
 * Saut au-delà duquel une avance n'est PAS une lecture continue mais un déplacement
 * (scrub, retour au début, reprise après pause longue) : le curseur se RECALE sans rien
 * jouer — rejouer d'un coup tous les sons enjambés ferait un mur de bruit. À 4x, un pas
 * d'animation avance de ~70 ms de rejeu : 1 s de marge ne peut pas confondre les deux.
 */
export const SOUND_RESYNC_JUMP_MS = 1_000

/**
 * Vitesse de lecture au-delà de laquelle le son se TAIT.
 *
 * POURQUOI. Un son tient ~1 s de temps réel quelle que soit la vitesse : à 4×, cette
 * seconde couvre 4 s de match, et les éliminations d'un même échange (2 à 4 s d'écart) se
 * recouvrent en permanence. Ce qu'on entendrait alors n'est plus le rythme du match mais
 * celui du lecteur. À 2×, elles restent distinctes — c'est la borne.
 */
export const SOUND_MAX_SPEED = 2

/** soundPlaysAtSpeed — le son a-t-il un sens à cette vitesse de lecture ? */
export function soundPlaysAtSpeed(multiplier: number): boolean {
  return multiplier <= SOUND_MAX_SPEED
}

/** Le curseur de lecture sonore : dernier instant servi, index du prochain événement. */
export interface SoundCursor {
  ms: number
  idx: number
}

/** resyncSoundCursor pose le curseur À l'instant donné : rien avant lui ne jouera. */
export function resyncSoundCursor(timeline: ReplaySoundEvent[], ms: number): SoundCursor {
  // Premier index strictement postérieur à `ms` (recherche binaire, timeline triée).
  let lo = 0
  let hi = timeline.length
  while (lo < hi) {
    const mid = (lo + hi) >> 1
    if (timeline[mid].ms <= ms) lo = mid + 1
    else hi = mid
  }
  return { ms, idx: lo }
}

/**
 * advanceSoundCursor avance le curseur à `nowMs` et rend les événements à jouer.
 *
 * Lecture continue (avance courte) : tout événement dans (cursor.ms, nowMs] part UNE
 * fois. Recul ou saut long : recalage silencieux — le son accompagne la lecture, il ne
 * raconte pas ce qu'on a enjambé.
 */
export function advanceSoundCursor(
  timeline: ReplaySoundEvent[],
  cursor: SoundCursor,
  nowMs: number,
): { cursor: SoundCursor; fire: ReplaySoundEvent[] } {
  if (nowMs < cursor.ms || nowMs - cursor.ms > SOUND_RESYNC_JUMP_MS) {
    return { cursor: resyncSoundCursor(timeline, nowMs), fire: [] }
  }
  const fire: ReplaySoundEvent[] = []
  let idx = cursor.idx
  while (idx < timeline.length && timeline[idx].ms <= nowMs) {
    fire.push(timeline[idx])
    idx++
  }
  return { cursor: { ms: nowMs, idx }, fire }
}
