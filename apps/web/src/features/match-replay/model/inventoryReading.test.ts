import { describe, expect, it } from 'vitest'

import {
  grenadeBoxAt,
  grenadeReadingAt,
  grenadesCarriedFrom,
  inventoryAt,
  selectedGrenadeFrom,
} from './inventoryReading'
import { testReplayDoc as doc } from '../test/testDoc'

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

/**
 * L'AXE DES GRENADES (schéma 20) — ce que ces cas verrouillent.
 *
 * Le lot 4.4 ajoute un SECOND canal sur la même grandeur : les paquets delta, transmis au
 * changement, qui rafraîchissent entre deux images-clés. Deux choses peuvent casser sans que
 * rien d'autre ne bouge — le repli sur un artefact ancien (la boîte se viderait) et la
 * préférence pour la lecture la plus récente (le gain de fraîcheur disparaîtrait).
 */
describe('grenadeReadingAt', () => {
  it('rend la lecture la PLUS RÉCENTE du slot, quel que soit le canal', () => {
    const d = doc({
      grenadeReads: [
        { t: 10, slot: 512, g: [0, 2, 0, 0], gs: 1, src: 'kf' },
        { t: 45, slot: 512, g: [0, 1, 0, 0], gs: 1, src: 'delta' },
        { t: 20, slot: 513, g: [1, 0, 0, 0], src: 'kf' },
      ],
    })
    const r = grenadeReadingAt(d, 512, 60)
    expect(r?.src, 'la lecture delta est plus récente que la kf').toBe('delta')
    expect(r?.g).toEqual([0, 1, 0, 0])
    expect(r?.age, "l'âge est compté en frames depuis la lecture").toBe(15)
  })

  it("rend null quand l'artefact ne porte pas l'axe — le repli est le point", () => {
    expect(grenadeReadingAt(doc({}), 512, 60)).toBeNull()
  })

  it('ignore les autres slots', () => {
    const d = doc({ grenadeReads: [{ t: 10, slot: 999, g: [1, 0, 0, 0], src: 'delta' }] })
    expect(grenadeReadingAt(d, 512, 60)).toBeNull()
  })
})

/**
 * grenadeBoxAt — LE DÉPARTAGE des deux sources de la boîte, éprouvé sans rendu.
 *
 * UNE LECTURE À VENIR NE PRIME JAMAIS UNE INFORMATION PASSÉE : même doctrine que la « lecture
 * vide À VENIR » ci-dessus, née du même défaut (slot 554 du film de référence — une plasma
 * affichée ~60 s avant sa première mesure, sous une infobulle « lu il y a X »).
 */
