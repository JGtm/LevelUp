/**
 * settingsMigration — LA REPRISE DES PRÉFÉRENCES D'AVANT LES PRÉRÉGLAGES SOUPLES, une fois, sans rien perdre.
 *
 * Les clés `replay-*` des réglages sont relues telles quelles (aucune ne change de nom) ; cette reprise ne règle que
 * ce qui change de sens ou disparaît :
 *
 *  1. LES TRACES DES VIES PRISES DU PRÉRÉGLAGE. Avant, trois traces (apparitions, morts, tueurs) sans clé écrite
 *     prenaient leur défaut du dernier préréglage choisi (« Combat » les allume : morts et tueurs ; « Déplacements » :
 *     apparitions), tant que les treize autres calques étaient encore à l'état de ce préréglage. Un préréglage
 *     couvre désormais ces traces lui-même et la règle de défaut a disparu : la reprise écrit une fois ce que la règle
 *     donnait, pour que la vue du lecteur reste la sienne ;
 *  2. LE GROUPE « VIES » NE SE REPLIE PLUS : la clé de son état replié est effacée ;
 *  3. LES TRAITS DU RELIEF, retenus par fond, n'ont plus qu'une valeur : leur reprise vit avec leur lecture
 *     (`lib/replay/useReliefLines`), partagée avec la Tactique et « Occupation du terrain ».
 *
 * La vue active reste sous sa clé d'avant (`replay-view-preset`) : un préréglage s'y lit comme avant. Le préréglage
 * ne verrouille plus rien — si l'état s'en écarte, la vue est « modifiée », et rien n'est réécrit.
 *
 * IDEMPOTENTE : une marque de version est écrite à la fin ; passée la marque, la reprise ne touche plus à rien.
 * Aucune valeur n'est jamais réécrite par-dessus une clé existante.
 */
import { persistPreference, readStoredFlag, readStoredRaw, removePreference } from '@/lib/replay/preferenceStore'

import { isPresetId, type PresetId } from './viewPresets'
import { VIEW_ACTIVE_KEY } from './settingsStore'

export const PREFS_VERSION_KEY = 'replay-prefs-version'
const PREFS_VERSION = '2'

/** L'ancienne clé de l'état replié du groupe « Vies ». */
const LIVES_GROUP_OPEN_KEY = 'replay-lives-group-open'

/** Les trois traces des vies : leur clé, et les préréglages d'avant qui l'allumaient. */
const LEGACY_TRACES: ReadonlyArray<{ key: string; on: readonly PresetId[] }> = [
  { key: 'replay-show-spawns', on: ['deplacements'] },
  { key: 'replay-show-deaths', on: ['combat'] },
  { key: 'replay-show-killers', on: ['combat'] },
]

const ALL: readonly PresetId[] = ['complet', 'combat', 'deplacements', 'objectif']
const CARRIED: readonly PresetId[] = ['complet', 'combat', 'objectif']

/**
 * Les treize calques d'avant les vies : leur clé, leur défaut, et les préréglages d'avant qui les allumaient
 * (la table d'alors, figée ici : c'est elle que la reprise rejoue).
 */
const LEGACY_LAYERS: ReadonlyArray<{ key: string; def: boolean; on: readonly PresetId[] }> = [
  { key: 'replay-show-aim', def: true, on: ['complet', 'combat'] },
  { key: 'replay-show-trail', def: true, on: ALL },
  { key: 'replay-show-zones', def: true, on: ['complet', 'deplacements', 'objectif'] },
  { key: 'replay-show-weapon-pads', def: true, on: ['complet'] },
  { key: 'replay-show-ground-weapons', def: true, on: ['complet'] },
  { key: 'replay-show-vehicles', def: true, on: ['complet', 'combat'] },
  { key: 'replay-show-placements', def: true, on: ['complet'] },
  { key: 'replay-show-dropped-placements', def: true, on: ['complet'] },
  { key: 'replay-show-mode-objectives', def: true, on: CARRIED },
  { key: 'replay-show-flag-carries', def: true, on: CARRIED },
  { key: 'replay-show-vip-crown', def: true, on: CARRIED },
  { key: 'replay-show-skull-carrier', def: true, on: CARRIED },
  { key: 'replay-show-bomb-carrier', def: true, on: CARRIED },
]

/** Écrit, pour chaque trace sans clé, ce que le préréglage retenu lui donnait — si les treize autres calques en sont encore à lui. */
function carryTracesFromPreset(): void {
  const remembered = readStoredRaw(VIEW_ACTIVE_KEY)
  if (!isPresetId(remembered)) return
  const stillThePreset = LEGACY_LAYERS.every((l) => readStoredFlag(l.key, l.def) === l.on.includes(remembered))
  if (!stillThePreset) return
  for (const trace of LEGACY_TRACES) {
    if (trace.on.includes(remembered) && readStoredRaw(trace.key) === null) persistPreference(trace.key, 'true')
  }
}

/** Reprend les préférences d'avant, une fois. */
export function migrateReplayPreferences(): void {
  if (readStoredRaw(PREFS_VERSION_KEY) === PREFS_VERSION) return
  carryTracesFromPreset()
  removePreference(LIVES_GROUP_OPEN_KEY)
  persistPreference(PREFS_VERSION_KEY, PREFS_VERSION)
}
