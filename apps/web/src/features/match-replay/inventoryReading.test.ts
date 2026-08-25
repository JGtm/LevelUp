import { describe, expect, it } from 'vitest'

import {
  grenadesCarried,
  inventoryAt,
  selectedGrenade,
} from './inventoryReading'
import type { ReplayInventoryReady } from './replayNormalize'
import { testReplayDoc as doc } from './test/testDoc'

/** Un inventaire tel que la frontière le livre : ce qui n'a pas été lu y est un tableau vide. */
function inv(over: Partial<ReplayInventoryReady> = {}): ReplayInventoryReady {
  return { t: 0, slot: 1, g: [], am: [], ...over }
}

describe('inventoryAt', () => {
  const d = doc({
    inventory: [
      // Marqueur de lecture : `gs` (type de grenade sélectionné). Il portait `a` (index de
      // capacité) jusqu'au schéma 6, qui a RETIRÉ ce champ de l'inventaire — la capacité vit
      // désormais dans son propre calque `abilities`, avec le RANG et non un index tronqué.
      // Ces cas ne testent pas la capacité : ils vérifient QUELLE lecture `inventoryAt` rend.
      { t: 10, slot: 512, g: [0, 2, 0, 0], gs: 4, d: 0 },
      { t: 200, slot: 512, g: [1, 0, 0, 0] },
      { t: 10, slot: 513, gs: 9 },
    ],
  })

  it('rend la dernière lecture du SLOT, avec son âge', () => {
    const r = inventoryAt(d, 512, 60)
    expect(r?.age).toBe(50)
    expect(r?.state.gs).toBe(4)
  })

  it('ne lit jamais l’inventaire d’un autre slot', () => {
    expect(inventoryAt(d, 513, 60)?.state.gs).toBe(9)
    const solo = doc({ inventory: [{ t: 10, slot: 513, gs: 9 }] })
    expect(inventoryAt(solo, 512, 5)).toBeNull()
  })

  it('avant la première image-clé de la vie : la lecture À VENIR, âge NÉGATIF publié tel quel', () => {
    // Même repli que loadoutAt (décision utilisateur 2026-08-12, au-delà du POC pour les
    // compteurs) : la dotation de spawn affichée avec son âge « à venir » informe mieux
    // que vingt secondes de vide. Un slot est une vie : aucun repli ne franchit une mort.
    const r = inventoryAt(d, 512, 5)
    expect(r?.age).toBe(-5)
    expect(r?.state.gs).toBe(4)
  })

  it('sans inventaire, rend null', () => {
    expect(inventoryAt(doc(), 512, 60)).toBeNull()
  })
})

describe('grenadesCarried', () => {
  // Les libellés du document sont BILINGUES depuis le schéma v2 : une seule table nomme
  // les rangs, et c'est le lecteur qui choisit sa langue.
  const labels = [
    { en: 'Frag', fr: 'Fragmentation' },
    { en: 'Plasma', fr: 'Plasma' },
    { en: 'Dynamo', fr: 'Dynamo' },
    { en: 'Spike', fr: 'Spike' },
  ]

  it('n’affiche que les types réellement portés', () => {
    // Le tableau publié est complet : un zéro y dit « ce type, aucune en réserve ». Montrer
    // quatre types dont trois à zéro noierait celui qui compte.
    const got = grenadesCarried(inv({ g: [0, 2, 0, 0] }), labels, 'fr')
    expect(got).toEqual([{ rank: 1, name: 'Plasma', count: 2 }])
  })

  it('rend le nom dans la langue du lecteur', () => {
    expect(grenadesCarried(inv({ g: [2, 0, 0, 0] }), labels, 'fr')[0].name).toBe('Fragmentation')
    expect(grenadesCarried(inv({ g: [2, 0, 0, 0] }), labels, 'en')[0].name).toBe('Frag')
  })

  it('sans compteurs lus, ne rend rien — jamais quatre types à zéro', () => {
    expect(grenadesCarried(inv(), labels, 'fr')).toEqual([])
  })

  it('sans table de noms, garde le rang plutôt qu’un nom inventé', () => {
    expect(grenadesCarried(inv({ g: [3, 0, 0, 0] }), undefined, 'fr')[0].name).toBe('rang 0')
  })
})

describe('selectedGrenade', () => {
  it('déduit le type quand il ne peut pas être un autre — et le dit DÉDUIT', () => {
    expect(selectedGrenade(inv({ g: [0, 0, 2, 0] }))).toEqual({ rank: 2, read: false })
  })

  it('la LECTURE du film (gs) prime la déduction, et se dit LUE', () => {
    expect(selectedGrenade(inv({ g: [1, 2, 0, 0], gs: 1 }))).toEqual({ rank: 1, read: true })
  })

  it('deux types portés sans sélecteur lu : INDÉTERMINÉ, jamais deviné', () => {
    // L'écran doit le dire (« sél. ? »), pas choisir : deviner afficherait une certitude
    // qu'on n'a pas.
    expect(selectedGrenade(inv({ g: [1, 2, 0, 0] }))).toBe('indeterminate')
  })

  it('un gs qui désigne un rang NON porté ne prime rien — la garde de cohérence du décodeur', () => {
    // Le décodeur publie gs sous masque == compteurs : ce cas ne doit pas arriver, et s'il
    // arrivait (artefact d'une autre version), on retombe sur la règle sans lecture.
    expect(selectedGrenade(inv({ g: [0, 0, 2, 0], gs: 1 }))).toEqual({ rank: 2, read: false })
  })

  it('ne désigne rien sans lecture', () => {
    expect(selectedGrenade(inv())).toBeNull()
  })
})

