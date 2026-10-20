import { describe, expect, it } from 'vitest'

import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import {
  grenadeBoxAt,
  grenadeReadingAt,
  grenadesCarriedFrom,
  inventoryAt,
  selectedGrenadeFrom,
} from './inventoryReading'
import { testReplayDoc as doc } from '../test/testDoc'

/** Une vie couvrant [start, end] sur un slot — même patron que rosterLogic.test.ts. */
function track(slot: number, xuid: string | undefined, start: number, end: number): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

describe('inventoryAt', () => {
  // Vie couvrante large sur les deux slots : les frames exercées ici (5, 60) y tombent
  // toutes, le seul facteur testé reste la présence/absence d'une lecture.
  const d = doc({
    tracks: [track(512, 'A', 0, 100), track(513, 'B', 0, 100)],
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
    const solo = doc({ tracks: [track(512, 'A', 0, 100)], inventory: [{ t: 10, slot: 513, gs: 9 }] })
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
    expect(inventoryAt(doc({ tracks: [track(512, 'A', 0, 100)] }), 512, 60)).toBeNull()
  })

  it('sans vie couvrante sur ce slot à cette image, rend null', () => {
    expect(inventoryAt(doc(), 512, 60)).toBeNull()
  })
})

describe('inventoryAt — une lecture VIDE n’efface plus la fiche', () => {
  // Le défaut mesuré : la lecture vide, étant la plus récente <= T, gagnait contre la lecture
  // PLEINE qui la précédait, et la ligne disparaissait pendant ~20 s. 17,4 % des lectures
  // publiées sont dans ce cas (mesure du 2026-08-24).
  const d = doc({
    tracks: [track(512, 'A', 0, 150), track(513, 'B', 0, 150)],
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
      tracks: [track(512, 'A', 0, 150), track(513, 'B', 0, 150)],
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
    const futur = doc({
      tracks: [track(512, 'A', 0, 100)],
      inventory: [{ t: 10, slot: 512, empty: 'quelque-chose' }],
    })
    expect(inventoryAt(futur, 512, 60)?.empty?.kind).toBe('unknown')
  })

  it('une lecture vide À VENIR n’affirme rien — pas de « Mort » avant la mort', () => {
    // Ronde 2 de la revue adversariale (2026-08-25) : quand la PREMIÈRE lecture d'un slot est
    // vide, nearestReading rend la lecture à venir (âge négatif) et le badge « Mort »
    // s'affichait de 7,5 à 19,1 s AVANT la lecture — 8 vies sur 90 du film de référence.
    // Comportement attendu : lecture ordinaire « à venir », sans état vide ni substitution.
    const ahead = doc({
      tracks: [track(512, 'A', 0, 100)],
      inventory: [{ t: 50, slot: 512, empty: 'dead' }],
    })
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
      tracks: [track(512, 'A', 0, 100)],
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
 * inventoryAt — BORNÉE À LA VIE EN COURS DU SLOT (correctif P0-2, 2026-09-06,
 * `.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md`).
 *
 * PAS UN ÉTAT D'IDENTITÉ (décision produit du 2026-09-06 : une vie est un humain ou un bot,
 * jamais une entité anonyme) : `null` dit seulement qu'aucune lecture n'a encore été observée
 * depuis le début de CETTE vie, jamais qu'elle serait « inconnue ».
 */
describe('inventoryAt — borné à la VIE EN COURS du slot (correctif P0-2)', () => {
  // Vie 1 (slot 512, frames 0-50) : inventaire lu à t=10 ET t=20 (pleine). Vie 2 (MÊME SLOT,
  // frames 60-150) : aucune lecture dans sa fenêtre.
  const recycled = doc({
    tracks: [track(512, 'A', 0, 50), track(512, 'B', 60, 150)],
    inventory: [{ t: 10, slot: 512, g: [1, 0, 0, 0] }],
  })

  it('ne reporte JAMAIS la lecture de la vie précédente', () => {
    // TEST PAR MUTATION : retirer la borne `s.t < life.start || s.t > life.end` de
    // `nearestReading` fait revenir l'ancien calcul — `best` retrouve la lecture de la vie 1
    // (âge 90) et ce test devient ROUGE.
    expect(inventoryAt(recycled, 512, 100)).toBeNull()
  })

  it('une lecture VIDE de la vie précédente ne comble pas non plus le vide de la vie courante', () => {
    // `lastFullBefore` doit lui aussi rester dans les bornes de la vie courante : sans la
    // borne `lifeStart`, il retrouverait la lecture PLEINE de la vie 1 pour « combler » la
    // lecture vide de la vie 2 — exactement le défaut symétrique de `nearestReading`.
    const d = doc({
      tracks: [track(512, 'A', 0, 50), track(512, 'B', 60, 150)],
      inventory: [
        { t: 10, slot: 512, g: [1, 0, 0, 0] },
        { t: 100, slot: 512, empty: 'dead' },
      ],
    })
    const r = inventoryAt(d, 512, 120)
    expect(r?.state.g).toEqual([])
    expect(r?.substituted).toBe(false)
  })

  it('une vie ANTÉRIEURE sans lecture propre ne capte JAMAIS l’inventaire d’une vie ULTÉRIEURE (borne haute, revue WEB-R1 C1)', () => {
    // Symétrique du premier cas de ce bloc : la SEULE lecture existante appartient à la vie 2
    // [60,150] (t=70), la requête tombe dans la vie 1 [0,50], qui n'a AUCUNE lecture propre.
    // Sans la moitié HAUTE de la borne (`s.t > life.end`), `ahead` capterait à tort cette
    // lecture future comme repli « à venir » de la vie 1 — potentiellement l'inventaire d'un
    // AUTRE joueur. TEST PAR MUTATION : retirer `|| s.t > life.end` du filtre de
    // `nearestReading` rend une lecture non nulle (âge -50) au lieu de `null`.
    const futureLifeOnly = doc({
      tracks: [track(512, 'A', 0, 50), track(512, 'B', 60, 150)],
      inventory: [{ t: 70, slot: 512, g: [1, 0, 0, 0] }],
    })
    expect(inventoryAt(futureLifeOnly, 512, 20)).toBeNull()
  })

  /**
   * `lastFullBefore` N'A PAS DE BORNE HAUTE QUI LUI SOIT PROPRE, ET C'EST UNE PREUVE, PAS UNE
   * LACUNE (revue WEB-R1, C1 — vérifié par mutation avant de l'écrire, pas seulement raisonné).
   *
   * Quand `frame` couvre une vie V, toute lecture d'une vie ULTÉRIEURE a nécessairement
   * `s.t >= vie_suivante.start > V.end >= t` (où `t` est le timestamp — déjà borné par
   * `nearestReading` — de la lecture vide que `lastFullBefore` cherche à combler). Son propre
   * filtre `s.t < t` exclut donc structurellement TOUTE lecture d'une vie ultérieure, avant même
   * `lifeStart` : il n'existe aucune ligne « borne haute » à retirer dans `lastFullBefore` pour
   * observer un rougissement — vérifié en retirant `|| s.t > life.end` de `nearestReading`
   * (la seule borne haute réelle de la chaîne) : le test ci-dessous reste VERT, parce que `best`
   * (la lecture VIDE propre à la vie courante, t=30) l'emporte déjà sur toute lecture future
   * candidate à `ahead`, quelle que soit la borne. Ce test verrouille donc le COMPORTEMENT
   * (aucune fuite observable), pas une ligne de code précise à mutation-tester : il n'y en a pas.
   */
  it('lastFullBefore ne remonte jamais à une lecture PLEINE d’une vie ULTÉRIEURE, même présente', () => {
    const d = doc({
      tracks: [track(512, 'A', 0, 50), track(512, 'B', 60, 150)],
      inventory: [
        { t: 30, slot: 512, empty: 'dead' }, // vie 1, propre — devient `best` sans ambiguïté
        { t: 70, slot: 512, g: [2, 0, 0, 0] }, // vie 2, PLEINE — ne doit jamais être vue depuis la vie 1
      ],
    })
    const r = inventoryAt(d, 512, 40)
    expect(r?.empty).toEqual({ kind: 'dead', age: 10 })
    expect(r?.substituted).toBe(false)
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
      tracks: [track(512, 'A', 0, 100)],
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
    expect(grenadeReadingAt(doc({ tracks: [track(512, 'A', 0, 100)] }), 512, 60)).toBeNull()
  })

  it('ignore les autres slots', () => {
    const d = doc({
      tracks: [track(512, 'A', 0, 100)],
      grenadeReads: [{ t: 10, slot: 999, g: [1, 0, 0, 0], src: 'delta' }],
    })
    expect(grenadeReadingAt(d, 512, 60)).toBeNull()
  })

  it('sans vie couvrante sur ce slot à cette image, rend null', () => {
    expect(grenadeReadingAt(doc({}), 512, 60)).toBeNull()
  })

  it('un slot RECYCLÉ ne reporte JAMAIS les grenades de la vie précédente (correctif P0-2)', () => {
    const d = doc({
      tracks: [track(512, 'A', 0, 50), track(512, 'B', 60, 150)],
      grenadeReads: [{ t: 10, slot: 512, g: [0, 2, 0, 0], src: 'kf' }],
    })
    expect(grenadeReadingAt(d, 512, 100)).toBeNull()
  })

  it('une vie ANTÉRIEURE sans lecture propre ne capte JAMAIS les grenades d’une vie ULTÉRIEURE (borne haute, revue WEB-R1 C1)', () => {
    // Symétrique du cas ci-dessus : la SEULE lecture existante appartient à la vie 2 [60,150]
    // (t=70), la requête tombe dans la vie 1 [0,50], sans lecture propre. TEST PAR MUTATION :
    // retirer `|| s.t > life.end` du filtre de `nearestReading` rend une lecture non nulle
    // (âge -50) au lieu de `null`.
    const futureLifeOnly = doc({
      tracks: [track(512, 'A', 0, 50), track(512, 'B', 60, 150)],
      grenadeReads: [{ t: 70, slot: 512, g: [0, 2, 0, 0], src: 'kf' }],
    })
    expect(grenadeReadingAt(futureLifeOnly, 512, 20)).toBeNull()
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
    tracks: [track(512, 'A', 0, 150)],
    inventory: [{ t: 0, slot: 512, g: [2, 0] }],
    grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
  })

  it('la lecture de l’axe PASSÉE gagne, avec son âge — c’est le gain du lot', () => {
    const d = doc({
      tracks: [track(512, 'A', 0, 150)],
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
    const d = doc({ tracks: [track(512, 'A', 0, 100)], inventory: [{ t: 0, slot: 512, g: [1, 2], gs: 1 }] })
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
    const d = doc({
      tracks: [track(512, 'A', 0, 150)],
      grenadeReads: [{ t: 90, slot: 512, g: [0, 5], src: 'delta' }],
    })
    expect(grenadeBoxAt(d, 512, 30, inventoryAt(d, 512, 30))?.age).toBe(-60)
  })

  it('un inventaire passé SANS compteurs lus ne départage rien — `g` vide = non lu', () => {
    // Un tableau vide dit « compteurs NON LUS », pas « aucune grenade » : ce n'est donc pas une
    // information passée sur les grenades, et la lecture à venir reste le seul état à montrer.
    const d = doc({
      tracks: [track(512, 'A', 0, 150)],
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
