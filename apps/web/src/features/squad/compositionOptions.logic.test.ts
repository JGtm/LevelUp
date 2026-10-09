/**
 * compositionOptions.logic — la liste du sélecteur de composition hors de la page Escouade.
 *
 * Ce que ces tests cadenassent : amis déclarés d'abord (et seuls quand il y en a), jamais un bot,
 * jamais le joueur consulté, repli sur les coéquipiers croisés sans ami déclaré, et un annuaire qui
 * traduit les profils, escouades et groupes en xuid.
 */
import { describe, expect, it } from 'vitest'

import type { EncounterDTO } from '@/lib/api/types'

import {
  annuaireDeComposition,
  coequipiersProposes,
  estUnBot,
  type SourcesDeComposition,
} from './compositionOptions.logic'

function rencontre(gamertag: string, xuid: string, asTeammate: number, asEnemy = 0): EncounterDTO {
  return {
    gamertag,
    xuid,
    match_count: asTeammate + asEnemy,
    as_teammate: asTeammate,
    as_enemy: asEnemy,
    avg_kda: null,
  }
}

const COEQUIPIERS: EncounterDTO[] = [
  rencontre('343 Meowlnir', 'bid(56.0)', 80, 20),
  rencontre('Inconnu', 'x-inconnu', 60),
  rencontre('Ami Rare', 'x-rare', 3),
  rencontre('Ami Frequent', 'x-frequent', 40),
  rencontre('JGtm', 'x-jgtm', 99),
]
const ADVERSAIRES: EncounterDTO[] = [rencontre('Rival', 'x-rival', 1, 30)]

function sources(partiel: Partial<SourcesDeComposition> = {}): SourcesDeComposition {
  return {
    coequipiers: COEQUIPIERS,
    adversaires: ADVERSAIRES,
    amis: [],
    identifies: [],
    joueur: { gamertag: 'JGtm', xuid: 'x-jgtm' },
    ...partiel,
  }
}

const proposes = (s: SourcesDeComposition) =>
  coequipiersProposes(s, annuaireDeComposition(s)).map((o) => o.gamertag)

describe('estUnBot', () => {
  it('reconnaît le xuid d’un bot, et lui seul', () => {
    expect(estUnBot('bid(3.0)')).toBe(true)
    expect(estUnBot('2533274823110022')).toBe(false)
  })
})

describe('coequipiersProposes', () => {
  it('AMIS DÉCLARÉS : eux seuls, les plus joués ensemble d’abord', () => {
    expect(proposes(sources({ amis: ['Ami Rare', 'Ami Frequent'] }))).toEqual(['Ami Frequent', 'Ami Rare'])
  })

  it('un ami connu par son seul profil suivi est proposé, sans compte de matchs', () => {
    const s = sources({
      amis: ['Ami Frequent', 'Profil'],
      identifies: [{ gamertag: 'Profil', xuid: 'x-profil' }],
    })
    const liste = coequipiersProposes(s, annuaireDeComposition(s))
    expect(liste.map((o) => [o.gamertag, o.encounter_count])).toEqual([
      ['Ami Frequent', 40],
      ['Profil', 0],
    ])
  })

  it('un ami qu’aucune source ne traduit en xuid n’est pas proposé', () => {
    expect(proposes(sources({ amis: ['Fantome', 'Ami Rare'] }))).toEqual(['Ami Rare'])
  })

  it('un ami déclaré deux fois (casse) n’apparaît qu’une fois', () => {
    expect(proposes(sources({ amis: ['Ami Rare', 'ami rare'] }))).toEqual(['Ami Rare'])
  })

  it('SANS AMI DÉCLARÉ : les coéquipiers croisés, bots, adversaires et joueur exclus', () => {
    expect(proposes(sources())).toEqual(['Inconnu', 'Ami Frequent', 'Ami Rare'])
  })

  it('JAMAIS un bot, même déclaré ami ou membre d’une escouade', () => {
    const s = sources({
      amis: ['343 Meowlnir', 'Ami Rare'],
      identifies: [{ gamertag: '343 Meowlnir', xuid: 'bid(56.0)' }],
    })
    expect(proposes(s)).toEqual(['Ami Rare'])
  })

  it('jamais le joueur consulté, par xuid comme par gamertag', () => {
    const s = sources({ amis: ['JGtm', 'Ami Rare'], identifies: [{ gamertag: 'jgtm', xuid: 'x-autre' }] })
    expect(proposes(s)).toEqual(['Ami Rare'])
  })
})

describe('annuaireDeComposition', () => {
  it('traduit les profils, escouades et groupes ; garde le compte des rencontres', () => {
    const annuaire = annuaireDeComposition(
      sources({
        identifies: [
          { gamertag: 'Profil', xuid: 'x-profil' },
          { gamertag: 'Ami Frequent', xuid: 'x-frequent' },
          { gamertag: undefined, xuid: 'x-sans-nom' },
        ],
      }),
    )
    const parNom = new Map(annuaire.map((o) => [o.gamertag, o]))
    expect(parNom.get('Profil')).toEqual({ gamertag: 'Profil', xuid: 'x-profil', encounter_count: 0 })
    expect(parNom.get('Ami Frequent')?.encounter_count).toBe(40)
    expect(parNom.get('Rival')?.xuid).toBe('x-rival')
    expect(parNom.has('343 Meowlnir')).toBe(false)
    expect(parNom.has('JGtm')).toBe(false)
    expect(annuaire.some((o) => o.xuid === 'x-sans-nom')).toBe(false)
  })
})
