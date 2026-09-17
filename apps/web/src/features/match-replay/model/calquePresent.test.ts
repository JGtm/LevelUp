/**
 * Tests — calquePresent (« ce calque a-t-il été produit ? », schéma 62, lot 4.2.2).
 *
 * CE QU'ILS PROTÈGENT. La valeur de ce module est de distinguer TROIS situations qu'un booléen
 * confondrait, et chacune est un artefact réel du parc : un document cuit au schéma 62 qui
 * DÉCLARE un calque, un document cuit au schéma 62 dont une garde de mode était fermée (CTF sur
 * un film Oddball : la passe n'a pas tourné), et un document cuit AVANT 62, qui ne répondait pas
 * encore à la question. Les trois cas sont donc montés séparément, et le troisième est celui
 * qu'un `?? false` ferait taire.
 *
 * LE CALQUE VIDE MAIS PRODUIT est le cas qui justifie tout le lot : `layers` porte l'entrée, le
 * tableau du document est vide, et la réponse est `produit` — « la passe a tourné, ce film n'en
 * portait aucun ». C'est exactement ce qu'une lecture du tableau seul ne peut pas dire.
 */
import { describe, expect, it } from 'vitest'

import { calquePresent, couchesDesCalquesProduits, revisionDuCalque } from './calquePresent'
import { goFixtureSchemaVersion } from '../test/goFixtures'
import { testReplayDoc } from '../test/testDoc'

/** La version que Go publie aujourd hui, et celle d AVANT : jamais un numero ecrit a la main
 * (garde `test/testDoc.guard.test.ts`, second volet du lot 0.B). */
const AVANT_LAYERS = goFixtureSchemaVersion() - 1

const LAYERS = {
  tracks: 'grammar-2026-09-15.42',
  zoneStates: 'grammar-2026-09-15.42',
  objectives: 'killsource-2026-09-17.2',
  matchId: `publication-${goFixtureSchemaVersion()}`,
}

describe('calquePresent', () => {
  it('rend `produit` quand `layers` porte l entrée', () => {
    const doc = testReplayDoc({ layers: LAYERS })
    expect(calquePresent(doc, 'zoneStates')).toBe('produit')
  })

  it('rend `produit` sur un calque DÉCLARÉ dont le tableau est VIDE — la passe a tourné', () => {
    const doc = testReplayDoc({ layers: LAYERS, zoneStates: [] })
    expect(doc.zoneStates).toHaveLength(0)
    expect(calquePresent(doc, 'zoneStates')).toBe('produit')
  })

  it('rend `nonProduit` quand `layers` est là et que l entrée manque', () => {
    const doc = testReplayDoc({ layers: LAYERS })
    expect(calquePresent(doc, 'flagCarries')).toBe('nonProduit')
  })

  it('rend `inconnu` quand `layers` est absent — artefact antérieur au schéma 62', () => {
    const doc = testReplayDoc({ schemaVersion: AVANT_LAYERS })
    expect(doc.layers).toBeUndefined()
    expect(calquePresent(doc, 'zoneStates')).toBe('inconnu')
  })

  it('ne comble JAMAIS l objet `layers` — la normalisation le laisse absent', () => {
    expect(testReplayDoc({ schemaVersion: AVANT_LAYERS }).layers).toBeUndefined()
  })
})

describe('revisionDuCalque', () => {
  it('rend la révision de la couche productrice', () => {
    expect(revisionDuCalque(testReplayDoc({ layers: LAYERS }), 'objectives')).toBe(
      'killsource-2026-09-17.2',
    )
  })

  it('rend `undefined` sur un calque non produit comme sur un document sans `layers`', () => {
    expect(revisionDuCalque(testReplayDoc({ layers: LAYERS }), 'vipCrown')).toBeUndefined()
    expect(revisionDuCalque(testReplayDoc({ schemaVersion: AVANT_LAYERS }), 'tracks')).toBeUndefined()
  })
})

describe('couchesDesCalquesProduits', () => {
  it('rend les révisions DISTINCTES, triées — une couche, pas un calque par entrée', () => {
    expect(couchesDesCalquesProduits(testReplayDoc({ layers: LAYERS }))).toEqual([
      'grammar-2026-09-15.42',
      'killsource-2026-09-17.2',
      `publication-${goFixtureSchemaVersion()}`,
    ])
  })

  it('rend une liste vide quand `layers` est absent', () => {
    expect(couchesDesCalquesProduits(testReplayDoc({ schemaVersion: AVANT_LAYERS }))).toEqual([])
  })
})
