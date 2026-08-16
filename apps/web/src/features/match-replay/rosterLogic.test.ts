import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ReplayInventoryReady, ReplayTrackReady } from './replayNormalize'
import { testReplayDoc as doc } from './test/testDoc'
import {
  buildPlayers,
  colorBySlot,
  grenadesCarried,
  groupByTeam,
  inventoryAt,
  loadoutAt,
  markBySlot,
  nameBySlot,
  playerName,
  playerStateAt,
  selectedGrenade,
  vitalityPresence,
} from './rosterLogic'

/** Présence standard : le document porte les deux champs (cas Halo Infinite décodé). */
const BOTH = { shield: true, health: true }

function track(
  slot: number,
  xuid: string | undefined,
  start: number,
  end: number,
): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0, sh: 1 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

function row(xuid: string, gamertag: string, side: string | null): MatchScoreboardRow {
  return {
    xuid,
    gamertag,
    team_side: side,
    is_me: false,
    rank: 1,
    score: 0,
    kills: 3,
    deaths: 2,
    assists: 1,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: 'Victoire',
  }
}

/** Un inventaire tel que la frontière le livre : ce qui n'a pas été lu y est un tableau vide. */
function inv(over: Partial<ReplayInventoryReady> = {}): ReplayInventoryReady {
  return { t: 0, slot: 1, g: [], am: [], ...over }
}

describe('buildPlayers', () => {
  it('regroupe les vies par joueur et joint le scoreboard par xuid', () => {
    const d = doc({
      roster: [
        { xuid: 'A', filmIndex: 0 },
        { xuid: 'B', filmIndex: 1 },
      ],
      tracks: [track(512, 'A', 0, 50), track(513, 'B', 0, 60), track(514, 'A', 70, 120)],
    })
    const players = buildPlayers(d, [row('A', 'Alpha', 'Eagle'), row('B', 'Bravo', 'Cobra')])
    expect(players).toHaveLength(2)
    const a = players.find((p) => p.xuid === 'A')!
    expect(a.lives).toHaveLength(2)
    expect(a.board?.gamertag).toBe('Alpha')
  })

  it('n’attribue AUCUN propriétaire à une vie anonyme', () => {
    // Une trace sans xuid continue d'exister sur la carte, mais elle n'ajoute de ligne à
    // personne — c'est la règle qui a fait supprimer le vote.
    const d = doc({ tracks: [track(512, undefined, 0, 50), track(513, 'A', 0, 60)] })
    const players = buildPlayers(d, [])
    expect(players).toHaveLength(1)
    expect(players[0].xuid).toBe('A')
    expect(players[0].lives).toHaveLength(1)
  })

  it('garde un joueur du film introuvable au scoreboard, sans ligne', () => {
    const d = doc({ roster: [{ xuid: 'Z', filmIndex: 0 }], tracks: [track(512, 'Z', 0, 50)] })
    const players = buildPlayers(d, [row('A', 'Alpha', 'Eagle')])
    expect(players[0].board).toBeUndefined()
  })

  it('retient le gamertag écrit par le FILM', () => {
    const d = doc({ roster: [{ xuid: 'Z', filmIndex: 0, name: 'Zulu' }] })
    expect(buildPlayers(d, [])[0].filmName).toBe('Zulu')
  })

  it('ordonne les vies dans le temps', () => {
    const d = doc({ tracks: [track(514, 'A', 70, 120), track(512, 'A', 0, 50)] })
    const [a] = buildPlayers(d, [])
    expect(a.lives.map((l) => l.startFrame)).toEqual([0, 70])
  })
})

describe('playerName', () => {
  it('préfère la base, qui suit un changement de pseudo', () => {
    const d = doc({ roster: [{ xuid: 'A', filmIndex: 0, name: 'AncienNom' }] })
    const [p] = buildPlayers(d, [row('A', 'NomActuel', 'Eagle')])
    expect(playerName(p)).toBe('NomActuel')
  })

  it('retombe sur le film plutôt que sur « inconnu »', () => {
    const d = doc({ roster: [{ xuid: 'A', filmIndex: 0, name: 'NomDuFilm' }] })
    expect(playerName(buildPlayers(d, [])[0])).toBe('NomDuFilm')
  })

  it('rend null quand aucune source ne nomme le joueur', () => {
    const d = doc({ roster: [{ xuid: 'A', filmIndex: 0 }] })
    expect(playerName(buildPlayers(d, [])[0])).toBeNull()
  })
})

describe('groupByTeam', () => {
  it('range par camp et isole ceux qui n’en ont pas', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(513, 'B', 0, 60), track(514, 'C', 0, 60)] })
    const groups = groupByTeam(buildPlayers(d, [row('A', 'Alpha', 'Eagle'), row('B', 'Bravo', 'Cobra')]))
    expect(groups.map((g) => g.side)).toEqual(['Cobra', 'Eagle', null])
    expect(groups[2].players[0].xuid).toBe('C')
  })
})

