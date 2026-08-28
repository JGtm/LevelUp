import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import type { ReplayTrackReady } from './replayNormalize'
import { testReplayDoc as doc } from './test/testDoc'
import {
  buildPlayers,
  buildSlotOwnership,
  colorResolver,
  colorResolverOrLast,
  groupByTeam,
  loadoutAt,
  markResolver,
  nameResolver,
  playerName,
  playerStateAt,
  sideResolver,
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

describe('colorResolver — la couleur appartient au JOUEUR à l’image, pas à la vie (D1)', () => {
  const teamColor = (ally: boolean) => (ally ? 'allie' : 'adverse')
  const own = (players: ReturnType<typeof buildPlayers>) => buildSlotOwnership(players)

  it('donne la MÊME couleur aux deux vies d’un même joueur, à toute image de la vie', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(514, 'A', 70, 120)] })
    const color = colorResolver(own(buildPlayers(d, [row('A', 'Alpha', 'Eagle')])), teamColor,
      () => true, 'neutre')
    expect(color(512, 25)).toBe('allie')
    expect(color(514, 90)).toBe('allie')
  })

  it('sépare les camps : allié et adversaire n’ont pas la même teinte', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(513, 'B', 0, 60)] })
    const players = buildPlayers(d, [row('A', 'Alpha', 'Eagle'), row('B', 'Bravo', 'Cobra')])
    const color = colorResolver(own(players), teamColor, (xuid) => xuid === 'A', 'neutre')
    expect(color(512, 25)).toBe('allie')
    expect(color(513, 25)).toBe('adverse')
  })

  it('une vie SANS propriétaire ne se colore pas — l’appelant y sert son encre neutre', () => {
    const d = doc({ tracks: [track(512, undefined, 0, 50)] })
    const color = colorResolver(own(buildPlayers(d, [])), teamColor, () => true, 'neutre')
    expect(color(512, 25)).toBeNull()
    expect(color(512, 25) ?? 'neutre').toBe('neutre')
  })

  it('un slot LIBRE à cette image (hors de toute vie) n’a pas de couleur', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50)] })
    const color = colorResolver(own(buildPlayers(d, [row('A', 'Alpha', 'Eagle')])), teamColor,
      () => true, 'neutre')
    expect(color(512, 25)).toBe('allie') // dans la vie
    expect(color(512, 999)).toBeNull() // après la vie
  })
})

describe('markResolver / nameResolver — l’identité résolue par slot ET par image', () => {
  const own = (players: ReturnType<typeof buildPlayers>) => buildSlotOwnership(players)

  it('porte la marque du propriétaire de la vie courante, rien pour les autres', () => {
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(514, 'A', 70, 120), track(513, 'B', 0, 60)] })
    const players = buildPlayers(d, [row('A', 'Alpha', 'Eagle'), row('B', 'Bravo', 'Cobra')])
    const mark = markResolver(own(players), new Map([['A', 'me' as const]]))
    expect(mark(512, 25)).toBe('me')
    expect(mark(514, 90)).toBe('me')
    expect(mark(513, 25)).toBeUndefined()
  })

  it('écrit le nom d’affichage du joueur, jamais un xuid brut', () => {
    const d = doc({ tracks: [track(512, '2533274800000000', 0, 50)] })
    const name = nameResolver(own(buildPlayers(d, [])))
    expect(name(512, 25)).toBe('Joueur 0000')
  })

  it('ne nomme pas une vie anonyme', () => {
    const d = doc({ tracks: [track(512, undefined, 0, 50)] })
    expect(nameResolver(own(buildPlayers(d, [])))(512, 25)).toBeNull()
  })
})

