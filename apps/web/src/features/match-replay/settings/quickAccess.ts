/**
 * quickAccess — L'ACCÈS RAPIDE : jusqu'à cinq interrupteurs épinglés au bord de la carte, à côté du bouton « Vue ».
 * Seuls les interrupteurs que la table des réglages dit épinglables (`SettingDef.pinnable`) s'épinglent.
 *
 * LOGIQUE PURE : aucun React, aucun stockage.
 */
import { defOf, SETTING_DEFS, type SettingId } from './settingsTable'

/** Combien de réglages au plus se posent au bord de la carte. */
export const PIN_MAX = 5

/**
 * Les réglages épinglés tant que le lecteur n'a rien choisi : la visée, la traînée, la carte de chaleur et le fil
 * posé sur le plan, puis les bulles de répliques.
 */
export const DEFAULT_PINS: readonly SettingId[] = ['aim', 'trail', 'heatmap', 'planFeed', 'chatterBubbles']

const PINNABLE: ReadonlySet<string> = new Set(SETTING_DEFS.filter((d) => d.pinnable).map((d) => d.id))

export function isPinnable(id: SettingId): boolean {
  return defOf(id).pinnable
}

/** Épingle ou retire un réglage ; `full` dit qu'un cinquième ne tient pas (rien n'est alors changé). */
export function togglePin(pins: readonly SettingId[], id: SettingId): { pins: readonly SettingId[]; full: boolean } {
  if (pins.includes(id)) return { pins: pins.filter((p) => p !== id), full: false }
  if (!isPinnable(id)) return { pins, full: false }
  if (pins.length >= PIN_MAX) return { pins, full: true }
  return { pins: [...pins, id], full: false }
}

/** Les épingles lues du stockage : seuls des réglages épinglables, sans doublon, au plus `PIN_MAX`. */
export function readPins(raw: unknown): readonly SettingId[] {
  if (!Array.isArray(raw)) return DEFAULT_PINS
  const out: SettingId[] = []
  for (const item of raw) {
    if (typeof item === 'string' && PINNABLE.has(item) && !out.includes(item as SettingId) && out.length < PIN_MAX) {
      out.push(item as SettingId)
    }
  }
  return out
}
