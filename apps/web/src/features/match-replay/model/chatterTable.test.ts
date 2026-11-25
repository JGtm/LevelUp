/// <reference types="node" />
// @vitest-environment node
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/locale'

import { racineDuDepot } from '../test/featureFiles'
import {
  CHATTER_TABLE_VERSION,
  chatterLanguage,
  chatterVoiceStem,
  EMPTY_CHATTER_TABLE,
  parseChatterTable,
} from './chatterTable'

describe('parseChatterTable', () => {
  it('lit une réplique : un texte par langue, les voix versées', () => {
    const t = parseChatterTable({
      version: CHATTER_TABLE_VERSION,
      sons: { '0000000a': { texte: { fr: 'Ennemi !', en: 'Enemy!' }, audio: ['fr'] } },
    })
    expect(t.get('0000000a')).toEqual({ texte: { fr: 'Ennemi !', en: 'Enemy!' }, audio: ['fr'] })
  })

  it('écarte une version inconnue, un document étranger et une entrée sans texte', () => {
    expect(parseChatterTable({ version: 99, sons: {} })).toBe(EMPTY_CHATTER_TABLE)
    expect(parseChatterTable(null)).toBe(EMPTY_CHATTER_TABLE)
    expect(parseChatterTable('x')).toBe(EMPTY_CHATTER_TABLE)
    const t = parseChatterTable({
      version: CHATTER_TABLE_VERSION,
      sons: { a: { texte: {}, audio: [] }, b: null, c: { texte: { fr: 'ok' } } },
    })
    expect([...t.keys()]).toEqual(['c'])
    expect(t.get('c')?.audio).toEqual([])
  })

  it("ne garde d'audio que les langues de l'application", () => {
    const t = parseChatterTable({
      version: CHATTER_TABLE_VERSION,
      sons: { a: { texte: { fr: 'x' }, audio: ['fr', 'xx', 3] } },
    })
    expect(t.get('a')?.audio).toEqual(['fr'])
  })
})

describe('chatterLanguage', () => {
  it("prend la langue de l'interface si la table la porte, sinon l'anglais, sinon rien", () => {
    expect(chatterLanguage({ texte: { fr: 'a', en: 'b' }, audio: [] }, 'fr')).toBe('fr')
    expect(chatterLanguage({ texte: { en: 'b' }, audio: [] }, 'fr')).toBe('en')
    expect(chatterLanguage({ texte: {}, audio: [] }, 'fr')).toBeNull()
  })
})

describe('chatterVoiceStem', () => {
  it('nomme le fichier par langue puis par son', () => {
    expect(chatterVoiceStem('0000000a', 'fr')).toBe('repliques/fr/0000000a')
  })
})

/**
 * LE FICHIER VERSÉ, relu hors ligne : chaque réplique a un texte dans chaque langue de `config/locales.toml`, et chaque voix
 * annoncée est un fichier du dépôt. La commande qui le produit (`cmd/chatter-build`) tient les mêmes invariants côté Go.
 */
describe('la table versée', () => {
  const dossier = resolve(racineDuDepot(), 'static', 'sounds', 'halo_infinite', 'repliques')
  const brut: unknown = JSON.parse(readFileSync(resolve(dossier, 'repliques.json'), 'utf8'))
  const table = parseChatterTable(brut)

  it('porte des répliques, toutes lisibles', () => {
    const declarees = Object.keys((brut as { sons: object }).sons).length
    expect(table.size).toBe(declarees)
    expect(table.size).toBeGreaterThan(0)
  })

  it('a un texte dans chaque langue de config/locales.toml', () => {
    const manquants = [...table.entries()].filter(([, e]) => KNOWN_LOCALES.some((l) => !e.texte[l])).map(([id]) => id)
    expect(manquants).toEqual([])
  })

  it('annonce seulement des voix versées, et verse seulement des voix annoncées', () => {
    const annoncees = new Set<string>()
    for (const [id, e] of table) for (const l of e.audio) annoncees.add(`${l}/${id}.ogg`)
    const absentes = [...annoncees].filter((f) => !existsSync(resolve(dossier, f)))
    expect(absentes).toEqual([])
  })
})