describe('buildSlotOwnership — le propriétaire d’un slot À UNE IMAGE (multi-manche)', () => {
  const teamColor = (ally: boolean) => (ally ? 'allie' : 'adverse')

  it('mono-manche : un slot à plusieurs vies du MÊME joueur rend ce joueur à toute image de ses vies', () => {
    // NEUTRALITÉ MONO-MANCHE : dans une manche, toutes les vies d’un slot sont au même joueur,
    // donc frame-aware == l’ancien « dernier gagnant » — le résultat est identique à toute image.
    const d = doc({ tracks: [track(512, 'A', 0, 50), track(512, 'A', 130, 190)] })
    const own = buildSlotOwnership(buildPlayers(d, [row('A', 'Alpha', 'Eagle')]))
    expect(own.ownerAtFrame(512, 10)?.xuid).toBe('A') // 1re vie
    expect(own.ownerAtFrame(512, 160)?.xuid).toBe('A') // 2de vie
    expect(own.ownerAtFrame(512, 80)).toBeNull() // entre deux vies : slot libre
  })

  it('multi-manche : un slot RÉATTRIBUÉ montre le joueur de LA MANCHE COURANTE (contre-épreuve)', () => {
    // Le bug rapporté « deux DinoR00 et pas de SHROOM » : slot 512 = SHROOM en manche 0 puis
    // DinoR00 en manche 2 ; slot 513 = DinoR00 partout. L’ancienne Map effondrée (DERNIER
    // gagnant) attribuait 512 à DinoR00 pour TOUT le match → deux DinoR00, SHROOM jamais montré.
    const d = doc({
      roster: [
        { xuid: 'S', filmIndex: 0 },
        { xuid: 'D', filmIndex: 1 },
      ],
      tracks: [
        track(512, 'S', 0, 50), // manche 0 : SHROOM
        track(512, 'D', 200, 250), // manche 2 : le slot revient à DinoR00
        track(513, 'D', 0, 250), // DinoR00 tient 513 tout du long
      ],
    })
    const players = buildPlayers(d, [row('S', 'SHROOM', 'Eagle'), row('D', 'DinoR00', 'Eagle')])
    const own = buildSlotOwnership(players)
    const name = nameResolver(own)
    // À 25 (manche 0) : 512 = SHROOM, 513 = DinoR00 → DEUX noms distincts, SHROOM PRÉSENT.
    expect(name(512, 25)).toBe('SHROOM')
    expect(name(513, 25)).toBe('DinoR00')
    // CONTRE-ÉPREUVE : le slot 512 change de propriétaire selon l’image — impossible avec une
    // Map figée, qui rendait le même nom aux deux images.
    expect(name(512, 220)).toBe('DinoR00')
    expect(name(512, 25)).not.toBe(name(512, 220))
    // L’ancien comportement (dernier gagnant) aurait rendu DinoR00 aux DEUX images ; la couleur
    // et le camp suivent la même règle par image.
    const color = colorResolver(own, teamColor, () => true, 'neutre')
    expect(color(512, 25)).toBe('allie')
    const side = sideResolver(own)
    expect(side(512, 25)).toBe('Eagle')
    expect(side(512, 80)).toBeNull() // manche 1 : le slot est libre entre les deux vies
  })

  it('vies triées : l’ordre d’insertion ne change pas la résolution', () => {
    // Les tracks arrivent dans le désordre ; l’index les trie par début.
    const d = doc({
      roster: [{ xuid: 'S', filmIndex: 0 }, { xuid: 'D', filmIndex: 1 }],
      tracks: [track(512, 'D', 200, 250), track(512, 'S', 0, 50)],
    })
    const own = buildSlotOwnership(buildPlayers(d, []))
    expect(own.ownerAtFrame(512, 25)?.xuid).toBe('S')
    expect(own.ownerAtFrame(512, 220)?.xuid).toBe('D')
  })
})

describe('ownerAtFrameOrLast — la FRONTIÈRE : vie couvrante, sinon la vie juste précédente', () => {
  it('image couverte par une vie : rend son propriétaire (comme ownerAtFrame)', () => {
    const d = doc({ tracks: [track(512, 'A', 10, 50)] })
    const own = buildSlotOwnership(buildPlayers(d, []))
    expect(own.ownerAtFrameOrLast(512, 30)?.xuid).toBe('A')
  })

  it('image = finVie + 1 (l’objet lâché à la mort) : rend le propriétaire de la vie qui vient de finir', () => {
    // C’est le cœur de la régression : un objet `dropped` porte t0 = finVie + 1. La résolution
    // STRICTE y rend null (encre neutre) ; la frontière rend le lâcheur.
    const d = doc({ tracks: [track(512, 'A', 10, 50)] })
    const own = buildSlotOwnership(buildPlayers(d, []))
    expect(own.ownerAtFrame(512, 51)).toBeNull() // strict : trou
    expect(own.ownerAtFrameOrLast(512, 51)?.xuid).toBe('A') // frontière : le lâcheur
  })

  it('trou entre deux vies d’un même slot (multi-manche) : la vie PRÉCÉDENTE, jamais la suivante', () => {
    // Slot 512 : A en manche 0 [0,50], B en manche 2 [200,250]. Dans le trou (frame 100),
    // la frontière rend A (le lâcheur d’alors), PAS B (la vie à venir) — donc PAS le
    // dernier-gagnant du match : « deux DinoR00 » n’est pas réintroduit.
    const d = doc({
      roster: [{ xuid: 'A', filmIndex: 0 }, { xuid: 'B', filmIndex: 1 }],
      tracks: [track(512, 'A', 0, 50), track(512, 'B', 200, 250)],
    })
    const own = buildSlotOwnership(buildPlayers(d, []))
    expect(own.ownerAtFrameOrLast(512, 100)?.xuid).toBe('A') // vie précédente
    expect(own.ownerAtFrameOrLast(512, 220)?.xuid).toBe('B') // vie couvrante (manche 2)
    expect(own.ownerAtFrameOrLast(512, 300)?.xuid).toBe('B') // après B : B est la dernière finie
  })

  it('avant la première vie du slot : aucune vie précédente → null', () => {
    const d = doc({ tracks: [track(512, 'A', 10, 50)] })
    const own = buildSlotOwnership(buildPlayers(d, []))
    expect(own.ownerAtFrameOrLast(512, 5)).toBeNull()
  })

  it('slot sans aucune vie : null', () => {
    const own = buildSlotOwnership(buildPlayers(doc({ tracks: [track(512, 'A', 10, 50)] }), []))
    expect(own.ownerAtFrameOrLast(999, 30)).toBeNull()
  })

  it('colorResolverOrLast : un objet lâché à finVie+1 prend la couleur d’équipe du lâcheur, pas le neutre', () => {
    const teamColor = (ally: boolean) => (ally ? 'allie' : 'adverse')
    const d = doc({ tracks: [track(512, 'A', 10, 50)] })
    const own = buildSlotOwnership(buildPlayers(d, [row('A', 'Alpha', 'Eagle')]))
    const strict = colorResolver(own, teamColor, () => true, 'neutre')
    const orLast = colorResolverOrLast(own, teamColor, () => true, 'neutre')
    expect(strict(512, 51)).toBeNull() // strict : trou → l’appelant tomberait sur le neutre
    expect(orLast(512, 51)).toBe('allie') // frontière : la couleur du lâcheur
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
