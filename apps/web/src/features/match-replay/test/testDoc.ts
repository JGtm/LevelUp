/**
 * testDoc.ts — LA fixture de document de rejeu, partagée par tous les tests de la feature.
 *
 * LE DOCUMENT DE TEST ENTRE PAR LA MÊME PORTE QUE CELUI DU SERVEUR : on décrit un document
 * de transport (champs facultatifs omis) et la frontière `normalizeReplayDocument` le
 * complète. Un test qui bâtirait directement la forme normalisée testerait un document que
 * le serveur n'envoie jamais.
 *
 * Règle « ≤ 2 copies » (CLAUDE.md n° 6) : ce helper est né quand la 3e copie de cette
 * fixture allait apparaître. Le garde-rail `testDoc.guard.test.ts` interdit d'en rebâtir
 * une à la main dans les tests de la feature.
 */
import type { ReplayDocument } from '@/lib/api/types'

import { normalizeReplayDocument, type ReplayDocumentReady } from '../replayNormalize'

/** Document de rejeu minimal valide, normalisé — surcharger ce que le test veut voir. */
export function testReplayDoc(over: Partial<ReplayDocument> = {}): ReplayDocumentReady {
  return normalizeReplayDocument({
    schemaVersion: 1,
    matchId: 'm',
    titleSlug: 'halo_infinite',
    frameCount: 200,
    bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
    tracks: [],
    ...over,
  })
}
