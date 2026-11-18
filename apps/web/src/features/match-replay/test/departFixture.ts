/// <reference types="node" />
/**
 * departFixture.ts — LE DOCUMENT D'UN DÉPART, tiré d'un rejeu réel et aminci.
 *
 * `fixtures/departs/depart_sans_mort.json` garde du rejeu d'origine ce qu'il faut pour lire un départ :
 * le tableau (présences), les pistes (deux points chacune) et les comptes à rebours de quatre joueurs
 * (trois humains, un bot). Tous les identifiants et les noms sont fictifs. Le joueur `fx-depart-1` meurt
 * à l'image 9539 (lâcher de commande, compte à rebours lu), réapparaît à 9783, son corps est supprimé à
 * 9785 sans lâcher ni compte à rebours, sa présence se ferme à 9937 et le bot (`Robot Un [bot]`) reprend
 * sa place à 9938. Le document dure 14 944 images de 33 ms.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import type { ReplayDocument } from '@/lib/api/types'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { featureRoot } from './featureFiles'
import { testReplayDoc } from './testDoc'

/** Le joueur qui quitte la partie. */
export const PARTANT = 'fx-depart-1'
/** Ses repères, en images du document. */
export const DEPART = { mort: 9539, reapparition: 9783, corpsSupprime: 9785, presenceFermee: 9937 } as const

/** Le document de transport, tel que le fichier le porte. */
export function departDocument(): ReplayDocument {
  const file = join(featureRoot(), 'test', 'fixtures', 'departs', 'depart_sans_mort.json')
  return JSON.parse(readFileSync(file, 'utf8')) as ReplayDocument
}

/** Le document normalisé, par la même porte que le serveur (`testReplayDoc`). */
export function departDoc(over: Partial<ReplayDocument> = {}): ReplayDocumentReady {
  return testReplayDoc({ ...departDocument(), ...over })
}
