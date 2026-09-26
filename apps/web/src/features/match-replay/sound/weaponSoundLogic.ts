/**
 * weaponSoundLogic.ts — LE CALCUL DES SONS D'ARMES DU REJEU, SANS WEBAUDIO.
 *
 * Les `.wav` extraits du jeu sont PURS : un fichier, toujours le même. Deux effets que le
 * moteur du jeu applique à chaque coup sont donc rejoués ici, côté app :
 *
 * - VARIATION : le jeu déplace volume et hauteur à chaque lecture, dans une fourchette
 *   déclarée par la bank (paquet RANGED, exporté par `cmd/weapon-sounds`). Sans elle, une
 *   rafale est le même échantillon répété à l'identique, ce qui s'entend immédiatement.
 * - DISTANCE : un tir lointain est plus faible ET plus sourd. Deux réglages, pas un.
 *
 * Tout ce qui est calculable sans navigateur vit ici, en fonctions pures testées : le
 * tirage, la conversion des unités et le mapping du curseur de distance. La lecture WebAudio
 * (replayAudio.ts) ne fait qu'assembler des nœuds à partir de ces valeurs ; les fourchettes
 * par arme vivent dans `weaponSoundVariations.ts` (généré avec les sons, même livraison).
 *
 * UNITÉS. Les fourchettes du manifeste sont des OFFSETS autour de la valeur nominale du
 * fichier : décibels pour le volume, centièmes de demi-ton pour la hauteur. Un gain en dB
 * devient un facteur linéaire par 10^(dB/20) ; des centièmes deviennent un `playbackRate`
 * par 2^(cents/1200) — la lecture accélérée monte la hauteur, c'est le même effet qu'un
 * pitch shift sur un échantillon, et c'est ce que fait le moteur.
 */

/** Une fourchette du manifeste : offsets signés, `bas <= haut`. */
export interface SoundRange {
  bas: number
  haut: number
}

/** Ce que le jeu fait varier à chaque lecture. Les deux champs sont optionnels. */
export interface SoundVariation {
  volume_db?: SoundRange
  pitch_cents?: SoundRange
}

/** Ce qu'une lecture applique au fichier pur. Neutre = `{ gainDb: 0, playbackRate: 1 }`. */
export interface SoundDraw {
  gainDb: number
  playbackRate: number
}

/**
 * Chaîne de distance. `null` = NEUTRE ABSOLU : le lecteur ne doit alors insérer AUCUN nœud
 * dans le chemin du signal, pas même un gain à 1 (exigence du plan — un filtre « neutre »
 * reste un filtre, et sa réponse n'est jamais parfaitement plate).
 */
export interface DistanceChain {
  gainDb: number
  cutoffHz: number
}

/** Atténuation à 100 % de distance. Un tir à l'autre bout de la carte reste audible. */
const DISTANCE_MAX_ATTENUATION_DB = -24
/** Coupure du passe-bas à 100 %. En deçà, le tir devient un bruit sourd non identifiable. */
const DISTANCE_MIN_CUTOFF_HZ = 500
/** Coupure à distance nulle : au-dessus de la bande audible, donc sans effet. */
const DISTANCE_MAX_CUTOFF_HZ = 20_000

/** Bornes du `playbackRate` acceptées par WebAudio en pratique (± 2 octaves suffisent). */
const MIN_PLAYBACK_RATE = 0.25
const MAX_PLAYBACK_RATE = 4

/** clampPercent ramène un réglage 0-100 dans ses bornes, 0 pour une valeur illisible. */
export function clampPercent(value: number): number {
  if (!Number.isFinite(value)) return 0
  if (value <= 0) return 0
  if (value >= 100) return 100
  return value
}

/**
 * normalizeRange rend une fourchette exploitable, ou `null`.
 *
 * Une borne non finie, ou les deux à zéro, valent absence : le son se joue pur. Les bornes
 * sont réordonnées — le manifeste les livre déjà triées, mais un fichier édité à la main ne
 * doit pas produire un tirage vide.
 */
export function normalizeRange(range: SoundRange | undefined | null): SoundRange | null {
  if (!range) return null
  const { bas, haut } = range
  if (!Number.isFinite(bas) || !Number.isFinite(haut)) return null
  if (bas === 0 && haut === 0) return null
  return bas <= haut ? { bas, haut } : { bas: haut, haut: bas }
}

/**
 * drawRange tire uniformément dans la fourchette réduite par `ratio` (0..1).
 *
 * Le réglage n'est pas un interrupteur : à 50 %, le jeu varie deux fois moins. Le ratio
 * s'applique donc aux BORNES, pas au résultat — sinon un réglage bas décalerait toujours
 * dans le même sens.
 */
export function drawRange(range: SoundRange | null, ratio: number, random: () => number): number {
  if (!range || ratio <= 0) return 0
  const bas = range.bas * ratio
  const haut = range.haut * ratio
  return bas + (haut - bas) * random()
}

/**
 * drawVariation rend le gain et la vitesse de lecture d'UNE lecture.
 *
 * Sans fourchette (manifeste absent, arme non couverte, réglage à 0), rend le neutre exact :
 * aucune erreur, aucun silence — le son se joue tel quel.
 */
export function drawVariation(
  variation: SoundVariation | undefined | null,
  variationPercent: number,
  random: () => number = Math.random,
): SoundDraw {
  const ratio = clampPercent(variationPercent) / 100
  if (!variation || ratio <= 0) return { gainDb: 0, playbackRate: 1 }
  const gainDb = drawRange(normalizeRange(variation.volume_db), ratio, random)
  const cents = drawRange(normalizeRange(variation.pitch_cents), ratio, random)
  return { gainDb, playbackRate: playbackRateFromCents(cents) }
}

/** gainFromDb convertit des décibels en facteur linéaire (celui d'un GainNode). */
export function gainFromDb(db: number): number {
  if (!Number.isFinite(db)) return 1
  return Math.pow(10, db / 20)
}

/**
 * playbackRateFromCents convertit des centièmes de demi-ton en vitesse de lecture.
 * Borné : une fourchette aberrante ne doit pas rendre un son inaudible ou infini.
 */
export function playbackRateFromCents(cents: number): number {
  if (!Number.isFinite(cents) || cents === 0) return 1
  const rate = Math.pow(2, cents / 1200)
  return Math.min(MAX_PLAYBACK_RATE, Math.max(MIN_PLAYBACK_RATE, rate))
}

/**
 * distanceChain rend l'atténuation et la coupure du passe-bas pour un réglage 0-100.
 *
 * À 0 : `null`, et le lecteur branche la source DIRECTEMENT sur la sortie. Le gain décroît
 * linéairement en décibels (l'oreille entend les dB, pas les facteurs) et la coupure
 * décroît GÉOMÉTRIQUEMENT (une octave est un rapport, pas une différence) : sans cela, la
 * première moitié du curseur ne s'entendrait presque pas et la seconde d'un coup.
 */
export function distanceChain(distancePercent: number): DistanceChain | null {
  const d = clampPercent(distancePercent) / 100
  if (d <= 0) return null
  const rapport = DISTANCE_MIN_CUTOFF_HZ / DISTANCE_MAX_CUTOFF_HZ
  return {
    gainDb: DISTANCE_MAX_ATTENUATION_DB * d,
    cutoffHz: DISTANCE_MAX_CUTOFF_HZ * Math.pow(rapport, d),
  }
}