describe('colorBySlot — la couleur appartient au JOUEUR, pas à la vie (D1)', () => {
  const teamColor = (ally: boolean) => (ally ? 'allie' : 'adverse')

  it('donne la MÊME couleur aux deux vies d’un même joueur', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(514, 'A', 70, 120)] })
    const colors = colorBySlot(buildPlayers(d, [row('A', 'Alpha', 'Eagle')]), teamColor,
      () => true, 'neutre')
    expect(colors.get(512)).toBe('allie')
    expect(colors.get(514)).toBe('allie')
  })

  it('sépare les camps : allié et adversaire n’ont pas la même teinte', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(513, 'B', 0, 60)] })
    const players = buildPlayers(d, [row('A', 'Alpha', 'Eagle'), row('B', 'Bravo', 'Cobra')])
    const colors = colorBySlot(players, teamColor, (xuid) => xuid === 'A', 'neutre')
    expect(colors.get(512)).toBe('allie')
    expect(colors.get(513)).toBe('adverse')
  })

  it('laisse une vie SANS propriétaire hors de la table — l’appelant y sert son encre neutre', () => {
    const d = doc({ tracks: [track(512, undefined, 0, 50)] })
    const colors = colorBySlot(buildPlayers(d, []), teamColor, () => true, 'neutre')
    expect(colors.has(512)).toBe(false)
    expect(colors.get(512) ?? 'neutre').toBe('neutre')
  })
})

describe('markBySlot / nameBySlot — l’identité redescend sur chaque vie', () => {
  it('porte la marque du propriétaire sur toutes ses vies, rien pour les autres', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(514, 'A', 70, 120), track(513, 'B', 0, 60)] })
    const players = buildPlayers(d, [row('A', 'Alpha', 'Eagle'), row('B', 'Bravo', 'Cobra')])
    const marks = markBySlot(players, new Map([['A', 'me' as const]]))
    expect(marks.get(512)).toBe('me')
    expect(marks.get(514)).toBe('me')
    expect(marks.get(513)).toBeUndefined()
  })

  it('écrit le nom d’affichage du joueur, jamais un xuid brut', () => {
    const d = doc({ tracks: [track(512, '2533274800000000', 0, 50)] })
    const names = nameBySlot(buildPlayers(d, []))
    expect(names.get(512)).toBe('Joueur 0000')
  })

  it('ne nomme pas une vie anonyme', () => {
    const d = doc({ tracks: [track(512, undefined, 0, 50)] })
    expect(nameBySlot(buildPlayers(d, [])).has(512)).toBe(false)
  })
})

describe('playerStateAt', () => {
  const d = doc({ tracks: [track(512, 'A', 0, 50), track(514, 'A', 130, 190)] })
  const [a] = buildPlayers(d, [])

  it('vivant : rend la vie en cours et le bouclier lu', () => {
    const s = playerStateAt(a, 10, BOTH)
    expect(s.alive).toBe(true)
    expect(s.life?.slot).toBe(512)
    expect(s.shield).toEqual({ value: 1, age: 10 })
  })

  it('mort : date la mort et LIT l’image du retour', () => {
    // Le retour est l'image de départ de la vie suivante, jamais une constante ajoutée.
    const s = playerStateAt(a, 90, BOTH)
    expect(s.alive).toBe(false)
    expect(s.sinceDeath).toBe(40)
    expect(s.respawnFrame).toBe(130)
  })

  it('sans vie suivante, le retour reste une LACUNE', () => {
    const solo = buildPlayers(doc({ tracks: [track(512, 'A', 0, 50)] }), [])[0]
    expect(playerStateAt(solo, 90, BOTH).respawnFrame).toBe(-1)
  })

  it('mort : aucun bouclier, jamais un zéro inventé', () => {
    // Zéro voudrait dire « bouclier brisé, mesuré ». Un mort n'a pas de mesure du tout.
    expect(playerStateAt(a, 90, BOTH).shield).toBeNull()
  })
})

