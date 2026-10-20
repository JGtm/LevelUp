/// <reference types="node" />
/**
 * testDoc.ts — LA fixture de document de rejeu, partagée par tous les tests de la feature.
 *
 * LE DOCUMENT DE TEST ENTRE PAR LA MÊME PORTE QUE CELUI DU SERVEUR : on décrit un document
 * de transport (champs facultatifs omis) et la frontière `normalizeReplayDocument` le
 * complète. Un test qui bâtirait directement la forme normalisée testerait un document que
 * le serveur n'envoie jamais.
 *
 * SA VERSION DE SCHÉMA VIENT DU PRODUCTEUR, plus d'un littéral (2026-09-13, lot 0.B). Elle
 * était figée à la version 1 — un numéro que le serveur n'a jamais servi, sous un
 * commentaire qui énonçait pourtant le principe inverse. Elle est désormais lue dans le
 * manifeste que Go écrit (`fixtures/go/manifest.json`, cf. `goFixtures.ts`) : le jour où la
 * cuisson monte de version, cette fixture suit sans que personne y pense.
 *
 * CE QU'ELLE N'EST PAS, ET POURQUOI ELLE RESTE MINIMALE. Elle n'est pas le document COMPLET
 * produit par Go : celui-ci vit dans `fixtures/go/`, pèse 3,3 Mio et sert le test de contrat
 * (`test/goFixtures.contract.test.ts`), qui le fait passer par la frontière et par les
 * logiques pures. Ici, ce qu'on veut est un document MINIMAL que chaque test surcharge de ce
 * qu'il observe — un document complet ferait dépendre 250 tests de valeurs qu'ils ne
 * décrivent pas. Les deux sont complémentaires : l'un prouve que la frontière tient sur le
 * document RÉEL, l'autre isole ce qu'un test veut montrer.
 *
 * Règle « ≤ 2 copies » (CLAUDE.md n° 6) : ce helper est né quand la 3e copie de cette
 * fixture allait apparaître. Le garde-rail `testDoc.guard.test.ts` interdit d'en rebâtir
 * une à la main dans les tests de la feature, et d'y écrire un numéro de schéma en dur.
 */
import type { ReplayDocument } from '@/lib/api/types'

import { normalizeReplayDocument, type ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { goFixtureSchemaVersion } from './goFixtures'

type RosterEntry = NonNullable<ReplayDocument['roster']>[number]

/**
 * Entrée de roster de test : `seat` est OPTIONNEL ici et vaut l'index de film par défaut — c'est
 * sa définition depuis le schéma 60 (lot 1.9.14 : le siège d'un joueur EST l'index que le film
 * écrit ; le document ne l'omet jamais). Les tests qui veulent un siège apparié (différent de
 * l'index) le posent explicitement.
 */
export type TestRosterEntry = Omit<RosterEntry, 'seat'> & { seat?: number }

/** Surcharges acceptées par [testReplayDoc] : le document, avec un roster de test. */
export type TestReplayDocOverrides = Omit<Partial<ReplayDocument>, 'roster'> & {
  roster?: TestRosterEntry[]
}

/** Document de rejeu minimal valide, normalisé — surcharger ce que le test veut voir. */
export function testReplayDoc(over: Partial<ReplayDocument> | TestReplayDocOverrides = {}): ReplayDocumentReady {
  const { roster, ...rest } = over as TestReplayDocOverrides
  return normalizeReplayDocument({
    schemaVersion: goFixtureSchemaVersion(),
    matchId: 'm',
    titleSlug: 'halo_infinite',
    frameCount: 200,
    bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
    tracks: [],
    ...rest,
    ...(roster ? { roster: roster.map((r) => ({ ...r, seat: r.seat ?? r.filmIndex })) } : {}),
  })
}
