/**
 * weaponSoundPlayer.ts — LE LECTEUR WEBAUDIO DES SONS D'ARMES DU REJEU 2D.
 *
 * Il n'assemble que des nœuds : tout le calcul est dans `weaponSoundLogic.ts`, pur et testé.
 *
 * LE CHEMIN DU SIGNAL EST MINIMAL PAR CONSTRUCTION. Un nœud n'est créé que s'il a quelque
 * chose à faire :
 *
 *	variation nulle ET distance à 0  ->  source -> destination        (neutre absolu)
 *	gain à appliquer                 ->  source -> gain -> …
 *	distance > 0                     ->  … -> passe-bas -> destination
 *
 * C'est une exigence du plan, et elle n'est pas cosmétique : un GainNode à 1 et un filtre
 * « neutre » ne sont jamais parfaitement transparents, et le réglage par défaut (distance à
 * 0 %) doit rendre le fichier extrait TEL QUEL.
 *
 * ABSENCE DE MANIFESTE = LECTURE PURE, PAS UNE ERREUR. Les `.wav` et leur `index.json`
 * n'existent pas encore (la livraison attend la fin du tri du chantier sons) : le lecteur
 * doit se charger, ne rien trouver, et rester silencieux sans jeter.
 */
import { staticAssetURL } from '@/lib/staticAssets'

import {
  distanceChain,
  drawVariation,
  gainFromDb,
  indexManifest,
  type WeaponSoundEntry,
  type WeaponSoundManifest,
} from './weaponSoundLogic'

/** Dossier des sons, miroir du sous-dossier `jeu/` des icônes extraites du jeu. */
const SOUND_DIR = 'sons'

/** Réglages d'instance, servis par la page d'admin (0-100 chacun). */
export interface WeaponSoundSettings {
  variationPercent: number
  distancePercent: number
}

/** Réglages d'usine : variation du jeu telle quelle, aucune distance. */
export const DEFAULT_WEAPON_SOUND_SETTINGS: WeaponSoundSettings = {
  variationPercent: 100,
  distancePercent: 0,
}

/** manifestURL compose l'URL du manifeste via le helper d'assets (jamais de `/static/` en dur). */
export function manifestURL(titleSlug?: string): string {
  return staticAssetURL('weapon', `${SOUND_DIR}/index`, '.json', titleSlug)
}

/** soundURL compose l'URL d'un `.wav` du manifeste. `fichier` porte déjà son extension. */
export function soundURL(fichier: string, titleSlug?: string): string {
  return staticAssetURL('weapon', `${SOUND_DIR}/${fichier}`, '', titleSlug)
}

/**
 * fetchWeaponSoundManifest lit le manifeste, ou rend `null`.
 *
 * `null` couvre les deux cas indiscernables ET équivalents pour le lecteur : le fichier
 * n'existe pas encore, ou le réseau a échoué. Dans les deux cas la conduite est la même —
 * aucun son, aucune exception, le rejeu continue.
 */
export async function fetchWeaponSoundManifest(
  fetchImpl: typeof fetch = fetch,
  titleSlug?: string,
): Promise<WeaponSoundManifest | null> {
  try {
    const res = await fetchImpl(manifestURL(titleSlug))
    if (!res.ok) return null
    const data = (await res.json()) as WeaponSoundManifest
    return Array.isArray(data?.sons) ? data : null
  } catch {
    return null
  }
}

/**
 * WeaponSoundPlayer joue un son d'arme par identifiant d'arme.
 *
 * L'`AudioContext` est fourni par l'appelant : un navigateur n'en autorise la création
 * qu'après un geste de l'utilisateur, et c'est au composant qui porte le bouton de lecture
 * de décider quand. Le lecteur, lui, reste testable avec un contexte factice.
 */
export class WeaponSoundPlayer {
  private readonly ctx: AudioContext
  private readonly random: () => number
  private settings: WeaponSoundSettings
  private entries = new Map<string, WeaponSoundEntry>()
  private buffers = new Map<string, AudioBuffer>()

  constructor(
    ctx: AudioContext,
    settings: WeaponSoundSettings = DEFAULT_WEAPON_SOUND_SETTINGS,
    random: () => number = Math.random,
  ) {
    this.ctx = ctx
    this.settings = settings
    this.random = random
  }

  /** setSettings applique les réglages d'admin ; la lecture suivante en tient compte. */
  setSettings(settings: WeaponSoundSettings): void {
    this.settings = settings
  }

  /** setManifest indexe les entrées. Un manifeste absent laisse le lecteur muet et sain. */
  setManifest(manifest: WeaponSoundManifest | null): void {
    this.entries = indexManifest(manifest)
  }

  /** armes rend les clés couvertes par le manifeste, dans l'ordre d'indexation. */
  armes(): string[] {
    return [...this.entries.keys()]
  }

  /**
   * preload télécharge et décode les sons des armes demandées.
   *
   * Le décodage est fait AVANT la lecture : décoder au moment du tir ajouterait une latence
   * variable, et un son de tir en retard est pire qu'un son absent. Une arme inconnue ou un
   * fichier illisible est ignoré — le rejeu ne s'arrête pas pour un son.
   */
  async preload(armes: string[], fetchImpl: typeof fetch = fetch, titleSlug?: string): Promise<number> {
    let charges = 0
    for (const arme of armes) {
      const entry = this.entries.get(arme)
      if (!entry || this.buffers.has(entry.fichier)) continue
      try {
        const res = await fetchImpl(soundURL(entry.fichier, titleSlug))
        if (!res.ok) continue
        const buffer = await this.ctx.decodeAudioData(await res.arrayBuffer())
        this.buffers.set(entry.fichier, buffer)
        charges++
      } catch {
        continue
      }
    }
    return charges
  }

  /**
   * play joue le son de l'arme. Rend `false` si rien n'est jouable (arme hors manifeste ou
   * son non préchargé) — un rejeu sans sons reste un rejeu.
   */
  play(arme: string): boolean {
    const entry = this.entries.get(arme)
    const buffer = entry ? this.buffers.get(entry.fichier) : undefined
    if (!entry || !buffer) return false

    const source = this.ctx.createBufferSource()
    source.buffer = buffer
    const tirage = drawVariation(entry.variation, this.settings.variationPercent, this.random)
    source.playbackRate.value = tirage.playbackRate

    this.brancher(source, tirage.gainDb)
    source.start()
    return true
  }

  /**
   * brancher relie la source à la sortie en n'insérant que les nœuds utiles.
   *
   * Le gain de variation et celui de distance sont additionnés EN DÉCIBELS avant d'être
   * convertis : deux GainNode en série feraient le même produit avec un nœud de plus.
   */
  private brancher(source: AudioBufferSourceNode, gainVariationDb: number): void {
    const distance = distanceChain(this.settings.distancePercent)
    const gainDb = gainVariationDb + (distance?.gainDb ?? 0)

    let noeud: AudioNode = source
    if (gainDb !== 0) {
      const gain = this.ctx.createGain()
      gain.gain.value = gainFromDb(gainDb)
      noeud.connect(gain)
      noeud = gain
    }
    if (distance) {
      const filtre = this.ctx.createBiquadFilter()
      filtre.type = 'lowpass'
      filtre.frequency.value = distance.cutoffHz
      noeud.connect(filtre)
      noeud = filtre
    }
    noeud.connect(this.ctx.destination)
  }
}