describe('inventoryAt — une lecture VIDE n’efface plus la fiche', () => {
  // Le défaut mesuré : la lecture vide, étant la plus récente <= T, gagnait contre la lecture
  // PLEINE qui la précédait, et la ligne disparaissait pendant ~20 s. 17,4 % des lectures
  // publiées sont dans ce cas (mesure du 2026-08-24).
  const d = doc({
    inventory: [
      { t: 10, slot: 512, g: [0, 2, 0, 0], gs: 1, d: 0 },
      { t: 100, slot: 512, empty: 'dead' },
      { t: 10, slot: 513, empty: 'unknown' },
    ],
  })

  it('rend la dernière lecture PLEINE, et l’état vide À CÔTÉ', () => {
    const r = inventoryAt(d, 512, 120)
    expect(r?.state.g).toEqual([0, 2, 0, 0])
    // L'âge est celui de la lecture PLEINE affichée, pas celui de la lecture vide : les deux
    // ne datent pas du même instant.
    expect(r?.age).toBe(110)
    expect(r?.empty).toEqual({ kind: 'dead', age: 20 })
    // La SUBSTITUTION est publiée : c'est elle qui autorise la fiche à dire « l'équipement
    // affiché est la dernière lecture pleine ».
    expect(r?.substituted).toBe(true)
  })

  it('sans lecture pleine antérieure, rend la lecture vide elle-même — jamais null', () => {
    const r = inventoryAt(d, 513, 60)
    expect(r).not.toBeNull()
    expect(r?.empty).toEqual({ kind: 'unknown', age: 50 })
    // AUCUNE substitution : `state` EST la lecture vide, et `age` est SON âge. Annoncer ici
    // une « dernière lecture pleine, lue il y a 50 » serait une infobulle mensongère — c'est
    // le défaut que ce drapeau ferme (revue adversariale, constat 4).
    expect(r?.substituted).toBe(false)
    expect(r?.age).toBe(50)
  })

  it('une lecture PLEINE ne porte aucun état vide', () => {
    expect(inventoryAt(d, 512, 60)?.empty).toBeUndefined()
  })

  it('ne remonte jamais au slot d’un autre joueur pour combler un vide', () => {
    const croise = doc({
      inventory: [
        { t: 10, slot: 513, g: [3, 0, 0, 0] },
        { t: 100, slot: 512, empty: 'dead' },
      ],
    })
    const r = inventoryAt(croise, 512, 120)
    expect(r?.state.g).toEqual([])
    expect(r?.empty?.kind).toBe('dead')
    expect(r?.substituted).toBe(false)
  })

  it('une étiquette inconnue d’un artefact futur se lit « indisponible », jamais « mort »', () => {
    // Écrire « mort » sur une valeur qu'on ne comprend pas serait affirmer à l'écran ce
    // qu'aucune pièce n'établit.
    const futur = doc({ inventory: [{ t: 10, slot: 512, empty: 'quelque-chose' }] })
    expect(inventoryAt(futur, 512, 60)?.empty?.kind).toBe('unknown')
  })

  it('une lecture vide À VENIR n’affirme rien — pas de « Mort » avant la mort', () => {
    // Ronde 2 de la revue adversariale (2026-08-25) : quand la PREMIÈRE lecture d'un slot est
    // vide, nearestReading rend la lecture à venir (âge négatif) et le badge « Mort »
    // s'affichait de 7,5 à 19,1 s AVANT la lecture — 8 vies sur 90 du film de référence.
    // Comportement attendu : lecture ordinaire « à venir », sans état vide ni substitution.
    const ahead = doc({ inventory: [{ t: 50, slot: 512, empty: 'dead' }] })
    const r = inventoryAt(ahead, 512, 10)
    expect(r).not.toBeNull()
    expect(r?.age).toBe(-40)
    expect(r?.empty).toBeUndefined()
    expect(r?.substituted).toBe(false)
  })

  it('une lecture vide à venir ne substitue pas un équipement pas encore ramassé', () => {
    // La vide (t=50) est la plus proche à venir ; une lecture pleine existe plus loin (t=80).
    // Sans la garde, lastFullBefore ne trouverait rien avant t=50 et le badge s'afficherait
    // quand même — avec elle, la lecture est rendue « à venir », sans état vide.
    const aheadFull = doc({
      inventory: [
        { t: 50, slot: 512, empty: 'dead' },
        { t: 80, slot: 512, g: [2, 0, 0, 0], d: 0 },
      ],
    })
    const r = inventoryAt(aheadFull, 512, 10)
    expect(r?.empty).toBeUndefined()
    expect(r?.substituted).toBe(false)
    expect(r?.age).toBe(-40)
    // La lecture rendue est bien la VIDE (t=50), pas la pleine future (t=80).
    expect(r?.state.g).toEqual([])
  })
})
