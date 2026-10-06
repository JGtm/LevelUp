/**
 * Tests — endMatchSound (ce qui sonne quand le match se termine).
 *
 * CE QU'ILS PROTÈGENT. Une erreur ici ne casse rien à l'exécution : elle annonce « Victoire »
 * à quelqu'un qui vient de perdre, ou parle anglais à une interface en français — en plein
 * écran de fin, sur le seul son que l'utilisateur écoutera jusqu'au bout. Les quatre confusions
 * plausibles sont couvertes : l'issue (la victoire de l'ADVERSAIRE n'est pas la mienne), la
 * langue, le cas sans équipes (FFA), et le tirage entre prises.
 *
 * LE TIRAGE EST INJECTÉ, jamais moqué : `rand` est un paramètre, les cas ci-dessous lui
 * donnent 0, 0,5 ou 0,999 et lisent la prise. Un `vi.spyOn(Math, 'random')` testerait le
 * double, pas la règle.
 */
import { describe, expect, it } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import {
  END_FFA_WIN_VOICE_STEMS,
  END_MUSIC_STEMS,
  END_VOICE_STEMS,
  endMatchSoundSpec,
  endMatchSoundStems,
  endMatchSounds,
} from './endMatchSound'

/**
 * Les équipes du FILM, par joueur (patron victoryLogic.test) : la lecture de fin ne regarde que
 * les camps et l'équipe du joueur de la page (`moi`). `reference` est celle que l'allégeance
 * porte (le point de vue de la page Rejeu) : la fin sonore ne doit pas la lire.
 */
function film(equipes: Array<[string, number | undefined]>, reference: string | null = null) {
  return buildFilmAllegiance(equipes.map(([xuid, team]) => ({ xuid, team, lives: [] }) as ReplayPlayer), reference)
}

/** Deux camps, le joueur de la page (`moi`) au camp 0 — le cas de référence de l'écran de fin. */
const DEUX_CAMPS = film([['moi', 0], ['pote', 0], ['eux-1', 1], ['eux-2', 1]])
/** Un FFA : le film ne donne aucune équipe (`-1`). */
const FFA = film([['moi', -1], ['a', -1], ['b', -1]])
/** La feuille : c'est sa ligne `is_me` (`moi`), pas sa première ligne, qui désigne la page. */
const FEUILLE = [
  { xuid: 'eux-1', is_me: false },
  { xuid: 'moi', is_me: true },
]

/** Le tirage figé sur la PREMIÈRE prise. */
const premiere = () => 0
/** Le tirage figé sur la DERNIÈRE prise, borne comprise (rand === 1 est hors contrat). */
const derniere = () => 0.999

describe('endMatchSounds — voix ET fanfare, selon l’issue', () => {
  it('victoire en français : la réplique de victoire et sa fanfare partent ensemble', () => {
    expect(endMatchSounds('win', false, 'fr', premiere)).toEqual([
      'end_victory_voice_fr_01',
      'end_victory_music_01',
    ])
  })

  it('défaite : c’est la réplique ET la musique de défaite, jamais celles de la victoire', () => {
    const joues = endMatchSounds('loss', false, 'fr', premiere)
    expect(joues).toEqual(['end_defeat_voice_fr_01', 'end_defeat_music_01'])
    expect(joues.join(' ')).not.toContain('victory')
  })

  it('égalité : sa propre réplique et sa propre fanfare', () => {
    expect(endMatchSounds('tie', false, 'fr', premiere)).toEqual([
      'end_tie_voice_fr_01',
      'end_tie_music_01',
    ])
  })
})

describe('endMatchSounds — la langue est celle de l’interface', () => {
  it('en anglais, la voix change ; la fanfare, elle, ne parle pas', () => {
    expect(endMatchSounds('win', false, 'en', premiere)).toEqual([
      'end_victory_voice_en_01',
      'end_victory_music_01',
    ])
    expect(endMatchSounds('loss', false, 'en', premiere)).toEqual([
      'end_defeat_voice_en_01',
      'end_defeat_music_01',
    ])
  })

  it('chaque issue a une voix dans LES DEUX langues (aucun pack troué)', () => {
    for (const issue of ['win', 'loss', 'tie'] as const) {
      for (const langue of ['fr', 'en'] as const) {
        expect(END_VOICE_STEMS[issue][langue].length, `${issue}/${langue}`).toBeGreaterThan(0)
      }
    }
  })
})

describe('endMatchSounds — le tirage entre prises', () => {
  it('deux prises françaises pour la victoire : le tirage les atteint toutes les deux', () => {
    expect(endMatchSounds('win', false, 'fr', premiere)[0]).toBe('end_victory_voice_fr_01')
    expect(endMatchSounds('win', false, 'fr', derniere)[0]).toBe('end_victory_voice_fr_02')
  })

  it('une seule prise : le tirage rend toujours la même, quelle que soit la valeur', () => {
    expect(endMatchSounds('win', false, 'en', premiere)[0]).toBe('end_victory_voice_en_01')
    expect(endMatchSounds('win', false, 'en', derniere)[0]).toBe('end_victory_voice_en_01')
  })

  it('un tirage à 1 (hors contrat de Math.random) ne sort JAMAIS de la liste', () => {
    for (const issue of ['win', 'loss', 'tie'] as const) {
      const voix = endMatchSounds(issue, false, 'fr', () => 1)[0]
      expect(END_VOICE_STEMS[issue].fr, `${issue}`).toContain(voix)
    }
  })
})