describe('playerStateAt — santé', () => {
  // La santé suit le MÊME contrat que le bouclier : report EN AVANT sur TOUTE la vie —
  // le flux est différentiel, non retransmis veut dire inchangé, et les points appartiennent
  // à la vie donc le report ne franchit jamais une mort. Ce qui vieillit s'ESTOMPE à
  // l'affichage (l'âge voyage avec la valeur). AVANT LA PREMIÈRE MESURE, la valeur juste
  // est 1,0 : on apparaît plein (règle du jeu, décision utilisateur 2026-08-12), gardé par
  // la PRÉSENCE du champ dans le document.
  const d = doc({
    tracks: [
      {
        slot: 512,
        team: -1,
        xuid: 'A',
        startFrame: 0,
        endFrame: 100,
        points: [
          { t: 0, x: 0, y: 0 }, // rien de transmis au départ de la vie
          { t: 40, x: 1, y: 1, hp: 0.3 }, // l'unique mesure de la vie
        ],
      },
    ],
  })
  const [a] = buildPlayers(d, [])

  it('reporte la mesure EN AVANT, avec son âge', () => {
    expect(playerStateAt(a, 50, BOTH).health).toEqual({ value: 0.3, age: 10 })
  })

  it('avant la première mesure : PLEIN d’apparition, jamais la mesure future peinte en arrière', () => {
    // La seule mesure de cette vie date de t=40 ; à t=20 la valeur juste est le plein du
    // spawn (1,0, âge 0) — pas 0,3, qui n'existe pas encore, et pas une lacune non plus.
    expect(playerStateAt(a, 20, BOTH).health).toEqual({ value: 1, age: 0 })
  })

  it('document SANS ce champ : rien — on n’invente pas une jauge pour un titre qui ne la transmet pas', () => {
    expect(playerStateAt(a, 20, { shield: true, health: false }).health).toBeNull()
  })

  it('le report tient JUSQU’À LA FIN DE LA VIE, avec son âge — c’est l’estompage qui dit le temps', () => {
    // Non retransmis = inchangé : couper le report à une constante peignait une fausse
    // lacune. La valeur reste, l'âge grandit, et l'affichage l'estompe.
    expect(playerStateAt(a, 80, BOTH).health).toEqual({ value: 0.3, age: 40 })
  })

  it('mort : aucune santé, jamais un zéro inventé', () => {
    expect(playerStateAt(a, 150, BOTH).health).toBeNull()
  })
})

describe('loadoutAt', () => {
  const d = doc({
    loadouts: [
      { t: 10, slot: 512, w: ['0xAAAA'] },
      { t: 200, slot: 512, w: ['0xBBBB'] },
      { t: 10, slot: 513, w: ['0xCCCC'] },
    ],
  })

  it('rend la dernière lecture du SLOT, avec son âge', () => {
    expect(loadoutAt(d, 512, 60)).toEqual({ weapons: ['0xAAAA'], age: 50 })
  })

  it('ne lit jamais le loadout d’un autre slot', () => {
    expect(loadoutAt(d, 513, 60)).toEqual({ weapons: ['0xCCCC'], age: 50 })
  })

  it('avant la première image-clé de la vie : la lecture À VENIR, âge NÉGATIF publié tel quel', () => {
    // Un slot EST une vie : la première lecture à venir du même slot appartient à cette vie
    // — c'est ce qui rend le repli sûr (doctrine du POC, 25,2 % de ses fiches). L'âge
    // négatif n'est jamais déguisé : l'affichage le dit « à venir » et estompe sur |âge|.
    expect(loadoutAt(d, 512, 5)).toEqual({ weapons: ['0xAAAA'], age: -5 })
  })

  it('une lecture PASSÉE prime toujours la lecture à venir', () => {
    // À frame 60 : la lecture passée (t=10) est rendue, pas la suivante (t=200).
    expect(loadoutAt(d, 512, 60)?.age).toBe(50)
  })

  it('le repli à venir ne lit JAMAIS le slot d’un autre', () => {
    const solo = doc({ loadouts: [{ t: 10, slot: 513, w: ['0xCCCC'] }] })
    expect(loadoutAt(solo, 512, 5)).toBeNull()
  })

  it('sans loadouts, rend null plutôt qu’un inventaire vide', () => {
    expect(loadoutAt(doc(), 512, 60)).toBeNull()
  })
})

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

describe('vitalityPresence — la garde multi-titre du plein d’apparition', () => {
  it('détecte chaque champ séparément, sur l’ensemble du document', () => {
    const d = doc({
      tracks: [
        track(512, 'A', 0, 50), // la fixture porte sh, jamais hp
        {
          slot: 513,
          team: -1,
          xuid: 'B',
          startFrame: 0,
          endFrame: 50,
          points: [{ t: 10, x: 0, y: 0, hp: 0.5 }],
        },
      ],
    })
    expect(vitalityPresence(d)).toEqual({ shield: true, health: true })
  })

  it('document sans AUCUNE vitalité (titre sans décodage film) : rien n’est présent', () => {
    const nu = doc({
      tracks: [
        { slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 50, points: [{ t: 0, x: 0, y: 0 }] },
      ],
    })
    expect(vitalityPresence(nu)).toEqual({ shield: false, health: false })
  })
})
