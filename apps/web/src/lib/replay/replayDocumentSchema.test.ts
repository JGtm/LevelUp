/**
 * replayDocumentSchema.test.ts — LE SCHÉMA À L'EXÉCUTION EST-IL ENCORE LE CONTRAT ?
 *
 * DEUX NIVEAUX DE PREUVE, ET LE PREMIER NE S'EXÉCUTE MÊME PAS.
 *
 *  1. À LA COMPILATION : les assertions de type ci-dessous confrontent le schéma zod au type
 *     GÉNÉRÉ depuis `api/openapi.yaml`, DANS LES DEUX SENS — l'ensemble des clés d'abord (une
 *     clé ajoutée, renommée ou retirée côté Go fait tomber `tsc -b`, donc la CI), puis
 *     l'assignabilité mutuelle (un champ dont la NATURE change ne passe plus).
 *  2. À L'EXÉCUTION : le schéma refuse ce que le compilateur ne voit plus une fois compilé —
 *     un champ renommé par le serveur, un tableau devenu objet, une borne manquante.
 *
 * CE QUE CES TESTS NE PRÉTENDENT PAS. Le schéma ne descend pas dans les éléments (cf. l'en-tête
 * du module) : un point de trajectoire mal formé passe. Ce n'est pas un oubli, c'est la
 * frontière écrite — la forme profonde est gardée côté producteur par l'empreinte réfléchie du
 * document (`replay/document_shape_test.go`).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import type { Equals, Expect } from '@/lib/types/typeEquality'

import {
  replayDocumentSchema,
  validateReplayDocument,
  type ReplayDocumentFromSchema,
} from './replayDocumentSchema'

/**
 * (1) LES CLÉS, DANS LES DEUX SENS. C'est l'assertion qui compte le plus : un calque ajouté à
 * la cuisson et publié au contrat DOIT entrer dans le schéma, sans quoi il traverserait la
 * frontière sans aucun contrôle de nature.
 */
type _MemesCles = Expect<Equals<keyof ReplayDocument, keyof ReplayDocumentFromSchema>>

/** Assignabilité d'un type vers un autre, en contrainte de compilation. */
type Assignable<A, B> = A extends B ? true : false

/** (2) Ce que le schéma décrit est un document du contrat... */
type _SchemaVersContrat = Expect<Assignable<ReplayDocumentFromSchema, ReplayDocument>>

/** ... et tout document du contrat est décrit par le schéma. */
type _ContratVersSchema = Expect<Assignable<ReplayDocument, ReplayDocumentFromSchema>>

/**
 * Le schéma n'exige qu'un NOMBRE ici : cette valeur ne prétend à aucune version, et ces tests
 * ne décrivent que la NATURE des champs. Le numéro qui compte — celui que le producteur écrit —
 * vit dans les fixtures produites par Go et n'a rien à faire dans ce fichier.
 */
const VERSION_QUELCONQUE = 7

/** Le document MINIMAL que le contrat impose : rien d'optionnel, tout le requis. */
function docMinimal(): Record<string, unknown> {
  return {
    schemaVersion: VERSION_QUELCONQUE,
    matchId: '000d5950',
    titleSlug: 'halo_infinite',
    frameCount: 200,
    bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
    tracks: [],
  }
}

describe('le contrat du document de rejeu, à la compilation', () => {
  it('couvre EXACTEMENT les clés du contrat généré — vérifié par tsc', () => {
    const memesCles: _MemesCles = true
    expect(memesCles).toBe(true)
  })

  it('décrit des documents du contrat, et tout document du contrat — vérifié par tsc', () => {
    const aller: _SchemaVersContrat = true
    const retour: _ContratVersSchema = true
    expect([aller, retour]).toEqual([true, true])
  })
})

describe('le contrat du document de rejeu, à l’exécution', () => {
  it('accepte le document minimal que le contrat impose', () => {
    expect(validateReplayDocument(docMinimal())).toBeNull()
  })

  it('accepte `tracks: null` — le contrat le déclare nullable, et un film muet en produit', () => {
    expect(validateReplayDocument({ ...docMinimal(), tracks: null })).toBeNull()
  })

  it('refuse un champ REQUIS renommé par le serveur — le cas que tsc ne voit plus', () => {
    const sansMatchID = docMinimal()
    delete sansMatchID.matchId
    const issue = validateReplayDocument({ ...sansMatchID, match_id: '000d5950' })
    expect(issue).not.toBeNull()
    expect(issue).toEqual({ kind: 'invalidField', path: 'matchId', detail: expect.any(String) })
  })

  it('refuse un calque dont la NATURE change (tableau devenu objet)', () => {
    const issue = validateReplayDocument({ ...docMinimal(), shots: { t: 1 } })
    expect(issue).not.toBeNull()
    expect(issue).toMatchObject({ kind: 'invalidField', path: 'shots' })
  })

  it('refuse une borne non numérique — une scène de taille NaN est une page blanche muette', () => {
    const issue = validateReplayDocument({
      ...docMinimal(),
      bounds: { minX: 0, minY: 0, maxX: '10', maxY: 10 },
    })
    expect(issue).not.toBeNull()
    expect(issue).toMatchObject({ kind: 'invalidField', path: 'bounds.maxX' })
  })

  it('accepte un calque ABSENT et un calque NUL : les deux disent « aucune donnée »', () => {
    expect(validateReplayDocument({ ...docMinimal(), shots: null })).toBeNull()
    expect(validateReplayDocument(docMinimal())).toBeNull()
  })

  it('ne LÈVE jamais, même sur une entrée qui n’est pas un objet', () => {
    expect(validateReplayDocument(null)).not.toBeNull()
    expect(validateReplayDocument('pas un document')).not.toBeNull()
    expect(validateReplayDocument(42)).not.toBeNull()
  })

  it('refuse une clé INCONNUE à la racine, et la NOMME (mode strict, constat C1)', () => {
    const issue = validateReplayDocument({ ...docMinimal(), shotz: [] })
    expect(issue).not.toBeNull()
    expect(issue).toEqual({ kind: 'unknownKeys', keys: ['shotz'] })
  })

  it('refuse une clé inconnue DANS les bornes — une borne renommée est une borne absente', () => {
    const issue = validateReplayDocument({
      ...docMinimal(),
      bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10, maxXX: 3 },
    })
    expect(issue).not.toBeNull()
    expect(issue).toEqual({ kind: 'unknownKeys', keys: ['maxXX'] })
  })

  it('laisse passer les éléments sans les inspecter — la frontière est écrite, pas devinée', () => {
    // Un tir dont la forme est absurde : le schéma ne descend pas, et c'est documenté.
    expect(validateReplayDocument({ ...docMinimal(), shots: [{ n_importe: 'quoi' }] })).toBeNull()
  })

  it('le schéma lui-même reste un objet analysable (contre-test du garde)', () => {
    expect(replayDocumentSchema.safeParse(docMinimal()).success).toBe(true)
  })
})