describe('endMatchSounds — le FFA n’a pas d’écran, mais il a une voix', () => {
  it('FFA gagné en français : « Vainqueur », avec la fanfare de victoire', () => {
    expect(endMatchSounds('win', true, 'fr', premiere)).toEqual([
      'end_winner_voice_fr_01',
      'end_victory_music_01',
    ])
  })

  it('FFA gagné en anglais : repli documenté sur « Victory » (pas de « Winner » isolé)', () => {
    expect(endMatchSounds('win', true, 'en', premiere)).toEqual([
      'end_victory_voice_en_01',
      'end_victory_music_01',
    ])
    expect(END_FFA_WIN_VOICE_STEMS.en).toEqual(['end_victory_voice_en_01'])
  })

  it('FFA perdu ou à égalité : rien — aucune réplique ne dit cela sans nommer d’équipe', () => {
    expect(endMatchSounds('loss', true, 'fr', premiere)).toEqual([])
    expect(endMatchSounds('tie', true, 'fr', premiere)).toEqual([])
    expect(endMatchSounds('loss', true, 'en', premiere)).toEqual([])
  })
})

describe('endMatchSoundSpec — la même lecture que l’écran de fin', () => {
  it('deux camps : l’issue du joueur de la page, sans drapeau FFA', () => {
    expect(endMatchSoundSpec(DEUX_CAMPS, FEUILLE, 3, 'fr')).toEqual({
      outcome: 'loss',
      ffa: false,
      locale: 'fr',
    })
  })

  it('ANCRÉE SUR LA PAGE : une allégeance vue d’un adversaire ne retourne pas l’issue annoncée', () => {
    // La page (`moi`, camp 0) a gagné ; le rejeu est regardé depuis `eux-1` (camp 1), et c'est
    // cette référence que porte l'allégeance. L'écran de fin suivrait ce point de vue ; la voix,
    // elle, annonce l'issue de la PAGE — jamais « Défaite » sur un match qu'elle a gagné.
    const vuDeLAdversaire = film([['moi', 0], ['pote', 0], ['eux-1', 1], ['eux-2', 1]], 'eux-1')
    expect(endMatchSoundSpec(vuDeLAdversaire, FEUILLE, 2, 'fr')).toEqual({
      outcome: 'win',
      ffa: false,
      locale: 'fr',
    })
  })

  it('égalité à deux camps : elle sonne, contrairement à l’égalité sans camps', () => {
    expect(endMatchSoundSpec(DEUX_CAMPS, FEUILLE, 1, 'en')).toEqual({
      outcome: 'tie',
      ffa: false,
      locale: 'en',
    })
    expect(endMatchSoundSpec(FFA, FEUILLE, 1, 'en')).toBeNull()
  })

  it('FFA gagné : le drapeau passe à true, et c’est lui qui choisit « Vainqueur »', () => {
    const spec = endMatchSoundSpec(FFA, FEUILLE, 2, 'fr')
    expect(spec).toEqual({ outcome: 'win', ffa: true, locale: 'fr' })
    expect(endMatchSounds(spec!.outcome, spec!.ffa, spec!.locale, premiere)[0]).toBe(
      'end_winner_voice_fr_01',
    )
  })

  it('FFA perdu : rien à annoncer', () => {
    expect(endMatchSoundSpec(FFA, FEUILLE, 3, 'fr')).toBeNull()
  })

  it('abandon (code 4), code absent, en-tête pas encore chargé : rien', () => {
    expect(endMatchSoundSpec(DEUX_CAMPS, FEUILLE, 4, 'fr')).toBeNull()
    expect(endMatchSoundSpec(DEUX_CAMPS, FEUILLE, undefined, 'fr')).toBeNull()
    expect(endMatchSoundSpec(film([]), [], undefined, 'fr')).toBeNull()
  })

  it('joueur de la page que le film ne situe pas : la victoire reste annonçable, sans équipe nommée', () => {
    // `readVictory` refuse (elle ne saurait pas quel camp habille l'écran) ; la VOIX, elle,
    // ne nomme personne — « Vainqueur » dit vrai sans rien supposer.
    expect(endMatchSoundSpec(film([['a', 0], ['b', 1]]), FEUILLE, 2, 'fr')).toEqual({
      outcome: 'win',
      ffa: true,
      locale: 'fr',
    })
  })
})

describe('endMatchSoundStems — ce que le lecteur doit précharger', () => {
  it('TOUTES les prises de l’issue, pas seulement celle qui sera tirée', () => {
    expect(endMatchSoundStems({ outcome: 'win', ffa: false, locale: 'fr' })).toEqual([
      'end_victory_voice_fr_01',
      'end_victory_voice_fr_02',
      'end_victory_music_01',
    ])
  })

  it('couvre exactement ce que le tirage peut rendre, pour chaque issue et chaque langue', () => {
    for (const issue of ['win', 'loss', 'tie'] as const) {
      for (const langue of ['fr', 'en'] as const) {
        for (const ffa of [false, true]) {
          const spec = { outcome: issue, ffa, locale: langue }
          const precharges = endMatchSoundStems(spec)
          for (const r of [0, 0.5, 0.999]) {
            for (const joue of endMatchSounds(issue, ffa, langue, () => r)) {
              expect(precharges, `${issue}/${langue}/ffa=${ffa}`).toContain(joue)
            }
          }
        }
      }
    }
  })

  it('rien à conclure : rien à charger', () => {
    expect(endMatchSoundStems(null)).toEqual([])
    expect(endMatchSoundStems({ outcome: 'loss', ffa: true, locale: 'fr' })).toEqual([])
  })

  it('la fanfare du FFA gagné est celle de la victoire (un seul fichier, pas deux)', () => {
    expect(endMatchSoundStems({ outcome: 'win', ffa: true, locale: 'fr' })).toContain(
      END_MUSIC_STEMS.win,
    )
  })
})
