import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import {
  normalizeReplayDocument,
  type ReplayDocumentReady,
  type ReplayInventoryReady,
  type ReplayTrackReady,
} from './replayNormalize'
import {
  buildPlayers,
  grenadesCarried,
  groupByTeam,
  inventoryAt,
  loadoutAt,
  playerName,
  playerStateAt,
  selectedGrenadeRank,
} from './rosterLogic'

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

// Comme en production, le document de test passe par la frontière : ce qu'on décrit ici est
// le document de transport, ce que les fonctions reçoivent est sa forme normalisée.
function doc(over: Partial<ReplayDocument> = {}): ReplayDocumentReady {
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

  it('donne une couleur STABLE par joueur, pas par vie', () => {
    const d = doc({
      roster: [
        { xuid: 'A', filmIndex: 0 },
        { xuid: 'B', filmIndex: 1 },
      ],
      tracks: [track(512, 'A', 0, 50), track(514, 'A', 70, 120), track(513, 'B', 0, 60)],
    })
    const players = buildPlayers(d, [])
    expect(players.map((p) => p.colorIndex)).toEqual([0, 1])
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

describe('playerStateAt', () => {
  const d = doc({ tracks: [track(512, 'A', 0, 50), track(514, 'A', 130, 190)] })
  const [a] = buildPlayers(d, [])

  it('vivant : rend la vie en cours et le bouclier lu', () => {
    const s = playerStateAt(a, 10, 100)
    expect(s.alive).toBe(true)
    expect(s.life?.slot).toBe(512)
    expect(s.shield).toEqual({ value: 1, age: 10 })
  })

  it('mort : date la mort et LIT l’image du retour', () => {
    // Le retour est l'image de départ de la vie suivante, jamais une constante ajoutée.
    const s = playerStateAt(a, 90, 100)
    expect(s.alive).toBe(false)
    expect(s.sinceDeath).toBe(40)
    expect(s.respawnFrame).toBe(130)
  })

  it('sans vie suivante, le retour reste une LACUNE', () => {
    const solo = buildPlayers(doc({ tracks: [track(512, 'A', 0, 50)] }), [])[0]
    expect(playerStateAt(solo, 90, 100).respawnFrame).toBe(-1)
  })

  it('mort : aucun bouclier, jamais un zéro inventé', () => {
    // Zéro voudrait dire « bouclier brisé, mesuré ». Un mort n'a pas de mesure du tout.
    expect(playerStateAt(a, 90, 100).shield).toBeNull()
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

  it('ne lit pas dans le futur', () => {
    expect(loadoutAt(d, 512, 5)).toBeNull()
  })

  it('sans loadouts, rend null plutôt qu’un inventaire vide', () => {
    expect(loadoutAt(doc(), 512, 60)).toBeNull()
  })
})

describe('inventoryAt', () => {
  const d = doc({
    inventory: [
      { t: 10, slot: 512, g: [0, 2, 0, 0], a: 4, d: 0 },
      { t: 200, slot: 512, g: [1, 0, 0, 0] },
      { t: 10, slot: 513, a: 9 },
    ],
  })

  it('rend la dernière lecture du SLOT, avec son âge', () => {
    const r = inventoryAt(d, 512, 60)
    expect(r?.age).toBe(50)
    expect(r?.state.a).toBe(4)
  })

  it('ne lit jamais l’inventaire d’un autre slot ni dans le futur', () => {
    expect(inventoryAt(d, 513, 60)?.state.a).toBe(9)
    expect(inventoryAt(d, 512, 5)).toBeNull()
  })

  it('sans inventaire, rend null', () => {
    expect(inventoryAt(doc(), 512, 60)).toBeNull()
  })
})

describe('grenadesCarried', () => {
  const labels = ['Fragmentation', 'Plasma', 'Dynamo', 'Spike']

  it('n’affiche que les types réellement portés', () => {
    // Le tableau publié est complet : un zéro y dit « ce type, aucune en réserve ». Montrer
    // quatre types dont trois à zéro noierait celui qui compte.
    const got = grenadesCarried(inv({ g: [0, 2, 0, 0] }), labels)
    expect(got).toEqual([{ rank: 1, name: 'Plasma', count: 2 }])
  })

  it('sans compteurs lus, ne rend rien — jamais quatre types à zéro', () => {
    expect(grenadesCarried(inv(), labels)).toEqual([])
  })

  it('sans table de noms, garde le rang plutôt qu’un nom inventé', () => {
    expect(grenadesCarried(inv({ g: [3, 0, 0, 0] }), undefined)[0].name).toBe('rang 0')
  })
})

describe('selectedGrenadeRank', () => {
  it('désigne le type quand il ne peut pas être un autre', () => {
    expect(selectedGrenadeRank(inv({ g: [0, 0, 2, 0] }))).toBe(2)
  })

  it('ne désigne RIEN quand deux types sont portés', () => {
    // Le sélecteur de type n'est pas dans ce qu'on sait lire : deviner afficherait une
    // certitude qu'on n'a pas.
    expect(selectedGrenadeRank(inv({ g: [1, 2, 0, 0] }))).toBeNull()
  })

  it('ne désigne rien sans lecture', () => {
    expect(selectedGrenadeRank(inv())).toBeNull()
  })
})