describe('grenadeBoxAt', () => {
  const withBoth = doc({
    inventory: [{ t: 0, slot: 512, g: [2, 0] }],
    grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
  })

  it('la lecture de l’axe PASSÉE gagne, avec son âge — c’est le gain du lot', () => {
    const d = doc({
      inventory: [{ t: 0, slot: 512, g: [2, 0] }],
      grenadeReads: [{ t: 60, slot: 512, g: [0, 3], src: 'delta' }],
    })
    expect(grenadeBoxAt(d, 512, 90, inventoryAt(d, 512, 90))).toEqual({
      g: [0, 3],
      gs: undefined,
      age: 30,
    })
  })

  it('sans axe (artefact ≤ 19), retombe sur l’inventaire — le repli est le point', () => {
    const d = doc({ inventory: [{ t: 0, slot: 512, g: [1, 2], gs: 1 }] })
    expect(grenadeBoxAt(d, 512, 60, inventoryAt(d, 512, 60))).toEqual({
      g: [1, 2],
      gs: 1,
      age: 60,
    })
  })

  it('lecture de l’axe À VENIR : les compteurs PASSÉS de l’inventaire priment', () => {
    expect(grenadeBoxAt(withBoth, 512, 30, inventoryAt(withBoth, 512, 30))).toEqual({
      g: [2, 0],
      gs: undefined,
      age: 30,
    })
  })

  it('lecture À VENIR sans rien de passé : elle s’affiche, âge NÉGATIF assumé', () => {
    const d = doc({ grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }] })
    expect(grenadeBoxAt(d, 512, 30, inventoryAt(d, 512, 30))?.age).toBe(-60)
  })

  it('un inventaire passé SANS compteurs lus ne départage rien — `g` vide = non lu', () => {
    // Un tableau vide dit « compteurs NON LUS », pas « aucune grenade » : ce n'est donc pas une
    // information passée sur les grenades, et la lecture à venir reste le seul état à montrer.
    const d = doc({
      inventory: [{ t: 0, slot: 512, g: [] }],
      grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
    })
    const box = grenadeBoxAt(d, 512, 30, inventoryAt(d, 512, 30))
    expect(box?.g).toEqual([0, 5])
    // L'ÂGE RESTE CELUI DE LA LECTURE À VENIR (négatif) : un mutant qui daterait ces compteurs
    // de l'inventaire passé (age 30) fabriquerait une boîte « il y a X » pour des compteurs
    // que rien n'a encore mesurés — exactement ce que le godoc de grenadeBoxAt interdit.
    expect(box?.age).toBe(-60)
  })

  it('rend null quand ni l’axe ni l’inventaire ne portent ce slot', () => {
    expect(grenadeBoxAt(doc({}), 512, 30, null)).toBeNull()
  })
})

describe('selectedGrenadeFrom', () => {
  it('retient la sélection LUE quand elle est cohérente avec les compteurs', () => {
    expect(selectedGrenadeFrom([0, 2, 0, 1], 3)).toEqual({ rank: 3, read: true })
  })

  it('DÉDUIT le type quand un seul est porté, et le dit', () => {
    expect(selectedGrenadeFrom([0, 2, 0, 0], undefined)).toEqual({ rank: 1, read: false })
  })

  it("reste indéterminé sur plusieurs types sans sélection lue — on ne devine pas", () => {
    expect(selectedGrenadeFrom([1, 2, 0, 0], undefined)).toBe('indeterminate')
  })

  it('rend null quand aucune grenade n est portée', () => {
    expect(selectedGrenadeFrom([0, 0, 0, 0], 2)).toBeNull()
  })

  it('ignore une sélection qui ne correspond à aucun compteur porté', () => {
    expect(selectedGrenadeFrom([0, 2, 0, 0], 3)).toEqual({ rank: 1, read: false })
  })
})

describe('grenadesCarriedFrom', () => {
  // Les libellés du document sont BILINGUES depuis le schéma v2 : une seule table nomme les
  // rangs, et c'est le lecteur qui choisit sa langue.
  const labels = [
    { en: 'Frag', fr: 'Fragmentation' },
    { en: 'Plasma', fr: 'Plasma' },
  ]

  it("n'affiche que les types réellement portés, et garde le rang sans table", () => {
    // Le tableau publié est complet : un zéro y dit « ce type, aucune en réserve ». Montrer
    // quatre types dont trois à zéro noierait celui qui compte.
    expect(grenadesCarriedFrom([0, 2, 0, 1], undefined, 'fr')).toEqual([
      { rank: 1, name: 'rang 1', count: 2 },
      { rank: 3, name: 'rang 3', count: 1 },
    ])
  })

  it('rend le nom dans la langue du lecteur', () => {
    expect(grenadesCarriedFrom([2, 0], labels, 'fr')[0].name).toBe('Fragmentation')
    expect(grenadesCarriedFrom([2, 0], labels, 'en')[0].name).toBe('Frag')
  })

  it('sans compteurs lus, ne rend rien — jamais quatre types à zéro', () => {
    expect(grenadesCarriedFrom([], labels, 'fr')).toEqual([])
  })
})
